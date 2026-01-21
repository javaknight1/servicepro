import React, {
  createContext,
  useContext,
  useReducer,
  useCallback,
  useEffect,
  useRef,
  ReactNode,
} from 'react';
import { KPIData, KPIState, KPIWebSocketMessage } from '../types/kpi';

// Action types
type KPIAction =
  | { type: 'SET_KPIS'; payload: KPIData[] }
  | { type: 'UPDATE_KPI'; payload: { id: string; data: Partial<KPIData> } }
  | { type: 'SET_LOADING'; payload: boolean }
  | { type: 'SET_ERROR'; payload: string | null }
  | { type: 'SET_CONNECTED'; payload: boolean }
  | { type: 'SET_LAST_UPDATED'; payload: string };

// Initial state
const initialState: KPIState = {
  kpis: {},
  loading: false,
  error: null,
  lastUpdated: null,
  isConnected: false,
};

// Reducer
function kpiReducer(state: KPIState, action: KPIAction): KPIState {
  switch (action.type) {
    case 'SET_KPIS':
      return {
        ...state,
        kpis: action.payload.reduce(
          (acc, kpi) => ({ ...acc, [kpi.id]: kpi }),
          {}
        ),
        loading: false,
        error: null,
        lastUpdated: new Date().toISOString(),
      };

    case 'UPDATE_KPI':
      if (!state.kpis[action.payload.id]) {
        return state;
      }
      return {
        ...state,
        kpis: {
          ...state.kpis,
          [action.payload.id]: {
            ...state.kpis[action.payload.id],
            ...action.payload.data,
            lastUpdated: new Date().toISOString(),
          },
        },
        lastUpdated: new Date().toISOString(),
      };

    case 'SET_LOADING':
      return { ...state, loading: action.payload };

    case 'SET_ERROR':
      return { ...state, error: action.payload, loading: false };

    case 'SET_CONNECTED':
      return { ...state, isConnected: action.payload };

    case 'SET_LAST_UPDATED':
      return { ...state, lastUpdated: action.payload };

    default:
      return state;
  }
}

// Context type
interface KPIContextType {
  state: KPIState;
  setKPIs: (kpis: KPIData[]) => void;
  updateKPI: (id: string, data: Partial<KPIData>) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  setConnected: (connected: boolean) => void;
  getKPI: (id: string) => KPIData | undefined;
  getKPIsByCategory: (category: string) => KPIData[];
  getAllKPIs: () => KPIData[];
}

// Create context
const KPIContext = createContext<KPIContextType | undefined>(undefined);

// Provider props
interface KPIProviderProps {
  children: ReactNode;
  /** WebSocket URL for real-time updates */
  wsUrl?: string;
  /** Enable WebSocket connection */
  enableWebSocket?: boolean;
  /** Auto-reconnect on disconnect */
  autoReconnect?: boolean;
  /** Reconnect interval in ms */
  reconnectInterval?: number;
}

// Provider component
export const KPIProvider: React.FC<KPIProviderProps> = ({
  children,
  wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${
    window.location.host
  }/api/v1/ws/kpi`,
  enableWebSocket = true,
  autoReconnect = true,
  reconnectInterval = 5000,
}) => {
  const [state, dispatch] = useReducer(kpiReducer, initialState);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const reconnectAttemptsRef = useRef(0);
  const maxReconnectAttempts = 10;

  // Actions
  const setKPIs = useCallback((kpis: KPIData[]) => {
    dispatch({ type: 'SET_KPIS', payload: kpis });
  }, []);

  const updateKPI = useCallback((id: string, data: Partial<KPIData>) => {
    dispatch({ type: 'UPDATE_KPI', payload: { id, data } });
  }, []);

  const setLoading = useCallback((loading: boolean) => {
    dispatch({ type: 'SET_LOADING', payload: loading });
  }, []);

  const setError = useCallback((error: string | null) => {
    dispatch({ type: 'SET_ERROR', payload: error });
  }, []);

  const setConnected = useCallback((connected: boolean) => {
    dispatch({ type: 'SET_CONNECTED', payload: connected });
  }, []);

  // Selectors
  const getKPI = useCallback((id: string) => state.kpis[id], [state.kpis]);

  const getKPIsByCategory = useCallback(
    (category: string) =>
      Object.values(state.kpis).filter((kpi) => kpi.category === category),
    [state.kpis]
  );

  const getAllKPIs = useCallback(() => Object.values(state.kpis), [state.kpis]);

  // WebSocket connection
  const connectWebSocket = useCallback(() => {
    if (!enableWebSocket || wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    try {
      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.log('KPI WebSocket connected');
        setConnected(true);
        reconnectAttemptsRef.current = 0;

        // Subscribe to KPI updates
        ws.send(JSON.stringify({ action: 'subscribe', channel: 'kpi' }));
      };

      ws.onclose = () => {
        console.log('KPI WebSocket disconnected');
        setConnected(false);

        // Auto-reconnect
        if (
          autoReconnect &&
          reconnectAttemptsRef.current < maxReconnectAttempts
        ) {
          reconnectAttemptsRef.current++;
          console.log(
            `Reconnecting KPI WebSocket (attempt ${reconnectAttemptsRef.current})`
          );
          reconnectTimeoutRef.current = setTimeout(
            connectWebSocket,
            reconnectInterval
          );
        }
      };

      ws.onerror = (error) => {
        console.error('KPI WebSocket error:', error);
      };

      ws.onmessage = (event) => {
        try {
          const message: KPIWebSocketMessage = JSON.parse(event.data);

          if (message.type === 'kpi_update') {
            updateKPI(message.kpi_id, {
              value: message.value,
              previousValue: message.previousValue,
              lastUpdated: message.timestamp,
              status: message.status,
            });
          }
        } catch (err) {
          console.error('Failed to parse KPI WebSocket message:', err);
        }
      };

      wsRef.current = ws;
    } catch (err) {
      console.error('Failed to create KPI WebSocket:', err);
    }
  }, [
    enableWebSocket,
    wsUrl,
    autoReconnect,
    reconnectInterval,
    setConnected,
    updateKPI,
  ]);

  // Connect WebSocket on mount
  useEffect(() => {
    if (enableWebSocket) {
      connectWebSocket();
    }

    return () => {
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [enableWebSocket, connectWebSocket]);

  const value: KPIContextType = {
    state,
    setKPIs,
    updateKPI,
    setLoading,
    setError,
    setConnected,
    getKPI,
    getKPIsByCategory,
    getAllKPIs,
  };

  return <KPIContext.Provider value={value}>{children}</KPIContext.Provider>;
};

// Hook to use KPI context
export const useKPIStore = (): KPIContextType => {
  const context = useContext(KPIContext);
  if (!context) {
    throw new Error('useKPIStore must be used within a KPIProvider');
  }
  return context;
};

export default KPIContext;

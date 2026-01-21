import { useEffect, useRef, useState, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';

export interface WebSocketMessage {
  type: string;
  quote_id: string;
  event: string;
  status: string;
  timestamp: string;
  data?: any;
}

export interface WebSocketOptions {
  /** WebSocket URL */
  url?: string;
  /** Auto-reconnect on disconnect */
  autoReconnect?: boolean;
  /** Reconnect delay in milliseconds */
  reconnectDelay?: number;
  /** Maximum reconnect attempts */
  maxReconnectAttempts?: number;
  /** Callback when connection opens */
  onOpen?: () => void;
  /** Callback when connection closes */
  onClose?: () => void;
  /** Callback when error occurs */
  onError?: (error: Event) => void;
  /** Callback when message is received */
  onMessage?: (message: WebSocketMessage) => void;
}

export interface WebSocketState {
  /** WebSocket connection */
  ws: WebSocket | null;
  /** Connection ready state */
  readyState: number;
  /** Is connected */
  isConnected: boolean;
  /** Last error */
  error: Event | null;
  /** Subscribe to quote updates */
  subscribe: (quoteId: string) => void;
  /** Unsubscribe from quote updates */
  unsubscribe: (quoteId: string) => void;
  /** Reconnect */
  reconnect: () => void;
}

const WS_BASE_URL =
  import.meta.env.VITE_WS_URL || 'ws://localhost:8080/api/v1/ws';

/**
 * Hook for WebSocket connection with auto-reconnect
 */
export const useWebSocket = (
  options: WebSocketOptions = {}
): WebSocketState => {
  const {
    url = WS_BASE_URL,
    autoReconnect = true,
    reconnectDelay = 3000,
    maxReconnectAttempts = 10,
    onOpen,
    onClose,
    onError,
    onMessage,
  } = options;

  const [ws, setWs] = useState<WebSocket | null>(null);
  const [readyState, setReadyState] = useState<number>(WebSocket.CONNECTING);
  const [error, setError] = useState<Event | null>(null);
  const reconnectAttempts = useRef(0);
  const reconnectTimeout = useRef<NodeJS.Timeout | null>(null);
  const queryClient = useQueryClient();

  const connect = useCallback(() => {
    try {
      const socket = new WebSocket(url);

      socket.onopen = () => {
        console.log('WebSocket connected');
        setReadyState(WebSocket.OPEN);
        reconnectAttempts.current = 0;
        onOpen?.();
      };

      socket.onclose = () => {
        console.log('WebSocket disconnected');
        setReadyState(WebSocket.CLOSED);
        onClose?.();

        // Auto-reconnect
        if (autoReconnect && reconnectAttempts.current < maxReconnectAttempts) {
          reconnectAttempts.current++;
          console.log(`Reconnecting... attempt ${reconnectAttempts.current}`);

          reconnectTimeout.current = setTimeout(() => {
            connect();
          }, reconnectDelay);
        }
      };

      socket.onerror = (event) => {
        console.error('WebSocket error:', event);
        setError(event);
        onError?.(event);
      };

      socket.onmessage = (event) => {
        try {
          const message: WebSocketMessage = JSON.parse(event.data);
          console.log('WebSocket message:', message);

          // Handle message
          onMessage?.(message);

          // Invalidate React Query cache for the quote
          if (message.quote_id) {
            queryClient.invalidateQueries(['quote', message.quote_id]);
            queryClient.invalidateQueries(['quotes']);
          }
        } catch (err) {
          console.error('Failed to parse WebSocket message:', err);
        }
      };

      setWs(socket);
    } catch (err) {
      console.error('Failed to create WebSocket:', err);
    }
  }, [
    url,
    autoReconnect,
    reconnectDelay,
    maxReconnectAttempts,
    onOpen,
    onClose,
    onError,
    onMessage,
    queryClient,
  ]);

  // Connect on mount
  useEffect(() => {
    connect();

    // Cleanup on unmount
    return () => {
      if (reconnectTimeout.current) {
        clearTimeout(reconnectTimeout.current);
      }
      if (ws) {
        ws.close();
      }
    };
  }, [connect]);

  // Update readyState when ws changes
  useEffect(() => {
    if (ws) {
      setReadyState(ws.readyState);
    }
  }, [ws]);

  const subscribe = useCallback(
    (quoteId: string) => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(
          JSON.stringify({
            action: 'subscribe',
            quote_id: quoteId,
          })
        );
      }
    },
    [ws]
  );

  const unsubscribe = useCallback(
    (quoteId: string) => {
      if (ws && ws.readyState === WebSocket.OPEN) {
        ws.send(
          JSON.stringify({
            action: 'unsubscribe',
            quote_id: quoteId,
          })
        );
      }
    },
    [ws]
  );

  const reconnect = useCallback(() => {
    if (ws) {
      ws.close();
    }
    reconnectAttempts.current = 0;
    connect();
  }, [ws, connect]);

  return {
    ws,
    readyState,
    isConnected: readyState === WebSocket.OPEN,
    error,
    subscribe,
    unsubscribe,
    reconnect,
  };
};

/**
 * Hook for subscribing to quote status updates
 */
export const useQuoteStatusUpdates = (
  quoteId: string | undefined,
  onStatusChange?: (message: WebSocketMessage) => void
) => {
  const [lastMessage, setLastMessage] = useState<WebSocketMessage | null>(null);

  const handleMessage = useCallback(
    (message: WebSocketMessage) => {
      if (message.quote_id === quoteId) {
        setLastMessage(message);
        onStatusChange?.(message);
      }
    },
    [quoteId, onStatusChange]
  );

  const { subscribe, unsubscribe, isConnected } = useWebSocket({
    onMessage: handleMessage,
  });

  useEffect(() => {
    if (quoteId && isConnected) {
      subscribe(quoteId);

      return () => {
        unsubscribe(quoteId);
      };
    }
  }, [quoteId, isConnected, subscribe, unsubscribe]);

  return {
    lastMessage,
    isConnected,
  };
};

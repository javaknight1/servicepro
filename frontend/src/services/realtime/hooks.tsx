/**
 * React Hooks for WebSocket Integration
 */

import {
  useState,
  useEffect,
  useCallback,
  useRef,
  useMemo,
  useContext,
  createContext,
  ReactNode,
} from 'react';
import {
  ConnectionStatus,
  ConnectionState,
  WebSocketMessage,
  ChannelSubscription,
  EntityType,
  EntityUpdateMessage,
  NotificationMessage,
  PresenceMessage,
  UseWebSocketOptions,
  UseWebSocketReturn,
  UseSubscriptionOptions,
  UseRealtimeEntityOptions,
} from '../../types/realtime';
import {
  WebSocketServiceImpl,
  getWebSocketService,
  createWebSocketService,
} from './WebSocketService';
import {
  UpdateHandlerManager,
  getUpdateHandlerManager,
} from './UpdateHandlers';
import {
  ConnectionManager,
  getConnectionManager,
  createConnectionManager,
} from './ConnectionManager';

// ============================================
// WebSocket Context
// ============================================

interface WebSocketContextValue {
  service: WebSocketServiceImpl | null;
  manager: ConnectionManager | null;
  handlers: UpdateHandlerManager;
  isConnected: boolean;
  status: ConnectionStatus;
  latency: number | null;
  error: Error | null;
  connect: () => Promise<void>;
  disconnect: () => void;
}

const WebSocketContext = createContext<WebSocketContextValue | null>(null);

// ============================================
// WebSocket Provider
// ============================================

interface WebSocketProviderProps {
  children: ReactNode;
  url: string;
  fallbackUrls?: string[];
  token?: string;
  autoConnect?: boolean;
  debug?: boolean;
}

export const WebSocketProvider: React.FC<WebSocketProviderProps> = ({
  children,
  url,
  fallbackUrls = [],
  token,
  autoConnect = true,
  debug = false,
}) => {
  const [isConnected, setIsConnected] = useState(false);
  const [status, setStatus] = useState<ConnectionStatus>('disconnected');
  const [latency, setLatency] = useState<number | null>(null);
  const [error, setError] = useState<Error | null>(null);

  const serviceRef = useRef<WebSocketServiceImpl | null>(null);
  const managerRef = useRef<ConnectionManager | null>(null);
  const handlersRef = useRef<UpdateHandlerManager>(getUpdateHandlerManager());

  // Initialize connection manager
  useEffect(() => {
    managerRef.current = createConnectionManager({
      primaryUrl: url,
      fallbackUrls,
      token,
      autoFailover: true,
      debug,
    });

    // Set up event listeners
    const manager = managerRef.current;

    manager.on('connected', () => {
      setIsConnected(true);
      setStatus('connected');
      setError(null);
      serviceRef.current = manager.getActiveConnection();
    });

    manager.on('disconnected', () => {
      setIsConnected(false);
      setStatus('disconnected');
    });

    manager.on('failover', () => {
      serviceRef.current = manager.getActiveConnection();
    });

    manager.on('health_check', (event) => {
      const health = event.data as { health: { latency: number | null } };
      setLatency(health.health.latency);
    });

    manager.on('error', (event) => {
      setError((event.data as { error: Error }).error);
    });

    // Auto connect
    if (autoConnect) {
      manager.connect().catch((err) => {
        setError(err);
        setStatus('failed');
      });
    }

    return () => {
      manager.disconnect();
    };
  }, [url, fallbackUrls, token, autoConnect, debug]);

  // Update token when it changes
  useEffect(() => {
    if (token && managerRef.current) {
      managerRef.current.setToken(token);
    }
  }, [token]);

  const connect = useCallback(async () => {
    if (managerRef.current) {
      setStatus('connecting');
      await managerRef.current.connect();
    }
  }, []);

  const disconnect = useCallback(() => {
    managerRef.current?.disconnect();
  }, []);

  const value = useMemo(
    () => ({
      service: serviceRef.current,
      manager: managerRef.current,
      handlers: handlersRef.current,
      isConnected,
      status,
      latency,
      error,
      connect,
      disconnect,
    }),
    [isConnected, status, latency, error, connect, disconnect]
  );

  return (
    <WebSocketContext.Provider value={value}>
      {children}
    </WebSocketContext.Provider>
  );
};

// ============================================
// useWebSocketContext Hook
// ============================================

export const useWebSocketContext = (): WebSocketContextValue => {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error(
      'useWebSocketContext must be used within a WebSocketProvider'
    );
  }
  return context;
};

// ============================================
// useWebSocket Hook
// ============================================

export const useWebSocket = (
  options?: UseWebSocketOptions
): UseWebSocketReturn => {
  const [isConnected, setIsConnected] = useState(false);
  const [status, setStatus] = useState<ConnectionStatus>('disconnected');
  const [latency, setLatency] = useState<number | null>(null);
  const [error, setError] = useState<Error | null>(null);

  const serviceRef = useRef<WebSocketServiceImpl | null>(null);
  const optionsRef = useRef(options);

  useEffect(() => {
    optionsRef.current = options;
  }, [options]);

  // Initialize service if URL provided
  useEffect(() => {
    if (!options?.url) return;

    const service = createWebSocketService({
      url: options.url,
      token: options.token,
      debug: false,
    });

    serviceRef.current = service;

    // Set up event listeners
    service.on('open', () => {
      setIsConnected(true);
      setStatus('connected');
      setError(null);
      optionsRef.current?.onConnect?.();
    });

    service.on('close', () => {
      setIsConnected(false);
      setStatus('disconnected');
      optionsRef.current?.onDisconnect?.();
    });

    service.on('error', (event) => {
      const err = (event.data as { error: Error }).error;
      setError(err);
      optionsRef.current?.onError?.(err);
    });

    service.on('message', (event) => {
      const message = (event.data as { message: WebSocketMessage }).message;
      optionsRef.current?.onMessage?.(message);
    });

    service.on('latency', (event) => {
      setLatency((event.data as { latency: number }).latency);
    });

    // Auto connect
    if (options.autoConnect !== false) {
      service.connect().catch((err) => {
        setError(err);
        setStatus('failed');
      });
    }

    return () => {
      service.disconnect();
    };
  }, [options?.url, options?.token, options?.autoConnect]);

  const connect = useCallback(async () => {
    if (serviceRef.current) {
      setStatus('connecting');
      await serviceRef.current.connect();
    }
  }, []);

  const disconnect = useCallback(() => {
    serviceRef.current?.disconnect();
  }, []);

  const send = useCallback(async (message: WebSocketMessage) => {
    if (serviceRef.current) {
      await serviceRef.current.send(message);
    }
  }, []);

  const subscribe = useCallback((subscription: ChannelSubscription) => {
    if (serviceRef.current) {
      return serviceRef.current.subscribe(subscription);
    }
    return () => {};
  }, []);

  return {
    isConnected,
    status,
    latency,
    error,
    connect,
    disconnect,
    send,
    subscribe,
  };
};

// ============================================
// useSubscription Hook
// ============================================

export function useSubscription<T = unknown>(
  options: UseSubscriptionOptions<T>
): {
  data: T | null;
  error: Error | null;
  isSubscribed: boolean;
} {
  const { service } = useWebSocketContext();
  const [data, setData] = useState<T | null>(null);
  const [error, setError] = useState<Error | null>(null);
  const [isSubscribed, setIsSubscribed] = useState(false);

  const optionsRef = useRef(options);
  useEffect(() => {
    optionsRef.current = options;
  }, [options]);

  useEffect(() => {
    if (!service || options.enabled === false) {
      setIsSubscribed(false);
      return;
    }

    const subscription: ChannelSubscription = {
      channel: options.channel,
      filters: options.filters,
      onMessage: (message) => {
        if ('data' in message) {
          setData(message.data as T);
          optionsRef.current.onMessage?.(message.data as T);
        }
      },
      onError: (err) => {
        setError(err);
      },
    };

    const unsubscribe = service.subscribe(subscription);
    setIsSubscribed(true);

    return () => {
      unsubscribe();
      setIsSubscribed(false);
    };
  }, [service, options.channel, options.filters, options.enabled]);

  return { data, error, isSubscribed };
}

// ============================================
// useRealtimeEntity Hook
// ============================================

export function useRealtimeEntity<T = unknown>(
  options: UseRealtimeEntityOptions<T>
): {
  data: T | null;
  isLoading: boolean;
  error: Error | null;
  refresh: () => void;
} {
  const { handlers } = useWebSocketContext();
  const [data, setData] = useState<T | null>(options.initialData ?? null);
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const optionsRef = useRef(options);
  useEffect(() => {
    optionsRef.current = options;
  }, [options]);

  // Subscribe to entity updates
  useEffect(() => {
    const unsubscribe = handlers.onEntityUpdate<T>(
      options.entity,
      (entityId, updateData, operation) => {
        // Filter by entityId if specified
        if (options.entityId && entityId !== options.entityId) {
          return;
        }

        switch (operation) {
          case 'create':
            optionsRef.current.onCreate?.(updateData);
            if (!options.entityId) {
              // For collections, add to data
              setData((prev) => {
                if (Array.isArray(prev)) {
                  return [...prev, updateData] as unknown as T;
                }
                return updateData;
              });
            }
            break;

          case 'update':
            setData(updateData);
            optionsRef.current.onUpdate?.(updateData);
            break;

          case 'delete':
            optionsRef.current.onDelete?.(entityId);
            if (options.entityId === entityId) {
              setData(null);
            } else if (Array.isArray(data)) {
              // For collections, remove from data
              setData((prev) => {
                if (Array.isArray(prev)) {
                  return prev.filter(
                    (item: unknown) => (item as { id: string }).id !== entityId
                  ) as unknown as T;
                }
                return prev;
              });
            }
            break;
        }
      }
    );

    return unsubscribe;
  }, [handlers, options.entity, options.entityId, data]);

  const refresh = useCallback(() => {
    setIsLoading(true);
    // Trigger a sync request - this would be handled by the application
    setIsLoading(false);
  }, []);

  return { data, isLoading, error, refresh };
}

// ============================================
// usePresence Hook
// ============================================

interface PresenceState {
  users: Map<
    string,
    { status: 'online' | 'away' | 'offline'; lastSeen?: number }
  >;
}

export function usePresence(): {
  users: PresenceState['users'];
  onlineCount: number;
  isUserOnline: (userId: string) => boolean;
} {
  const { handlers } = useWebSocketContext();
  const [users, setUsers] = useState<PresenceState['users']>(new Map());

  useEffect(() => {
    const unsubscribe = handlers.onPresence((message) => {
      setUsers((prev) => {
        const next = new Map(prev);
        next.set(message.userId, {
          status: message.status,
          lastSeen: message.lastSeen,
        });
        return next;
      });
    });

    return unsubscribe;
  }, [handlers]);

  const onlineCount = useMemo(() => {
    return Array.from(users.values()).filter((u) => u.status === 'online')
      .length;
  }, [users]);

  const isUserOnline = useCallback(
    (userId: string) => {
      return users.get(userId)?.status === 'online';
    },
    [users]
  );

  return { users, onlineCount, isUserOnline };
}

// ============================================
// useNotifications Hook
// ============================================

export function useNotifications(options?: {
  maxNotifications?: number;
  onNotification?: (notification: NotificationMessage) => void;
}): {
  notifications: NotificationMessage[];
  unreadCount: number;
  markAsRead: (id: string) => void;
  clearAll: () => void;
} {
  const { handlers } = useWebSocketContext();
  const [notifications, setNotifications] = useState<NotificationMessage[]>([]);
  const [readIds, setReadIds] = useState<Set<string>>(new Set());

  const maxNotifications = options?.maxNotifications ?? 50;

  useEffect(() => {
    const unsubscribe = handlers.onNotification((notification) => {
      setNotifications((prev) => {
        const next = [notification, ...prev];
        return next.slice(0, maxNotifications);
      });
      options?.onNotification?.(notification);
    });

    return unsubscribe;
  }, [handlers, maxNotifications, options]);

  const unreadCount = useMemo(() => {
    return notifications.filter((n) => !readIds.has(n.id)).length;
  }, [notifications, readIds]);

  const markAsRead = useCallback((id: string) => {
    setReadIds((prev) => new Set(prev).add(id));
  }, []);

  const clearAll = useCallback(() => {
    setNotifications([]);
    setReadIds(new Set());
  }, []);

  return { notifications, unreadCount, markAsRead, clearAll };
}

// ============================================
// useConnectionHealth Hook
// ============================================

export function useConnectionHealth(): {
  isHealthy: boolean;
  latency: number | null;
  status: ConnectionStatus;
  reconnectAttempts: number;
} {
  const { manager, status, latency } = useWebSocketContext();
  const [reconnectAttempts, setReconnectAttempts] = useState(0);

  useEffect(() => {
    if (!manager) return;

    const state = manager.getConnectionState();
    if (state) {
      setReconnectAttempts(state.reconnectAttempts);
    }
  }, [manager, status]);

  const isHealthy = useMemo(() => {
    return status === 'connected' && (latency === null || latency < 5000);
  }, [status, latency]);

  return { isHealthy, latency, status, reconnectAttempts };
}

// ============================================
// useRealtimeSync Hook
// ============================================

export function useRealtimeSync<T = unknown>(
  entity: EntityType,
  options?: {
    initialData?: T[];
    onSync?: (data: T[], hasMore: boolean) => void;
  }
): {
  data: T[];
  isLoading: boolean;
  hasMore: boolean;
  loadMore: () => void;
} {
  const { handlers } = useWebSocketContext();
  const [data, setData] = useState<T[]>(options?.initialData ?? []);
  const [isLoading, setIsLoading] = useState(false);
  const [hasMore, setHasMore] = useState(false);
  const [cursor, setCursor] = useState<string | undefined>();

  useEffect(() => {
    const unsubscribe = handlers.onSync<T>(
      entity,
      (_, syncData, more, nextCursor) => {
        setData((prev) => [...prev, ...syncData]);
        setHasMore(more);
        setCursor(nextCursor);
        setIsLoading(false);
        options?.onSync?.(syncData, more);
      }
    );

    return unsubscribe;
  }, [handlers, entity, options]);

  const loadMore = useCallback(() => {
    if (isLoading || !hasMore) return;
    setIsLoading(true);
    // Application should trigger sync request with cursor
  }, [isLoading, hasMore]);

  return { data, isLoading, hasMore, loadMore };
}

// ============================================
// useLiveUpdates Hook - Combines multiple real-time features
// ============================================

export function useLiveUpdates<T = unknown>(options: {
  entity: EntityType;
  entityId?: string;
  channel?: string;
  initialData?: T;
  enablePresence?: boolean;
  enableNotifications?: boolean;
}): {
  data: T | null;
  isConnected: boolean;
  presence: ReturnType<typeof usePresence> | null;
  notifications: ReturnType<typeof useNotifications> | null;
} {
  const { isConnected } = useWebSocketContext();

  const entityResult = useRealtimeEntity<T>({
    entity: options.entity,
    entityId: options.entityId,
    initialData: options.initialData,
  });

  // Always call hooks unconditionally to follow React rules of hooks
  const presenceResult = usePresence();
  const notificationsResult = useNotifications();

  const presence = options.enablePresence ? presenceResult : null;
  const notifications = options.enableNotifications
    ? notificationsResult
    : null;

  return {
    data: entityResult.data,
    isConnected,
    presence,
    notifications,
  };
}

export default {
  WebSocketProvider,
  useWebSocketContext,
  useWebSocket,
  useSubscription,
  useRealtimeEntity,
  usePresence,
  useNotifications,
  useConnectionHealth,
  useRealtimeSync,
  useLiveUpdates,
};

import { useEffect, useCallback, useMemo, useRef } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { KPIData, KPIResponse, KPIWebSocketMessage } from '../types/kpi';
import { useKPIStore } from '../store/kpiStore';
import api from '../services/api';

// API endpoints
const KPI_ENDPOINT = '/api/v1/kpis';

// Query keys
export const kpiKeys = {
  all: ['kpis'] as const,
  list: () => [...kpiKeys.all, 'list'] as const,
  detail: (id: string) => [...kpiKeys.all, 'detail', id] as const,
  category: (category: string) =>
    [...kpiKeys.all, 'category', category] as const,
  dashboard: () => [...kpiKeys.all, 'dashboard'] as const,
};

// API functions
const fetchKPIs = async (): Promise<KPIData[]> => {
  const response = await api.get<KPIResponse>(KPI_ENDPOINT);
  return response.data.data;
};

const fetchKPIById = async (id: string): Promise<KPIData> => {
  const response = await api.get<KPIData>(`${KPI_ENDPOINT}/${id}`);
  return response.data;
};

const fetchKPIsByCategory = async (category: string): Promise<KPIData[]> => {
  const response = await api.get<KPIData[]>(
    `${KPI_ENDPOINT}/category/${category}`
  );
  return response.data;
};

const fetchDashboardKPIs = async (): Promise<KPIData[]> => {
  const response = await api.get<KPIResponse>(`${KPI_ENDPOINT}/dashboard`);
  return response.data.data;
};

// Hook options
interface UseKPIOptions {
  /** Enable automatic refetch */
  refetchOnMount?: boolean;
  /** Refetch interval in ms (0 to disable) */
  refetchInterval?: number;
  /** Sync with KPI store */
  syncWithStore?: boolean;
  /** Enable WebSocket updates */
  enableRealTime?: boolean;
}

// Main hook for fetching all KPIs
export const useKPIs = (options: UseKPIOptions = {}) => {
  const {
    refetchOnMount = true,
    refetchInterval = 0,
    syncWithStore = true,
  } = options;

  const { setKPIs, setLoading, setError } = useKPIStore();
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: kpiKeys.list(),
    queryFn: fetchKPIs,
    refetchOnMount,
    refetchInterval: refetchInterval > 0 ? refetchInterval : false,
    staleTime: 30000, // Consider data stale after 30 seconds
  });

  // Sync with store
  useEffect(() => {
    if (syncWithStore) {
      if (query.isLoading) {
        setLoading(true);
      } else if (query.isError) {
        setError(
          query.error instanceof Error
            ? query.error.message
            : 'Failed to fetch KPIs'
        );
      } else if (query.data) {
        setKPIs(query.data);
      }
    }
  }, [
    query.data,
    query.isLoading,
    query.isError,
    query.error,
    syncWithStore,
    setKPIs,
    setLoading,
    setError,
  ]);

  const refresh = useCallback(() => {
    return queryClient.invalidateQueries({ queryKey: kpiKeys.list() });
  }, [queryClient]);

  return {
    ...query,
    kpis: query.data || [],
    refresh,
  };
};

// Hook for fetching a single KPI
export const useKPI = (id: string, options: UseKPIOptions = {}) => {
  const { refetchOnMount = true, refetchInterval = 0 } = options;

  const query = useQuery({
    queryKey: kpiKeys.detail(id),
    queryFn: () => fetchKPIById(id),
    enabled: !!id,
    refetchOnMount,
    refetchInterval: refetchInterval > 0 ? refetchInterval : false,
    staleTime: 30000,
  });

  return {
    ...query,
    kpi: query.data,
  };
};

// Hook for fetching KPIs by category
export const useKPIsByCategory = (
  category: string,
  options: UseKPIOptions = {}
) => {
  const { refetchOnMount = true, refetchInterval = 0 } = options;

  const query = useQuery({
    queryKey: kpiKeys.category(category),
    queryFn: () => fetchKPIsByCategory(category),
    enabled: !!category,
    refetchOnMount,
    refetchInterval: refetchInterval > 0 ? refetchInterval : false,
    staleTime: 30000,
  });

  return {
    ...query,
    kpis: query.data || [],
  };
};

// Hook for dashboard KPIs with real-time updates
export const useDashboardKPIs = (options: UseKPIOptions = {}) => {
  const {
    refetchOnMount = true,
    refetchInterval = 60000, // Refetch every minute by default
    syncWithStore = true,
    enableRealTime = true,
  } = options;

  const { setKPIs, updateKPI, setLoading, setError, state } = useKPIStore();
  const queryClient = useQueryClient();
  const wsRef = useRef<WebSocket | null>(null);

  const query = useQuery({
    queryKey: kpiKeys.dashboard(),
    queryFn: fetchDashboardKPIs,
    refetchOnMount,
    refetchInterval: refetchInterval > 0 ? refetchInterval : false,
    staleTime: 30000,
  });

  // Sync with store
  useEffect(() => {
    if (syncWithStore) {
      if (query.isLoading) {
        setLoading(true);
      } else if (query.isError) {
        setError(
          query.error instanceof Error
            ? query.error.message
            : 'Failed to fetch KPIs'
        );
      } else if (query.data) {
        setKPIs(query.data);
      }
    }
  }, [
    query.data,
    query.isLoading,
    query.isError,
    query.error,
    syncWithStore,
    setKPIs,
    setLoading,
    setError,
  ]);

  // WebSocket for real-time updates
  useEffect(() => {
    if (!enableRealTime) return;

    const wsUrl = `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${
      window.location.host
    }/api/v1/ws/kpi`;

    try {
      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        console.log('Dashboard KPI WebSocket connected');
        ws.send(
          JSON.stringify({ action: 'subscribe', channel: 'dashboard-kpi' })
        );
      };

      ws.onmessage = (event) => {
        try {
          const message: KPIWebSocketMessage = JSON.parse(event.data);

          if (message.type === 'kpi_update') {
            // Update in store
            updateKPI(message.kpi_id, {
              value: message.value,
              previousValue: message.previousValue,
              lastUpdated: message.timestamp,
              status: message.status,
            });

            // Also update React Query cache
            queryClient.setQueryData<KPIData[]>(kpiKeys.dashboard(), (old) => {
              if (!old) return old;
              return old.map((kpi) =>
                kpi.id === message.kpi_id
                  ? {
                      ...kpi,
                      value: message.value,
                      previousValue: message.previousValue,
                      lastUpdated: message.timestamp,
                      status: message.status,
                    }
                  : kpi
              );
            });
          }
        } catch (err) {
          console.error('Failed to parse KPI message:', err);
        }
      };

      ws.onerror = (error) => {
        console.error('Dashboard KPI WebSocket error:', error);
      };

      wsRef.current = ws;

      return () => {
        ws.close();
      };
    } catch (err) {
      console.error('Failed to create Dashboard KPI WebSocket:', err);
    }
  }, [enableRealTime, updateKPI, queryClient]);

  const refresh = useCallback(() => {
    return queryClient.invalidateQueries({ queryKey: kpiKeys.dashboard() });
  }, [queryClient]);

  // Merge query data with store data for real-time updates
  const kpis = useMemo(() => {
    if (syncWithStore && Object.keys(state.kpis).length > 0) {
      return Object.values(state.kpis);
    }
    return query.data || [];
  }, [syncWithStore, state.kpis, query.data]);

  return {
    ...query,
    kpis,
    isConnected: state.isConnected,
    refresh,
  };
};

// Hook for polling KPI updates
export const useKPIPolling = (
  id: string,
  intervalMs: number = 5000,
  options: { enabled?: boolean } = {}
) => {
  const { enabled = true } = options;
  const { updateKPI } = useKPIStore();
  const queryClient = useQueryClient();

  const query = useQuery({
    queryKey: kpiKeys.detail(id),
    queryFn: () => fetchKPIById(id),
    enabled: enabled && !!id,
    refetchInterval: intervalMs,
    staleTime: intervalMs / 2,
  });

  // Update store when data changes
  useEffect(() => {
    if (query.data) {
      updateKPI(id, query.data);
    }
  }, [query.data, id, updateKPI]);

  const stopPolling = useCallback(() => {
    queryClient.cancelQueries({ queryKey: kpiKeys.detail(id) });
  }, [queryClient, id]);

  return {
    ...query,
    kpi: query.data,
    stopPolling,
  };
};

// Utility hook for computing derived KPI values
export const useKPIDerived = (kpis: KPIData[]) => {
  return useMemo(() => {
    const total = kpis.reduce((sum, kpi) => sum + kpi.value, 0);
    const average = kpis.length > 0 ? total / kpis.length : 0;

    const byCategory = kpis.reduce(
      (acc, kpi) => {
        const category = kpi.category || 'uncategorized';
        if (!acc[category]) {
          acc[category] = [];
        }
        acc[category].push(kpi);
        return acc;
      },
      {} as Record<string, KPIData[]>
    );

    const byStatus = kpis.reduce(
      (acc, kpi) => {
        const status = kpi.status || 'neutral';
        if (!acc[status]) {
          acc[status] = 0;
        }
        acc[status]++;
        return acc;
      },
      {} as Record<string, number>
    );

    return {
      total,
      average,
      count: kpis.length,
      byCategory,
      byStatus,
    };
  }, [kpis]);
};

export default useKPIs;

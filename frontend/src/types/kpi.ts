/**
 * KPI Widget Types and Interfaces
 */

// Display format for KPI values
export type KPIFormat =
  | 'number'
  | 'currency'
  | 'percentage'
  | 'duration'
  | 'compact'
  | 'decimal';

// Trend direction
export type TrendDirection = 'up' | 'down' | 'neutral';

// KPI status for visual indication
export type KPIStatus = 'success' | 'warning' | 'error' | 'info' | 'neutral';

// Size variants
export type KPISize = 'sm' | 'md' | 'lg' | 'xl';

// KPI data structure
export interface KPIData {
  /** Unique identifier */
  id: string;
  /** Display label */
  label: string;
  /** Current value */
  value: number;
  /** Previous period value for comparison */
  previousValue?: number;
  /** Target value */
  target?: number;
  /** Format for display */
  format: KPIFormat;
  /** Currency code (for currency format) */
  currency?: string;
  /** Decimal places to show */
  decimals?: number;
  /** Prefix (e.g., "$") */
  prefix?: string;
  /** Suffix (e.g., "%", "hrs") */
  suffix?: string;
  /** Optional description/tooltip text */
  description?: string;
  /** Status indicator */
  status?: KPIStatus;
  /** Icon name or component */
  icon?: string | React.ReactNode;
  /** Last updated timestamp */
  lastUpdated?: string;
  /** Category for grouping */
  category?: string;
}

// KPI trend calculation result
export interface KPITrend {
  direction: TrendDirection;
  percentage: number;
  isPositive: boolean;
  label: string;
}

// KPI comparison to target
export interface KPITargetProgress {
  percentage: number;
  isOnTrack: boolean;
  remaining: number;
}

// Widget props
export interface KPIWidgetProps {
  /** KPI data to display */
  data: KPIData;
  /** Show trend indicator */
  showTrend?: boolean;
  /** Show target progress */
  showTarget?: boolean;
  /** Show sparkline chart */
  showSparkline?: boolean;
  /** Historical data for sparkline */
  sparklineData?: number[];
  /** Loading state */
  loading?: boolean;
  /** Error state */
  error?: string | null;
  /** Size variant */
  size?: KPISize;
  /** Enable real-time updates */
  realTime?: boolean;
  /** Update interval in ms (for polling) */
  updateInterval?: number;
  /** Click handler */
  onClick?: () => void;
  /** Custom className */
  className?: string;
  /** Animate value changes */
  animate?: boolean;
  /** Show tooltip on hover */
  showTooltip?: boolean;
  /** Compact mode (hide label) */
  compact?: boolean;
}

// KPI Grid props for displaying multiple KPIs
export interface KPIGridProps {
  /** Array of KPI widgets */
  kpis: KPIData[];
  /** Number of columns */
  columns?: 1 | 2 | 3 | 4 | 6;
  /** Show trend for all */
  showTrends?: boolean;
  /** Gap between items */
  gap?: 'sm' | 'md' | 'lg';
  /** Loading state for all */
  loading?: boolean;
  /** Error state */
  error?: string | null;
  /** Custom className */
  className?: string;
}

// API response for KPI data
export interface KPIResponse {
  data: KPIData[];
  lastUpdated: string;
  period: {
    start: string;
    end: string;
  };
}

// WebSocket message for KPI updates
export interface KPIWebSocketMessage {
  type: 'kpi_update';
  kpi_id: string;
  value: number;
  previousValue?: number;
  timestamp: string;
  status?: KPIStatus;
}

// KPI Store state
export interface KPIState {
  kpis: Record<string, KPIData>;
  loading: boolean;
  error: string | null;
  lastUpdated: string | null;
  isConnected: boolean;
}

// KPI Store actions
export interface KPIActions {
  setKPIs: (kpis: KPIData[]) => void;
  updateKPI: (id: string, data: Partial<KPIData>) => void;
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;
  setConnected: (connected: boolean) => void;
  refreshKPIs: () => Promise<void>;
}

// Helper functions for formatting
export const formatKPIValue = (
  value: number,
  format: KPIFormat,
  options?: {
    currency?: string;
    decimals?: number;
    locale?: string;
  }
): string => {
  const { currency = 'USD', decimals = 0, locale = 'en-US' } = options || {};

  switch (format) {
    case 'currency':
      return new Intl.NumberFormat(locale, {
        style: 'currency',
        currency,
        minimumFractionDigits: decimals,
        maximumFractionDigits: decimals,
      }).format(value);

    case 'percentage':
      return new Intl.NumberFormat(locale, {
        style: 'percent',
        minimumFractionDigits: decimals,
        maximumFractionDigits: decimals,
      }).format(value / 100);

    case 'compact':
      return new Intl.NumberFormat(locale, {
        notation: 'compact',
        compactDisplay: 'short',
        maximumFractionDigits: 1,
      }).format(value);

    case 'decimal':
      return new Intl.NumberFormat(locale, {
        minimumFractionDigits: decimals,
        maximumFractionDigits: decimals,
      }).format(value);

    case 'duration':
      const hours = Math.floor(value / 60);
      const minutes = value % 60;
      if (hours > 0) {
        return `${hours}h ${minutes}m`;
      }
      return `${minutes}m`;

    case 'number':
    default:
      return new Intl.NumberFormat(locale).format(value);
  }
};

export const calculateTrend = (
  current: number,
  previous: number | undefined
): KPITrend => {
  if (previous === undefined || previous === 0) {
    return {
      direction: 'neutral',
      percentage: 0,
      isPositive: true,
      label: 'No data',
    };
  }

  const change = ((current - previous) / Math.abs(previous)) * 100;
  const direction: TrendDirection =
    change > 0 ? 'up' : change < 0 ? 'down' : 'neutral';
  const isPositive = change >= 0;

  return {
    direction,
    percentage: Math.abs(change),
    isPositive,
    label: `${isPositive ? '+' : ''}${change.toFixed(1)}%`,
  };
};

export const calculateTargetProgress = (
  current: number,
  target: number | undefined
): KPITargetProgress | null => {
  if (target === undefined || target === 0) {
    return null;
  }

  const percentage = (current / target) * 100;
  const remaining = target - current;
  const isOnTrack = percentage >= 100 || (percentage >= 80 && remaining > 0);

  return {
    percentage: Math.min(percentage, 100),
    isOnTrack,
    remaining,
  };
};

export const getStatusColor = (status: KPIStatus): string => {
  const colors: Record<KPIStatus, string> = {
    success: 'text-green-600',
    warning: 'text-amber-600',
    error: 'text-red-600',
    info: 'text-blue-600',
    neutral: 'text-gray-600',
  };
  return colors[status];
};

export const getStatusBgColor = (status: KPIStatus): string => {
  const colors: Record<KPIStatus, string> = {
    success: 'bg-green-50',
    warning: 'bg-amber-50',
    error: 'bg-red-50',
    info: 'bg-blue-50',
    neutral: 'bg-gray-50',
  };
  return colors[status];
};

export const getTrendColor = (
  direction: TrendDirection,
  isPositiveGood: boolean = true
): string => {
  if (direction === 'neutral') return 'text-gray-500';

  const isGood = direction === 'up' ? isPositiveGood : !isPositiveGood;
  return isGood ? 'text-green-600' : 'text-red-600';
};

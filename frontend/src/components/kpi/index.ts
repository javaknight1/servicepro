// Main components
export { default as KPIWidget } from './KPIWidget';
export { default as KPIGrid } from './KPIGrid';

// Sub-components
export {
  KPIWidgetSkeleton,
  KPIWidgetError,
  Sparkline,
  TargetProgress,
} from './KPIWidget';

// Types (re-export from types for convenience)
export type {
  KPIData,
  KPIFormat,
  KPISize,
  KPIStatus,
  KPITrend,
  KPITargetProgress,
  KPIWidgetProps,
  KPIGridProps,
  KPIState,
  KPIActions,
  KPIWebSocketMessage,
  TrendDirection,
} from '../../types/kpi';

// Utility functions
export {
  formatKPIValue,
  calculateTrend,
  calculateTargetProgress,
  getStatusColor,
  getStatusBgColor,
  getTrendColor,
} from '../../types/kpi';

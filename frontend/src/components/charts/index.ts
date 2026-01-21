// Main chart components
export { default as BaseChart } from './BaseChart';
export { default as LineChart, useLineChartData } from './LineChart';
export {
  default as BarChart,
  HorizontalBarChart,
  StackedBarChart,
  useBarChartData,
} from './BarChart';
export {
  default as PieChart,
  DoughnutChart,
  HalfDoughnutChart,
  usePieChartData,
} from './PieChart';
export {
  default as AreaChart,
  StackedAreaChart,
  GradientAreaChart,
  useAreaChartData,
} from './AreaChart';

// Sub-components
export { ChartSkeleton, ChartError, ExportButton } from './BaseChart';

// Utilities
export * from './chartUtils';

// Types (re-export for convenience)
export type {
  // Base types
  BaseChartProps,
  ChartRef,
  ChartTheme,
  AnimationPreset,
  LegendPosition,
  TooltipMode,
  ExportFormat,
  DataPoint,
  DatasetConfig,
  // Chart-specific props
  LineChartProps,
  BarChartProps,
  PieChartProps,
  AreaChartProps,
} from '../../types/chart';

// Constants
export {
  chartColors,
  defaultColorPalette,
  animationConfigs,
  themeConfigs,
} from '../../types/chart';

// Utility functions from types
export {
  getColorWithOpacity,
  generateGradient,
  formatNumber,
  formatCurrency,
  formatPercentage,
  formatCompact,
} from '../../types/chart';

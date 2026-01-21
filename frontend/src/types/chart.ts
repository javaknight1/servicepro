/**
 * Chart Components Types and Interfaces
 * Built on Chart.js with React integration
 */

import type {
  ChartData,
  ChartOptions,
  ChartType,
  Plugin,
  ScriptableContext,
  TooltipItem,
  AnimationSpec,
  Chart,
} from 'chart.js';

// Color palette for charts
export const chartColors = {
  primary: {
    main: 'rgb(59, 130, 246)',
    light: 'rgba(59, 130, 246, 0.5)',
    lighter: 'rgba(59, 130, 246, 0.1)',
  },
  secondary: {
    main: 'rgb(107, 114, 128)',
    light: 'rgba(107, 114, 128, 0.5)',
    lighter: 'rgba(107, 114, 128, 0.1)',
  },
  success: {
    main: 'rgb(34, 197, 94)',
    light: 'rgba(34, 197, 94, 0.5)',
    lighter: 'rgba(34, 197, 94, 0.1)',
  },
  warning: {
    main: 'rgb(245, 158, 11)',
    light: 'rgba(245, 158, 11, 0.5)',
    lighter: 'rgba(245, 158, 11, 0.1)',
  },
  error: {
    main: 'rgb(239, 68, 68)',
    light: 'rgba(239, 68, 68, 0.5)',
    lighter: 'rgba(239, 68, 68, 0.1)',
  },
  info: {
    main: 'rgb(14, 165, 233)',
    light: 'rgba(14, 165, 233, 0.5)',
    lighter: 'rgba(14, 165, 233, 0.1)',
  },
};

// Default color palette for datasets
export const defaultColorPalette = [
  'rgb(59, 130, 246)', // Blue
  'rgb(34, 197, 94)', // Green
  'rgb(245, 158, 11)', // Amber
  'rgb(239, 68, 68)', // Red
  'rgb(168, 85, 247)', // Purple
  'rgb(14, 165, 233)', // Sky
  'rgb(249, 115, 22)', // Orange
  'rgb(236, 72, 153)', // Pink
  'rgb(20, 184, 166)', // Teal
  'rgb(132, 204, 22)', // Lime
];

// Chart theme
export type ChartTheme = 'light' | 'dark';

// Animation presets
export type AnimationPreset =
  | 'none'
  | 'default'
  | 'smooth'
  | 'bounce'
  | 'elastic';

// Legend position
export type LegendPosition = 'top' | 'bottom' | 'left' | 'right' | 'hidden';

// Tooltip mode
export type TooltipMode = 'index' | 'point' | 'nearest' | 'dataset';

// Export format
export type ExportFormat = 'png' | 'jpeg' | 'svg' | 'pdf';

// Data point for charts
export interface DataPoint {
  x: number | string | Date;
  y: number;
  label?: string;
}

// Dataset configuration
export interface DatasetConfig {
  label: string;
  data: number[] | DataPoint[];
  backgroundColor?: string | string[];
  borderColor?: string;
  borderWidth?: number;
  fill?: boolean | string;
  tension?: number;
  pointRadius?: number;
  pointHoverRadius?: number;
  hidden?: boolean;
  order?: number;
  yAxisID?: string;
  stack?: string;
}

// Base chart props shared by all chart types
export interface BaseChartProps {
  /** Chart title */
  title?: string;
  /** Chart subtitle */
  subtitle?: string;
  /** Width (CSS value or number for pixels) */
  width?: string | number;
  /** Height (CSS value or number for pixels) */
  height?: string | number;
  /** Maintain aspect ratio */
  maintainAspectRatio?: boolean;
  /** Aspect ratio (width/height) */
  aspectRatio?: number;
  /** Show legend */
  showLegend?: boolean;
  /** Legend position */
  legendPosition?: LegendPosition;
  /** Show grid lines */
  showGrid?: boolean;
  /** Show X axis */
  showXAxis?: boolean;
  /** Show Y axis */
  showYAxis?: boolean;
  /** Animation preset */
  animation?: AnimationPreset;
  /** Animation duration in ms */
  animationDuration?: number;
  /** Enable tooltips */
  enableTooltips?: boolean;
  /** Tooltip mode */
  tooltipMode?: TooltipMode;
  /** Theme */
  theme?: ChartTheme;
  /** Loading state */
  loading?: boolean;
  /** Error state */
  error?: string | null;
  /** Custom className */
  className?: string;
  /** Enable zoom/pan */
  enableZoom?: boolean;
  /** Enable export functionality */
  enableExport?: boolean;
  /** Custom plugins */
  plugins?: Plugin[];
  /** Callback when chart is ready */
  onReady?: (chart: Chart) => void;
  /** Callback on click */
  onClick?: (event: MouseEvent, elements: any[], chart: Chart) => void;
  /** Callback on hover */
  onHover?: (event: MouseEvent, elements: any[], chart: Chart) => void;
  /** Accessible label */
  ariaLabel?: string;
}

// Line chart specific props
export interface LineChartProps extends BaseChartProps {
  /** Labels for X axis */
  labels: string[];
  /** Datasets to display */
  datasets: DatasetConfig[];
  /** Line tension (0 = straight, 1 = curved) */
  tension?: number;
  /** Fill area under line */
  fill?: boolean;
  /** Show points on line */
  showPoints?: boolean;
  /** Point style */
  pointStyle?: 'circle' | 'cross' | 'triangle' | 'rect' | 'star';
  /** Stepped line */
  stepped?: boolean | 'before' | 'after' | 'middle';
  /** Y axis min value */
  yMin?: number;
  /** Y axis max value */
  yMax?: number;
  /** Format Y axis values */
  yAxisFormat?: (value: number) => string;
  /** Format X axis values */
  xAxisFormat?: (value: string) => string;
  /** Format tooltip values */
  tooltipFormat?: (
    value: number,
    label: string,
    datasetLabel: string
  ) => string;
}

// Bar chart specific props
export interface BarChartProps extends BaseChartProps {
  /** Labels for X axis */
  labels: string[];
  /** Datasets to display */
  datasets: DatasetConfig[];
  /** Horizontal bar chart */
  horizontal?: boolean;
  /** Stack bars */
  stacked?: boolean;
  /** Bar thickness in pixels */
  barThickness?: number;
  /** Maximum bar thickness */
  maxBarThickness?: number;
  /** Bar border radius */
  borderRadius?: number;
  /** Group bars */
  grouped?: boolean;
  /** Y axis min value */
  yMin?: number;
  /** Y axis max value */
  yMax?: number;
  /** Format Y axis values */
  yAxisFormat?: (value: number) => string;
  /** Format tooltip values */
  tooltipFormat?: (
    value: number,
    label: string,
    datasetLabel: string
  ) => string;
}

// Pie/Doughnut chart specific props
export interface PieChartProps extends BaseChartProps {
  /** Labels for segments */
  labels: string[];
  /** Data values */
  data: number[];
  /** Colors for segments */
  colors?: string[];
  /** Doughnut chart (hollow center) */
  doughnut?: boolean;
  /** Cutout percentage for doughnut */
  cutout?: number | string;
  /** Rotation angle */
  rotation?: number;
  /** Circumference (for partial pie) */
  circumference?: number;
  /** Show percentage in tooltip */
  showPercentage?: boolean;
  /** Center label (for doughnut) */
  centerLabel?: string;
  /** Center value (for doughnut) */
  centerValue?: string | number;
  /** Format tooltip values */
  tooltipFormat?: (value: number, label: string, percentage: number) => string;
}

// Area chart specific props
export interface AreaChartProps extends BaseChartProps {
  /** Labels for X axis */
  labels: string[];
  /** Datasets to display */
  datasets: DatasetConfig[];
  /** Stack areas */
  stacked?: boolean;
  /** Line tension */
  tension?: number;
  /** Fill opacity */
  fillOpacity?: number;
  /** Y axis min value */
  yMin?: number;
  /** Y axis max value */
  yMax?: number;
  /** Format Y axis values */
  yAxisFormat?: (value: number) => string;
  /** Format tooltip values */
  tooltipFormat?: (
    value: number,
    label: string,
    datasetLabel: string
  ) => string;
}

// Chart ref type for imperative handle
export interface ChartRef {
  /** Get the Chart.js instance */
  getChart: () => Chart | null;
  /** Export chart as image */
  exportAsImage: (format?: ExportFormat, filename?: string) => Promise<void>;
  /** Reset zoom */
  resetZoom: () => void;
  /** Update chart data */
  updateData: (labels: string[], datasets: DatasetConfig[]) => void;
  /** Toggle dataset visibility */
  toggleDataset: (index: number) => void;
  /** Destroy chart */
  destroy: () => void;
}

// Animation configurations
export const animationConfigs: Record<
  AnimationPreset,
  AnimationSpec<ChartType> | false
> = {
  none: false,
  default: {
    duration: 750,
    easing: 'easeOutQuart',
  },
  smooth: {
    duration: 1000,
    easing: 'easeInOutQuart',
  },
  bounce: {
    duration: 1200,
    easing: 'easeOutBounce',
  },
  elastic: {
    duration: 1500,
    easing: 'easeOutElastic',
  },
};

// Theme configurations
export const themeConfigs: Record<
  ChartTheme,
  { grid: string; text: string; background: string }
> = {
  light: {
    grid: 'rgba(0, 0, 0, 0.1)',
    text: 'rgb(55, 65, 81)',
    background: 'rgba(255, 255, 255, 0.9)',
  },
  dark: {
    grid: 'rgba(255, 255, 255, 0.1)',
    text: 'rgb(229, 231, 235)',
    background: 'rgba(17, 24, 39, 0.9)',
  },
};

// Utility functions
export const getColorWithOpacity = (color: string, opacity: number): string => {
  if (color.startsWith('rgb(')) {
    return color.replace('rgb(', 'rgba(').replace(')', `, ${opacity})`);
  }
  if (color.startsWith('rgba(')) {
    return color.replace(/,\s*[\d.]+\)$/, `, ${opacity})`);
  }
  return color;
};

export const generateGradient = (
  ctx: CanvasRenderingContext2D,
  color: string,
  height: number
): CanvasGradient => {
  const gradient = ctx.createLinearGradient(0, 0, 0, height);
  gradient.addColorStop(0, getColorWithOpacity(color, 0.4));
  gradient.addColorStop(1, getColorWithOpacity(color, 0.05));
  return gradient;
};

export const formatNumber = (
  value: number,
  options?: Intl.NumberFormatOptions
): string => {
  return new Intl.NumberFormat('en-US', options).format(value);
};

export const formatCurrency = (value: number, currency = 'USD'): string => {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency,
    minimumFractionDigits: 0,
    maximumFractionDigits: 0,
  }).format(value);
};

export const formatPercentage = (value: number, decimals = 1): string => {
  return `${value.toFixed(decimals)}%`;
};

export const formatCompact = (value: number): string => {
  return new Intl.NumberFormat('en-US', {
    notation: 'compact',
    compactDisplay: 'short',
  }).format(value);
};

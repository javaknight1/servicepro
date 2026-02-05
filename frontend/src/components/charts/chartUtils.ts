/**
 * Chart Utilities
 * Helper functions for chart data generation, formatting, and manipulation
 */

import {
  DatasetConfig,
  defaultColorPalette,
  getColorWithOpacity,
  formatNumber,
  formatCurrency,
  formatPercentage,
  formatCompact,
} from '../../types/chart';
import {
  downloadCSV as downloadCSVFile,
  downloadJSON as downloadJSONFile,
} from '../../utils/fileDownload';

// ============================================
// Mock Data Generators
// ============================================

/**
 * Generate random number within range
 */
export const randomInRange = (min: number, max: number): number => {
  return Math.floor(Math.random() * (max - min + 1)) + min;
};

/**
 * Generate random data array
 */
export const generateRandomData = (
  length: number,
  min: number,
  max: number
): number[] => {
  return Array.from({ length }, () => randomInRange(min, max));
};

/**
 * Generate trending data (with natural growth/decline pattern)
 */
export const generateTrendingData = (
  length: number,
  startValue: number,
  trend: 'up' | 'down' | 'stable' = 'up',
  volatility: number = 0.1
): number[] => {
  const data: number[] = [startValue];
  const trendMultiplier = trend === 'up' ? 1.02 : trend === 'down' ? 0.98 : 1;

  for (let i = 1; i < length; i++) {
    const prevValue = data[i - 1];
    const randomFactor = 1 + (Math.random() - 0.5) * volatility * 2;
    const newValue = prevValue * trendMultiplier * randomFactor;
    data.push(Math.round(newValue));
  }

  return data;
};

/**
 * Generate seasonal data pattern
 */
export const generateSeasonalData = (
  months: number = 12,
  baseValue: number = 1000,
  seasonalAmplitude: number = 0.3
): number[] => {
  return Array.from({ length: months }, (_, i) => {
    const seasonal = Math.sin((i / 12) * 2 * Math.PI) * seasonalAmplitude;
    const random = (Math.random() - 0.5) * 0.1;
    return Math.round(baseValue * (1 + seasonal + random));
  });
};

/**
 * Generate mock time series labels
 */
export const generateTimeLabels = (
  count: number,
  type: 'months' | 'weeks' | 'days' | 'hours' = 'months',
  startDate: Date = new Date()
): string[] => {
  const labels: string[] = [];
  const date = new Date(startDate);

  const formatters = {
    months: (d: Date) =>
      d.toLocaleDateString('en-US', { month: 'short', year: '2-digit' }),
    weeks: (d: Date) => `Week ${Math.ceil(d.getDate() / 7)}`,
    days: (d: Date) =>
      d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }),
    hours: (d: Date) =>
      d.toLocaleTimeString('en-US', { hour: 'numeric', hour12: true }),
  };

  const increments = {
    months: () => date.setMonth(date.getMonth() + 1),
    weeks: () => date.setDate(date.getDate() + 7),
    days: () => date.setDate(date.getDate() + 1),
    hours: () => date.setHours(date.getHours() + 1),
  };

  for (let i = 0; i < count; i++) {
    labels.push(formatters[type](date));
    increments[type]();
  }

  return labels;
};

/**
 * Generate mock dataset for testing
 */
export const generateMockDataset = (
  label: string,
  length: number,
  options: {
    min?: number;
    max?: number;
    color?: string;
    trend?: 'up' | 'down' | 'stable';
  } = {}
): DatasetConfig => {
  const { min = 100, max = 1000, color, trend } = options;

  let data: number[];
  if (trend) {
    data = generateTrendingData(length, randomInRange(min, max), trend);
  } else {
    data = generateRandomData(length, min, max);
  }

  return {
    label,
    data,
    borderColor: color,
    backgroundColor: color ? getColorWithOpacity(color, 0.1) : undefined,
  };
};

/**
 * Generate complete mock chart data
 */
export const generateMockChartData = (
  datasetCount: number = 2,
  pointCount: number = 12,
  labelType: 'months' | 'weeks' | 'days' = 'months'
) => {
  const labels = generateTimeLabels(pointCount, labelType);
  const datasets = Array.from({ length: datasetCount }, (_, i) => ({
    label: `Series ${i + 1}`,
    data: generateRandomData(pointCount, 100, 1000),
    borderColor: defaultColorPalette[i % defaultColorPalette.length],
  }));

  return { labels, datasets };
};

// ============================================
// Data Transformation Utilities
// ============================================

/**
 * Calculate percentage change between values
 */
export const calculateChange = (
  current: number,
  previous: number
): { value: number; percentage: number; isPositive: boolean } => {
  const change = current - previous;
  const percentage = previous !== 0 ? (change / previous) * 100 : 0;
  return {
    value: change,
    percentage,
    isPositive: change >= 0,
  };
};

/**
 * Normalize data to 0-100 range
 */
export const normalizeData = (data: number[]): number[] => {
  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;
  return data.map((value) => ((value - min) / range) * 100);
};

/**
 * Calculate moving average
 */
export const movingAverage = (data: number[], window: number = 3): number[] => {
  return data.map((_, index, arr) => {
    const start = Math.max(0, index - window + 1);
    const values = arr.slice(start, index + 1);
    return values.reduce((sum, val) => sum + val, 0) / values.length;
  });
};

/**
 * Calculate cumulative sum
 */
export const cumulativeSum = (data: number[]): number[] => {
  let sum = 0;
  return data.map((value) => (sum += value));
};

/**
 * Group data by category
 */
export const groupDataByCategory = <
  T extends { category: string; value: number },
>(
  items: T[]
): { labels: string[]; data: number[] } => {
  const groups = items.reduce(
    (acc, item) => {
      acc[item.category] = (acc[item.category] || 0) + item.value;
      return acc;
    },
    {} as Record<string, number>
  );

  return {
    labels: Object.keys(groups),
    data: Object.values(groups),
  };
};

/**
 * Sort datasets by total value
 */
export const sortDatasetsByTotal = (
  datasets: DatasetConfig[],
  order: 'asc' | 'desc' = 'desc'
): DatasetConfig[] => {
  return [...datasets].sort((a, b) => {
    const totalA = (a.data as number[]).reduce((sum, val) => sum + val, 0);
    const totalB = (b.data as number[]).reduce((sum, val) => sum + val, 0);
    return order === 'desc' ? totalB - totalA : totalA - totalB;
  });
};

// ============================================
// Formatting Utilities
// ============================================

/**
 * Format axis tick value based on type
 */
export const createAxisFormatter = (
  type: 'number' | 'currency' | 'percentage' | 'compact'
): ((value: number) => string) => {
  switch (type) {
    case 'currency':
      return formatCurrency;
    case 'percentage':
      return (v) => formatPercentage(v);
    case 'compact':
      return formatCompact;
    default:
      return (v) => formatNumber(v);
  }
};

/**
 * Create tooltip formatter
 */
export const createTooltipFormatter = (
  type: 'number' | 'currency' | 'percentage',
  options?: { includeLabel?: boolean; includeDataset?: boolean }
): ((value: number, label: string, dataset: string) => string) => {
  const { includeLabel = true, includeDataset = true } = options || {};

  return (value, label, dataset) => {
    let formatted = '';
    if (includeDataset) formatted += `${dataset}: `;
    if (includeLabel) formatted += `${label} - `;

    switch (type) {
      case 'currency':
        formatted += formatCurrency(value);
        break;
      case 'percentage':
        formatted += formatPercentage(value);
        break;
      default:
        formatted += formatNumber(value);
    }

    return formatted;
  };
};

// ============================================
// Color Utilities
// ============================================

/**
 * Generate color palette with opacity variants
 */
export const generateColorPalette = (
  baseColors: string[] = defaultColorPalette,
  opacities: number[] = [1, 0.7, 0.4, 0.1]
): Record<string, string[]> => {
  return baseColors.reduce(
    (acc, color, index) => {
      acc[`color${index + 1}`] = opacities.map((opacity) =>
        getColorWithOpacity(color, opacity)
      );
      return acc;
    },
    {} as Record<string, string[]>
  );
};

/**
 * Get contrasting text color for background
 */
export const getContrastColor = (bgColor: string): string => {
  // Simple luminance calculation
  const rgb = bgColor.match(/\d+/g);
  if (!rgb || rgb.length < 3) return '#000000';

  const luminance =
    (parseInt(rgb[0]) * 299 + parseInt(rgb[1]) * 587 + parseInt(rgb[2]) * 114) /
    1000;

  return luminance > 128 ? '#000000' : '#ffffff';
};

// ============================================
// Animation Utilities
// ============================================

/**
 * Create staggered animation delay
 */
export const createStaggeredDelay = (
  index: number,
  totalItems: number,
  totalDuration: number = 1000
): number => {
  return (index / totalItems) * (totalDuration / 2);
};

/**
 * Easing functions
 */
export const easings = {
  linear: (t: number) => t,
  easeIn: (t: number) => t * t,
  easeOut: (t: number) => t * (2 - t),
  easeInOut: (t: number) => (t < 0.5 ? 2 * t * t : -1 + (4 - 2 * t) * t),
  bounce: (t: number) => {
    if (t < 1 / 2.75) return 7.5625 * t * t;
    if (t < 2 / 2.75) return 7.5625 * (t -= 1.5 / 2.75) * t + 0.75;
    if (t < 2.5 / 2.75) return 7.5625 * (t -= 2.25 / 2.75) * t + 0.9375;
    return 7.5625 * (t -= 2.625 / 2.75) * t + 0.984375;
  },
};

// ============================================
// Export Utilities
// ============================================

/**
 * Download data as CSV
 */
export const downloadCSV = (
  labels: string[],
  datasets: DatasetConfig[],
  filename: string = 'chart-data.csv'
): void => {
  // Build CSV header
  const headers = ['Label', ...datasets.map((d) => d.label)];

  // Build CSV rows
  const rows = labels.map((label, index) => [
    label,
    ...datasets.map((d) => (d.data as number[])[index]),
  ]);

  // Combine
  const csv = [headers, ...rows].map((row) => row.join(',')).join('\n');

  // Download
  downloadCSVFile(csv, filename);
};

/**
 * Download data as JSON
 */
export const downloadJSON = (
  labels: string[],
  datasets: DatasetConfig[],
  filename: string = 'chart-data.json'
): void => {
  const data = {
    labels,
    datasets: datasets.map((d) => ({
      label: d.label,
      data: d.data,
    })),
  };

  downloadJSONFile(data, filename);
};

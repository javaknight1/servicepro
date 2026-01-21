import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Import chart components
import LineChart from '../LineChart';
import BarChart, { HorizontalBarChart, StackedBarChart } from '../BarChart';
import PieChart, { DoughnutChart } from '../PieChart';
import AreaChart, { StackedAreaChart } from '../AreaChart';
import { ChartSkeleton, ChartError } from '../BaseChart';

// Import utilities
import {
  generateRandomData,
  generateTrendingData,
  generateSeasonalData,
  generateTimeLabels,
  generateMockDataset,
  generateMockChartData,
  calculateChange,
  normalizeData,
  movingAverage,
  cumulativeSum,
  groupDataByCategory,
  createAxisFormatter,
  downloadCSV,
  downloadJSON,
} from '../chartUtils';

import {
  formatNumber,
  formatCurrency,
  formatPercentage,
  formatCompact,
  getColorWithOpacity,
  defaultColorPalette,
} from '../../../types/chart';

// Mock Chart.js
vi.mock('chart.js', async () => {
  const actual = await vi.importActual('chart.js');
  return {
    ...actual,
    Chart: {
      register: vi.fn(),
    },
  };
});

// Mock react-chartjs-2
vi.mock('react-chartjs-2', () => ({
  Chart: vi.fn(({ data, options, type }) => (
    <div data-testid="mock-chart" data-type={type}>
      <span data-testid="chart-labels">{JSON.stringify(data.labels)}</span>
      <span data-testid="chart-datasets">{JSON.stringify(data.datasets)}</span>
    </div>
  )),
}));

// ============================================
// Chart Component Tests
// ============================================

describe('LineChart', () => {
  const defaultProps = {
    labels: ['Jan', 'Feb', 'Mar', 'Apr', 'May'],
    datasets: [
      { label: 'Revenue', data: [100, 200, 150, 300, 250] },
      { label: 'Costs', data: [80, 90, 100, 120, 110] },
    ],
  };

  it('should render with required props', () => {
    render(<LineChart {...defaultProps} />);
    expect(screen.getByTestId('chart-container')).toBeInTheDocument();
  });

  it('should render with title', () => {
    render(<LineChart {...defaultProps} title="Monthly Performance" />);
    expect(screen.getByTestId('mock-chart')).toBeInTheDocument();
  });

  it('should show loading skeleton when loading', () => {
    render(<LineChart {...defaultProps} loading />);
    expect(screen.getByTestId('chart-skeleton')).toBeInTheDocument();
  });

  it('should show error state when error prop is provided', () => {
    render(<LineChart {...defaultProps} error="Failed to load data" />);
    expect(screen.getByTestId('chart-error')).toBeInTheDocument();
    expect(screen.getByText('Failed to load data')).toBeInTheDocument();
  });

  it('should pass correct data structure', () => {
    render(<LineChart {...defaultProps} />);
    const datasetsEl = screen.getByTestId('chart-datasets');
    const datasets = JSON.parse(datasetsEl.textContent || '[]');
    expect(datasets).toHaveLength(2);
    expect(datasets[0].label).toBe('Revenue');
  });

  it('should apply tension to lines', () => {
    render(<LineChart {...defaultProps} tension={0.5} />);
    const datasetsEl = screen.getByTestId('chart-datasets');
    const datasets = JSON.parse(datasetsEl.textContent || '[]');
    expect(datasets[0].tension).toBe(0.5);
  });

  it('should set fill option', () => {
    render(<LineChart {...defaultProps} fill />);
    const datasetsEl = screen.getByTestId('chart-datasets');
    const datasets = JSON.parse(datasetsEl.textContent || '[]');
    expect(datasets[0].fill).toBe(true);
  });

  it('should hide points when showPoints is false', () => {
    render(<LineChart {...defaultProps} showPoints={false} />);
    const datasetsEl = screen.getByTestId('chart-datasets');
    const datasets = JSON.parse(datasetsEl.textContent || '[]');
    expect(datasets[0].pointRadius).toBe(0);
  });
});

describe('BarChart', () => {
  const defaultProps = {
    labels: ['Q1', 'Q2', 'Q3', 'Q4'],
    datasets: [{ label: 'Sales', data: [1200, 1900, 3000, 5000] }],
  };

  it('should render with required props', () => {
    render(<BarChart {...defaultProps} />);
    expect(screen.getByTestId('chart-container')).toBeInTheDocument();
  });

  it('should render as horizontal bar chart', () => {
    render(<HorizontalBarChart {...defaultProps} />);
    expect(screen.getByTestId('mock-chart')).toBeInTheDocument();
  });

  it('should render as stacked bar chart', () => {
    const stackedProps = {
      ...defaultProps,
      datasets: [
        { label: 'Product A', data: [100, 200, 150, 250] },
        { label: 'Product B', data: [80, 120, 100, 150] },
      ],
    };
    render(<StackedBarChart {...stackedProps} />);
    expect(screen.getByTestId('mock-chart')).toBeInTheDocument();
  });

  it('should apply border radius to bars', () => {
    render(<BarChart {...defaultProps} borderRadius={8} />);
    const datasetsEl = screen.getByTestId('chart-datasets');
    const datasets = JSON.parse(datasetsEl.textContent || '[]');
    expect(datasets[0].borderRadius).toBe(8);
  });

  it('should limit bar thickness', () => {
    render(<BarChart {...defaultProps} maxBarThickness={40} />);
    const datasetsEl = screen.getByTestId('chart-datasets');
    const datasets = JSON.parse(datasetsEl.textContent || '[]');
    expect(datasets[0].maxBarThickness).toBe(40);
  });
});

describe('PieChart', () => {
  const defaultProps = {
    labels: ['Desktop', 'Mobile', 'Tablet'],
    data: [60, 30, 10],
  };

  it('should render with required props', () => {
    render(<PieChart {...defaultProps} />);
    expect(screen.getByTestId('chart-container')).toBeInTheDocument();
  });

  it('should render as doughnut chart', () => {
    render(<DoughnutChart {...defaultProps} />);
    expect(screen.getByTestId('mock-chart')).toHaveAttribute(
      'data-type',
      'doughnut'
    );
  });

  it('should apply custom colors', () => {
    const colors = ['#ff0000', '#00ff00', '#0000ff'];
    render(<PieChart {...defaultProps} colors={colors} />);
    const datasetsEl = screen.getByTestId('chart-datasets');
    const datasets = JSON.parse(datasetsEl.textContent || '[]');
    expect(datasets[0].backgroundColor).toEqual(colors);
  });

  it('should set cutout for doughnut', () => {
    render(<PieChart {...defaultProps} doughnut cutout="70%" />);
    expect(screen.getByTestId('mock-chart')).toBeInTheDocument();
  });
});

describe('AreaChart', () => {
  const defaultProps = {
    labels: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri'],
    datasets: [{ label: 'Visitors', data: [1200, 1900, 1600, 2100, 1800] }],
  };

  it('should render with required props', () => {
    render(<AreaChart {...defaultProps} />);
    expect(screen.getByTestId('chart-container')).toBeInTheDocument();
  });

  it('should render as stacked area chart', () => {
    const stackedProps = {
      ...defaultProps,
      datasets: [
        { label: 'Organic', data: [500, 700, 600, 800, 750] },
        { label: 'Paid', data: [700, 1200, 1000, 1300, 1050] },
      ],
    };
    render(<StackedAreaChart {...stackedProps} />);
    expect(screen.getByTestId('mock-chart')).toBeInTheDocument();
  });

  it('should set fill to true by default', () => {
    render(<AreaChart {...defaultProps} />);
    const datasetsEl = screen.getByTestId('chart-datasets');
    const datasets = JSON.parse(datasetsEl.textContent || '[]');
    expect(datasets[0].fill).toBe(true);
  });
});

// ============================================
// Chart Skeleton & Error Tests
// ============================================

describe('ChartSkeleton', () => {
  it('should render with animation class', () => {
    render(<ChartSkeleton />);
    const skeleton = screen.getByTestId('chart-skeleton');
    expect(skeleton).toHaveClass('animate-pulse');
  });

  it('should apply custom height', () => {
    render(<ChartSkeleton height={400} />);
    const skeleton = screen.getByTestId('chart-skeleton');
    expect(skeleton).toHaveStyle({ height: '400px' });
  });
});

describe('ChartError', () => {
  it('should render error message', () => {
    render(<ChartError error="Test error" />);
    expect(screen.getByText('Test error')).toBeInTheDocument();
  });

  it('should show retry button when onRetry is provided', () => {
    const onRetry = vi.fn();
    render(<ChartError error="Error" onRetry={onRetry} />);

    const retryButton = screen.getByText('Try again');
    fireEvent.click(retryButton);

    expect(onRetry).toHaveBeenCalledTimes(1);
  });
});

// ============================================
// Data Generator Tests
// ============================================

describe('Data Generators', () => {
  describe('generateRandomData', () => {
    it('should generate array of specified length', () => {
      const data = generateRandomData(10, 0, 100);
      expect(data).toHaveLength(10);
    });

    it('should generate values within range', () => {
      const data = generateRandomData(100, 50, 100);
      data.forEach((value) => {
        expect(value).toBeGreaterThanOrEqual(50);
        expect(value).toBeLessThanOrEqual(100);
      });
    });
  });

  describe('generateTrendingData', () => {
    it('should generate upward trending data', () => {
      const data = generateTrendingData(10, 100, 'up', 0);
      const first = data[0];
      const last = data[data.length - 1];
      expect(last).toBeGreaterThan(first);
    });

    it('should generate downward trending data', () => {
      const data = generateTrendingData(10, 100, 'down', 0);
      const first = data[0];
      const last = data[data.length - 1];
      expect(last).toBeLessThan(first);
    });
  });

  describe('generateSeasonalData', () => {
    it('should generate 12 months of data by default', () => {
      const data = generateSeasonalData();
      expect(data).toHaveLength(12);
    });

    it('should generate values around base value', () => {
      const baseValue = 1000;
      const data = generateSeasonalData(12, baseValue, 0.2);
      const avg = data.reduce((sum, val) => sum + val, 0) / data.length;
      expect(avg).toBeGreaterThan(baseValue * 0.7);
      expect(avg).toBeLessThan(baseValue * 1.3);
    });
  });

  describe('generateTimeLabels', () => {
    it('should generate month labels', () => {
      const labels = generateTimeLabels(6, 'months');
      expect(labels).toHaveLength(6);
    });

    it('should generate day labels', () => {
      const labels = generateTimeLabels(7, 'days');
      expect(labels).toHaveLength(7);
    });
  });

  describe('generateMockDataset', () => {
    it('should create dataset with label', () => {
      const dataset = generateMockDataset('Test', 5);
      expect(dataset.label).toBe('Test');
      expect(dataset.data).toHaveLength(5);
    });

    it('should apply custom color', () => {
      const dataset = generateMockDataset('Test', 5, { color: '#ff0000' });
      expect(dataset.borderColor).toBe('#ff0000');
    });
  });

  describe('generateMockChartData', () => {
    it('should generate complete chart data', () => {
      const { labels, datasets } = generateMockChartData(3, 12);
      expect(labels).toHaveLength(12);
      expect(datasets).toHaveLength(3);
    });
  });
});

// ============================================
// Data Transformation Tests
// ============================================

describe('Data Transformations', () => {
  describe('calculateChange', () => {
    it('should calculate positive change', () => {
      const result = calculateChange(150, 100);
      expect(result.value).toBe(50);
      expect(result.percentage).toBe(50);
      expect(result.isPositive).toBe(true);
    });

    it('should calculate negative change', () => {
      const result = calculateChange(80, 100);
      expect(result.value).toBe(-20);
      expect(result.percentage).toBe(-20);
      expect(result.isPositive).toBe(false);
    });
  });

  describe('normalizeData', () => {
    it('should normalize data to 0-100 range', () => {
      const data = [10, 50, 100];
      const normalized = normalizeData(data);
      expect(normalized[0]).toBe(0);
      expect(normalized[2]).toBe(100);
    });
  });

  describe('movingAverage', () => {
    it('should calculate moving average', () => {
      const data = [10, 20, 30, 40, 50];
      const ma = movingAverage(data, 3);
      expect(ma[2]).toBe(20); // (10 + 20 + 30) / 3
      expect(ma[4]).toBe(40); // (30 + 40 + 50) / 3
    });
  });

  describe('cumulativeSum', () => {
    it('should calculate cumulative sum', () => {
      const data = [10, 20, 30];
      const cumSum = cumulativeSum(data);
      expect(cumSum).toEqual([10, 30, 60]);
    });
  });

  describe('groupDataByCategory', () => {
    it('should group items by category', () => {
      const items = [
        { category: 'A', value: 10 },
        { category: 'B', value: 20 },
        { category: 'A', value: 15 },
      ];
      const result = groupDataByCategory(items);
      expect(result.labels).toContain('A');
      expect(result.labels).toContain('B');
      expect(result.data[result.labels.indexOf('A')]).toBe(25);
    });
  });
});

// ============================================
// Formatting Tests
// ============================================

describe('Formatting Functions', () => {
  describe('formatNumber', () => {
    it('should format number with commas', () => {
      expect(formatNumber(1234567)).toBe('1,234,567');
    });
  });

  describe('formatCurrency', () => {
    it('should format as USD by default', () => {
      const result = formatCurrency(1234);
      expect(result).toContain('$');
      expect(result).toContain('1,234');
    });
  });

  describe('formatPercentage', () => {
    it('should format as percentage', () => {
      expect(formatPercentage(25.5, 1)).toBe('25.5%');
    });
  });

  describe('formatCompact', () => {
    it('should format large numbers compactly', () => {
      const result = formatCompact(1500000);
      expect(result).toMatch(/1\.5M|1,500K/);
    });
  });

  describe('createAxisFormatter', () => {
    it('should create currency formatter', () => {
      const formatter = createAxisFormatter('currency');
      expect(formatter(1000)).toContain('$');
    });

    it('should create percentage formatter', () => {
      const formatter = createAxisFormatter('percentage');
      expect(formatter(50)).toContain('%');
    });
  });
});

// ============================================
// Color Utility Tests
// ============================================

describe('Color Utilities', () => {
  describe('getColorWithOpacity', () => {
    it('should convert rgb to rgba', () => {
      const result = getColorWithOpacity('rgb(255, 0, 0)', 0.5);
      expect(result).toBe('rgba(255, 0, 0, 0.5)');
    });

    it('should update rgba opacity', () => {
      const result = getColorWithOpacity('rgba(255, 0, 0, 1)', 0.3);
      expect(result).toBe('rgba(255, 0, 0, 0.3)');
    });
  });

  describe('defaultColorPalette', () => {
    it('should have multiple colors', () => {
      expect(defaultColorPalette.length).toBeGreaterThan(5);
    });
  });
});

// ============================================
// Export Utility Tests
// ============================================

describe('Export Utilities', () => {
  let createObjectURLMock: ReturnType<typeof vi.fn>;
  let revokeObjectURLMock: ReturnType<typeof vi.fn>;
  let clickMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    createObjectURLMock = vi.fn(() => 'blob:test');
    revokeObjectURLMock = vi.fn();
    clickMock = vi.fn();

    global.URL.createObjectURL = createObjectURLMock;
    global.URL.revokeObjectURL = revokeObjectURLMock;

    vi.spyOn(document, 'createElement').mockImplementation((tag) => {
      if (tag === 'a') {
        return {
          click: clickMock,
          href: '',
          download: '',
        } as any;
      }
      return document.createElement(tag);
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('downloadCSV', () => {
    it('should create CSV blob and trigger download', () => {
      const labels = ['A', 'B', 'C'];
      const datasets = [{ label: 'Data', data: [1, 2, 3] }];

      downloadCSV(labels, datasets, 'test.csv');

      expect(createObjectURLMock).toHaveBeenCalled();
      expect(clickMock).toHaveBeenCalled();
      expect(revokeObjectURLMock).toHaveBeenCalled();
    });
  });

  describe('downloadJSON', () => {
    it('should create JSON blob and trigger download', () => {
      const labels = ['A', 'B', 'C'];
      const datasets = [{ label: 'Data', data: [1, 2, 3] }];

      downloadJSON(labels, datasets, 'test.json');

      expect(createObjectURLMock).toHaveBeenCalled();
      expect(clickMock).toHaveBeenCalled();
      expect(revokeObjectURLMock).toHaveBeenCalled();
    });
  });
});

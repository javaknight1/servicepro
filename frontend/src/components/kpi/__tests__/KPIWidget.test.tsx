import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import KPIWidget, {
  KPIWidgetSkeleton,
  KPIWidgetError,
  Sparkline,
  TargetProgress,
} from '../KPIWidget';
import { KPIData, KPITargetProgress } from '../../../types/kpi';

describe('KPIWidget', () => {
  const mockKPI: KPIData = {
    id: 'test-kpi-1',
    label: 'Total Revenue',
    value: 125000,
    previousValue: 100000,
    format: 'currency',
    currency: 'USD',
    description: 'Total revenue this month',
    status: 'success',
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Basic Rendering', () => {
    it('should render the KPI label and value', () => {
      render(<KPIWidget data={mockKPI} />);

      expect(screen.getByText('Total Revenue')).toBeInTheDocument();
      expect(screen.getByTestId('kpi-value')).toBeInTheDocument();
    });

    it('should format currency values correctly', () => {
      render(<KPIWidget data={mockKPI} />);

      const valueElement = screen.getByTestId('kpi-value');
      expect(valueElement.textContent).toContain('$');
      expect(valueElement.textContent).toContain('125,000');
    });

    it('should format percentage values correctly', () => {
      const percentageKPI: KPIData = {
        ...mockKPI,
        id: 'percentage-kpi',
        label: 'Conversion Rate',
        value: 25.5,
        format: 'percentage',
      };

      render(<KPIWidget data={percentageKPI} />);

      const valueElement = screen.getByTestId('kpi-value');
      expect(valueElement.textContent).toContain('%');
    });

    it('should format compact numbers correctly', () => {
      const compactKPI: KPIData = {
        ...mockKPI,
        id: 'compact-kpi',
        label: 'Page Views',
        value: 1500000,
        format: 'compact',
      };

      render(<KPIWidget data={compactKPI} />);

      const valueElement = screen.getByTestId('kpi-value');
      expect(valueElement.textContent).toMatch(/1\.5M|1,500K/);
    });

    it('should format duration values correctly', () => {
      const durationKPI: KPIData = {
        ...mockKPI,
        id: 'duration-kpi',
        label: 'Average Response Time',
        value: 125, // 2h 5m
        format: 'duration',
      };

      render(<KPIWidget data={durationKPI} />);

      const valueElement = screen.getByTestId('kpi-value');
      expect(valueElement.textContent).toContain('2h 5m');
    });

    it('should apply prefix and suffix correctly', () => {
      const prefixSuffixKPI: KPIData = {
        ...mockKPI,
        id: 'prefix-suffix-kpi',
        value: 45,
        format: 'number',
        prefix: '~',
        suffix: ' users',
      };

      render(<KPIWidget data={prefixSuffixKPI} />);

      const valueElement = screen.getByTestId('kpi-value');
      expect(valueElement.textContent).toContain('~');
      expect(valueElement.textContent).toContain('users');
    });
  });

  describe('Trend Indicator', () => {
    it('should show trend indicator when showTrend is true', () => {
      render(<KPIWidget data={mockKPI} showTrend />);

      expect(screen.getByTestId('kpi-trend')).toBeInTheDocument();
    });

    it('should not show trend indicator when showTrend is false', () => {
      render(<KPIWidget data={mockKPI} showTrend={false} />);

      expect(screen.queryByTestId('kpi-trend')).not.toBeInTheDocument();
    });

    it('should calculate trend percentage correctly', () => {
      render(<KPIWidget data={mockKPI} showTrend />);

      const trendElement = screen.getByTestId('kpi-trend');
      // 125000 vs 100000 = +25%
      expect(trendElement.textContent).toContain('+25');
    });

    it('should show green for upward trend', () => {
      render(<KPIWidget data={mockKPI} showTrend />);

      const trendElement = screen.getByTestId('kpi-trend');
      expect(trendElement).toHaveClass('text-green-600');
    });

    it('should show red for downward trend', () => {
      const downTrendKPI: KPIData = {
        ...mockKPI,
        value: 80000,
        previousValue: 100000,
      };

      render(<KPIWidget data={downTrendKPI} showTrend />);

      const trendElement = screen.getByTestId('kpi-trend');
      expect(trendElement).toHaveClass('text-red-600');
    });

    it('should not show trend when previousValue is undefined', () => {
      const noPrevKPI: KPIData = {
        ...mockKPI,
        previousValue: undefined,
      };

      render(<KPIWidget data={noPrevKPI} showTrend />);

      expect(screen.queryByTestId('kpi-trend')).not.toBeInTheDocument();
    });
  });

  describe('Loading State', () => {
    it('should show skeleton when loading', () => {
      render(<KPIWidget data={mockKPI} loading />);

      expect(screen.getByTestId('kpi-skeleton')).toBeInTheDocument();
      expect(screen.queryByTestId('kpi-widget')).not.toBeInTheDocument();
    });

    it('should not show skeleton when not loading', () => {
      render(<KPIWidget data={mockKPI} loading={false} />);

      expect(screen.queryByTestId('kpi-skeleton')).not.toBeInTheDocument();
      expect(screen.getByTestId('kpi-widget')).toBeInTheDocument();
    });
  });

  describe('Error State', () => {
    it('should show error message when error is provided', () => {
      render(<KPIWidget data={mockKPI} error="Failed to load data" />);

      expect(screen.getByTestId('kpi-error')).toBeInTheDocument();
      expect(screen.getByText('Failed to load data')).toBeInTheDocument();
    });

    it('should not show widget when error is present', () => {
      render(<KPIWidget data={mockKPI} error="Error" />);

      expect(screen.queryByTestId('kpi-widget')).not.toBeInTheDocument();
    });
  });

  describe('Size Variants', () => {
    it('should apply small size classes', () => {
      render(<KPIWidget data={mockKPI} size="sm" />);

      const widget = screen.getByTestId('kpi-widget');
      expect(widget).toHaveClass('p-3');
    });

    it('should apply medium size classes by default', () => {
      render(<KPIWidget data={mockKPI} />);

      const widget = screen.getByTestId('kpi-widget');
      expect(widget).toHaveClass('p-4');
    });

    it('should apply large size classes', () => {
      render(<KPIWidget data={mockKPI} size="lg" />);

      const widget = screen.getByTestId('kpi-widget');
      expect(widget).toHaveClass('p-5');
    });

    it('should apply extra large size classes', () => {
      render(<KPIWidget data={mockKPI} size="xl" />);

      const widget = screen.getByTestId('kpi-widget');
      expect(widget).toHaveClass('p-6');
    });
  });

  describe('Interactivity', () => {
    it('should call onClick when clicked', () => {
      const handleClick = vi.fn();
      render(<KPIWidget data={mockKPI} onClick={handleClick} />);

      fireEvent.click(screen.getByTestId('kpi-widget'));

      expect(handleClick).toHaveBeenCalledTimes(1);
    });

    it('should have cursor-pointer class when onClick is provided', () => {
      const handleClick = vi.fn();
      render(<KPIWidget data={mockKPI} onClick={handleClick} />);

      expect(screen.getByTestId('kpi-widget')).toHaveClass('cursor-pointer');
    });

    it('should be keyboard accessible when clickable', () => {
      const handleClick = vi.fn();
      render(<KPIWidget data={mockKPI} onClick={handleClick} />);

      const widget = screen.getByTestId('kpi-widget');
      expect(widget).toHaveAttribute('tabIndex', '0');
      expect(widget).toHaveAttribute('role', 'button');

      fireEvent.keyDown(widget, { key: 'Enter' });
      expect(handleClick).toHaveBeenCalledTimes(1);

      fireEvent.keyDown(widget, { key: ' ' });
      expect(handleClick).toHaveBeenCalledTimes(2);
    });
  });

  describe('Target Progress', () => {
    it('should show target progress when showTarget is true', () => {
      const targetKPI: KPIData = {
        ...mockKPI,
        target: 150000,
      };

      render(<KPIWidget data={targetKPI} showTarget />);

      expect(screen.getByTestId('kpi-target-progress')).toBeInTheDocument();
    });

    it('should not show target progress when target is undefined', () => {
      render(<KPIWidget data={mockKPI} showTarget />);

      expect(
        screen.queryByTestId('kpi-target-progress')
      ).not.toBeInTheDocument();
    });

    it('should calculate progress percentage correctly', () => {
      const targetKPI: KPIData = {
        ...mockKPI,
        value: 75000,
        target: 100000,
      };

      render(<KPIWidget data={targetKPI} showTarget />);

      expect(screen.getByText('75%')).toBeInTheDocument();
    });
  });

  describe('Sparkline', () => {
    it('should show sparkline when showSparkline is true and data is provided', () => {
      const sparklineData = [100, 120, 115, 130, 125, 140, 135];

      render(
        <KPIWidget data={mockKPI} showSparkline sparklineData={sparklineData} />
      );

      expect(screen.getByTestId('kpi-sparkline')).toBeInTheDocument();
    });

    it('should not show sparkline when data is insufficient', () => {
      render(<KPIWidget data={mockKPI} showSparkline sparklineData={[100]} />);

      expect(screen.queryByTestId('kpi-sparkline')).not.toBeInTheDocument();
    });

    it('should not show sparkline when showSparkline is false', () => {
      const sparklineData = [100, 120, 115, 130, 125, 140, 135];

      render(
        <KPIWidget
          data={mockKPI}
          showSparkline={false}
          sparklineData={sparklineData}
        />
      );

      expect(screen.queryByTestId('kpi-sparkline')).not.toBeInTheDocument();
    });
  });

  describe('Status', () => {
    it('should apply success status styling', () => {
      const successKPI: KPIData = { ...mockKPI, status: 'success' };

      render(<KPIWidget data={successKPI} />);

      const widget = screen.getByTestId('kpi-widget');
      expect(widget).toHaveClass('bg-green-50');
    });

    it('should apply warning status styling', () => {
      const warningKPI: KPIData = { ...mockKPI, status: 'warning' };

      render(<KPIWidget data={warningKPI} />);

      const widget = screen.getByTestId('kpi-widget');
      expect(widget).toHaveClass('bg-amber-50');
    });

    it('should apply error status styling', () => {
      const errorKPI: KPIData = { ...mockKPI, status: 'error' };

      render(<KPIWidget data={errorKPI} />);

      const widget = screen.getByTestId('kpi-widget');
      expect(widget).toHaveClass('bg-red-50');
    });
  });

  describe('Last Updated', () => {
    it('should show last updated timestamp when provided', () => {
      const lastUpdatedKPI: KPIData = {
        ...mockKPI,
        lastUpdated: '2024-01-15T10:30:00Z',
      };

      render(<KPIWidget data={lastUpdatedKPI} />);

      expect(screen.getByTestId('kpi-last-updated')).toBeInTheDocument();
      expect(screen.getByText(/Updated/)).toBeInTheDocument();
    });

    it('should not show last updated when not provided', () => {
      render(<KPIWidget data={mockKPI} />);

      expect(screen.queryByTestId('kpi-last-updated')).not.toBeInTheDocument();
    });
  });

  describe('Custom className', () => {
    it('should apply custom className', () => {
      render(<KPIWidget data={mockKPI} className="custom-class" />);

      expect(screen.getByTestId('kpi-widget')).toHaveClass('custom-class');
    });
  });

  describe('Compact Mode', () => {
    it('should not show label in compact mode', () => {
      render(<KPIWidget data={mockKPI} compact />);

      expect(screen.queryByText('Total Revenue')).not.toBeInTheDocument();
    });
  });
});

describe('KPIWidgetSkeleton', () => {
  it('should render skeleton with animation', () => {
    render(<KPIWidgetSkeleton />);

    const skeleton = screen.getByTestId('kpi-skeleton');
    expect(skeleton).toHaveClass('animate-pulse');
  });

  it('should apply size classes', () => {
    render(<KPIWidgetSkeleton size="lg" />);

    const skeleton = screen.getByTestId('kpi-skeleton');
    expect(skeleton).toHaveClass('h-32');
  });
});

describe('KPIWidgetError', () => {
  it('should render error message', () => {
    render(<KPIWidgetError error="Test error message" />);

    expect(screen.getByText('Test error message')).toBeInTheDocument();
  });

  it('should show retry button when onRetry is provided', () => {
    const handleRetry = vi.fn();
    render(<KPIWidgetError error="Error" onRetry={handleRetry} />);

    const retryButton = screen.getByText('Try again');
    expect(retryButton).toBeInTheDocument();

    fireEvent.click(retryButton);
    expect(handleRetry).toHaveBeenCalledTimes(1);
  });

  it('should not show retry button when onRetry is not provided', () => {
    render(<KPIWidgetError error="Error" />);

    expect(screen.queryByText('Try again')).not.toBeInTheDocument();
  });
});

describe('Sparkline', () => {
  it('should render SVG with correct path', () => {
    const data = [10, 20, 15, 25, 30];
    render(<Sparkline data={data} />);

    const svg = screen.getByTestId('kpi-sparkline');
    expect(svg.tagName.toLowerCase()).toBe('svg');
    expect(svg.querySelector('polyline')).toBeInTheDocument();
    expect(svg.querySelector('circle')).toBeInTheDocument();
  });

  it('should use green color for upward trend', () => {
    const data = [10, 20, 15, 25, 30]; // Last > second-to-last
    render(<Sparkline data={data} />);

    const polyline = screen
      .getByTestId('kpi-sparkline')
      .querySelector('polyline');
    expect(polyline).toHaveAttribute('stroke', '#22c55e'); // green
  });

  it('should use red color for downward trend', () => {
    const data = [30, 25, 20, 15, 10]; // Last < second-to-last
    render(<Sparkline data={data} />);

    const polyline = screen
      .getByTestId('kpi-sparkline')
      .querySelector('polyline');
    expect(polyline).toHaveAttribute('stroke', '#ef4444'); // red
  });

  it('should return null for insufficient data', () => {
    const { container } = render(<Sparkline data={[10]} />);

    expect(container.firstChild).toBeNull();
  });
});

describe('TargetProgress', () => {
  it('should render progress bar', () => {
    const progress: KPITargetProgress = {
      percentage: 75,
      isOnTrack: true,
      remaining: 25000,
    };

    render(<TargetProgress progress={progress} />);

    expect(screen.getByText('75%')).toBeInTheDocument();
    expect(screen.getByText('Progress')).toBeInTheDocument();
  });

  it('should use green for on-track progress', () => {
    const progress: KPITargetProgress = {
      percentage: 85,
      isOnTrack: true,
      remaining: 15000,
    };

    const { container } = render(<TargetProgress progress={progress} />);

    const progressBar = container.querySelector('.bg-green-500');
    expect(progressBar).toBeInTheDocument();
  });

  it('should use amber for off-track progress', () => {
    const progress: KPITargetProgress = {
      percentage: 50,
      isOnTrack: false,
      remaining: 50000,
    };

    const { container } = render(<TargetProgress progress={progress} />);

    const progressBar = container.querySelector('.bg-amber-500');
    expect(progressBar).toBeInTheDocument();
  });

  it('should cap progress at 100%', () => {
    const progress: KPITargetProgress = {
      percentage: 120,
      isOnTrack: true,
      remaining: -20000,
    };

    const { container } = render(<TargetProgress progress={progress} />);

    const progressBar = container.querySelector('.h-full');
    expect(progressBar).toHaveStyle({ width: '100%' });
  });
});

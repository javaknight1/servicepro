/* eslint-disable react-refresh/only-export-components */
import React, {
  useRef,
  useEffect,
  useCallback,
  forwardRef,
  useImperativeHandle,
  useState,
  memo,
} from 'react';
import {
  Chart as ChartJS,
  ChartData,
  ChartOptions,
  ChartType,
  registerables,
} from 'chart.js';
import { Chart } from 'react-chartjs-2';
import { cn } from '../../utils/cn';
import {
  BaseChartProps,
  ChartRef,
  ExportFormat,
  DatasetConfig,
  animationConfigs,
  themeConfigs,
  LegendPosition,
} from '../../types/chart';

// Register Chart.js components
ChartJS.register(...registerables);

// Loading skeleton component
const ChartSkeleton: React.FC<{ height?: string | number }> = ({
  height = 300,
}) => (
  <div
    className="animate-pulse bg-gray-100 rounded-lg flex items-center justify-center"
    style={{ height: typeof height === 'number' ? `${height}px` : height }}
    data-testid="chart-skeleton"
  >
    <div className="text-gray-400">
      <svg
        className="w-12 h-12"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={1.5}
          d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
        />
      </svg>
    </div>
  </div>
);

// Error state component
const ChartError: React.FC<{ error: string; onRetry?: () => void }> = ({
  error,
  onRetry,
}) => (
  <div
    className="bg-red-50 border border-red-200 rounded-lg p-6 text-center"
    data-testid="chart-error"
  >
    <div className="text-red-600 mb-2">
      <svg
        className="w-8 h-8 mx-auto"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          strokeLinecap="round"
          strokeLinejoin="round"
          strokeWidth={2}
          d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
        />
      </svg>
    </div>
    <p className="text-red-700 font-medium mb-1">Failed to render chart</p>
    <p className="text-red-600 text-sm mb-3">{error}</p>
    {onRetry && (
      <button
        onClick={onRetry}
        className="text-red-700 text-sm underline hover:no-underline"
      >
        Try again
      </button>
    )}
  </div>
);

// Export button component
const ExportButton: React.FC<{
  onExport: (format: ExportFormat) => void;
  disabled?: boolean;
}> = ({ onExport, disabled }) => {
  const [isOpen, setIsOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };
    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const formats: { value: ExportFormat; label: string }[] = [
    { value: 'png', label: 'PNG Image' },
    { value: 'jpeg', label: 'JPEG Image' },
    { value: 'svg', label: 'SVG Vector' },
    { value: 'pdf', label: 'PDF Document' },
  ];

  return (
    <div className="relative inline-block" ref={menuRef}>
      <button
        onClick={() => setIsOpen(!isOpen)}
        disabled={disabled}
        className={cn(
          'flex items-center gap-1 px-2 py-1 text-xs rounded border transition-colors',
          disabled
            ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
            : 'bg-white text-gray-600 border-gray-300 hover:bg-gray-50'
        )}
        data-testid="export-button"
      >
        <svg
          className="w-3.5 h-3.5"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
          />
        </svg>
        Export
      </button>

      {isOpen && (
        <div className="absolute right-0 mt-1 w-36 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-50">
          {formats.map((format) => (
            <button
              key={format.value}
              onClick={() => {
                onExport(format.value);
                setIsOpen(false);
              }}
              className="w-full px-3 py-1.5 text-left text-sm text-gray-700 hover:bg-gray-100"
            >
              {format.label}
            </button>
          ))}
        </div>
      )}
    </div>
  );
};

// Legend position mapping
const getLegendConfig = (position: LegendPosition) => {
  if (position === 'hidden') {
    return { display: false };
  }
  return {
    display: true,
    position: position as 'top' | 'bottom' | 'left' | 'right',
  };
};

// Props for the internal base chart
interface InternalBaseChartProps extends BaseChartProps {
  type: ChartType;
  data: ChartData;
  options?: ChartOptions;
}

// Base Chart Component
const BaseChart = forwardRef<ChartRef, InternalBaseChartProps>(
  (
    {
      type,
      data,
      options: customOptions,
      title,
      subtitle,
      width,
      height = 300,
      maintainAspectRatio = true,
      aspectRatio = 2,
      legendPosition = 'top',
      showGrid = true,
      showXAxis = true,
      showYAxis = true,
      animation = 'default',
      animationDuration,
      enableTooltips = true,
      tooltipMode = 'index',
      theme = 'light',
      loading = false,
      error = null,
      className,
      enableExport = false,
      plugins: customPlugins = [],
      onReady,
      onClick,
      onHover,
      ariaLabel,
    },
    ref
  ) => {
    const chartRef = useRef<ChartJS | null>(null);
    const containerRef = useRef<HTMLDivElement>(null);
    const [chartError, setChartError] = useState<string | null>(null);

    // Get theme colors
    const themeColors = themeConfigs[theme];

    // Build chart options
    const chartOptions: ChartOptions = {
      responsive: true,
      maintainAspectRatio,
      aspectRatio,
      animation: animationDuration
        ? { duration: animationDuration }
        : animationConfigs[animation],
      interaction: {
        mode: tooltipMode,
        intersect: false,
      },
      plugins: {
        legend: getLegendConfig(legendPosition),
        tooltip: {
          enabled: enableTooltips,
          backgroundColor: themeColors.background,
          titleColor: themeColors.text,
          bodyColor: themeColors.text,
          borderColor: themeColors.grid,
          borderWidth: 1,
          padding: 12,
          cornerRadius: 8,
          displayColors: true,
        },
        title: {
          display: !!title,
          text: title || '',
          color: themeColors.text,
          font: {
            size: 16,
            weight: 'bold',
          },
          padding: { bottom: subtitle ? 0 : 16 },
        },
        subtitle: {
          display: !!subtitle,
          text: subtitle || '',
          color: theme === 'dark' ? 'rgb(156, 163, 175)' : 'rgb(107, 114, 128)',
          font: {
            size: 12,
          },
          padding: { bottom: 16 },
        },
      },
      scales:
        type !== 'pie' && type !== 'doughnut'
          ? {
              x: {
                display: showXAxis,
                grid: {
                  display: showGrid,
                  color: themeColors.grid,
                },
                ticks: {
                  color: themeColors.text,
                },
              },
              y: {
                display: showYAxis,
                grid: {
                  display: showGrid,
                  color: themeColors.grid,
                },
                ticks: {
                  color: themeColors.text,
                },
              },
            }
          : undefined,
      onClick: onClick
        ? (event, elements, chart) => {
            onClick(event.native as MouseEvent, elements, chart);
          }
        : undefined,
      onHover: onHover
        ? (event, elements, chart) => {
            onHover(event.native as MouseEvent, elements, chart);
          }
        : undefined,
      ...customOptions,
    };

    // Merge custom options deeply
    if (customOptions?.plugins) {
      chartOptions.plugins = {
        ...chartOptions.plugins,
        ...customOptions.plugins,
      };
    }
    if (customOptions?.scales) {
      chartOptions.scales = {
        ...chartOptions.scales,
        ...customOptions.scales,
      };
    }

    // Export chart as image
    const exportAsImage = useCallback(
      async (format: ExportFormat = 'png', filename?: string) => {
        const chart = chartRef.current;
        if (!chart) return;

        const canvas = chart.canvas;
        const defaultFilename = `chart-${Date.now()}`;

        try {
          if (format === 'svg') {
            // For SVG, we need to convert canvas to SVG
            const svgData = canvas.toDataURL('image/svg+xml');
            const link = document.createElement('a');
            link.download = `${filename || defaultFilename}.svg`;
            link.href = svgData;
            link.click();
          } else if (format === 'pdf') {
            // For PDF, we'd typically use a library like jsPDF
            // For now, export as PNG
            const imageData = canvas.toDataURL('image/png', 1.0);
            const link = document.createElement('a');
            link.download = `${filename || defaultFilename}.png`;
            link.href = imageData;
            link.click();
          } else {
            const mimeType = format === 'jpeg' ? 'image/jpeg' : 'image/png';
            const imageData = canvas.toDataURL(mimeType, 1.0);
            const link = document.createElement('a');
            link.download = `${filename || defaultFilename}.${format}`;
            link.href = imageData;
            link.click();
          }
        } catch (err) {
          console.error('Failed to export chart:', err);
          setChartError('Failed to export chart');
        }
      },
      []
    );

    // Imperative handle for parent components
    useImperativeHandle(
      ref,
      () => ({
        getChart: () => chartRef.current,
        exportAsImage,
        resetZoom: () => {
          // Would require zoom plugin
          // eslint-disable-next-line no-console
          console.log('Zoom reset not implemented');
        },
        updateData: (labels: string[], datasets: DatasetConfig[]) => {
          const chart = chartRef.current;
          if (chart) {
            chart.data.labels = labels;
            // eslint-disable-next-line @typescript-eslint/no-explicit-any
            chart.data.datasets = datasets as any;
            chart.update();
          }
        },
        toggleDataset: (index: number) => {
          const chart = chartRef.current;
          if (chart) {
            const meta = chart.getDatasetMeta(index);
            meta.hidden = !meta.hidden;
            chart.update();
          }
        },
        destroy: () => {
          chartRef.current?.destroy();
        },
      }),
      [exportAsImage]
    );

    // Notify when chart is ready
    useEffect(() => {
      if (chartRef.current && onReady) {
        onReady(chartRef.current);
      }
    }, [onReady]);

    // Container style
    const containerStyle: React.CSSProperties = {
      width: typeof width === 'number' ? `${width}px` : width,
      height: typeof height === 'number' ? `${height}px` : height,
    };

    // Render loading state
    if (loading) {
      return <ChartSkeleton height={height} />;
    }

    // Render error state
    if (error || chartError) {
      return <ChartError error={error || chartError || 'Unknown error'} />;
    }

    return (
      <div
        ref={containerRef}
        className={cn(
          'relative',
          theme === 'dark' ? 'bg-gray-900' : 'bg-white',
          className
        )}
        style={containerStyle}
        data-testid="chart-container"
      >
        {/* Export button */}
        {enableExport && (
          <div className="absolute top-2 right-2 z-10">
            <ExportButton
              onExport={exportAsImage}
              disabled={!chartRef.current}
            />
          </div>
        )}

        {/* Chart */}
        <Chart
          ref={(instance) => {
            chartRef.current = instance || null;
          }}
          type={type}
          data={data}
          options={chartOptions}
          plugins={customPlugins}
          aria-label={ariaLabel || title || 'Chart'}
        />
      </div>
    );
  }
);

BaseChart.displayName = 'BaseChart';

export default memo(BaseChart);
export { ChartSkeleton, ChartError, ExportButton, getLegendConfig };

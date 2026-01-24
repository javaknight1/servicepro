/* eslint-disable react-refresh/only-export-components */
import React, { useMemo, forwardRef, memo } from 'react';
import { ChartData, ChartOptions, Plugin } from 'chart.js';
import BaseChart from './BaseChart';
import {
  PieChartProps,
  ChartRef,
  defaultColorPalette,
  getColorWithOpacity,
  formatPercentage,
} from '../../types/chart';

/**
 * PieChart Component
 *
 * A flexible pie/doughnut chart component built on Chart.js with features like:
 * - Pie and doughnut variants
 * - Customizable colors per segment
 * - Center label for doughnut charts
 * - Percentage display in tooltips
 * - Partial pie with custom circumference
 * - Rotation control
 *
 * @example
 * ```tsx
 * <PieChart
 *   labels={['Desktop', 'Mobile', 'Tablet']}
 *   data={[60, 30, 10]}
 *   title="Device Distribution"
 *   doughnut
 *   centerLabel="Total"
 *   centerValue="100%"
 * />
 * ```
 */
const PieChart = forwardRef<ChartRef, PieChartProps>(
  (
    {
      labels,
      data,
      colors = defaultColorPalette,
      doughnut = false,
      cutout = '60%',
      rotation = 0,
      circumference = 360,
      showPercentage = true,
      centerLabel,
      centerValue,
      tooltipFormat,
      ...baseProps
    },
    ref
  ) => {
    // Calculate total for percentages
    const total = useMemo(
      () => data.reduce((sum, val) => sum + val, 0),
      [data]
    );

    // Build chart data
    const chartData: ChartData<'pie' | 'doughnut'> = useMemo(() => {
      const backgroundColors = labels.map(
        (_, index) => colors[index % colors.length]
      );
      const hoverBackgroundColors = backgroundColors.map((color) =>
        getColorWithOpacity(color, 0.9)
      );

      return {
        labels,
        datasets: [
          {
            data,
            backgroundColor: backgroundColors,
            hoverBackgroundColor: hoverBackgroundColors,
            borderColor: '#fff',
            borderWidth: 2,
            hoverBorderWidth: 3,
            hoverOffset: 8,
          },
        ],
      };
    }, [labels, data, colors]);

    // Center label plugin for doughnut charts
    const centerLabelPlugin: Plugin<'doughnut'> = useMemo(
      () => ({
        id: 'centerLabel',
        afterDraw: (chart) => {
          if (!centerLabel && !centerValue) return;
          if (!doughnut) return;

          const { ctx, width, height } = chart;
          ctx.save();

          // Draw center label
          if (centerLabel) {
            ctx.font = '12px Arial';
            ctx.fillStyle = '#6b7280';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText(centerLabel, width / 2, height / 2 - 10);
          }

          // Draw center value
          if (centerValue) {
            ctx.font = 'bold 24px Arial';
            ctx.fillStyle = '#111827';
            ctx.textAlign = 'center';
            ctx.textBaseline = 'middle';
            ctx.fillText(String(centerValue), width / 2, height / 2 + 10);
          }

          ctx.restore();
        },
      }),
      [centerLabel, centerValue, doughnut]
    );

    // Build chart options
    const chartOptions: ChartOptions<'pie' | 'doughnut'> = useMemo(() => {
      const options: ChartOptions<'pie' | 'doughnut'> = {
        cutout: doughnut ? cutout : 0,
        rotation: rotation,
        circumference: circumference,
        plugins: {
          tooltip: {
            callbacks: {
              label: function (context) {
                const value = context.raw as number;
                const label = context.label;
                const percentage = (value / total) * 100;

                if (tooltipFormat) {
                  return tooltipFormat(value, label, percentage);
                }

                if (showPercentage) {
                  return `${label}: ${value.toLocaleString()} (${formatPercentage(
                    percentage
                  )})`;
                }

                return `${label}: ${value.toLocaleString()}`;
              },
            },
          },
        },
      };

      return options;
    }, [
      doughnut,
      cutout,
      rotation,
      circumference,
      total,
      showPercentage,
      tooltipFormat,
    ]);

    // Combine plugins
    const plugins = useMemo(() => {
      const allPlugins = [...(baseProps.plugins || [])];
      if (doughnut && (centerLabel || centerValue)) {
        allPlugins.push(centerLabelPlugin as Plugin);
      }
      return allPlugins;
    }, [
      baseProps.plugins,
      doughnut,
      centerLabel,
      centerValue,
      centerLabelPlugin,
    ]);

    return (
      <BaseChart
        ref={ref}
        type={doughnut ? 'doughnut' : 'pie'}
        data={chartData}
        options={chartOptions}
        {...baseProps}
        plugins={plugins}
        showGrid={false}
        showXAxis={false}
        showYAxis={false}
      />
    );
  }
);

PieChart.displayName = 'PieChart';

export default memo(PieChart);

// Doughnut chart convenience component
export const DoughnutChart = forwardRef<
  ChartRef,
  Omit<PieChartProps, 'doughnut'>
>((props, ref) => {
  return <PieChart ref={ref} {...props} doughnut />;
});

DoughnutChart.displayName = 'DoughnutChart';

// Half doughnut convenience component
export const HalfDoughnutChart = forwardRef<
  ChartRef,
  Omit<PieChartProps, 'doughnut' | 'rotation' | 'circumference'>
>((props, ref) => {
  return (
    <PieChart
      ref={ref}
      {...props}
      doughnut
      rotation={-90}
      circumference={180}
    />
  );
});

HalfDoughnutChart.displayName = 'HalfDoughnutChart';

// Convenience hook for pie chart data
export const usePieChartData = (
  items: { label: string; value: number }[],
  colorPalette = defaultColorPalette
) => {
  return useMemo(() => {
    return {
      labels: items.map((item) => item.label),
      data: items.map((item) => item.value),
      colors: items.map(
        (_, index) => colorPalette[index % colorPalette.length]
      ),
    };
  }, [items, colorPalette]);
};

import React, { useMemo, forwardRef, memo } from 'react';
import { ChartData, ChartOptions } from 'chart.js';
import BaseChart from './BaseChart';
import {
  BarChartProps,
  ChartRef,
  defaultColorPalette,
  getColorWithOpacity,
} from '../../types/chart';

/**
 * BarChart Component
 *
 * A flexible bar chart component built on Chart.js with features like:
 * - Vertical and horizontal orientations
 * - Stacked and grouped bars
 * - Customizable bar appearance
 * - Custom axis formatting
 * - Tooltip customization
 *
 * @example
 * ```tsx
 * <BarChart
 *   labels={['Q1', 'Q2', 'Q3', 'Q4']}
 *   datasets={[
 *     { label: 'Sales', data: [1200, 1900, 3000, 5000] },
 *     { label: 'Costs', data: [900, 1200, 1800, 2500] },
 *   ]}
 *   title="Quarterly Performance"
 *   stacked={false}
 *   enableExport
 * />
 * ```
 */
const BarChart = forwardRef<ChartRef, BarChartProps>(
  (
    {
      labels,
      datasets,
      horizontal = false,
      stacked = false,
      barThickness,
      maxBarThickness = 50,
      borderRadius = 4,
      grouped = true,
      yMin,
      yMax,
      yAxisFormat,
      tooltipFormat,
      ...baseProps
    },
    ref
  ) => {
    // Build chart data
    const chartData: ChartData<'bar'> = useMemo(() => {
      return {
        labels,
        datasets: datasets.map((dataset, index) => {
          const color =
            dataset.borderColor ||
            defaultColorPalette[index % defaultColorPalette.length];
          const bgColor =
            dataset.backgroundColor || getColorWithOpacity(color, 0.8);

          return {
            label: dataset.label,
            data: dataset.data as number[],
            backgroundColor: bgColor,
            borderColor: color,
            borderWidth: dataset.borderWidth ?? 0,
            borderRadius: borderRadius,
            barThickness: barThickness,
            maxBarThickness: maxBarThickness,
            hidden: dataset.hidden,
            order: dataset.order,
            stack: dataset.stack || (stacked ? 'stack' : undefined),
            yAxisID: dataset.yAxisID,
          };
        }),
      };
    }, [
      labels,
      datasets,
      borderRadius,
      barThickness,
      maxBarThickness,
      stacked,
    ]);

    // Build chart options
    const chartOptions: ChartOptions<'bar'> = useMemo(() => {
      const options: ChartOptions<'bar'> = {
        indexAxis: horizontal ? 'y' : 'x',
        scales: {
          x: {
            stacked: stacked,
            grid: {
              display: !horizontal,
            },
          },
          y: {
            stacked: stacked,
            min: yMin,
            max: yMax,
            grid: {
              display: horizontal,
            },
            ticks: yAxisFormat
              ? {
                  callback: function (value) {
                    return yAxisFormat(value as number);
                  },
                }
              : undefined,
          },
        },
        plugins: {
          tooltip: tooltipFormat
            ? {
                callbacks: {
                  label: function (context) {
                    const value = context.parsed[horizontal ? 'x' : 'y'];
                    const label = context.label;
                    const datasetLabel = context.dataset.label || '';
                    return tooltipFormat(value, label, datasetLabel);
                  },
                },
              }
            : undefined,
        },
      };

      return options;
    }, [horizontal, stacked, yMin, yMax, yAxisFormat, tooltipFormat]);

    return (
      <BaseChart
        ref={ref}
        type="bar"
        data={chartData}
        options={chartOptions}
        {...baseProps}
      />
    );
  }
);

BarChart.displayName = 'BarChart';

export default memo(BarChart);

// Horizontal bar chart convenience component
export const HorizontalBarChart = forwardRef<
  ChartRef,
  Omit<BarChartProps, 'horizontal'>
>((props, ref) => {
  return <BarChart ref={ref} {...props} horizontal />;
});

HorizontalBarChart.displayName = 'HorizontalBarChart';

// Stacked bar chart convenience component
export const StackedBarChart = forwardRef<
  ChartRef,
  Omit<BarChartProps, 'stacked'>
>((props, ref) => {
  return <BarChart ref={ref} {...props} stacked />;
});

StackedBarChart.displayName = 'StackedBarChart';

// Convenience hook for bar chart data generation
export const useBarChartData = (
  rawData: { label: string; values: number[] }[],
  labels: string[]
) => {
  return useMemo(() => {
    return {
      labels,
      datasets: rawData.map((item, index) => ({
        label: item.label,
        data: item.values,
        backgroundColor: getColorWithOpacity(
          defaultColorPalette[index % defaultColorPalette.length],
          0.8
        ),
      })),
    };
  }, [rawData, labels]);
};

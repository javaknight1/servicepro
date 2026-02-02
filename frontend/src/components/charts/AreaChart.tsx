/* eslint-disable react-refresh/only-export-components */
import React, { useMemo, forwardRef, memo, useRef, useEffect } from 'react';
import { ChartData, ChartOptions } from 'chart.js';
import BaseChart from './BaseChart';
import {
  AreaChartProps,
  ChartRef,
  defaultColorPalette,
  getColorWithOpacity,
  generateGradient,
} from '../../types/chart';

/**
 * AreaChart Component
 *
 * A flexible area chart component built on Chart.js with features like:
 * - Gradient fills
 * - Stacked areas
 * - Customizable opacity
 * - Smooth line tension
 * - Custom axis formatting
 * - Tooltip customization
 *
 * @example
 * ```tsx
 * <AreaChart
 *   labels={['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun']}
 *   datasets={[
 *     { label: 'Revenue', data: [1000, 1200, 1100, 1400, 1300, 1600] },
 *     { label: 'Costs', data: [800, 750, 900, 850, 950, 900] },
 *   ]}
 *   title="Revenue vs Costs"
 *   stacked
 *   enableExport
 * />
 * ```
 */
const AreaChart = forwardRef<ChartRef, AreaChartProps>(
  (
    {
      labels,
      datasets,
      stacked = false,
      tension = 0.4,
      fillOpacity = 0.3,
      yMin,
      yMax,
      yAxisFormat,
      tooltipFormat,
      ...baseProps
    },
    ref
  ) => {
    // Build chart data with gradients
    const chartData: ChartData<'line'> = useMemo(() => {
      return {
        labels,
        datasets: datasets.map((dataset, index) => {
          const color =
            dataset.borderColor ||
            defaultColorPalette[index % defaultColorPalette.length];

          return {
            label: dataset.label,
            data: dataset.data as number[],
            borderColor: color,
            backgroundColor: getColorWithOpacity(color, fillOpacity),
            borderWidth: dataset.borderWidth ?? 2,
            fill: true,
            tension: dataset.tension ?? tension,
            pointRadius: 0,
            pointHoverRadius: 6,
            pointBackgroundColor: color,
            pointBorderColor: '#fff',
            pointBorderWidth: 2,
            hidden: dataset.hidden,
            order: dataset.order,
            stack: dataset.stack || (stacked ? 'stack' : undefined),
            yAxisID: dataset.yAxisID,
          };
        }),
      };
    }, [labels, datasets, tension, fillOpacity, stacked]);

    // Build chart options
    const chartOptions: ChartOptions<'line'> = useMemo(() => {
      const options: ChartOptions<'line'> = {
        scales: {
          x: {
            grid: {
              display: false,
            },
          },
          y: {
            stacked: stacked,
            min: yMin,
            max: yMax,
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
                    const value = context.parsed.y ?? 0;
                    const label = context.label;
                    const datasetLabel = context.dataset.label || '';
                    return tooltipFormat(value, label, datasetLabel);
                  },
                },
              }
            : undefined,
          filler: {
            propagate: true,
          },
        },
        interaction: {
          mode: 'index',
          intersect: false,
        },
        elements: {
          line: {
            tension: tension,
          },
        },
      };

      return options;
    }, [stacked, yMin, yMax, yAxisFormat, tooltipFormat, tension]);

    return (
      <BaseChart
        ref={ref}
        type="line"
        data={chartData}
        options={chartOptions}
        {...baseProps}
      />
    );
  }
);

AreaChart.displayName = 'AreaChart';

export default memo(AreaChart);

// Stacked area chart convenience component
export const StackedAreaChart = forwardRef<
  ChartRef,
  Omit<AreaChartProps, 'stacked'>
>((props, ref) => {
  return <AreaChart ref={ref} {...props} stacked />;
});

StackedAreaChart.displayName = 'StackedAreaChart';

// Convenience hook for area chart data generation
export const useAreaChartData = (
  rawData: { label: string; values: number[] }[],
  labels: string[],
  fillOpacity = 0.3
) => {
  return useMemo(() => {
    return {
      labels,
      datasets: rawData.map((item, index) => ({
        label: item.label,
        data: item.values,
        borderColor: defaultColorPalette[index % defaultColorPalette.length],
        backgroundColor: getColorWithOpacity(
          defaultColorPalette[index % defaultColorPalette.length],
          fillOpacity
        ),
        fill: true,
      })),
    };
  }, [rawData, labels, fillOpacity]);
};

// Gradient area chart with canvas gradient fills
export const GradientAreaChart = forwardRef<ChartRef, AreaChartProps>(
  (props, ref) => {
    const internalRef = useRef<ChartRef>(null);

    useEffect(() => {
      const chart = internalRef.current?.getChart();
      if (!chart) return;

      const ctx = chart.ctx;
      const chartArea = chart.chartArea;

      if (!chartArea) return;

      // Apply gradients to datasets
      chart.data.datasets.forEach((dataset, index) => {
        const color = defaultColorPalette[index % defaultColorPalette.length];
        const gradient = generateGradient(
          ctx,
          color,
          chartArea.bottom - chartArea.top
        );
        dataset.backgroundColor = gradient;
      });

      chart.update();
    }, []);

    // Combine refs
    React.useImperativeHandle(ref, () => internalRef.current!, []);

    return <AreaChart ref={internalRef} {...props} />;
  }
);

GradientAreaChart.displayName = 'GradientAreaChart';

/* eslint-disable react-refresh/only-export-components */
import React, { useMemo, forwardRef, memo } from 'react';
import { ChartData, ChartOptions } from 'chart.js';
import BaseChart from './BaseChart';
import {
  LineChartProps,
  ChartRef,
  defaultColorPalette,
  getColorWithOpacity,
} from '../../types/chart';

/**
 * LineChart Component
 *
 * A flexible line chart component built on Chart.js with features like:
 * - Multiple datasets support
 * - Customizable line tension and fill
 * - Point styling options
 * - Stepped lines support
 * - Custom axis formatting
 * - Tooltip customization
 *
 * @example
 * ```tsx
 * <LineChart
 *   labels={['Jan', 'Feb', 'Mar', 'Apr']}
 *   datasets={[
 *     { label: 'Revenue', data: [1000, 1200, 1400, 1600] },
 *     { label: 'Expenses', data: [800, 850, 900, 950] },
 *   ]}
 *   title="Monthly Performance"
 *   showLegend
 *   enableExport
 * />
 * ```
 */
const LineChart = forwardRef<ChartRef, LineChartProps>(
  (
    {
      labels,
      datasets,
      tension = 0.4,
      fill = false,
      showPoints = true,
      pointStyle = 'circle',
      stepped = false,
      yMin,
      yMax,
      yAxisFormat,
      xAxisFormat,
      tooltipFormat,
      ...baseProps
    },
    ref
  ) => {
    // Build chart data
    const chartData: ChartData<'line'> = useMemo(() => {
      return {
        labels,
        datasets: datasets.map((dataset, index) => {
          const color =
            dataset.borderColor ||
            defaultColorPalette[index % defaultColorPalette.length];
          const bgColor =
            dataset.backgroundColor || getColorWithOpacity(color, 0.1);

          return {
            label: dataset.label,
            data: dataset.data as number[],
            borderColor: color,
            backgroundColor: fill ? bgColor : 'transparent',
            borderWidth: dataset.borderWidth ?? 2,
            fill: dataset.fill ?? fill,
            tension: dataset.tension ?? tension,
            pointRadius: showPoints ? (dataset.pointRadius ?? 4) : 0,
            pointHoverRadius: showPoints ? (dataset.pointHoverRadius ?? 6) : 0,
            pointStyle: pointStyle,
            pointBackgroundColor: color,
            pointBorderColor: '#fff',
            pointBorderWidth: 2,
            stepped: stepped,
            hidden: dataset.hidden,
            order: dataset.order,
            yAxisID: dataset.yAxisID,
          };
        }),
      };
    }, [labels, datasets, tension, fill, showPoints, pointStyle, stepped]);

    // Build chart options
    const chartOptions: ChartOptions<'line'> = useMemo(() => {
      const options: ChartOptions<'line'> = {
        scales: {
          y: {
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
          x: {
            ticks: xAxisFormat
              ? {
                  callback: function (value, index) {
                    const label = labels[index];
                    return xAxisFormat(label);
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
        },
      };

      return options;
    }, [yMin, yMax, yAxisFormat, xAxisFormat, tooltipFormat, labels]);

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

LineChart.displayName = 'LineChart';

export default memo(LineChart);

// Convenience hook for line chart data generation
export const useLineChartData = (
  rawData: { label: string; values: number[] }[],
  labels: string[]
) => {
  return useMemo(() => {
    return {
      labels,
      datasets: rawData.map((item, index) => ({
        label: item.label,
        data: item.values,
        borderColor: defaultColorPalette[index % defaultColorPalette.length],
      })),
    };
  }, [rawData, labels]);
};

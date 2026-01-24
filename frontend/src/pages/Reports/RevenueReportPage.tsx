import React, { useState, useEffect, useCallback } from 'react';
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js';
import { Line, Bar, Doughnut } from 'react-chartjs-2';
import {
  TrendingUp,
  TrendingDown,
  DollarSign,
  Users,
  ShoppingCart,
  Download,
  RefreshCw,
  AlertCircle,
} from 'lucide-react';
import { format, subDays } from 'date-fns';

import { DashboardLayout } from '@components/layout';
import DateRangePicker from '../../components/reports/DateRangePicker';
import revenueService from '../../services/revenueService';
import {
  RevenueReportResponse,
  RevenueReportQuery,
  PeriodType,
} from '../../types/revenue';

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  BarElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
  Filler
);

const RevenueReportPage: React.FC = () => {
  const [report, setReport] = useState<RevenueReportResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [startDate, setStartDate] = useState<Date | null>(
    subDays(new Date(), 30)
  );
  const [endDate, setEndDate] = useState<Date | null>(new Date());
  const [periodType, setPeriodType] = useState<PeriodType>('daily');
  const [isExporting, setIsExporting] = useState(false);

  const fetchReport = useCallback(async () => {
    if (!startDate || !endDate) return;

    setLoading(true);
    setError(null);

    try {
      const query: RevenueReportQuery = {
        start_date: startDate.toISOString(),
        end_date: endDate.toISOString(),
        period_type: periodType,
        currency: 'USD',
      };

      const data = await revenueService.getRevenueReport(query);
      setReport(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch report');
    } finally {
      setLoading(false);
    }
  }, [startDate, endDate, periodType]);

  useEffect(() => {
    fetchReport();
  }, [fetchReport]);

  const handleDateChange = (start: Date | null, end: Date | null) => {
    setStartDate(start);
    setEndDate(end);
  };

  const handleExportCSV = async () => {
    if (!startDate || !endDate) return;

    setIsExporting(true);
    try {
      await revenueService.downloadCSVExport(
        {
          start_date: startDate.toISOString(),
          end_date: endDate.toISOString(),
          period_type: periodType,
        },
        `revenue_report_${format(new Date(), 'yyyy-MM-dd')}.csv`
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to export report');
    } finally {
      setIsExporting(false);
    }
  };

  const handleExportJSON = async () => {
    if (!startDate || !endDate) return;

    setIsExporting(true);
    try {
      await revenueService.downloadJSONExport(
        {
          start_date: startDate.toISOString(),
          end_date: endDate.toISOString(),
          period_type: periodType,
        },
        `revenue_report_${format(new Date(), 'yyyy-MM-dd')}.json`
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to export report');
    } finally {
      setIsExporting(false);
    }
  };

  // Format currency for display
  const formatCurrency = (value: string | number): string => {
    const num = typeof value === 'string' ? parseFloat(value) : value;
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 2,
    }).format(num);
  };

  // Format percentage
  const formatPercentage = (value: string | number): string => {
    const num = typeof value === 'string' ? parseFloat(value) : value;
    return `${num >= 0 ? '+' : ''}${num.toFixed(2)}%`;
  };

  // Prepare line chart data
  const getLineChartData = () => {
    if (!report?.time_series) return null;

    return {
      labels: report.time_series.map((dp) =>
        format(
          new Date(dp.period),
          periodType === 'daily' ? 'MMM d' : 'MMM yyyy'
        )
      ),
      datasets: [
        {
          label: 'Gross Revenue',
          data: report.time_series.map((dp) => parseFloat(dp.gross_revenue)),
          borderColor: 'rgb(59, 130, 246)',
          backgroundColor: 'rgba(59, 130, 246, 0.1)',
          fill: true,
          tension: 0.4,
        },
        {
          label: 'Net Revenue',
          data: report.time_series.map((dp) => parseFloat(dp.net_revenue)),
          borderColor: 'rgb(34, 197, 94)',
          backgroundColor: 'rgba(34, 197, 94, 0.1)',
          fill: true,
          tension: 0.4,
        },
      ],
    };
  };

  // Prepare category pie chart data
  const getCategoryChartData = () => {
    if (!report?.by_category || report.by_category.length === 0) return null;

    const colors = [
      'rgb(59, 130, 246)',
      'rgb(34, 197, 94)',
      'rgb(234, 179, 8)',
      'rgb(239, 68, 68)',
      'rgb(168, 85, 247)',
      'rgb(236, 72, 153)',
      'rgb(20, 184, 166)',
      'rgb(249, 115, 22)',
    ];

    return {
      labels: report.by_category.map((cat) => cat.category || 'Uncategorized'),
      datasets: [
        {
          data: report.by_category.map((cat) => parseFloat(cat.revenue)),
          backgroundColor: colors.slice(0, report.by_category.length),
          borderWidth: 2,
          borderColor: '#fff',
        },
      ],
    };
  };

  // Prepare transaction bar chart data
  const getTransactionChartData = () => {
    if (!report?.time_series) return null;

    return {
      labels: report.time_series.map((dp) =>
        format(
          new Date(dp.period),
          periodType === 'daily' ? 'MMM d' : 'MMM yyyy'
        )
      ),
      datasets: [
        {
          label: 'Transactions',
          data: report.time_series.map((dp) => dp.transaction_count),
          backgroundColor: 'rgba(168, 85, 247, 0.8)',
          borderColor: 'rgb(168, 85, 247)',
          borderWidth: 1,
        },
      ],
    };
  };

  const lineChartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        position: 'top' as const,
      },
      title: {
        display: true,
        text: 'Revenue Over Time',
      },
      tooltip: {
        mode: 'index' as const,
        intersect: false,
        callbacks: {
          label: (context: {
            dataset: { label?: string };
            parsed: { y: number | null };
          }) => {
            return `${context.dataset.label}: ${formatCurrency(
              context.parsed.y ?? 0
            )}`;
          },
        },
      },
    },
    scales: {
      y: {
        beginAtZero: true,
        ticks: {
          callback: (value: number | string) => formatCurrency(value),
        },
      },
    },
  };

  const barChartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        display: false,
      },
      title: {
        display: true,
        text: 'Transaction Volume',
      },
    },
    scales: {
      y: {
        beginAtZero: true,
      },
    },
  };

  const pieChartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        position: 'right' as const,
      },
      title: {
        display: true,
        text: 'Revenue by Category',
      },
    },
  };

  if (loading) {
    return (
      <DashboardLayout>
        <div className="flex items-center justify-center min-h-screen">
          <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600"></div>
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout>
      <div className="p-6 max-w-7xl mx-auto">
        {/* Header */}
        <div className="mb-8">
          <h1 className="text-2xl font-bold text-gray-900">Revenue Report</h1>
          <p className="text-gray-600 mt-1">
            Track and analyze your revenue performance
          </p>
        </div>

        {/* Controls */}
        <div className="flex flex-wrap items-center gap-4 mb-6 p-4 bg-white rounded-lg shadow-sm border border-gray-200">
          <DateRangePicker
            startDate={startDate}
            endDate={endDate}
            onDateChange={handleDateChange}
          />

          <select
            value={periodType}
            onChange={(e) => setPeriodType(e.target.value as PeriodType)}
            className="px-4 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="daily">Daily</option>
            <option value="weekly">Weekly</option>
            <option value="monthly">Monthly</option>
            <option value="quarterly">Quarterly</option>
            <option value="yearly">Yearly</option>
          </select>

          <button
            onClick={fetchReport}
            disabled={loading}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </button>

          <div className="ml-auto flex items-center gap-2">
            <button
              onClick={handleExportCSV}
              disabled={isExporting}
              className="flex items-center gap-2 px-4 py-2 border border-gray-300 rounded-lg text-sm hover:bg-gray-50 disabled:opacity-50"
            >
              <Download className="w-4 h-4" />
              Export CSV
            </button>
            <button
              onClick={handleExportJSON}
              disabled={isExporting}
              className="flex items-center gap-2 px-4 py-2 border border-gray-300 rounded-lg text-sm hover:bg-gray-50 disabled:opacity-50"
            >
              <Download className="w-4 h-4" />
              Export JSON
            </button>
          </div>
        </div>

        {/* Error message */}
        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg flex items-center gap-3">
            <AlertCircle className="w-5 h-5 text-red-500" />
            <span className="text-red-700">{error}</span>
          </div>
        )}

        {report && (
          <>
            {/* Summary Cards */}
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-8">
              {/* Total Revenue */}
              <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-gray-600">Total Revenue</p>
                    <p className="text-2xl font-bold text-gray-900">
                      {formatCurrency(report.summary.total_gross_revenue)}
                    </p>
                  </div>
                  <div className="p-3 bg-blue-100 rounded-full">
                    <DollarSign className="w-6 h-6 text-blue-600" />
                  </div>
                </div>
                <div className="mt-4 flex items-center">
                  {parseFloat(report.summary.growth_rate) >= 0 ? (
                    <TrendingUp className="w-4 h-4 text-green-500 mr-1" />
                  ) : (
                    <TrendingDown className="w-4 h-4 text-red-500 mr-1" />
                  )}
                  <span
                    className={`text-sm font-medium ${
                      parseFloat(report.summary.growth_rate) >= 0
                        ? 'text-green-600'
                        : 'text-red-600'
                    }`}
                  >
                    {formatPercentage(report.summary.growth_rate)}
                  </span>
                  <span className="text-sm text-gray-500 ml-1">
                    vs previous period
                  </span>
                </div>
              </div>

              {/* Net Revenue */}
              <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-gray-600">Net Revenue</p>
                    <p className="text-2xl font-bold text-gray-900">
                      {formatCurrency(report.summary.total_net_revenue)}
                    </p>
                  </div>
                  <div className="p-3 bg-green-100 rounded-full">
                    <TrendingUp className="w-6 h-6 text-green-600" />
                  </div>
                </div>
                <p className="mt-4 text-sm text-gray-500">
                  After fees: {formatCurrency(report.summary.total_fees)}
                </p>
              </div>

              {/* Transactions */}
              <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-gray-600">Total Transactions</p>
                    <p className="text-2xl font-bold text-gray-900">
                      {report.summary.transaction_count.toLocaleString()}
                    </p>
                  </div>
                  <div className="p-3 bg-purple-100 rounded-full">
                    <ShoppingCart className="w-6 h-6 text-purple-600" />
                  </div>
                </div>
                <p className="mt-4 text-sm text-gray-500">
                  Avg:{' '}
                  {formatCurrency(report.summary.average_transaction_value)}
                </p>
              </div>

              {/* Customers */}
              <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-gray-600">Unique Customers</p>
                    <p className="text-2xl font-bold text-gray-900">
                      {report.summary.unique_customer_count.toLocaleString()}
                    </p>
                  </div>
                  <div className="p-3 bg-orange-100 rounded-full">
                    <Users className="w-6 h-6 text-orange-600" />
                  </div>
                </div>
                <p className="mt-4 text-sm text-gray-500">
                  Refunds: {report.summary.refund_count}
                </p>
              </div>
            </div>

            {/* Charts Grid */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
              {/* Revenue Line Chart */}
              <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                <div className="h-80">
                  {getLineChartData() && (
                    <Line
                      data={getLineChartData()!}
                      options={lineChartOptions}
                    />
                  )}
                </div>
              </div>

              {/* Category Pie Chart */}
              <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                <div className="h-80">
                  {getCategoryChartData() ? (
                    <Doughnut
                      data={getCategoryChartData()!}
                      options={pieChartOptions}
                    />
                  ) : (
                    <div className="flex items-center justify-center h-full text-gray-500">
                      No category data available
                    </div>
                  )}
                </div>
              </div>

              {/* Transaction Volume Bar Chart */}
              <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200 lg:col-span-2">
                <div className="h-64">
                  {getTransactionChartData() && (
                    <Bar
                      data={getTransactionChartData()!}
                      options={barChartOptions}
                    />
                  )}
                </div>
              </div>
            </div>

            {/* Top Customers Table */}
            {report.top_customers && report.top_customers.length > 0 && (
              <div className="bg-white rounded-lg shadow-sm border border-gray-200">
                <div className="px-6 py-4 border-b border-gray-200">
                  <h2 className="text-lg font-semibold text-gray-900">
                    Top Customers by Revenue
                  </h2>
                </div>
                <div className="overflow-x-auto">
                  <table className="min-w-full divide-y divide-gray-200">
                    <thead className="bg-gray-50">
                      <tr>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Customer
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Total Revenue
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Transactions
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          Avg Value
                        </th>
                        <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                          First Transaction
                        </th>
                      </tr>
                    </thead>
                    <tbody className="bg-white divide-y divide-gray-200">
                      {report.top_customers.map((customer, index) => (
                        <tr key={customer.user_id} className="hover:bg-gray-50">
                          <td className="px-6 py-4 whitespace-nowrap">
                            <div className="flex items-center">
                              <div className="w-8 h-8 bg-blue-100 rounded-full flex items-center justify-center mr-3">
                                <span className="text-sm font-medium text-blue-600">
                                  {index + 1}
                                </span>
                              </div>
                              <div>
                                <div className="text-sm font-medium text-gray-900">
                                  {customer.user_name || 'Unknown'}
                                </div>
                                <div className="text-sm text-gray-500">
                                  {customer.user_email ||
                                    customer.user_id.slice(0, 8)}
                                </div>
                              </div>
                            </div>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap">
                            <span className="text-sm font-medium text-gray-900">
                              {formatCurrency(customer.total_revenue)}
                            </span>
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {customer.transaction_count}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {formatCurrency(customer.average_transaction_value)}
                          </td>
                          <td className="px-6 py-4 whitespace-nowrap text-sm text-gray-500">
                            {format(
                              new Date(customer.first_transaction_date),
                              'MMM d, yyyy'
                            )}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {/* Report Metadata */}
            <div className="mt-6 text-sm text-gray-500 text-right">
              Generated at{' '}
              {format(new Date(report.metadata.generated_at), 'PPpp')} |{' '}
              {report.metadata.data_points} data points |{' '}
              {report.metadata.execution_time_ms}ms
            </div>
          </>
        )}
      </div>
    </DashboardLayout>
  );
};

export default RevenueReportPage;

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
  Users,
  UserPlus,
  UserMinus,
  TrendingUp,
  TrendingDown,
  AlertTriangle,
  Download,
  RefreshCw,
  AlertCircle,
  MapPin,
} from 'lucide-react';
import { format, subMonths } from 'date-fns';

import { DashboardLayout } from '@components/layout';
import DateRangePicker from '../../components/reports/DateRangePicker';
import customerReportService from '../../services/customerReportService';
import {
  CustomerReportResponse,
  CustomerReportQuery,
  CustomerSegment,
  getSegmentDisplayName,
  getSegmentColor,
} from '../../types/customer';

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

const CustomerReportPage: React.FC = () => {
  const [report, setReport] = useState<CustomerReportResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [startDate, setStartDate] = useState<Date | null>(
    subMonths(new Date(), 12)
  );
  const [endDate, setEndDate] = useState<Date | null>(new Date());
  const [isExporting, setIsExporting] = useState(false);
  const [selectedSegment, setSelectedSegment] = useState<CustomerSegment | ''>(
    ''
  );

  const fetchReport = useCallback(async () => {
    setLoading(true);
    setError(null);

    try {
      const query: CustomerReportQuery = {
        start_date: startDate?.toISOString(),
        end_date: endDate?.toISOString(),
      };

      if (selectedSegment) {
        query.segment = selectedSegment;
      }

      const data = await customerReportService.getCustomerReport(query);
      setReport(data);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch report');
    } finally {
      setLoading(false);
    }
  }, [startDate, endDate, selectedSegment]);

  useEffect(() => {
    fetchReport();
  }, [fetchReport]);

  const handleDateChange = (start: Date | null, end: Date | null) => {
    setStartDate(start);
    setEndDate(end);
  };

  const handleExportCSV = async () => {
    setIsExporting(true);
    try {
      await customerReportService.downloadCSVExport(
        {
          start_date: startDate?.toISOString(),
          end_date: endDate?.toISOString(),
          segment: selectedSegment || undefined,
        },
        `customer_report_${format(new Date(), 'yyyy-MM-dd')}.csv`
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to export report');
    } finally {
      setIsExporting(false);
    }
  };

  const formatCurrency = (value: string | number): string => {
    const num = typeof value === 'string' ? parseFloat(value) : value;
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
      minimumFractionDigits: 0,
      maximumFractionDigits: 0,
    }).format(num);
  };

  const formatPercentage = (value: string | number): string => {
    const num = typeof value === 'string' ? parseFloat(value) : value;
    return `${num >= 0 ? '+' : ''}${num.toFixed(1)}%`;
  };

  // Prepare segment pie chart data
  const getSegmentChartData = () => {
    if (!report?.segment_breakdown) return null;

    return {
      labels: report.segment_breakdown.map((seg) => seg.display_name),
      datasets: [
        {
          data: report.segment_breakdown.map((seg) => seg.count),
          backgroundColor: report.segment_breakdown.map((seg) => seg.color),
          borderWidth: 2,
          borderColor: '#fff',
        },
      ],
    };
  };

  // Prepare acquisition trend chart data
  const getAcquisitionChartData = () => {
    if (!report?.acquisition_trend) return null;

    return {
      labels: report.acquisition_trend.map((t) =>
        format(new Date(t.period), 'MMM yyyy')
      ),
      datasets: [
        {
          label: 'New Customers',
          data: report.acquisition_trend.map((t) => t.new_customers),
          borderColor: 'rgb(59, 130, 246)',
          backgroundColor: 'rgba(59, 130, 246, 0.1)',
          fill: true,
          tension: 0.4,
        },
        {
          label: 'Active Customers',
          data: report.acquisition_trend.map((t) => t.active_customers),
          borderColor: 'rgb(34, 197, 94)',
          backgroundColor: 'rgba(34, 197, 94, 0.1)',
          fill: true,
          tension: 0.4,
        },
      ],
    };
  };

  // Prepare geography bar chart data
  const getGeographyChartData = () => {
    if (!report?.geography_breakdown) return null;

    const topStates = report.geography_breakdown.slice(0, 10);

    return {
      labels: topStates.map((g) => g.state),
      datasets: [
        {
          label: 'Customers',
          data: topStates.map((g) => g.customer_count),
          backgroundColor: 'rgba(59, 130, 246, 0.8)',
          borderColor: 'rgb(59, 130, 246)',
          borderWidth: 1,
        },
      ],
    };
  };

  const chartOptions = {
    responsive: true,
    maintainAspectRatio: false,
    plugins: {
      legend: {
        position: 'top' as const,
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
        text: 'Customer Segmentation',
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
          <h1 className="text-2xl font-bold text-gray-900">Customer Report</h1>
          <p className="text-gray-600 mt-1">
            Analyze customer segments, trends, and engagement metrics
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
            value={selectedSegment}
            onChange={(e) =>
              setSelectedSegment(e.target.value as CustomerSegment | '')
            }
            className="px-4 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="">All Segments</option>
            <option value="high_value">High Value</option>
            <option value="regular">Regular</option>
            <option value="occasional">Occasional</option>
            <option value="at_risk">At Risk</option>
            <option value="churned">Churned</option>
            <option value="new">New</option>
          </select>

          <button
            onClick={fetchReport}
            disabled={loading}
            className="flex items-center gap-2 px-4 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 disabled:opacity-50"
          >
            <RefreshCw className={`w-4 h-4 ${loading ? 'animate-spin' : ''}`} />
            Refresh
          </button>

          <div className="ml-auto">
            <button
              onClick={handleExportCSV}
              disabled={isExporting}
              className="flex items-center gap-2 px-4 py-2 border border-gray-300 rounded-lg text-sm hover:bg-gray-50 disabled:opacity-50"
            >
              <Download className="w-4 h-4" />
              Export CSV
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
              {/* Total Customers */}
              <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-gray-600">Total Customers</p>
                    <p className="text-2xl font-bold text-gray-900">
                      {report.summary.total_customers.toLocaleString()}
                    </p>
                  </div>
                  <div className="p-3 bg-blue-100 rounded-full">
                    <Users className="w-6 h-6 text-blue-600" />
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
                  <span className="text-sm text-gray-500 ml-1">growth</span>
                </div>
              </div>

              {/* New Customers */}
              <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-gray-600">New This Period</p>
                    <p className="text-2xl font-bold text-gray-900">
                      {report.summary.new_customers_this_period.toLocaleString()}
                    </p>
                  </div>
                  <div className="p-3 bg-green-100 rounded-full">
                    <UserPlus className="w-6 h-6 text-green-600" />
                  </div>
                </div>
                <p className="mt-4 text-sm text-gray-500">
                  Active: {report.summary.active_customers.toLocaleString()}
                </p>
              </div>

              {/* At Risk */}
              <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-gray-600">At Risk</p>
                    <p className="text-2xl font-bold text-gray-900">
                      {report.at_risk_customers?.length || 0}
                    </p>
                  </div>
                  <div className="p-3 bg-red-100 rounded-full">
                    <AlertTriangle className="w-6 h-6 text-red-600" />
                  </div>
                </div>
                <p className="mt-4 text-sm text-gray-500">
                  Churned: {report.summary.churned_customers}
                </p>
              </div>

              {/* Total Revenue */}
              <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                <div className="flex items-center justify-between">
                  <div>
                    <p className="text-sm text-gray-600">Total Revenue</p>
                    <p className="text-2xl font-bold text-gray-900">
                      {formatCurrency(report.summary.total_revenue)}
                    </p>
                  </div>
                  <div className="p-3 bg-purple-100 rounded-full">
                    <TrendingUp className="w-6 h-6 text-purple-600" />
                  </div>
                </div>
                <p className="mt-4 text-sm text-gray-500">
                  Avg:{' '}
                  {formatCurrency(report.summary.average_revenue_per_customer)}
                  /customer
                </p>
              </div>
            </div>

            {/* Charts Grid */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mb-8">
              {/* Segmentation Pie Chart */}
              <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                <div className="h-80">
                  {getSegmentChartData() ? (
                    <Doughnut
                      data={getSegmentChartData()!}
                      options={pieChartOptions}
                    />
                  ) : (
                    <div className="flex items-center justify-center h-full text-gray-500">
                      No segment data available
                    </div>
                  )}
                </div>
              </div>

              {/* Acquisition Trend Line Chart */}
              <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200">
                <h3 className="text-lg font-semibold text-gray-900 mb-4">
                  Customer Acquisition Trend
                </h3>
                <div className="h-72">
                  {getAcquisitionChartData() ? (
                    <Line
                      data={getAcquisitionChartData()!}
                      options={chartOptions}
                    />
                  ) : (
                    <div className="flex items-center justify-center h-full text-gray-500">
                      No trend data available
                    </div>
                  )}
                </div>
              </div>

              {/* Geography Bar Chart */}
              <div className="bg-white p-6 rounded-lg shadow-sm border border-gray-200 lg:col-span-2">
                <h3 className="text-lg font-semibold text-gray-900 mb-4">
                  <MapPin className="w-5 h-5 inline mr-2" />
                  Geographic Distribution (Top 10 States)
                </h3>
                <div className="h-64">
                  {getGeographyChartData() ? (
                    <Bar
                      data={getGeographyChartData()!}
                      options={chartOptions}
                    />
                  ) : (
                    <div className="flex items-center justify-center h-full text-gray-500">
                      No geography data available
                    </div>
                  )}
                </div>
              </div>
            </div>

            {/* Segment Breakdown Cards */}
            {report.segment_breakdown &&
              report.segment_breakdown.length > 0 && (
                <div className="mb-8">
                  <h2 className="text-lg font-semibold text-gray-900 mb-4">
                    Segment Breakdown
                  </h2>
                  <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
                    {report.segment_breakdown.map((seg) => (
                      <div
                        key={seg.segment}
                        className="bg-white p-4 rounded-lg shadow-sm border border-gray-200"
                        style={{
                          borderLeftColor: seg.color,
                          borderLeftWidth: 4,
                        }}
                      >
                        <p className="text-sm font-medium text-gray-600">
                          {seg.display_name}
                        </p>
                        <p className="text-xl font-bold text-gray-900">
                          {seg.count}
                        </p>
                        <p className="text-xs text-gray-500">
                          {formatCurrency(seg.average_revenue)} avg
                        </p>
                      </div>
                    ))}
                  </div>
                </div>
              )}

            {/* Tables Section */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
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
                          <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                            Customer
                          </th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                            Segment
                          </th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                            Revenue
                          </th>
                          <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                            Jobs
                          </th>
                        </tr>
                      </thead>
                      <tbody className="bg-white divide-y divide-gray-200">
                        {report.top_customers.slice(0, 5).map((customer) => (
                          <tr
                            key={customer.customer_id}
                            className="hover:bg-gray-50"
                          >
                            <td className="px-4 py-3 whitespace-nowrap">
                              <div className="text-sm font-medium text-gray-900">
                                {customer.company_name ||
                                  `${customer.first_name} ${customer.last_name}`}
                              </div>
                              <div className="text-xs text-gray-500">
                                {customer.email}
                              </div>
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap">
                              <span
                                className="inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium"
                                style={{
                                  backgroundColor: `${getSegmentColor(
                                    customer.segment
                                  )}20`,
                                  color: getSegmentColor(customer.segment),
                                }}
                              >
                                {getSegmentDisplayName(customer.segment)}
                              </span>
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm font-medium text-gray-900">
                              {formatCurrency(customer.total_revenue)}
                            </td>
                            <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                              {customer.total_jobs}
                            </td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  </div>
                </div>
              )}

              {/* At-Risk Customers Table */}
              {report.at_risk_customers &&
                report.at_risk_customers.length > 0 && (
                  <div className="bg-white rounded-lg shadow-sm border border-gray-200">
                    <div className="px-6 py-4 border-b border-gray-200 flex items-center gap-2">
                      <AlertTriangle className="w-5 h-5 text-red-500" />
                      <h2 className="text-lg font-semibold text-gray-900">
                        At-Risk Customers
                      </h2>
                    </div>
                    <div className="overflow-x-auto">
                      <table className="min-w-full divide-y divide-gray-200">
                        <thead className="bg-gray-50">
                          <tr>
                            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                              Customer
                            </th>
                            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                              Revenue
                            </th>
                            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                              Last Activity
                            </th>
                            <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase">
                              Days Inactive
                            </th>
                          </tr>
                        </thead>
                        <tbody className="bg-white divide-y divide-gray-200">
                          {report.at_risk_customers
                            .slice(0, 5)
                            .map((customer) => (
                              <tr
                                key={customer.customer_id}
                                className="hover:bg-gray-50"
                              >
                                <td className="px-4 py-3 whitespace-nowrap">
                                  <div className="text-sm font-medium text-gray-900">
                                    {customer.company_name ||
                                      `${customer.first_name} ${customer.last_name}`}
                                  </div>
                                  <div className="text-xs text-gray-500">
                                    {customer.email}
                                  </div>
                                </td>
                                <td className="px-4 py-3 whitespace-nowrap text-sm font-medium text-gray-900">
                                  {formatCurrency(customer.total_revenue)}
                                </td>
                                <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                                  {customer.last_activity_date
                                    ? format(
                                        new Date(customer.last_activity_date),
                                        'MMM d, yyyy'
                                      )
                                    : 'N/A'}
                                </td>
                                <td className="px-4 py-3 whitespace-nowrap">
                                  <span className="text-sm font-medium text-red-600">
                                    {customer.days_since_last_activity} days
                                  </span>
                                </td>
                              </tr>
                            ))}
                        </tbody>
                      </table>
                    </div>
                  </div>
                )}
            </div>

            {/* Report Metadata */}
            <div className="mt-6 text-sm text-gray-500 text-right">
              Generated at{' '}
              {format(new Date(report.metadata.generated_at), 'PPpp')} |{' '}
              {report.metadata.total_records} total customers |{' '}
              {report.metadata.execution_time_ms}ms
            </div>
          </>
        )}
      </div>
    </DashboardLayout>
  );
};

export default CustomerReportPage;

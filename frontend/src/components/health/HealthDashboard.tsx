/**
 * Health Dashboard Component
 *
 * Displays real-time health status of all system components
 */

import { useState, useEffect, useCallback } from 'react';

// Types
interface CheckResult {
  name: string;
  status: 'healthy' | 'unhealthy' | 'degraded' | 'unknown';
  message?: string;
  duration_ms: number;
  timestamp: string;
  details?: Record<string, unknown>;
  error?: string;
  last_success?: string;
  last_failure?: string;
}

interface HealthReport {
  status: 'healthy' | 'unhealthy' | 'degraded' | 'unknown';
  timestamp: string;
  version?: string;
  environment?: string;
  uptime_seconds: number;
  checks: Record<string, CheckResult>;
}

interface UptimeStats {
  uptime_seconds: number;
  start_time: string;
  availability_pct: Record<string, number>;
  total_checks: Record<string, number>;
  failed_checks: Record<string, number>;
  current_status: Record<string, string>;
}

interface StatusChangeEvent {
  name: string;
  old_status: string;
  new_status: string;
  timestamp: string;
}

interface HealthDashboardProps {
  apiEndpoint?: string;
  refreshInterval?: number;
  showDetails?: boolean;
  onStatusChange?: (status: string) => void;
}

// Status color mapping
const statusColors = {
  healthy: 'bg-green-500',
  unhealthy: 'bg-red-500',
  degraded: 'bg-yellow-500',
  unknown: 'bg-gray-500',
};

const statusTextColors = {
  healthy: 'text-green-600',
  unhealthy: 'text-red-600',
  degraded: 'text-yellow-600',
  unknown: 'text-gray-600',
};

const statusBgColors = {
  healthy: 'bg-green-100',
  unhealthy: 'bg-red-100',
  degraded: 'bg-yellow-100',
  unknown: 'bg-gray-100',
};

// Format uptime
function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);

  if (days > 0) {
    return `${days}d ${hours}h ${minutes}m`;
  }
  if (hours > 0) {
    return `${hours}h ${minutes}m`;
  }
  return `${minutes}m`;
}

// Format timestamp
function formatTimestamp(timestamp: string): string {
  return new Date(timestamp).toLocaleString();
}

// Status Badge Component
function StatusBadge({ status }: { status: string }) {
  const normalizedStatus = status as keyof typeof statusColors;
  return (
    <span
      className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${
        statusBgColors[normalizedStatus] || statusBgColors.unknown
      } ${statusTextColors[normalizedStatus] || statusTextColors.unknown}`}
    >
      <span
        className={`w-2 h-2 mr-1.5 rounded-full ${
          statusColors[normalizedStatus] || statusColors.unknown
        }`}
      />
      {status.toUpperCase()}
    </span>
  );
}

// Check Card Component
function CheckCard({
  check,
  showDetails,
}: {
  check: CheckResult;
  showDetails: boolean;
}) {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="border rounded-lg p-4 bg-white shadow-sm">
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-3">
          <div
            className={`w-3 h-3 rounded-full ${
              statusColors[check.status] || statusColors.unknown
            }`}
          />
          <div>
            <h3 className="font-medium text-gray-900">{check.name}</h3>
            {check.message && (
              <p className="text-sm text-gray-500">{check.message}</p>
            )}
          </div>
        </div>
        <div className="flex items-center space-x-4">
          <span className="text-sm text-gray-500">{check.duration_ms}ms</span>
          <StatusBadge status={check.status} />
          {showDetails && check.details && (
            <button
              onClick={() => setExpanded(!expanded)}
              className="text-sm text-blue-600 hover:text-blue-800"
            >
              {expanded ? 'Hide' : 'Details'}
            </button>
          )}
        </div>
      </div>

      {check.error && (
        <div className="mt-2 p-2 bg-red-50 rounded text-sm text-red-700">
          {check.error}
        </div>
      )}

      {expanded && check.details && (
        <div className="mt-3 p-3 bg-gray-50 rounded text-sm">
          <pre className="overflow-auto text-xs">
            {JSON.stringify(check.details, null, 2)}
          </pre>
        </div>
      )}

      <div className="mt-2 text-xs text-gray-400">
        Last checked: {formatTimestamp(check.timestamp)}
      </div>
    </div>
  );
}

// Event Log Component
function EventLog({ events }: { events: StatusChangeEvent[] }) {
  if (events.length === 0) {
    return (
      <div className="text-sm text-gray-500 text-center py-4">
        No recent status changes
      </div>
    );
  }

  return (
    <div className="space-y-2">
      {events.map((event, index) => (
        <div
          key={index}
          className="flex items-center justify-between p-2 bg-gray-50 rounded text-sm"
        >
          <div className="flex items-center space-x-2">
            <span className="font-medium">{event.name}</span>
            <span className="text-gray-400">→</span>
            <StatusBadge status={event.new_status} />
          </div>
          <span className="text-xs text-gray-400">
            {formatTimestamp(event.timestamp)}
          </span>
        </div>
      ))}
    </div>
  );
}

// Main Dashboard Component
export function HealthDashboard({
  apiEndpoint = '/health/dashboard',
  refreshInterval = 30000,
  showDetails = true,
  onStatusChange,
}: HealthDashboardProps) {
  const [report, setReport] = useState<HealthReport | null>(null);
  const [uptimeStats, setUptimeStats] = useState<UptimeStats | null>(null);
  const [events, setEvents] = useState<StatusChangeEvent[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastStatus, setLastStatus] = useState<string | null>(null);

  const fetchHealth = useCallback(async () => {
    try {
      const response = await fetch(apiEndpoint);
      if (!response.ok) {
        throw new Error(`HTTP error! status: ${response.status}`);
      }
      const data = await response.json();

      setReport({
        status: data.status,
        timestamp: data.timestamp,
        version: data.version,
        environment: data.environment,
        uptime_seconds: data.uptime,
        checks: data.checks,
      });

      if (data.availability) {
        setUptimeStats({
          uptime_seconds: data.uptime,
          start_time: data.timestamp,
          availability_pct: data.availability,
          total_checks: {},
          failed_checks: {},
          current_status: {},
        });
      }

      if (data.recent_events) {
        setEvents(data.recent_events);
      }

      // Notify on status change
      if (onStatusChange && lastStatus !== null && lastStatus !== data.status) {
        onStatusChange(data.status);
      }
      setLastStatus(data.status);

      setError(null);
    } catch (err) {
      setError(
        err instanceof Error ? err.message : 'Failed to fetch health status'
      );
    } finally {
      setLoading(false);
    }
  }, [apiEndpoint, lastStatus, onStatusChange]);

  useEffect(() => {
    fetchHealth();
    const interval = setInterval(fetchHealth, refreshInterval);
    return () => clearInterval(interval);
  }, [fetchHealth, refreshInterval]);

  if (loading && !report) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600" />
      </div>
    );
  }

  if (error && !report) {
    return (
      <div className="p-4 bg-red-50 rounded-lg">
        <div className="flex items-center">
          <div className="flex-shrink-0">
            <svg
              className="h-5 w-5 text-red-400"
              viewBox="0 0 20 20"
              fill="currentColor"
            >
              <path
                fillRule="evenodd"
                d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                clipRule="evenodd"
              />
            </svg>
          </div>
          <div className="ml-3">
            <h3 className="text-sm font-medium text-red-800">
              Failed to load health status
            </h3>
            <p className="text-sm text-red-700 mt-1">{error}</p>
          </div>
        </div>
      </div>
    );
  }

  if (!report) return null;

  const checks = Object.values(report.checks);

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">System Health</h1>
          <p className="text-sm text-gray-500">
            {report.environment && `Environment: ${report.environment}`}
            {report.version && ` | Version: ${report.version}`}
          </p>
        </div>
        <div className="text-right">
          <StatusBadge status={report.status} />
          <p className="text-sm text-gray-500 mt-1">
            Uptime: {formatUptime(report.uptime_seconds)}
          </p>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4">
        <div className="bg-white rounded-lg shadow p-4">
          <div className="text-sm font-medium text-gray-500">Total Checks</div>
          <div className="mt-1 text-2xl font-semibold text-gray-900">
            {checks.length}
          </div>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <div className="text-sm font-medium text-gray-500">Healthy</div>
          <div className="mt-1 text-2xl font-semibold text-green-600">
            {checks.filter((c) => c.status === 'healthy').length}
          </div>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <div className="text-sm font-medium text-gray-500">Degraded</div>
          <div className="mt-1 text-2xl font-semibold text-yellow-600">
            {checks.filter((c) => c.status === 'degraded').length}
          </div>
        </div>
        <div className="bg-white rounded-lg shadow p-4">
          <div className="text-sm font-medium text-gray-500">Unhealthy</div>
          <div className="mt-1 text-2xl font-semibold text-red-600">
            {checks.filter((c) => c.status === 'unhealthy').length}
          </div>
        </div>
      </div>

      {/* Checks Grid */}
      <div>
        <h2 className="text-lg font-medium text-gray-900 mb-4">
          Health Checks
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          {checks.map((check) => (
            <CheckCard
              key={check.name}
              check={check}
              showDetails={showDetails}
            />
          ))}
        </div>
      </div>

      {/* Availability */}
      {uptimeStats && Object.keys(uptimeStats.availability_pct).length > 0 && (
        <div>
          <h2 className="text-lg font-medium text-gray-900 mb-4">
            Availability
          </h2>
          <div className="bg-white rounded-lg shadow overflow-hidden">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Check
                  </th>
                  <th className="px-6 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Availability
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {Object.entries(uptimeStats.availability_pct).map(
                  ([name, pct]) => (
                    <tr key={name}>
                      <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-gray-900">
                        {name}
                      </td>
                      <td className="px-6 py-4 whitespace-nowrap">
                        <div className="flex items-center">
                          <div className="w-full bg-gray-200 rounded-full h-2.5 mr-2">
                            <div
                              className={`h-2.5 rounded-full ${
                                pct >= 99
                                  ? 'bg-green-500'
                                  : pct >= 95
                                    ? 'bg-yellow-500'
                                    : 'bg-red-500'
                              }`}
                              style={{ width: `${pct}%` }}
                            />
                          </div>
                          <span className="text-sm text-gray-600">
                            {pct.toFixed(2)}%
                          </span>
                        </div>
                      </td>
                    </tr>
                  )
                )}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* Recent Events */}
      <div>
        <h2 className="text-lg font-medium text-gray-900 mb-4">
          Recent Events
        </h2>
        <div className="bg-white rounded-lg shadow p-4">
          <EventLog events={events} />
        </div>
      </div>

      {/* Footer */}
      <div className="text-sm text-gray-400 text-center">
        Last updated: {formatTimestamp(report.timestamp)}
        {error && (
          <span className="text-red-500 ml-2">(Update failed: {error})</span>
        )}
      </div>
    </div>
  );
}

export default HealthDashboard;

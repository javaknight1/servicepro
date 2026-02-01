import React, { useEffect, useState, useCallback } from 'react';
import api from '../../services/api';
import {
  ConflictCheckerProps,
  ConflictCheckRequest,
  ConflictCheckResponse,
} from './types';

/**
 * ConflictChecker Component
 *
 * Real-time conflict detection for schedule creation/updates.
 * Automatically checks for conflicts when schedule details change.
 *
 * Features:
 * - Automatic conflict checking on data change
 * - Debounced API calls to reduce load
 * - Loading and error states
 * - Callback notifications for conflicts
 *
 * @example
 * ```tsx
 * <ConflictChecker
 *   jobId="123"
 *   startTime={new Date('2024-01-15T09:00:00')}
 *   endTime={new Date('2024-01-15T12:00:00')}
 *   assignedTechIds={['tech-1', 'tech-2']}
 *   onConflictDetected={(response) => console.log(response)}
 *   autoCheck={true}
 *   showSuggestions={true}
 * />
 * ```
 */
export const ConflictChecker: React.FC<ConflictCheckerProps> = ({
  scheduleId,
  jobId,
  startTime,
  endTime,
  assignedTechIds,
  location,
  onConflictDetected,
  onConflictResolved,
  autoCheck = true,
  showSuggestions: _showSuggestions = true,
}) => {
  const [isChecking, setIsChecking] = useState(false);
  const [conflictResponse, setConflictResponse] =
    useState<ConflictCheckResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  /**
   * Check for conflicts
   */
  const checkConflicts = useCallback(async () => {
    // Validate inputs
    if (!jobId || !startTime || !endTime || assignedTechIds.length === 0) {
      return;
    }

    setIsChecking(true);
    setError(null);

    try {
      const request: ConflictCheckRequest = {
        scheduleId,
        jobId,
        startTime: startTime.toISOString(),
        endTime: endTime.toISOString(),
        assignedTechIds,
        location,
      };

      const response = await api.post<ConflictCheckResponse>(
        '/v1/conflicts/check',
        request
      );

      const data = response.data;
      setConflictResponse(data);

      // Notify parent component
      if (data.hasConflicts && onConflictDetected) {
        onConflictDetected(data);
      } else if (!data.hasConflicts && onConflictResolved) {
        onConflictResolved();
      }
    } catch (err) {
      const errorMessage =
        err instanceof Error ? err.message : 'Failed to check conflicts';
      setError(errorMessage);
      // eslint-disable-next-line no-console
      console.error('Conflict check error:', err);
    } finally {
      setIsChecking(false);
    }
  }, [
    scheduleId,
    jobId,
    startTime,
    endTime,
    assignedTechIds,
    location,
    onConflictDetected,
    onConflictResolved,
  ]);

  /**
   * Auto-check for conflicts when inputs change
   * Debounced by 500ms to avoid excessive API calls
   */
  useEffect(() => {
    if (!autoCheck) return;

    const timer = setTimeout(() => {
      checkConflicts();
    }, 500);

    return () => clearTimeout(timer);
  }, [autoCheck, checkConflicts]);

  /**
   * Manual check trigger
   */
  const handleManualCheck = () => {
    checkConflicts();
  };

  // Don't render anything if not showing UI
  if (!conflictResponse && !isChecking && !error) {
    return null;
  }

  return (
    <div className="conflict-checker">
      {/* Loading State */}
      {isChecking && (
        <div className="flex items-center gap-2 p-3 bg-blue-50 border border-blue-200 rounded-lg">
          <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600"></div>
          <span className="text-sm text-blue-700">
            Checking for conflicts...
          </span>
        </div>
      )}

      {/* Error State */}
      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg">
          <div className="flex items-center gap-2">
            <svg
              className="w-5 h-5 text-red-500"
              fill="currentColor"
              viewBox="0 0 20 20"
            >
              <path
                fillRule="evenodd"
                d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                clipRule="evenodd"
              />
            </svg>
            <span className="text-sm text-red-700">{error}</span>
          </div>
          <button
            onClick={handleManualCheck}
            className="mt-2 text-sm text-red-600 hover:text-red-800 underline"
          >
            Try again
          </button>
        </div>
      )}

      {/* Conflict Results */}
      {conflictResponse && !isChecking && (
        <div className="space-y-2">
          {conflictResponse.hasConflicts ? (
            <div className="p-3 bg-yellow-50 border border-yellow-200 rounded-lg">
              <div className="flex items-start gap-2">
                <svg
                  className="w-5 h-5 text-yellow-600 mt-0.5"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path
                    fillRule="evenodd"
                    d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                    clipRule="evenodd"
                  />
                </svg>
                <div className="flex-1">
                  <h4 className="text-sm font-semibold text-yellow-800">
                    {conflictResponse.conflicts.length} Conflict
                    {conflictResponse.conflicts.length !== 1 ? 's' : ''}{' '}
                    Detected
                  </h4>
                  <p className="text-sm text-yellow-700 mt-1">
                    This schedule conflicts with existing schedules. Review the
                    conflicts below.
                  </p>
                </div>
              </div>
            </div>
          ) : (
            <div className="p-3 bg-green-50 border border-green-200 rounded-lg">
              <div className="flex items-center gap-2">
                <svg
                  className="w-5 h-5 text-green-600"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path
                    fillRule="evenodd"
                    d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z"
                    clipRule="evenodd"
                  />
                </svg>
                <span className="text-sm text-green-700 font-medium">
                  No conflicts detected
                </span>
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
};

export default ConflictChecker;

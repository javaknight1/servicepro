import React, { useState } from 'react';
import {
  ConflictAlertProps,
  ConflictDetail,
  ConflictSeverity,
  ConflictType,
  ResolutionSuggestion,
} from './types';
import { formatEpochTimeUTC } from '@/utils/epoch';

/**
 * Get severity color classes
 */
const getSeverityColor = (severity: ConflictSeverity): string => {
  switch (severity) {
    case ConflictSeverity.LOW:
      return 'bg-blue-50 border-blue-200 text-blue-800';
    case ConflictSeverity.MEDIUM:
      return 'bg-yellow-50 border-yellow-200 text-yellow-800';
    case ConflictSeverity.HIGH:
      return 'bg-orange-50 border-orange-200 text-orange-800';
    case ConflictSeverity.CRITICAL:
      return 'bg-red-50 border-red-200 text-red-800';
    default:
      return 'bg-gray-50 border-gray-200 text-gray-800';
  }
};

/**
 * Get severity badge color
 */
const getSeverityBadgeColor = (severity: ConflictSeverity): string => {
  switch (severity) {
    case ConflictSeverity.LOW:
      return 'bg-blue-100 text-blue-800';
    case ConflictSeverity.MEDIUM:
      return 'bg-yellow-100 text-yellow-800';
    case ConflictSeverity.HIGH:
      return 'bg-orange-100 text-orange-800';
    case ConflictSeverity.CRITICAL:
      return 'bg-red-100 text-red-800';
    default:
      return 'bg-gray-100 text-gray-800';
  }
};

/**
 * Get conflict type icon
 */
const getConflictTypeIcon = (type: ConflictType): JSX.Element => {
  switch (type) {
    case ConflictType.TECHNICIAN_OVERLAP:
      return (
        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
          <path d="M9 6a3 3 0 11-6 0 3 3 0 016 0zM17 6a3 3 0 11-6 0 3 3 0 016 0zM12.93 17c.046-.327.07-.66.07-1a6.97 6.97 0 00-1.5-4.33A5 5 0 0119 16v1h-6.07zM6 11a5 5 0 015 5v1H1v-1a5 5 0 015-5z" />
        </svg>
      );
    case ConflictType.TIME_OVERLAP:
      return (
        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-12a1 1 0 10-2 0v4a1 1 0 00.293.707l2.828 2.829a1 1 0 101.415-1.415L11 9.586V6z"
            clipRule="evenodd"
          />
        </svg>
      );
    case ConflictType.BUSINESS_HOURS:
      return (
        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M6 2a1 1 0 00-1 1v1H4a2 2 0 00-2 2v10a2 2 0 002 2h12a2 2 0 002-2V6a2 2 0 00-2-2h-1V3a1 1 0 10-2 0v1H7V3a1 1 0 00-1-1zm0 5a1 1 0 000 2h8a1 1 0 100-2H6z"
            clipRule="evenodd"
          />
        </svg>
      );
    case ConflictType.WORKLOAD_EXCESS:
      return (
        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M3 3a1 1 0 000 2v8a2 2 0 002 2h2.586l-1.293 1.293a1 1 0 101.414 1.414L10 15.414l2.293 2.293a1 1 0 001.414-1.414L12.414 15H15a2 2 0 002-2V5a1 1 0 100-2H3zm11 4a1 1 0 10-2 0v4a1 1 0 102 0V7zm-3 1a1 1 0 10-2 0v3a1 1 0 102 0V8zM8 9a1 1 0 00-2 0v2a1 1 0 102 0V9z"
            clipRule="evenodd"
          />
        </svg>
      );
    default:
      return (
        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
          <path
            fillRule="evenodd"
            d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z"
            clipRule="evenodd"
          />
        </svg>
      );
  }
};

/**
 * Format time for display (epoch seconds → "9:00 AM")
 */
const formatTime = (epoch: number): string => {
  return formatEpochTimeUTC(epoch);
};

/**
 * ConflictDetail Component - Shows individual conflict
 */
const ConflictDetailCard: React.FC<{ conflict: ConflictDetail }> = ({
  conflict,
}) => {
  const [isExpanded, setIsExpanded] = useState(false);

  return (
    <div
      className={`border rounded-lg p-3 ${getSeverityColor(conflict.severity)}`}
    >
      <div className="flex items-start gap-3">
        <div className="mt-0.5">
          {getConflictTypeIcon(conflict.conflict_type)}
        </div>
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span
              className={`px-2 py-0.5 rounded text-xs font-medium ${getSeverityBadgeColor(
                conflict.severity
              )}`}
            >
              {conflict.severity.toUpperCase()}
            </span>
            <span className="text-xs text-gray-500">
              {conflict.conflict_type.replace(/_/g, ' ').toUpperCase()}
            </span>
          </div>
          <p className="text-sm font-medium">{conflict.description}</p>

          {/* Conflicting Schedule Details */}
          {conflict.conflicting_with && (
            <div className="mt-2">
              <button
                onClick={() => setIsExpanded(!isExpanded)}
                className="text-xs text-blue-600 hover:text-blue-800 font-medium flex items-center gap-1"
              >
                {isExpanded ? 'Hide' : 'Show'} details
                <svg
                  className={`w-4 h-4 transition-transform ${
                    isExpanded ? 'rotate-180' : ''
                  }`}
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path
                    fillRule="evenodd"
                    d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z"
                    clipRule="evenodd"
                  />
                </svg>
              </button>

              {isExpanded && (
                <div className="mt-2 p-2 bg-white bg-opacity-50 rounded text-xs space-y-1">
                  <div>
                    <span className="font-medium">Schedule:</span>{' '}
                    {conflict.conflicting_with.title}
                  </div>
                  {conflict.conflicting_with.job_number && (
                    <div>
                      <span className="font-medium">Job:</span>{' '}
                      {conflict.conflicting_with.job_number}
                    </div>
                  )}
                  <div>
                    <span className="font-medium">Time:</span>{' '}
                    {formatTime(conflict.conflicting_with.start_time)} -{' '}
                    {formatTime(conflict.conflicting_with.end_time)}
                  </div>
                  {conflict.conflicting_with.location && (
                    <div>
                      <span className="font-medium">Location:</span>{' '}
                      {conflict.conflicting_with.location}
                    </div>
                  )}
                  {conflict.time_overlap && (
                    <div>
                      <span className="font-medium">Overlap:</span>{' '}
                      {conflict.time_overlap.overlap_minutes} minutes
                    </div>
                  )}
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

/**
 * ResolutionSuggestions Component
 */
const ResolutionSuggestionsCard: React.FC<{
  suggestions: ResolutionSuggestion[];
  onSelectSuggestion?: (suggestion: ResolutionSuggestion) => void;
}> = ({ suggestions, onSelectSuggestion }) => {
  if (!suggestions || suggestions.length === 0) return null;

  // Sort by priority
  const sortedSuggestions = [...suggestions].sort(
    (a, b) => a.priority - b.priority
  );

  return (
    <div className="bg-blue-50 border border-blue-200 rounded-lg p-3">
      <h4 className="text-sm font-semibold text-blue-900 mb-2 flex items-center gap-2">
        <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
          <path d="M11 3a1 1 0 10-2 0v1a1 1 0 102 0V3zM15.657 5.757a1 1 0 00-1.414-1.414l-.707.707a1 1 0 001.414 1.414l.707-.707zM18 10a1 1 0 01-1 1h-1a1 1 0 110-2h1a1 1 0 011 1zM5.05 6.464A1 1 0 106.464 5.05l-.707-.707a1 1 0 00-1.414 1.414l.707.707zM5 10a1 1 0 01-1 1H3a1 1 0 110-2h1a1 1 0 011 1zM8 16v-1h4v1a2 2 0 11-4 0zM12 14c.015-.34.208-.646.477-.859a4 4 0 10-4.954 0c.27.213.462.519.476.859h4.002z" />
        </svg>
        Suggested Resolutions
      </h4>
      <div className="space-y-2">
        {sortedSuggestions.map((suggestion, index) => (
          <div
            key={index}
            className="bg-white rounded p-2 border border-blue-100 hover:border-blue-300 transition-colors"
          >
            <div className="flex items-start justify-between gap-2">
              <div className="flex-1">
                <p className="text-sm font-medium text-gray-900">
                  {suggestion.description}
                </p>
                <p className="text-xs text-gray-600 mt-0.5">
                  Priority: {suggestion.priority} | Type: {suggestion.type}
                </p>
              </div>
              {onSelectSuggestion && (
                <button
                  onClick={() => onSelectSuggestion(suggestion)}
                  className="px-3 py-1 bg-blue-600 text-white text-xs font-medium rounded hover:bg-blue-700 transition-colors"
                >
                  Apply
                </button>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

/**
 * ConflictAlert Component
 *
 * Displays conflict details and resolution suggestions in a user-friendly format.
 *
 * @example
 * ```tsx
 * <ConflictAlert
 *   conflicts={conflictResponse.conflicts}
 *   suggestions={conflictResponse.suggestions}
 *   onDismiss={() => setShowAlert(false)}
 *   onSelectSuggestion={(suggestion) => handleSuggestion(suggestion)}
 * />
 * ```
 */
export const ConflictAlert: React.FC<ConflictAlertProps> = ({
  conflicts,
  suggestions,
  onDismiss,
  onSelectSuggestion,
  className = '',
}) => {
  if (!conflicts || conflicts.length === 0) {
    return null;
  }

  // Group conflicts by severity
  const criticalConflicts = conflicts.filter(
    (c) => c.severity === ConflictSeverity.CRITICAL
  );
  const highConflicts = conflicts.filter(
    (c) => c.severity === ConflictSeverity.HIGH
  );
  const mediumConflicts = conflicts.filter(
    (c) => c.severity === ConflictSeverity.MEDIUM
  );
  const lowConflicts = conflicts.filter(
    (c) => c.severity === ConflictSeverity.LOW
  );

  return (
    <div className={`conflict-alert space-y-3 ${className}`}>
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <svg
            className="w-6 h-6 text-red-600"
            fill="currentColor"
            viewBox="0 0 20 20"
          >
            <path
              fillRule="evenodd"
              d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
              clipRule="evenodd"
            />
          </svg>
          <h3 className="text-lg font-semibold text-gray-900">
            {conflicts.length} Scheduling Conflict
            {conflicts.length !== 1 ? 's' : ''} Detected
          </h3>
        </div>
        {onDismiss && (
          <button
            onClick={onDismiss}
            className="text-gray-400 hover:text-gray-600 transition-colors"
            aria-label="Dismiss"
          >
            <svg className="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
              <path
                fillRule="evenodd"
                d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                clipRule="evenodd"
              />
            </svg>
          </button>
        )}
      </div>

      {/* Conflicts List */}
      <div className="space-y-2">
        {/* Critical Conflicts */}
        {criticalConflicts.map((conflict, index) => (
          <ConflictDetailCard key={`critical-${index}`} conflict={conflict} />
        ))}

        {/* High Conflicts */}
        {highConflicts.map((conflict, index) => (
          <ConflictDetailCard key={`high-${index}`} conflict={conflict} />
        ))}

        {/* Medium Conflicts */}
        {mediumConflicts.map((conflict, index) => (
          <ConflictDetailCard key={`medium-${index}`} conflict={conflict} />
        ))}

        {/* Low Conflicts */}
        {lowConflicts.map((conflict, index) => (
          <ConflictDetailCard key={`low-${index}`} conflict={conflict} />
        ))}
      </div>

      {/* Resolution Suggestions */}
      {suggestions && suggestions.length > 0 && (
        <ResolutionSuggestionsCard
          suggestions={suggestions}
          onSelectSuggestion={onSelectSuggestion}
        />
      )}
    </div>
  );
};

export default ConflictAlert;

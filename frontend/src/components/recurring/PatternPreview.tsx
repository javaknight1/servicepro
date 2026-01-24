/* eslint-disable react-refresh/only-export-components */
import { useState, useEffect } from 'react';
import { PatternPreviewProps, RecurrenceType, DAYS_OF_WEEK } from './types';

/**
 * Format date for display
 */
const formatDate = (date: Date): string => {
  return date.toLocaleDateString('en-US', {
    weekday: 'short',
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
};

/**
 * Format time for display
 */
const formatTime = (timeStr: string): string => {
  const [hours, minutes] = timeStr.split(':');
  const hour = parseInt(hours);
  const ampm = hour >= 12 ? 'PM' : 'AM';
  const displayHour = hour % 12 || 12;
  return `${displayHour}:${minutes} ${ampm}`;
};

/**
 * Generate preview occurrences client-side
 */
const generatePreviewOccurrences = (
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  pattern: any,
  previewMonths: number
): Date[] => {
  const occurrences: Date[] = [];
  const startDate = new Date(pattern.startDate);
  const endDate = pattern.endDate
    ? new Date(pattern.endDate)
    : new Date(startDate);
  endDate.setMonth(endDate.getMonth() + previewMonths);

  const maxOccurrences = pattern.occurrences || 50;
  const current = new Date(startDate);

  const interval = pattern.interval || 1;

  while (current <= endDate && occurrences.length < maxOccurrences) {
    switch (pattern.recurrenceType) {
      case RecurrenceType.DAILY:
        occurrences.push(new Date(current));
        current.setDate(current.getDate() + interval);
        break;

      case RecurrenceType.WEEKLY:
        if (pattern.daysOfWeek && pattern.daysOfWeek.length > 0) {
          // Check each day of the week
          for (let i = 0; i < 7 * interval; i++) {
            const checkDate = new Date(current);
            checkDate.setDate(checkDate.getDate() + i);

            if (checkDate > endDate) break;
            if (checkDate < startDate) continue;

            const dayOfWeek = checkDate.getDay();
            if (pattern.daysOfWeek.includes(dayOfWeek)) {
              occurrences.push(new Date(checkDate));
              if (occurrences.length >= maxOccurrences) break;
            }
          }
          current.setDate(current.getDate() + 7 * interval);
        }
        break;

      case RecurrenceType.MONTHLY:
        if (pattern.dayOfMonth) {
          const year = current.getFullYear();
          const month = current.getMonth();
          const daysInMonth = new Date(year, month + 1, 0).getDate();
          const day = Math.min(pattern.dayOfMonth, daysInMonth);

          const occurrence = new Date(year, month, day);
          if (occurrence >= startDate && occurrence <= endDate) {
            occurrences.push(occurrence);
          }
          current.setMonth(current.getMonth() + interval);
        }
        break;

      case RecurrenceType.YEARLY:
        if (pattern.monthOfYear && pattern.dayOfMonth) {
          const year = current.getFullYear();
          const occurrence = new Date(
            year,
            pattern.monthOfYear - 1,
            pattern.dayOfMonth
          );
          if (occurrence >= startDate && occurrence <= endDate) {
            occurrences.push(occurrence);
          }
          current.setFullYear(current.getFullYear() + interval);
        }
        break;

      default:
        return occurrences;
    }

    // Safety check to prevent infinite loop
    if (occurrences.length > 1000) break;
  }

  return occurrences;
};

/**
 * Pattern Description Generator
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const generateDescription = (pattern: any): string => {
  const parts: string[] = [];

  // Frequency
  if (pattern.interval === 1) {
    parts.push(pattern.recurrenceType);
  } else {
    parts.push(`Every ${pattern.interval} ${pattern.recurrenceType}s`);
  }

  // Days of week (for weekly)
  if (pattern.recurrenceType === RecurrenceType.WEEKLY && pattern.daysOfWeek) {
    const dayNames = pattern.daysOfWeek
      .map((d: number) => DAYS_OF_WEEK.find((day) => day.value === d)?.label)
      .join(', ');
    parts.push(`on ${dayNames}`);
  }

  // Day of month (for monthly)
  if (pattern.recurrenceType === RecurrenceType.MONTHLY && pattern.dayOfMonth) {
    parts.push(`on day ${pattern.dayOfMonth}`);
  }

  // Month and day (for yearly)
  if (pattern.recurrenceType === RecurrenceType.YEARLY) {
    if (pattern.monthOfYear && pattern.dayOfMonth) {
      const month = new Date(2000, pattern.monthOfYear - 1, 1).toLocaleString(
        'en-US',
        { month: 'long' }
      );
      parts.push(`on ${month} ${pattern.dayOfMonth}`);
    }
  }

  // Time
  if (pattern.timeStart && pattern.timeEnd) {
    parts.push(
      `from ${formatTime(pattern.timeStart)} to ${formatTime(pattern.timeEnd)}`
    );
  }

  // End condition
  if (pattern.endDate) {
    parts.push(`until ${new Date(pattern.endDate).toLocaleDateString()}`);
  } else if (pattern.occurrences) {
    parts.push(`for ${pattern.occurrences} occurrences`);
  }

  return parts.join(' ');
};

/**
 * PatternPreview Component
 *
 * Displays a visual preview of recurring pattern occurrences.
 *
 * Features:
 * - Calendar grid view of occurrences
 * - List view of upcoming occurrences
 * - Human-readable pattern description
 * - Configurable preview duration
 *
 * @example
 * ```tsx
 * <PatternPreview
 *   pattern={formData}
 *   previewMonths={3}
 *   showSkipped={true}
 * />
 * ```
 */
export const PatternPreview: React.FC<PatternPreviewProps> = ({
  pattern,
  previewMonths = 3,
  _showSkipped = false,
}) => {
  const [occurrences, setOccurrences] = useState<Date[]>([]);
  const [viewMode, setViewMode] = useState<'list' | 'calendar'>('list');

  useEffect(() => {
    const generated = generatePreviewOccurrences(pattern, previewMonths);
    setOccurrences(generated);
  }, [pattern, previewMonths]);

  const description = generateDescription(pattern);

  return (
    <div className="bg-white rounded-lg shadow p-6 space-y-4">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-semibold text-gray-900">Pattern Preview</h3>
        <div className="flex gap-2">
          <button
            onClick={() => setViewMode('list')}
            className={`px-3 py-1 rounded text-sm font-medium ${
              viewMode === 'list'
                ? 'bg-blue-600 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            List
          </button>
          <button
            onClick={() => setViewMode('calendar')}
            className={`px-3 py-1 rounded text-sm font-medium ${
              viewMode === 'calendar'
                ? 'bg-blue-600 text-white'
                : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
            }`}
          >
            Calendar
          </button>
        </div>
      </div>

      {/* Description */}
      <div className="p-3 bg-blue-50 border border-blue-200 rounded-md">
        <p className="text-sm text-blue-900">
          <span className="font-semibold">Pattern:</span> {description}
        </p>
      </div>

      {/* Stats */}
      <div className="grid grid-cols-3 gap-4">
        <div className="p-3 bg-gray-50 rounded-md">
          <div className="text-2xl font-bold text-gray-900">
            {occurrences.length}
          </div>
          <div className="text-sm text-gray-600">Occurrences</div>
        </div>
        <div className="p-3 bg-gray-50 rounded-md">
          <div className="text-2xl font-bold text-gray-900">
            {previewMonths}
          </div>
          <div className="text-sm text-gray-600">Months Preview</div>
        </div>
        <div className="p-3 bg-gray-50 rounded-md">
          <div className="text-2xl font-bold text-gray-900">
            {pattern.timeStart && pattern.timeEnd
              ? `${formatTime(pattern.timeStart)} - ${formatTime(
                  pattern.timeEnd
                )}`
              : '-'}
          </div>
          <div className="text-sm text-gray-600">Time Range</div>
        </div>
      </div>

      {/* List View */}
      {viewMode === 'list' && (
        <div className="space-y-2 max-h-96 overflow-y-auto">
          {occurrences.length === 0 ? (
            <div className="text-center py-8 text-gray-500">
              No occurrences to display. Please configure the recurrence
              pattern.
            </div>
          ) : (
            occurrences.map((date, index) => (
              <div
                key={index}
                className="flex items-center justify-between p-3 bg-gray-50 hover:bg-gray-100 rounded-md transition-colors"
              >
                <div className="flex items-center gap-3">
                  <div className="flex-shrink-0 w-12 h-12 bg-blue-100 rounded-md flex items-center justify-center">
                    <div className="text-center">
                      <div className="text-xs font-medium text-blue-600">
                        {date
                          .toLocaleString('en-US', { month: 'short' })
                          .toUpperCase()}
                      </div>
                      <div className="text-lg font-bold text-blue-900">
                        {date.getDate()}
                      </div>
                    </div>
                  </div>
                  <div>
                    <div className="font-medium text-gray-900">
                      {formatDate(date)}
                    </div>
                    <div className="text-sm text-gray-600">
                      {pattern.timeStart &&
                        `${formatTime(pattern.timeStart)} - ${formatTime(
                          pattern.timeEnd
                        )}`}
                    </div>
                  </div>
                </div>
                <div className="text-sm text-gray-500">#{index + 1}</div>
              </div>
            ))
          )}
        </div>
      )}

      {/* Calendar View */}
      {viewMode === 'calendar' && (
        <div className="p-4 bg-gray-50 rounded-md">
          <div className="grid grid-cols-7 gap-2 mb-2">
            {['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'].map((day) => (
              <div
                key={day}
                className="text-center text-xs font-medium text-gray-600"
              >
                {day}
              </div>
            ))}
          </div>
          <div className="text-sm text-gray-600 text-center py-4">
            Calendar view coming soon
          </div>
        </div>
      )}

      {/* Info */}
      {occurrences.length > 0 && (
        <div className="text-sm text-gray-600">
          Showing next {occurrences.length} occurrence
          {occurrences.length !== 1 ? 's' : ''}
          {pattern.occurrences && ` (max ${pattern.occurrences})`}
        </div>
      )}
    </div>
  );
};

export default PatternPreview;

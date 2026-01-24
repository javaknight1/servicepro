import { useState, useEffect } from 'react';
import {
  RecurringFormProps,
  RecurringPatternRequest,
  RecurrenceType,
  DAYS_OF_WEEK,
  MONTHS,
} from './types';

/**
 * RecurringForm Component
 *
 * Form for creating and editing recurring schedule patterns.
 *
 * Features:
 * - Daily, Weekly, Monthly, Yearly recurrence options
 * - Custom interval configuration
 * - End date or occurrence limit
 * - Days of week selection for weekly patterns
 * - Day of month for monthly/yearly patterns
 * - Time range selection
 * - Technician assignment
 *
 * @example
 * ```tsx
 * <RecurringForm
 *   onSubmit={(data) => createPattern(data)}
 *   onCancel={() => navigate('/schedules')}
 *   isLoading={isCreating}
 * />
 * ```
 */
export const RecurringForm: React.FC<RecurringFormProps> = ({
  initialData,
  onSubmit,
  onCancel,
  isLoading = false,
}) => {
  const [formData, setFormData] = useState<RecurringPatternRequest>({
    title: initialData?.title || '',
    description: initialData?.description || '',
    recurrenceType: initialData?.recurrenceType || RecurrenceType.WEEKLY,
    startDate: initialData?.startDate || new Date().toISOString().split('T')[0],
    endDate: initialData?.endDate,
    interval: initialData?.interval || 1,
    daysOfWeek: initialData?.daysOfWeek || [],
    dayOfMonth: initialData?.dayOfMonth,
    monthOfYear: initialData?.monthOfYear,
    occurrences: initialData?.occurrences,
    timeStart: initialData?.timeStart || '09:00',
    timeEnd: initialData?.timeEnd || '17:00',
    assignedTechIds: initialData?.assignedTechIds || [],
    location: initialData?.location || '',
    jobTemplateId: initialData?.jobTemplateId,
  });

  const [endMode, setEndMode] = useState<'never' | 'date' | 'count'>(
    initialData?.endDate ? 'date' : initialData?.occurrences ? 'count' : 'never'
  );

  const [errors, setErrors] = useState<Record<string, string>>({});

  // Update end date/occurrences based on mode
  useEffect(() => {
    if (endMode === 'never') {
      setFormData((prev) => ({
        ...prev,
        endDate: undefined,
        occurrences: undefined,
      }));
    } else if (endMode === 'date') {
      setFormData((prev) => ({ ...prev, occurrences: undefined }));
    } else if (endMode === 'count') {
      setFormData((prev) => ({ ...prev, endDate: undefined }));
    }
  }, [endMode]);

  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const handleInputChange = (
    field: keyof RecurringPatternRequest,
    value: any
  ) => {
    setFormData((prev) => ({ ...prev, [field]: value }));
    // Clear error for this field
    if (errors[field]) {
      setErrors((prev) => {
        const newErrors = { ...prev };
        delete newErrors[field];
        return newErrors;
      });
    }
  };

  const handleDayToggle = (day: number) => {
    setFormData((prev) => {
      const daysOfWeek = prev.daysOfWeek || [];
      const newDays = daysOfWeek.includes(day)
        ? daysOfWeek.filter((d) => d !== day)
        : [...daysOfWeek, day].sort();
      return { ...prev, daysOfWeek: newDays };
    });
  };

  const validate = (): boolean => {
    const newErrors: Record<string, string> = {};

    if (!formData.title.trim()) {
      newErrors.title = 'Title is required';
    }

    if (
      formData.recurrenceType === RecurrenceType.WEEKLY &&
      (!formData.daysOfWeek || formData.daysOfWeek.length === 0)
    ) {
      newErrors.daysOfWeek = 'Select at least one day for weekly recurrence';
    }

    if (
      formData.recurrenceType === RecurrenceType.MONTHLY &&
      !formData.dayOfMonth
    ) {
      newErrors.dayOfMonth = 'Day of month is required for monthly recurrence';
    }

    if (formData.recurrenceType === RecurrenceType.YEARLY) {
      if (!formData.monthOfYear) {
        newErrors.monthOfYear = 'Month is required for yearly recurrence';
      }
      if (!formData.dayOfMonth) {
        newErrors.dayOfMonth = 'Day is required for yearly recurrence';
      }
    }

    if (formData.timeEnd <= formData.timeStart) {
      newErrors.timeEnd = 'End time must be after start time';
    }

    if (formData.endDate && formData.endDate < formData.startDate) {
      newErrors.endDate = 'End date must be after start date';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();

    if (!validate()) {
      return;
    }

    onSubmit(formData);
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-6 max-w-4xl">
      {/* Basic Information */}
      <div className="bg-white rounded-lg shadow p-6 space-y-4">
        <h3 className="text-lg font-semibold text-gray-900">
          Basic Information
        </h3>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Title <span className="text-red-500">*</span>
          </label>
          <input
            type="text"
            value={formData.title}
            onChange={(e) => handleInputChange('title', e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
            placeholder="Weekly Maintenance"
          />
          {errors.title && (
            <p className="mt-1 text-sm text-red-600">{errors.title}</p>
          )}
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Description
          </label>
          <textarea
            value={formData.description}
            onChange={(e) => handleInputChange('description', e.target.value)}
            rows={3}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
            placeholder="Regular maintenance schedule for HVAC systems"
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-1">
            Location
          </label>
          <input
            type="text"
            value={formData.location}
            onChange={(e) => handleInputChange('location', e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
            placeholder="123 Main St, City"
          />
        </div>
      </div>

      {/* Recurrence Pattern */}
      <div className="bg-white rounded-lg shadow p-6 space-y-4">
        <h3 className="text-lg font-semibold text-gray-900">
          Recurrence Pattern
        </h3>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Recurrence Type <span className="text-red-500">*</span>
            </label>
            <select
              value={formData.recurrenceType}
              onChange={(e) =>
                handleInputChange(
                  'recurrenceType',
                  e.target.value as RecurrenceType
                )
              }
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
            >
              <option value={RecurrenceType.DAILY}>Daily</option>
              <option value={RecurrenceType.WEEKLY}>Weekly</option>
              <option value={RecurrenceType.MONTHLY}>Monthly</option>
              <option value={RecurrenceType.YEARLY}>Yearly</option>
            </select>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Repeat Every <span className="text-red-500">*</span>
            </label>
            <div className="flex items-center gap-2">
              <input
                type="number"
                min="1"
                value={formData.interval}
                onChange={(e) =>
                  handleInputChange('interval', parseInt(e.target.value))
                }
                className="w-20 px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
              />
              <span className="text-sm text-gray-600">
                {formData.recurrenceType === RecurrenceType.DAILY && 'day(s)'}
                {formData.recurrenceType === RecurrenceType.WEEKLY && 'week(s)'}
                {formData.recurrenceType === RecurrenceType.MONTHLY &&
                  'month(s)'}
                {formData.recurrenceType === RecurrenceType.YEARLY && 'year(s)'}
              </span>
            </div>
          </div>
        </div>

        {/* Weekly: Days of Week */}
        {formData.recurrenceType === RecurrenceType.WEEKLY && (
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-2">
              Repeat On <span className="text-red-500">*</span>
            </label>
            <div className="flex gap-2">
              {DAYS_OF_WEEK.map((day) => (
                <button
                  key={day.value}
                  type="button"
                  onClick={() => handleDayToggle(day.value)}
                  className={`px-3 py-2 rounded-md text-sm font-medium transition-colors ${
                    formData.daysOfWeek?.includes(day.value)
                      ? 'bg-blue-600 text-white'
                      : 'bg-gray-100 text-gray-700 hover:bg-gray-200'
                  }`}
                >
                  {day.short}
                </button>
              ))}
            </div>
            {errors.daysOfWeek && (
              <p className="mt-1 text-sm text-red-600">{errors.daysOfWeek}</p>
            )}
          </div>
        )}

        {/* Monthly: Day of Month */}
        {formData.recurrenceType === RecurrenceType.MONTHLY && (
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Day of Month <span className="text-red-500">*</span>
            </label>
            <input
              type="number"
              min="1"
              max="31"
              value={formData.dayOfMonth || ''}
              onChange={(e) =>
                handleInputChange('dayOfMonth', parseInt(e.target.value))
              }
              className="w-32 px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
              placeholder="15"
            />
            {errors.dayOfMonth && (
              <p className="mt-1 text-sm text-red-600">{errors.dayOfMonth}</p>
            )}
          </div>
        )}

        {/* Yearly: Month and Day */}
        {formData.recurrenceType === RecurrenceType.YEARLY && (
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Month <span className="text-red-500">*</span>
              </label>
              <select
                value={formData.monthOfYear || ''}
                onChange={(e) =>
                  handleInputChange('monthOfYear', parseInt(e.target.value))
                }
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
              >
                <option value="">Select month</option>
                {MONTHS.map((month) => (
                  <option key={month.value} value={month.value}>
                    {month.label}
                  </option>
                ))}
              </select>
              {errors.monthOfYear && (
                <p className="mt-1 text-sm text-red-600">
                  {errors.monthOfYear}
                </p>
              )}
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700 mb-1">
                Day <span className="text-red-500">*</span>
              </label>
              <input
                type="number"
                min="1"
                max="31"
                value={formData.dayOfMonth || ''}
                onChange={(e) =>
                  handleInputChange('dayOfMonth', parseInt(e.target.value))
                }
                className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
                placeholder="15"
              />
              {errors.dayOfMonth && (
                <p className="mt-1 text-sm text-red-600">{errors.dayOfMonth}</p>
              )}
            </div>
          </div>
        )}
      </div>

      {/* Time and Duration */}
      <div className="bg-white rounded-lg shadow p-6 space-y-4">
        <h3 className="text-lg font-semibold text-gray-900">
          Time and Duration
        </h3>

        <div className="grid grid-cols-3 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Start Date <span className="text-red-500">*</span>
            </label>
            <input
              type="date"
              value={formData.startDate}
              onChange={(e) => handleInputChange('startDate', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Start Time <span className="text-red-500">*</span>
            </label>
            <input
              type="time"
              value={formData.timeStart}
              onChange={(e) => handleInputChange('timeStart', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              End Time <span className="text-red-500">*</span>
            </label>
            <input
              type="time"
              value={formData.timeEnd}
              onChange={(e) => handleInputChange('timeEnd', e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500"
            />
            {errors.timeEnd && (
              <p className="mt-1 text-sm text-red-600">{errors.timeEnd}</p>
            )}
          </div>
        </div>
      </div>

      {/* End Condition */}
      <div className="bg-white rounded-lg shadow p-6 space-y-4">
        <h3 className="text-lg font-semibold text-gray-900">End Condition</h3>

        <div className="space-y-3">
          <label className="flex items-center">
            <input
              type="radio"
              checked={endMode === 'never'}
              onChange={() => setEndMode('never')}
              className="mr-2"
            />
            <span className="text-sm text-gray-700">Never ends</span>
          </label>

          <label className="flex items-center gap-2">
            <input
              type="radio"
              checked={endMode === 'date'}
              onChange={() => setEndMode('date')}
              className="mr-2"
            />
            <span className="text-sm text-gray-700">Ends on</span>
            <input
              type="date"
              value={formData.endDate || ''}
              onChange={(e) => handleInputChange('endDate', e.target.value)}
              disabled={endMode !== 'date'}
              className="px-3 py-1 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500 disabled:bg-gray-100"
            />
          </label>
          {errors.endDate && (
            <p className="ml-6 text-sm text-red-600">{errors.endDate}</p>
          )}

          <label className="flex items-center gap-2">
            <input
              type="radio"
              checked={endMode === 'count'}
              onChange={() => setEndMode('count')}
              className="mr-2"
            />
            <span className="text-sm text-gray-700">Ends after</span>
            <input
              type="number"
              min="1"
              value={formData.occurrences || ''}
              onChange={(e) =>
                handleInputChange('occurrences', parseInt(e.target.value))
              }
              disabled={endMode !== 'count'}
              className="w-20 px-3 py-1 border border-gray-300 rounded-md focus:ring-blue-500 focus:border-blue-500 disabled:bg-gray-100"
            />
            <span className="text-sm text-gray-700">occurrence(s)</span>
          </label>
        </div>
      </div>

      {/* Actions */}
      <div className="flex justify-end gap-3">
        {onCancel && (
          <button
            type="button"
            onClick={onCancel}
            disabled={isLoading}
            className="px-4 py-2 border border-gray-300 rounded-md text-gray-700 hover:bg-gray-50 disabled:opacity-50"
          >
            Cancel
          </button>
        )}
        <button
          type="submit"
          disabled={isLoading}
          className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50 flex items-center gap-2"
        >
          {isLoading && (
            <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white"></div>
          )}
          {initialData ? 'Update Pattern' : 'Create Pattern'}
        </button>
      </div>
    </form>
  );
};

export default RecurringForm;

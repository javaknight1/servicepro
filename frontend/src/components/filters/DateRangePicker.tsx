import React, { useState, useRef, useEffect, useCallback, memo } from 'react';
import { cn } from '../../utils/cn';
import {
  DateRangePickerProps,
  DateRange,
  DateRangePreset,
  dateRangePresets,
} from '../../types/filter';

// Calendar icon
const CalendarIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg
    className={className}
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
    strokeWidth={2}
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"
    />
  </svg>
);

// Chevron icons
const ChevronLeftIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg
    className={className}
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M15 19l-7-7 7-7"
    />
  </svg>
);

const ChevronRightIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg
    className={className}
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M9 5l7 7-7 7"
    />
  </svg>
);

// Clear icon
const XIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg
    className={className}
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      strokeWidth={2}
      d="M6 18L18 6M6 6l12 12"
    />
  </svg>
);

// Month names
const MONTHS = [
  'January',
  'February',
  'March',
  'April',
  'May',
  'June',
  'July',
  'August',
  'September',
  'October',
  'November',
  'December',
];

// Day names
const DAYS = ['Su', 'Mo', 'Tu', 'We', 'Th', 'Fr', 'Sa'];

// Get days in month
const getDaysInMonth = (year: number, month: number): number => {
  return new Date(year, month + 1, 0).getDate();
};

// Get first day of month (0 = Sunday)
const getFirstDayOfMonth = (year: number, month: number): number => {
  return new Date(year, month, 1).getDay();
};

// Check if date is in range
const isInRange = (
  date: Date,
  start: Date | null,
  end: Date | null
): boolean => {
  if (!start || !end) return false;
  return date >= start && date <= end;
};

// Check if dates are same day
const isSameDay = (date1: Date, date2: Date): boolean => {
  return (
    date1.getFullYear() === date2.getFullYear() &&
    date1.getMonth() === date2.getMonth() &&
    date1.getDate() === date2.getDate()
  );
};

// Format date for display
const formatDate = (date: Date | null): string => {
  if (!date) return '';
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
};

// Calendar component
interface CalendarProps {
  month: number;
  year: number;
  selectedStart: Date | null;
  selectedEnd: Date | null;
  hoverDate: Date | null;
  minDate?: Date;
  maxDate?: Date;
  onDateClick: (date: Date) => void;
  onDateHover: (date: Date | null) => void;
  onMonthChange: (month: number, year: number) => void;
}

const Calendar: React.FC<CalendarProps> = memo(
  ({
    month,
    year,
    selectedStart,
    selectedEnd,
    hoverDate,
    minDate,
    maxDate,
    onDateClick,
    onDateHover,
    onMonthChange,
  }) => {
    const daysInMonth = getDaysInMonth(year, month);
    const firstDay = getFirstDayOfMonth(year, month);
    const today = new Date();
    today.setHours(0, 0, 0, 0);

    const handlePrevMonth = () => {
      if (month === 0) {
        onMonthChange(11, year - 1);
      } else {
        onMonthChange(month - 1, year);
      }
    };

    const handleNextMonth = () => {
      if (month === 11) {
        onMonthChange(0, year + 1);
      } else {
        onMonthChange(month + 1, year);
      }
    };

    const isDateDisabled = (date: Date): boolean => {
      if (minDate && date < minDate) return true;
      if (maxDate && date > maxDate) return true;
      return false;
    };

    const getDateClasses = (date: Date): string => {
      const isDisabled = isDateDisabled(date);
      const isToday = isSameDay(date, today);
      const isStart = selectedStart && isSameDay(date, selectedStart);
      const isEnd = selectedEnd && isSameDay(date, selectedEnd);
      const isSelected = isStart || isEnd;

      // Determine if date is in range (between start and end, or between start and hover)
      let inRange = false;
      if (selectedStart && selectedEnd) {
        inRange = isInRange(date, selectedStart, selectedEnd);
      } else if (selectedStart && hoverDate && !selectedEnd) {
        const rangeStart =
          selectedStart < hoverDate ? selectedStart : hoverDate;
        const rangeEnd = selectedStart < hoverDate ? hoverDate : selectedStart;
        inRange = isInRange(date, rangeStart, rangeEnd);
      }

      return cn(
        'w-9 h-9 flex items-center justify-center text-sm rounded-full transition-colors',
        isDisabled && 'text-gray-300 cursor-not-allowed',
        !isDisabled && !isSelected && 'hover:bg-gray-100 cursor-pointer',
        isToday && !isSelected && 'font-bold text-blue-600',
        isSelected && 'bg-blue-600 text-white font-medium',
        inRange && !isSelected && 'bg-blue-50',
        isStart && selectedEnd && 'rounded-r-none',
        isEnd && selectedStart && 'rounded-l-none'
      );
    };

    // Generate calendar grid
    const days = [];

    // Empty cells before first day
    for (let i = 0; i < firstDay; i++) {
      days.push(<div key={`empty-${i}`} className="w-9 h-9" />);
    }

    // Days of month
    for (let day = 1; day <= daysInMonth; day++) {
      const date = new Date(year, month, day);
      date.setHours(0, 0, 0, 0);
      const disabled = isDateDisabled(date);

      days.push(
        <button
          key={day}
          type="button"
          disabled={disabled}
          className={getDateClasses(date)}
          onClick={() => !disabled && onDateClick(date)}
          onMouseEnter={() => !disabled && onDateHover(date)}
          onMouseLeave={() => onDateHover(null)}
          data-testid={`calendar-day-${day}`}
        >
          {day}
        </button>
      );
    }

    return (
      <div className="w-64" data-testid="calendar">
        {/* Header */}
        <div className="flex items-center justify-between mb-4">
          <button
            type="button"
            onClick={handlePrevMonth}
            className="p-1 hover:bg-gray-100 rounded-full"
            data-testid="prev-month"
          >
            <ChevronLeftIcon className="w-5 h-5 text-gray-600" />
          </button>
          <span className="font-medium text-gray-900">
            {MONTHS[month]} {year}
          </span>
          <button
            type="button"
            onClick={handleNextMonth}
            className="p-1 hover:bg-gray-100 rounded-full"
            data-testid="next-month"
          >
            <ChevronRightIcon className="w-5 h-5 text-gray-600" />
          </button>
        </div>

        {/* Day names */}
        <div className="grid grid-cols-7 gap-0 mb-2">
          {DAYS.map((day) => (
            <div
              key={day}
              className="w-9 h-8 flex items-center justify-center text-xs font-medium text-gray-500"
            >
              {day}
            </div>
          ))}
        </div>

        {/* Days grid */}
        <div className="grid grid-cols-7 gap-0">{days}</div>
      </div>
    );
  }
);

Calendar.displayName = 'Calendar';

// Main DateRangePicker component
const DateRangePicker: React.FC<DateRangePickerProps> = ({
  value,
  onChange,
  label,
  placeholder = 'Select date range',
  minDate,
  maxDate,
  presets = dateRangePresets,
  _showTime = false,
  disabled = false,
  className,
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [hoverDate, setHoverDate] = useState<Date | null>(null);
  const [selectingStart, setSelectingStart] = useState(true);
  const [tempRange, setTempRange] = useState<DateRange>(
    value || { start: null, end: null }
  );

  const containerRef = useRef<HTMLDivElement>(null);
  const today = new Date();

  // Initialize calendar months
  const [leftMonth, setLeftMonth] = useState(
    value?.start ? value.start.getMonth() : today.getMonth()
  );
  const [leftYear, setLeftYear] = useState(
    value?.start ? value.start.getFullYear() : today.getFullYear()
  );

  // Right calendar is always one month ahead
  const rightMonth = leftMonth === 11 ? 0 : leftMonth + 1;
  const rightYear = leftMonth === 11 ? leftYear + 1 : leftYear;

  // Close on click outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        containerRef.current &&
        !containerRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Update temp range when value changes
  useEffect(() => {
    setTempRange(value || { start: null, end: null });
  }, [value]);

  const handleDateClick = useCallback(
    (date: Date) => {
      if (selectingStart) {
        // Starting new selection
        setTempRange({ start: date, end: null });
        setSelectingStart(false);
      } else {
        // Completing selection
        const start = tempRange.start;
        if (start && date >= start) {
          const newRange = { start, end: date };
          setTempRange(newRange);
          onChange(newRange);
          setIsOpen(false);
        } else if (start) {
          // User clicked before start date, swap
          const newRange = { start: date, end: start };
          setTempRange(newRange);
          onChange(newRange);
          setIsOpen(false);
        }
        setSelectingStart(true);
      }
    },
    [selectingStart, tempRange.start, onChange]
  );

  const handlePresetClick = useCallback(
    (preset: DateRangePreset) => {
      const range = preset.getValue();
      setTempRange(range);
      onChange(range);
      setIsOpen(false);
    },
    [onChange]
  );

  const handleClear = useCallback(
    (e: React.MouseEvent) => {
      e.stopPropagation();
      setTempRange({ start: null, end: null });
      onChange(null);
      setSelectingStart(true);
    },
    [onChange]
  );

  const handleOpen = () => {
    if (!disabled) {
      setIsOpen(true);
      setTempRange(value || { start: null, end: null });
      setSelectingStart(true);
    }
  };

  const handleLeftMonthChange = (month: number, year: number) => {
    setLeftMonth(month);
    setLeftYear(year);
  };

  // Display value
  const displayValue =
    value?.start && value?.end
      ? `${formatDate(value.start)} - ${formatDate(value.end)}`
      : placeholder;

  return (
    <div
      ref={containerRef}
      className={cn('relative', className)}
      data-testid="date-range-picker"
    >
      {/* Label */}
      {label && (
        <label className="block text-sm font-medium text-gray-700 mb-1">
          {label}
        </label>
      )}

      {/* Trigger button */}
      <button
        type="button"
        onClick={handleOpen}
        disabled={disabled}
        className={cn(
          'w-full flex items-center justify-between px-3 py-2 border rounded-lg text-left transition-colors',
          disabled
            ? 'bg-gray-100 text-gray-400 cursor-not-allowed'
            : 'bg-white border-gray-300 hover:border-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500',
          isOpen && 'ring-2 ring-blue-500 border-blue-500'
        )}
        data-testid="date-range-trigger"
      >
        <div className="flex items-center gap-2">
          <CalendarIcon className="w-5 h-5 text-gray-400" />
          <span
            className={cn(
              'text-sm',
              value?.start ? 'text-gray-900' : 'text-gray-400'
            )}
          >
            {displayValue}
          </span>
        </div>
        {value?.start && !disabled && (
          <button
            type="button"
            onClick={handleClear}
            className="p-0.5 hover:bg-gray-100 rounded"
            data-testid="clear-button"
          >
            <XIcon className="w-4 h-4 text-gray-400" />
          </button>
        )}
      </button>

      {/* Dropdown */}
      {isOpen && (
        <div
          className="absolute z-50 mt-1 bg-white rounded-lg shadow-xl border border-gray-200 p-4"
          data-testid="date-range-dropdown"
        >
          <div className="flex gap-6">
            {/* Presets */}
            {presets.length > 0 && (
              <div className="border-r border-gray-200 pr-4">
                <div className="text-xs font-medium text-gray-500 uppercase mb-2">
                  Quick Select
                </div>
                <div className="space-y-1">
                  {presets.map((preset) => (
                    <button
                      key={preset.label}
                      type="button"
                      onClick={() => handlePresetClick(preset)}
                      className="w-full text-left px-2 py-1.5 text-sm text-gray-700 hover:bg-gray-100 rounded"
                      data-testid={`preset-${preset.label
                        .toLowerCase()
                        .replace(/\s/g, '-')}`}
                    >
                      {preset.label}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {/* Calendars */}
            <div className="flex gap-4">
              <Calendar
                month={leftMonth}
                year={leftYear}
                selectedStart={tempRange.start}
                selectedEnd={tempRange.end}
                hoverDate={hoverDate}
                minDate={minDate}
                maxDate={maxDate}
                onDateClick={handleDateClick}
                onDateHover={setHoverDate}
                onMonthChange={handleLeftMonthChange}
              />
              <Calendar
                month={rightMonth}
                year={rightYear}
                selectedStart={tempRange.start}
                selectedEnd={tempRange.end}
                hoverDate={hoverDate}
                minDate={minDate}
                maxDate={maxDate}
                onDateClick={handleDateClick}
                onDateHover={setHoverDate}
                onMonthChange={(m, y) => {
                  // Adjust left calendar when right changes
                  if (m === 0) {
                    handleLeftMonthChange(11, y - 1);
                  } else {
                    handleLeftMonthChange(m - 1, y);
                  }
                }}
              />
            </div>
          </div>

          {/* Footer */}
          <div className="flex items-center justify-between mt-4 pt-4 border-t border-gray-200">
            <div className="text-sm text-gray-600">
              {selectingStart ? 'Select start date' : 'Select end date'}
            </div>
            <div className="flex gap-2">
              <button
                type="button"
                onClick={() => setIsOpen(false)}
                className="px-3 py-1.5 text-sm text-gray-600 hover:bg-gray-100 rounded"
              >
                Cancel
              </button>
              {tempRange.start && tempRange.end && (
                <button
                  type="button"
                  onClick={() => {
                    onChange(tempRange);
                    setIsOpen(false);
                  }}
                  className="px-3 py-1.5 text-sm bg-blue-600 text-white rounded hover:bg-blue-700"
                >
                  Apply
                </button>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

export default memo(DateRangePicker);

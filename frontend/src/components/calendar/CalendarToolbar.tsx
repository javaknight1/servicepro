import React from 'react';
import { CalendarToolbarProps, CalendarView, CALENDAR_VIEWS } from './types';
import { toolbarStyles, cn } from './styles';
import { ChevronLeft, ChevronRight, Calendar } from 'lucide-react';

/**
 * Custom toolbar for calendar navigation and view selection
 */
export const CalendarToolbar: React.FC<CalendarToolbarProps> = ({
  date,
  view,
  onNavigate,
  onViewChange,
  label,
}) => {
  return (
    <div className={toolbarStyles.container}>
      {/* Navigation Controls */}
      <div className={toolbarStyles.navigationGroup}>
        <button
          type="button"
          onClick={() => onNavigate('PREV')}
          className={toolbarStyles.navButton}
          aria-label="Previous"
        >
          <ChevronLeft className="w-5 h-5" />
        </button>

        <button
          type="button"
          onClick={() => onNavigate('TODAY')}
          className={toolbarStyles.todayButton}
          aria-label="Today"
        >
          <Calendar className="w-4 h-4 inline-block mr-2" />
          Today
        </button>

        <button
          type="button"
          onClick={() => onNavigate('NEXT')}
          className={toolbarStyles.navButton}
          aria-label="Next"
        >
          <ChevronRight className="w-5 h-5" />
        </button>
      </div>

      {/* Date Label */}
      <div className={toolbarStyles.label}>{label}</div>

      {/* View Selector */}
      <div className={toolbarStyles.viewGroup}>
        {CALENDAR_VIEWS.map((viewOption) => (
          <button
            key={viewOption}
            type="button"
            onClick={() => onViewChange(viewOption)}
            className={
              view === viewOption
                ? toolbarStyles.viewButtonActive
                : toolbarStyles.viewButton
            }
            aria-label={`${viewOption} view`}
            aria-pressed={view === viewOption}
          >
            {viewOption.charAt(0).toUpperCase() + viewOption.slice(1)}
          </button>
        ))}
      </div>
    </div>
  );
};

/**
 * Memoized version of CalendarToolbar for performance
 */
export default React.memo(CalendarToolbar);

import React from 'react';
import { CalendarEventProps, JobPriority, JOB_PRIORITY_COLORS } from './types';
import { eventStyles, cn } from './styles';
import { Clock, MapPin, AlertCircle } from 'lucide-react';

/**
 * Custom event component for calendar
 * Renders job information with status colors and metadata
 */
export const CalendarEvent: React.FC<CalendarEventProps> = ({
  event,
  title,
}) => {
  const { jobNumber, priority, location, start, end } = event;

  // Format time range
  const formatTime = (date: Date): string => {
    return new Intl.DateTimeFormat('en-US', {
      hour: 'numeric',
      minute: '2-digit',
      hour12: true,
    }).format(date);
  };

  const timeRange = `${formatTime(start)} - ${formatTime(end)}`;

  // Get priority badge color
  const getPriorityBadgeColor = (priorityLevel?: JobPriority): string => {
    if (!priorityLevel) return '';
    return JOB_PRIORITY_COLORS[priorityLevel];
  };

  return (
    <div className={eventStyles.container}>
      {/* Title and Job Number */}
      <div className="flex items-start justify-between gap-1">
        <div className="flex-1 min-w-0">
          <div className={eventStyles.title} title={title}>
            {title}
          </div>
          <div className={eventStyles.jobNumber} title={jobNumber}>
            #{jobNumber}
          </div>
        </div>

        {/* Priority Badge */}
        {priority && (
          <div
            className={cn(
              eventStyles.priorityBadge,
              getPriorityBadgeColor(priority),
              'shrink-0'
            )}
            title={`Priority: ${priority}`}
          >
            {priority === JobPriority.URGENT && (
              <AlertCircle className="inline-block w-3 h-3 mr-1" />
            )}
            {priority.charAt(0).toUpperCase()}
          </div>
        )}
      </div>

      {/* Time Range - Hidden in month view */}
      <div
        className={cn(
          eventStyles.time,
          'hidden sm:flex items-center gap-1 mt-1'
        )}
      >
        <Clock className="w-3 h-3" />
        <span>{timeRange}</span>
      </div>

      {/* Location - Hidden in compact views */}
      {location && (
        <div
          className={cn(
            eventStyles.location,
            'hidden md:flex items-center gap-1'
          )}
        >
          <MapPin className="w-3 h-3" />
          <span title={location}>{location}</span>
        </div>
      )}
    </div>
  );
};

/**
 * Memoized version of CalendarEvent for performance
 */
export default React.memo(CalendarEvent);

import { useEffect, useRef } from 'react';
import { format } from 'date-fns';
import { X, Clock, MapPin, User, Briefcase, ExternalLink } from 'lucide-react';
import { JobEvent, CalendarJobStatus, JOB_STATUS_COLORS } from './types';
import { cn } from './styles';

interface CalendarEventPopoverProps {
  event: JobEvent;
  position: { x: number; y: number };
  onClose: () => void;
  onEdit: (event: JobEvent) => void;
}

const STATUS_LABELS: Record<CalendarJobStatus, string> = {
  [CalendarJobStatus.SCHEDULED]: 'Scheduled',
  [CalendarJobStatus.IN_PROGRESS]: 'In Progress',
  [CalendarJobStatus.ON_HOLD]: 'On Hold',
  [CalendarJobStatus.COMPLETED]: 'Completed',
  [CalendarJobStatus.CANCELLED]: 'Cancelled',
};

export function CalendarEventPopover({
  event,
  position,
  onClose,
  onEdit,
}: CalendarEventPopoverProps) {
  const popoverRef = useRef<HTMLDivElement>(null);

  // Close on click outside
  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (
        popoverRef.current &&
        !popoverRef.current.contains(e.target as Node)
      ) {
        onClose();
      }
    }

    // Close on escape key
    function handleEscape(e: KeyboardEvent) {
      if (e.key === 'Escape') {
        onClose();
      }
    }

    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleEscape);

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [onClose]);

  // Adjust position to keep popover in viewport
  useEffect(() => {
    if (popoverRef.current) {
      const rect = popoverRef.current.getBoundingClientRect();
      const viewportWidth = window.innerWidth;
      const viewportHeight = window.innerHeight;

      // Adjust horizontal position if overflowing
      if (rect.right > viewportWidth - 20) {
        popoverRef.current.style.left = `${viewportWidth - rect.width - 20}px`;
      }

      // Adjust vertical position if overflowing
      if (rect.bottom > viewportHeight - 20) {
        popoverRef.current.style.top = `${viewportHeight - rect.height - 20}px`;
      }
    }
  }, [position]);

  const statusColor = JOB_STATUS_COLORS[event.status] || 'bg-gray-500';

  return (
    <div
      ref={popoverRef}
      className="fixed z-50 bg-white rounded-lg shadow-xl border border-neutral-200 w-80 animate-in fade-in zoom-in-95 duration-150"
      style={{
        left: position.x,
        top: position.y,
      }}
    >
      {/* Header */}
      <div className="flex items-start justify-between p-4 border-b border-neutral-100">
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2 mb-1">
            <span
              className={cn(
                'inline-block w-2 h-2 rounded-full flex-shrink-0',
                statusColor
              )}
            />
            <span className="text-xs font-medium text-neutral-500 uppercase tracking-wide">
              {STATUS_LABELS[event.status]}
            </span>
          </div>
          <h3 className="font-semibold text-neutral-900 truncate">
            {event.title}
          </h3>
          <p className="text-sm text-primary-600 font-mono">
            {event.jobNumber}
          </p>
        </div>
        <button
          onClick={onClose}
          className="p-1 rounded-md text-neutral-400 hover:text-neutral-600 hover:bg-neutral-100 transition-colors"
          aria-label="Close"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Content */}
      <div className="p-4 space-y-3">
        {/* Time */}
        <div className="flex items-center gap-3 text-sm">
          <Clock className="h-4 w-4 text-neutral-400 flex-shrink-0" />
          <span className="text-neutral-700">
            {format(event.start, 'MMM d, yyyy')} &middot;{' '}
            {format(event.start, 'h:mm a')} - {format(event.end, 'h:mm a')}
          </span>
        </div>

        {/* Customer */}
        {event.customerName && (
          <div className="flex items-center gap-3 text-sm">
            <User className="h-4 w-4 text-neutral-400 flex-shrink-0" />
            <span className="text-neutral-700">{event.customerName}</span>
          </div>
        )}

        {/* Location */}
        {event.location && (
          <div className="flex items-start gap-3 text-sm">
            <MapPin className="h-4 w-4 text-neutral-400 flex-shrink-0 mt-0.5" />
            <span className="text-neutral-700">{event.location}</span>
          </div>
        )}

        {/* Description */}
        {event.description && (
          <div className="flex items-start gap-3 text-sm">
            <Briefcase className="h-4 w-4 text-neutral-400 flex-shrink-0 mt-0.5" />
            <span className="text-neutral-600 line-clamp-2">
              {event.description}
            </span>
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="p-4 border-t border-neutral-100 bg-neutral-50 rounded-b-lg">
        <button
          onClick={() => onEdit(event)}
          className="w-full flex items-center justify-center gap-2 px-4 py-2 bg-primary-600 text-white text-sm font-medium rounded-lg hover:bg-primary-700 transition-colors"
        >
          <ExternalLink className="h-4 w-4" />
          View Job Details
        </button>
      </div>
    </div>
  );
}

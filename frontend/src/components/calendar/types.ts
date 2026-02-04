import { Event as RBCEvent } from 'react-big-calendar';
import { JobStatus as FullJobStatus } from '@services/jobService';

/**
 * Calendar job status enum (subset of full JobStatus for display purposes)
 * Maps the 9 backend statuses to 5 visual categories for the calendar
 */
export enum CalendarJobStatus {
  SCHEDULED = 'scheduled',
  IN_PROGRESS = 'in_progress',
  ON_HOLD = 'on_hold',
  COMPLETED = 'completed',
  CANCELLED = 'cancelled',
}

/**
 * Maps full job status to calendar display status
 * This keeps the calendar visuals simple while supporting all backend statuses
 */
export function mapToCalendarStatus(status: FullJobStatus): CalendarJobStatus {
  switch (status) {
    case FullJobStatus.NEW:
    case FullJobStatus.SCHEDULED:
      return CalendarJobStatus.SCHEDULED;
    case FullJobStatus.EN_ROUTE:
    case FullJobStatus.IN_PROGRESS:
      return CalendarJobStatus.IN_PROGRESS;
    case FullJobStatus.ON_HOLD:
      return CalendarJobStatus.ON_HOLD;
    case FullJobStatus.COMPLETED:
    case FullJobStatus.INVOICED:
    case FullJobStatus.PAID:
      return CalendarJobStatus.COMPLETED;
    case FullJobStatus.CANCELLED:
      return CalendarJobStatus.CANCELLED;
    default:
      return CalendarJobStatus.SCHEDULED;
  }
}

// Re-export for backwards compatibility
export { CalendarJobStatus as JobStatus };

/**
 * Job priority levels
 */
export enum JobPriority {
  LOW = 'low',
  NORMAL = 'normal',
  HIGH = 'high',
  URGENT = 'urgent',
}

/**
 * Calendar view type
 */
export type CalendarView = 'day' | 'week' | 'month' | 'agenda';

/**
 * Calendar resource (e.g., technician, equipment)
 */
export interface CalendarResource {
  id: string;
  title: string;
  color?: string;
}

/**
 * Job event interface for calendar
 */
export interface JobEvent extends RBCEvent {
  id: string;
  title: string;
  start: Date;
  end: Date;
  jobNumber: string;
  status: CalendarJobStatus;
  priority?: JobPriority;
  description?: string;
  assignedTo?: string[];
  customerId?: string;
  customerName?: string;
  location?: string;
  resource?: CalendarResource | string;
}

/**
 * Drag and drop event data
 */
export interface DragDropEvent {
  event: JobEvent;
  start: Date;
  end: Date;
  isAllDay: boolean;
}

/**
 * Resize event data
 */
export interface ResizeEvent {
  event: JobEvent;
  start: Date;
  end: Date;
}

/**
 * Event click data including position for popover
 */
export interface EventClickData {
  event: JobEvent;
  position: { x: number; y: number };
}

/**
 * Calendar props
 */
export interface CalendarProps {
  events: JobEvent[];
  onEventClick?: (data: EventClickData) => void;
  onEventDrop?: (data: DragDropEvent) => Promise<void>;
  onEventResize?: (data: ResizeEvent) => Promise<void>;
  onSelectSlot?: (slotInfo: { start: Date; end: Date; slots: Date[] }) => void;
  onNavigate?: (date: Date) => void;
  onViewChange?: (view: CalendarView) => void;
  defaultView?: CalendarView;
  defaultDate?: Date;
  isLoading?: boolean;
  className?: string;
}

/**
 * Calendar toolbar props
 */
export interface CalendarToolbarProps {
  date: Date;
  view: CalendarView;
  onNavigate: (action: 'PREV' | 'NEXT' | 'TODAY') => void;
  onViewChange: (view: CalendarView) => void;
  label: string;
}

/**
 * Calendar event component props
 */
export interface CalendarEventProps {
  event: JobEvent;
  title: string;
}

/**
 * Job status color mapping
 */
export const JOB_STATUS_COLORS: Record<CalendarJobStatus, string> = {
  [CalendarJobStatus.SCHEDULED]: 'bg-blue-500',
  [CalendarJobStatus.IN_PROGRESS]: 'bg-yellow-500',
  [CalendarJobStatus.ON_HOLD]: 'bg-orange-500',
  [CalendarJobStatus.COMPLETED]: 'bg-green-500',
  [CalendarJobStatus.CANCELLED]: 'bg-gray-500',
};

/**
 * Job status border colors
 */
export const JOB_STATUS_BORDER_COLORS: Record<CalendarJobStatus, string> = {
  [CalendarJobStatus.SCHEDULED]: 'border-blue-600',
  [CalendarJobStatus.IN_PROGRESS]: 'border-yellow-600',
  [CalendarJobStatus.ON_HOLD]: 'border-orange-600',
  [CalendarJobStatus.COMPLETED]: 'border-green-600',
  [CalendarJobStatus.CANCELLED]: 'border-gray-600',
};

/**
 * Job status text colors
 */
export const JOB_STATUS_TEXT_COLORS: Record<CalendarJobStatus, string> = {
  [CalendarJobStatus.SCHEDULED]: 'text-blue-700',
  [CalendarJobStatus.IN_PROGRESS]: 'text-yellow-700',
  [CalendarJobStatus.ON_HOLD]: 'text-orange-700',
  [CalendarJobStatus.COMPLETED]: 'text-green-700',
  [CalendarJobStatus.CANCELLED]: 'text-gray-700',
};

/**
 * Job priority colors
 */
export const JOB_PRIORITY_COLORS: Record<JobPriority, string> = {
  [JobPriority.LOW]: 'bg-gray-200',
  [JobPriority.NORMAL]: 'bg-blue-200',
  [JobPriority.HIGH]: 'bg-orange-200',
  [JobPriority.URGENT]: 'bg-red-200',
};

/**
 * Calendar views configuration
 */
export const CALENDAR_VIEWS: CalendarView[] = [
  'month',
  'week',
  'day',
  'agenda',
];

/**
 * Error boundary state
 */
export interface CalendarErrorBoundaryState {
  hasError: boolean;
  error: Error | null;
}

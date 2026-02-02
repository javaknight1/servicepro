import { Event as RBCEvent } from 'react-big-calendar';

/**
 * Job status enum matching backend
 */
export enum JobStatus {
  SCHEDULED = 'scheduled',
  IN_PROGRESS = 'in_progress',
  ON_HOLD = 'on_hold',
  COMPLETED = 'completed',
  CANCELLED = 'cancelled',
}

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
  status: JobStatus;
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
 * Calendar props
 */
export interface CalendarProps {
  events: JobEvent[];
  onEventClick?: (event: JobEvent) => void;
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
export const JOB_STATUS_COLORS: Record<JobStatus, string> = {
  [JobStatus.SCHEDULED]: 'bg-blue-500',
  [JobStatus.IN_PROGRESS]: 'bg-yellow-500',
  [JobStatus.ON_HOLD]: 'bg-orange-500',
  [JobStatus.COMPLETED]: 'bg-green-500',
  [JobStatus.CANCELLED]: 'bg-gray-500',
};

/**
 * Job status border colors
 */
export const JOB_STATUS_BORDER_COLORS: Record<JobStatus, string> = {
  [JobStatus.SCHEDULED]: 'border-blue-600',
  [JobStatus.IN_PROGRESS]: 'border-yellow-600',
  [JobStatus.ON_HOLD]: 'border-orange-600',
  [JobStatus.COMPLETED]: 'border-green-600',
  [JobStatus.CANCELLED]: 'border-gray-600',
};

/**
 * Job status text colors
 */
export const JOB_STATUS_TEXT_COLORS: Record<JobStatus, string> = {
  [JobStatus.SCHEDULED]: 'text-blue-700',
  [JobStatus.IN_PROGRESS]: 'text-yellow-700',
  [JobStatus.ON_HOLD]: 'text-orange-700',
  [JobStatus.COMPLETED]: 'text-green-700',
  [JobStatus.CANCELLED]: 'text-gray-700',
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

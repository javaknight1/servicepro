/**
 * Calendar Component Exports
 *
 * Central export point for all calendar-related components, types, and utilities
 */

// Main Components
export { Calendar, default } from './Calendar';
export { CalendarEvent } from './CalendarEvent';
export { CalendarToolbar } from './CalendarToolbar';

// Types
export type {
  JobEvent,
  CalendarView,
  CalendarProps,
  CalendarEventProps,
  CalendarToolbarProps,
  DragDropEvent,
  ResizeEvent,
  CalendarErrorBoundaryState,
} from './types';

export {
  JobStatus,
  JobPriority,
  JOB_STATUS_COLORS,
  JOB_STATUS_BORDER_COLORS,
  JOB_STATUS_TEXT_COLORS,
  JOB_PRIORITY_COLORS,
  CALENDAR_VIEWS,
} from './types';

// Styles and utilities
export {
  calendarStyles,
  getEventStyles,
  toolbarStyles,
  eventStyles,
  cn,
} from './styles';

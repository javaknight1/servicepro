import { JobStatus } from './types';

/**
 * Base calendar styles using Tailwind classes
 */
export const calendarStyles = {
  container: 'h-full w-full bg-white rounded-lg shadow-lg',
  loadingOverlay:
    'absolute inset-0 bg-white bg-opacity-75 flex items-center justify-center z-50',
  loadingSpinner:
    'animate-spin h-12 w-12 border-4 border-blue-500 border-t-transparent rounded-full',
};

/**
 * Event styles based on status
 */
export const getEventStyles = (status: JobStatus): string => {
  const baseStyles =
    'text-white rounded px-2 py-1 text-sm font-medium transition-all duration-200 cursor-pointer';

  const statusStyles: Record<JobStatus, string> = {
    [JobStatus.SCHEDULED]:
      'bg-blue-500 hover:bg-blue-600 border-l-4 border-blue-700',
    [JobStatus.IN_PROGRESS]:
      'bg-yellow-500 hover:bg-yellow-600 border-l-4 border-yellow-700',
    [JobStatus.ON_HOLD]:
      'bg-orange-500 hover:bg-orange-600 border-l-4 border-orange-700',
    [JobStatus.COMPLETED]:
      'bg-green-500 hover:bg-green-600 border-l-4 border-green-700',
    [JobStatus.CANCELLED]:
      'bg-gray-500 hover:bg-gray-600 border-l-4 border-gray-700',
  };

  return `${baseStyles} ${statusStyles[status]}`;
};

/**
 * React Big Calendar custom styles (CSS-in-JS)
 */
export const rbcStyles = {
  '.rbc-calendar': {
    fontFamily: 'Inter, system-ui, sans-serif',
    height: '100%',
  },
  '.rbc-header': {
    padding: '12px 6px',
    fontWeight: '600',
    fontSize: '0.875rem',
    color: '#374151',
    borderBottom: '2px solid #E5E7EB',
    backgroundColor: '#F9FAFB',
  },
  '.rbc-today': {
    backgroundColor: '#EFF6FF',
  },
  '.rbc-off-range-bg': {
    backgroundColor: '#F9FAFB',
  },
  '.rbc-event': {
    padding: '4px 8px',
    borderRadius: '4px',
    border: 'none',
    fontSize: '0.875rem',
    boxShadow: '0 1px 3px rgba(0, 0, 0, 0.1)',
  },
  '.rbc-event.rbc-selected': {
    backgroundColor: 'inherit',
    boxShadow: '0 0 0 3px rgba(59, 130, 246, 0.5)',
  },
  '.rbc-event-label': {
    fontSize: '0.75rem',
    fontWeight: '500',
  },
  '.rbc-event-content': {
    fontSize: '0.875rem',
    lineHeight: '1.25rem',
  },
  '.rbc-slot-selection': {
    backgroundColor: 'rgba(59, 130, 246, 0.1)',
    border: '2px dashed #3B82F6',
  },
  '.rbc-day-slot .rbc-time-slot': {
    borderTop: '1px solid #E5E7EB',
  },
  '.rbc-time-header-content': {
    borderLeft: '1px solid #E5E7EB',
  },
  '.rbc-time-content': {
    borderTop: '2px solid #E5E7EB',
  },
  '.rbc-current-time-indicator': {
    backgroundColor: '#EF4444',
    height: '2px',
  },
  '.rbc-agenda-view': {
    padding: '16px',
  },
  '.rbc-agenda-table': {
    borderSpacing: '0',
    width: '100%',
  },
  '.rbc-agenda-date-cell': {
    padding: '12px 16px',
    fontWeight: '600',
    backgroundColor: '#F9FAFB',
    borderBottom: '1px solid #E5E7EB',
  },
  '.rbc-agenda-time-cell': {
    padding: '12px 16px',
    color: '#6B7280',
    fontSize: '0.875rem',
  },
  '.rbc-agenda-event-cell': {
    padding: '12px 16px',
  },
};

/**
 * Toolbar styles
 */
export const toolbarStyles = {
  container:
    'flex flex-col sm:flex-row items-center justify-between gap-4 mb-6 p-4 bg-white rounded-lg shadow',
  navigationGroup: 'flex items-center gap-2',
  navButton:
    'px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-md hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 transition-colors',
  todayButton:
    'px-6 py-2 text-sm font-medium text-white bg-blue-600 border border-transparent rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2 transition-colors',
  label: 'text-lg font-semibold text-gray-900 min-w-[200px] text-center',
  viewGroup: 'flex items-center gap-1 bg-gray-100 rounded-lg p-1',
  viewButton:
    'px-3 py-1.5 text-sm font-medium text-gray-700 rounded-md hover:bg-white hover:shadow-sm transition-all',
  viewButtonActive:
    'px-3 py-1.5 text-sm font-medium text-blue-700 bg-white rounded-md shadow-sm',
};

/**
 * Event component styles
 */
export const eventStyles = {
  container: 'w-full h-full flex flex-col justify-center overflow-hidden',
  title: 'font-semibold truncate text-xs sm:text-sm',
  jobNumber: 'text-xs opacity-90 truncate',
  location: 'text-xs opacity-75 truncate mt-0.5',
  time: 'text-xs opacity-75',
  priorityBadge: 'inline-block px-1.5 py-0.5 text-xs font-medium rounded',
};

/**
 * Loading skeleton styles
 */
export const skeletonStyles = {
  container: 'animate-pulse space-y-4 p-4',
  header: 'h-12 bg-gray-200 rounded',
  row: 'h-24 bg-gray-100 rounded',
  grid: 'grid grid-cols-7 gap-2',
  cell: 'h-32 bg-gray-50 rounded',
};

/**
 * Error boundary styles
 */
export const errorBoundaryStyles = {
  container:
    'flex flex-col items-center justify-center h-full p-8 bg-red-50 rounded-lg',
  icon: 'w-16 h-16 text-red-500 mb-4',
  title: 'text-xl font-semibold text-red-900 mb-2',
  message: 'text-red-700 text-center mb-6',
  button:
    'px-6 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2 transition-colors',
};

/**
 * Responsive breakpoints for calendar
 */
export const responsiveStyles = {
  mobile: {
    eventFontSize: '0.75rem',
    eventPadding: '2px 4px',
    toolbarFlexDirection: 'column',
  },
  tablet: {
    eventFontSize: '0.875rem',
    eventPadding: '4px 8px',
    toolbarFlexDirection: 'row',
  },
  desktop: {
    eventFontSize: '0.875rem',
    eventPadding: '4px 8px',
    toolbarFlexDirection: 'row',
  },
};

/**
 * Animation classes
 */
export const animationStyles = {
  fadeIn: 'animate-fadeIn',
  slideIn: 'animate-slideIn',
  pulse: 'animate-pulse',
  spin: 'animate-spin',
};

/**
 * Utility function to combine class names
 */
export const cn = (...classes: (string | boolean | undefined)[]): string => {
  return classes.filter(Boolean).join(' ');
};

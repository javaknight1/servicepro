import React, {
  useState,
  useCallback,
  useMemo,
  Component,
  ErrorInfo,
} from 'react';
import {
  Calendar as BigCalendar,
  dateFnsLocalizer,
  Views,
  SlotInfo,
  View,
  EventProps,
} from 'react-big-calendar';
import { format, parse, startOfWeek, getDay } from 'date-fns';
import { enUS } from 'date-fns/locale';
import withDragAndDrop from 'react-big-calendar/lib/addons/dragAndDrop';
import 'react-big-calendar/lib/css/react-big-calendar.css';
import 'react-big-calendar/lib/addons/dragAndDrop/styles.css';

import {
  CalendarProps,
  CalendarView,
  JobEvent,
  DragDropEvent,
  ResizeEvent,
  CalendarErrorBoundaryState,
} from './types';
import { CalendarEvent } from './CalendarEvent';
import { CalendarToolbar } from './CalendarToolbar';
import { calendarStyles, getEventStyles, cn } from './styles';
import { AlertTriangle, RefreshCw } from 'lucide-react';

// Setup the localizer for react-big-calendar
const locales = {
  'en-US': enUS,
};

const localizer = dateFnsLocalizer({
  format,
  parse,
  startOfWeek,
  getDay,
  locales,
});

// Create drag-and-drop enabled calendar with proper typing
const DnDCalendar = withDragAndDrop<JobEvent>(
  BigCalendar as React.ComponentType<any>
);

/**
 * Error Boundary for Calendar Component
 */
class CalendarErrorBoundary extends Component<
  { children: React.ReactNode; onReset?: () => void },
  CalendarErrorBoundaryState
> {
  constructor(props: { children: React.ReactNode; onReset?: () => void }) {
    super(props);
    this.state = { hasError: false, error: null };
  }

  static getDerivedStateFromError(error: Error): CalendarErrorBoundaryState {
    return { hasError: true, error };
  }

  componentDidCatch(error: Error, errorInfo: ErrorInfo) {
    console.error('Calendar Error:', error, errorInfo);
  }

  handleReset = () => {
    this.setState({ hasError: false, error: null });
    this.props.onReset?.();
  };

  render() {
    if (this.state.hasError) {
      return (
        <div className="flex flex-col items-center justify-center h-full p-8 bg-red-50 rounded-lg">
          <AlertTriangle className="w-16 h-16 text-red-500 mb-4" />
          <h2 className="text-xl font-semibold text-red-900 mb-2">
            Calendar Error
          </h2>
          <p className="text-red-700 text-center mb-6">
            {this.state.error?.message ||
              'Something went wrong with the calendar.'}
          </p>
          <button
            onClick={this.handleReset}
            className="px-6 py-2 bg-red-600 text-white rounded-md hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500 focus:ring-offset-2 transition-colors"
          >
            <RefreshCw className="w-4 h-4 inline-block mr-2" />
            Reload Calendar
          </button>
        </div>
      );
    }

    return this.props.children;
  }
}

/**
 * Loading Skeleton Component
 */
const CalendarSkeleton: React.FC = () => (
  <div className="animate-pulse space-y-4 p-4">
    <div className="h-12 bg-gray-200 rounded" />
    <div className="grid grid-cols-7 gap-2">
      {Array.from({ length: 35 }).map((_, i) => (
        <div key={i} className="h-32 bg-gray-50 rounded" />
      ))}
    </div>
  </div>
);

/**
 * Main Calendar Component
 */
export const Calendar: React.FC<CalendarProps> = ({
  events,
  onEventClick,
  onEventDrop,
  onEventResize,
  onSelectSlot,
  onNavigate,
  onViewChange,
  defaultView = 'month',
  defaultDate = new Date(),
  isLoading = false,
  className,
}) => {
  const [currentDate, setCurrentDate] = useState<Date>(defaultDate);
  const [currentView, setCurrentView] = useState<CalendarView>(defaultView);

  // Event style getter
  const eventStyleGetter = useCallback((event: JobEvent) => {
    return {
      className: getEventStyles(event.status),
    };
  }, []);

  // Handle event drop (drag and drop)
  const handleEventDrop = useCallback(
    async ({ event, start, end, isAllDay }: any) => {
      if (!onEventDrop) return;

      const dragData: DragDropEvent = {
        event: event as JobEvent,
        start: new Date(start),
        end: new Date(end),
        isAllDay: isAllDay || false,
      };

      try {
        await onEventDrop(dragData);
      } catch (error) {
        console.error('Failed to update event:', error);
        // Optionally show a toast notification here
      }
    },
    [onEventDrop]
  );

  // Handle event resize
  const handleEventResize = useCallback(
    async ({ event, start, end }: any) => {
      if (!onEventResize) return;

      const resizeData: ResizeEvent = {
        event: event as JobEvent,
        start: new Date(start),
        end: new Date(end),
      };

      try {
        await onEventResize(resizeData);
      } catch (error) {
        console.error('Failed to resize event:', error);
        // Optionally show a toast notification here
      }
    },
    [onEventResize]
  );

  // Handle slot selection
  const handleSelectSlot = useCallback(
    (slotInfo: SlotInfo) => {
      if (!onSelectSlot) return;

      onSelectSlot({
        start: slotInfo.start,
        end: slotInfo.end,
        slots: slotInfo.slots,
      });
    },
    [onSelectSlot]
  );

  // Handle event click
  const handleSelectEvent = useCallback(
    (event: JobEvent) => {
      if (onEventClick) {
        onEventClick(event);
      }
    },
    [onEventClick]
  );

  // Handle navigation
  const handleNavigate = useCallback(
    (newDate: Date) => {
      setCurrentDate(newDate);
      if (onNavigate) {
        onNavigate(newDate);
      }
    },
    [onNavigate]
  );

  // Handle view change
  const handleViewChange = useCallback(
    (view: View) => {
      const calView = view as CalendarView;
      setCurrentView(calView);
      if (onViewChange) {
        onViewChange(calView);
      }
    },
    [onViewChange]
  );

  // Custom event component
  const EventComponent = useCallback(
    (props: EventProps<JobEvent>) => (
      <CalendarEvent
        event={props.event}
        title={String(props.title || props.event.title)}
      />
    ),
    []
  );

  // Custom toolbar component
  const ToolbarComponent = useCallback(
    (toolbarProps: any) => (
      <CalendarToolbar
        date={toolbarProps.date}
        view={currentView}
        onNavigate={toolbarProps.onNavigate}
        onViewChange={handleViewChange}
        label={toolbarProps.label}
      />
    ),
    [currentView, handleViewChange]
  );

  // Memoize calendar components
  const components = useMemo(
    () => ({
      event: EventComponent,
      toolbar: ToolbarComponent,
    }),
    [EventComponent, ToolbarComponent]
  );

  // Calendar formats
  const formats = useMemo(
    () => ({
      timeGutterFormat: 'h:mm A',
      eventTimeRangeFormat: ({ start, end }: { start: Date; end: Date }) =>
        `${format(start, 'h:mm a')} - ${format(end, 'h:mm a')}`,
      agendaTimeRangeFormat: ({ start, end }: { start: Date; end: Date }) =>
        `${format(start, 'h:mm a')} - ${format(end, 'h:mm a')}`,
      dayHeaderFormat: 'EEEE, MMM dd',
      monthHeaderFormat: 'MMMM yyyy',
      agendaHeaderFormat: ({ start, end }: { start: Date; end: Date }) =>
        `${format(start, 'MMM dd')} - ${format(end, 'MMM dd, yyyy')}`,
    }),
    []
  );

  return (
    <CalendarErrorBoundary>
      <div className={cn('relative', calendarStyles.container, className)}>
        {/* Loading Overlay */}
        {isLoading && (
          <div className={calendarStyles.loadingOverlay}>
            <div className={calendarStyles.loadingSpinner} />
          </div>
        )}

        {/* Calendar */}
        {!isLoading ? (
          <DnDCalendar
            localizer={localizer}
            events={events}
            startAccessor={(event: JobEvent) => event.start}
            endAccessor={(event: JobEvent) => event.end}
            date={currentDate}
            onNavigate={handleNavigate}
            view={currentView as View}
            onView={handleViewChange}
            views={['month', 'week', 'day', 'agenda'] as View[]}
            onSelectEvent={(event: JobEvent) => handleSelectEvent(event)}
            onSelectSlot={handleSelectSlot}
            onEventDrop={handleEventDrop}
            onEventResize={handleEventResize}
            selectable
            resizable
            popup
            showMultiDayTimes
            step={30}
            timeslots={2}
            defaultDate={defaultDate}
            defaultView={defaultView as View}
            components={components}
            eventPropGetter={(event: JobEvent) => eventStyleGetter(event)}
            formats={formats}
            style={{ height: '100%', minHeight: '600px' }}
            className="custom-calendar"
          />
        ) : (
          <CalendarSkeleton />
        )}
      </div>
    </CalendarErrorBoundary>
  );
};

/**
 * Memoized version of Calendar for performance
 */
export default React.memo(Calendar);

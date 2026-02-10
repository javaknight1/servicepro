import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  startOfMonth,
  endOfMonth,
  startOfWeek,
  endOfWeek,
  addMonths,
  subMonths,
} from 'date-fns';
import { DashboardLayout } from '@components/layout';
import { Calendar } from '@components/calendar';
import { CalendarEventPopover } from '@components/calendar/CalendarEventPopover';
import {
  JobEvent,
  CalendarView,
  EventClickData,
  mapToCalendarStatus,
} from '@components/calendar/types';
import { jobService, Job } from '@services/jobService';
import { epochToDate } from '@/utils/epoch';
import { CalendarDays, List, AlertCircle } from 'lucide-react';
import { Button } from '@components/shared';

/**
 * Converts a Job from the API to a JobEvent for the calendar
 */
function jobToCalendarEvent(job: Job): JobEvent | null {
  // Skip jobs without scheduled times
  if (!job.scheduled_start_at) {
    return null;
  }

  const start = epochToDate(job.scheduled_start_at);
  // Default to 1 hour if no end time
  const end = job.scheduled_end_at
    ? epochToDate(job.scheduled_end_at)
    : new Date(start.getTime() + 60 * 60 * 1000);

  return {
    id: job.id,
    title: job.title,
    start,
    end,
    jobNumber: job.job_number,
    status: mapToCalendarStatus(job.status),
    priority: job.priority,
    description: job.description,
    customerId: job.customer_id,
    customerName: job.customer
      ? `${job.customer.first_name} ${job.customer.last_name}`
      : undefined,
    location: job.service_address
      ? `${job.service_address.street}, ${job.service_address.city}`
      : undefined,
    assignedTo: job.assignments?.map((a) => a.user_id) || [],
  };
}

/**
 * Calculate the date range to fetch based on current date
 * Adds buffer to ensure we have events for navigation
 */
function getDateRange(date: Date): { start: Date; end: Date } {
  // Always fetch a bit more than needed to handle view transitions
  const monthStart = startOfMonth(date);
  const monthEnd = endOfMonth(date);

  // Add buffer for week view overflow
  const start = startOfWeek(subMonths(monthStart, 1));
  const end = endOfWeek(addMonths(monthEnd, 1));

  return { start, end };
}

export function JobCalendarPage() {
  const navigate = useNavigate();
  const [events, setEvents] = useState<JobEvent[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [currentDate, setCurrentDate] = useState(new Date());
  const [currentView, setCurrentView] = useState<CalendarView>('month');
  const [selectedEvent, setSelectedEvent] = useState<{
    event: JobEvent;
    position: { x: number; y: number };
  } | null>(null);

  const loadJobs = useCallback(async (date: Date) => {
    setIsLoading(true);
    setError(null);

    try {
      const { start, end } = getDateRange(date);
      const jobs = await jobService.getScheduledJobs(start, end);

      const calendarEvents = jobs
        .map(jobToCalendarEvent)
        .filter((event): event is JobEvent => event !== null);

      setEvents(calendarEvents);
    } catch (err) {
      console.error('Failed to load scheduled jobs:', err);
      setError('Failed to load scheduled jobs. Please try again.');
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    loadJobs(currentDate);
  }, [currentDate, loadJobs]);

  const handleEventClick = useCallback((data: EventClickData) => {
    setSelectedEvent({
      event: data.event,
      position: data.position,
    });
  }, []);

  const handleClosePopover = useCallback(() => {
    setSelectedEvent(null);
  }, []);

  const handleEditJob = useCallback(
    (event: JobEvent) => {
      navigate(`/jobs/${event.id}`);
    },
    [navigate]
  );

  const handleNavigate = useCallback((date: Date) => {
    setCurrentDate(date);
  }, []);

  const handleViewChange = useCallback((view: CalendarView) => {
    setCurrentView(view);
  }, []);

  return (
    <DashboardLayout>
      {/* Event Popover */}
      {selectedEvent && (
        <CalendarEventPopover
          event={selectedEvent.event}
          position={selectedEvent.position}
          onClose={handleClosePopover}
          onEdit={handleEditJob}
        />
      )}

      <div className="p-6 lg:p-8 space-y-6">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold text-neutral-900 flex items-center gap-2">
              <CalendarDays className="h-7 w-7 text-primary-600" />
              Job Calendar
            </h1>
            <p className="mt-1 text-sm text-neutral-500">
              View scheduled jobs on the calendar
            </p>
          </div>

          <div className="flex gap-2">
            <Button
              variant="outline"
              onClick={() => navigate('/jobs')}
              className="flex items-center gap-2"
            >
              <List className="h-4 w-4" />
              List View
            </Button>
          </div>
        </div>

        {/* Error message */}
        {error && (
          <div className="bg-red-50 border border-red-200 rounded-lg p-4 flex items-center gap-3">
            <AlertCircle className="h-5 w-5 text-red-500 flex-shrink-0" />
            <p className="text-sm text-red-700">{error}</p>
            <button
              onClick={() => loadJobs(currentDate)}
              className="ml-auto text-sm text-red-600 hover:text-red-800 font-medium"
            >
              Retry
            </button>
          </div>
        )}

        {/* Calendar */}
        <div className="bg-white rounded-lg shadow-sm border border-neutral-200 p-4 overflow-hidden">
          <Calendar
            events={events}
            onEventClick={handleEventClick}
            onNavigate={handleNavigate}
            onViewChange={handleViewChange}
            defaultView={currentView}
            defaultDate={currentDate}
            isLoading={isLoading}
            className="h-[650px]"
          />
        </div>

        {/* Legend */}
        <div className="bg-white rounded-lg shadow-sm border border-neutral-200 p-4">
          <h3 className="text-sm font-medium text-neutral-700 mb-3">
            Status Legend
          </h3>
          <div className="flex flex-wrap gap-4">
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-blue-500" />
              <span className="text-sm text-neutral-600">Scheduled / New</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-yellow-500" />
              <span className="text-sm text-neutral-600">
                In Progress / En Route
              </span>
            </div>
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-orange-500" />
              <span className="text-sm text-neutral-600">On Hold</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-green-500" />
              <span className="text-sm text-neutral-600">
                Completed / Invoiced / Paid
              </span>
            </div>
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-gray-500" />
              <span className="text-sm text-neutral-600">Cancelled</span>
            </div>
          </div>
        </div>
      </div>
    </DashboardLayout>
  );
}

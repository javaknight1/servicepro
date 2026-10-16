import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Calendar } from '../../../src/components/calendar/Calendar';
import {
  JobEvent,
  JobStatus,
  JobPriority,
} from '../../../src/components/calendar/types';

/**
 * Integration tests for Calendar component
 * These tests verify the interaction between calendar components and user workflows
 */
// Fixtures live in January 2024; render the calendar on that month.
const DEFAULT_DATE = new Date(2024, 0, 15);

describe('Calendar Integration Tests', () => {
  const mockEvents: JobEvent[] = [
    {
      id: '1',
      title: 'Install HVAC System',
      start: new Date(2024, 0, 15, 9, 0),
      end: new Date(2024, 0, 15, 12, 0),
      jobNumber: 'JOB-001',
      status: JobStatus.SCHEDULED,
      priority: JobPriority.HIGH,
      location: '123 Main St',
      customerName: 'John Doe',
      assignedTo: ['tech1'],
    },
    {
      id: '2',
      title: 'Fix Plumbing Issue',
      start: new Date(2024, 0, 16, 14, 0),
      end: new Date(2024, 0, 16, 16, 0),
      jobNumber: 'JOB-002',
      status: JobStatus.IN_PROGRESS,
      priority: JobPriority.URGENT,
      location: '456 Oak Ave',
      customerName: 'Jane Smith',
      assignedTo: ['tech2'],
    },
    {
      id: '3',
      title: 'Electrical Inspection',
      start: new Date(2024, 0, 17, 10, 0),
      end: new Date(2024, 0, 17, 11, 0),
      jobNumber: 'JOB-003',
      status: JobStatus.COMPLETED,
      location: '789 Pine Rd',
      customerName: 'Bob Johnson',
      assignedTo: ['tech3'],
    },
  ];

  const mockHandlers = {
    onEventClick: vi.fn(),
    onEventDrop: vi.fn().mockResolvedValue(undefined),
    onEventResize: vi.fn().mockResolvedValue(undefined),
    onSelectSlot: vi.fn(),
    onNavigate: vi.fn(),
    onViewChange: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Complete User Workflows', () => {
    it('should handle complete view switching workflow', async () => {
      const user = userEvent.setup();
      render(
        <Calendar
          defaultDate={DEFAULT_DATE}
          events={mockEvents}
          onViewChange={mockHandlers.onViewChange}
        />
      );

      // Start in month view
      expect(screen.getByText('Month')).toHaveAttribute('aria-pressed', 'true');

      // Switch to week view
      await user.click(screen.getByText('Week'));
      expect(mockHandlers.onViewChange).toHaveBeenCalledWith('week');

      // Switch to day view
      await user.click(screen.getByText('Day'));
      expect(mockHandlers.onViewChange).toHaveBeenCalledWith('day');

      // Switch to agenda view
      await user.click(screen.getByText('Agenda'));
      expect(mockHandlers.onViewChange).toHaveBeenCalledWith('agenda');

      // Switch back to month view
      await user.click(screen.getByText('Month'));
      expect(mockHandlers.onViewChange).toHaveBeenCalledWith('month');
    });

    it('should handle complete navigation workflow', async () => {
      const user = userEvent.setup();
      render(
        <Calendar
          defaultDate={DEFAULT_DATE}
          events={mockEvents}
          onNavigate={mockHandlers.onNavigate}
        />
      );

      // Navigate forward
      const nextButton = screen.getByLabelText('Next');
      await user.click(nextButton);
      expect(mockHandlers.onNavigate).toHaveBeenCalledWith(expect.any(Date));

      // Navigate backward
      const prevButton = screen.getByLabelText('Previous');
      await user.click(prevButton);
      expect(mockHandlers.onNavigate).toHaveBeenCalledWith(expect.any(Date));

      // Navigate to today
      const todayButton = screen.getByLabelText('Today');
      await user.click(todayButton);
      expect(mockHandlers.onNavigate).toHaveBeenCalledWith(expect.any(Date));
    });

    it('should handle event click and view details workflow', async () => {
      const user = userEvent.setup();
      render(
        <Calendar
          defaultDate={DEFAULT_DATE}
          events={mockEvents}
          onEventClick={mockHandlers.onEventClick}
        />
      );

      // Click on first event
      const event = screen.getByText(mockEvents[0].title);
      await user.click(event);

      expect(mockHandlers.onEventClick).toHaveBeenCalledWith(
        expect.objectContaining({
          event: expect.objectContaining({
            id: mockEvents[0].id,
            title: mockEvents[0].title,
            jobNumber: mockEvents[0].jobNumber,
          }),
          position: expect.any(Object),
        })
      );
    });
  });

  // Drag-and-drop rescheduling, resizing and slot selection are disabled in
  // the Calendar until conflict detection lands (TODO T027), so there are no
  // DnD/resize/slot-selection workflows to exercise here.

  describe('Combined Workflows', () => {
    it('should handle view change and navigation together', async () => {
      const user = userEvent.setup();
      render(
        <Calendar
          defaultDate={DEFAULT_DATE}
          events={mockEvents}
          onViewChange={mockHandlers.onViewChange}
          onNavigate={mockHandlers.onNavigate}
        />
      );

      // Change to week view
      await user.click(screen.getByText('Week'));
      expect(mockHandlers.onViewChange).toHaveBeenCalledWith('week');

      // Navigate forward
      await user.click(screen.getByLabelText('Next'));
      expect(mockHandlers.onNavigate).toHaveBeenCalled();

      // Change to day view
      await user.click(screen.getByText('Day'));
      expect(mockHandlers.onViewChange).toHaveBeenCalledWith('day');
    });

    it('should handle loading state during interactions', async () => {
      const { rerender } = render(
        <Calendar
          defaultDate={DEFAULT_DATE}
          events={mockEvents}
          isLoading={false}
        />
      );

      // Verify calendar is visible
      expect(document.querySelector('.rbc-calendar')).toBeInTheDocument();

      // Set loading state
      rerender(
        <Calendar
          defaultDate={DEFAULT_DATE}
          events={mockEvents}
          isLoading={true}
        />
      );

      // Verify loading skeleton is shown
      expect(screen.getByTestId('calendar-skeleton')).toBeInTheDocument();

      // Remove loading state
      rerender(
        <Calendar
          defaultDate={DEFAULT_DATE}
          events={mockEvents}
          isLoading={false}
        />
      );

      // Verify calendar is visible again
      expect(document.querySelector('.rbc-calendar')).toBeInTheDocument();
    });
  });

  describe('Real-world Scenarios', () => {
    it('should let a technician open a job from the calendar', async () => {
      const user = userEvent.setup();
      render(
        <Calendar
          defaultDate={DEFAULT_DATE}
          events={mockEvents}
          onEventClick={mockHandlers.onEventClick}
        />
      );

      await user.click(screen.getByText(mockEvents[0].title));
      expect(mockHandlers.onEventClick).toHaveBeenCalledWith(
        expect.objectContaining({
          event: expect.objectContaining({ id: mockEvents[0].id }),
        })
      );
    });

    it('should handle manager viewing weekly schedule', async () => {
      const user = userEvent.setup();
      render(
        <Calendar
          defaultDate={DEFAULT_DATE}
          events={mockEvents}
          defaultView="week"
          onViewChange={mockHandlers.onViewChange}
          onNavigate={mockHandlers.onNavigate}
        />
      );

      // Navigate through weeks
      await user.click(screen.getByLabelText('Next'));
      expect(mockHandlers.onNavigate).toHaveBeenCalled();

      await user.click(screen.getByLabelText('Previous'));
      expect(mockHandlers.onNavigate).toHaveBeenCalled();

      // Go back to today
      await user.click(screen.getByLabelText('Today'));
      expect(mockHandlers.onNavigate).toHaveBeenCalled();
    });

    it('should handle viewing different job statuses', async () => {
      render(<Calendar defaultDate={DEFAULT_DATE} events={mockEvents} />);

      // Verify all job statuses are visible
      expect(screen.getByText('Install HVAC System')).toBeInTheDocument(); // Scheduled
      expect(screen.getByText('Fix Plumbing Issue')).toBeInTheDocument(); // In Progress
      expect(screen.getByText('Electrical Inspection')).toBeInTheDocument(); // Completed
    });
  });

  describe('Performance and Responsiveness', () => {
    it('should handle rapid view changes', async () => {
      const user = userEvent.setup();
      render(
        <Calendar
          defaultDate={DEFAULT_DATE}
          events={mockEvents}
          onViewChange={mockHandlers.onViewChange}
        />
      );

      // Rapidly change views
      await user.click(screen.getByText('Week'));
      await user.click(screen.getByText('Day'));
      await user.click(screen.getByText('Agenda'));
      await user.click(screen.getByText('Month'));

      expect(mockHandlers.onViewChange).toHaveBeenCalledTimes(4);
    });

    it('should handle many events efficiently', async () => {
      const manyEvents: JobEvent[] = Array.from({ length: 100 }, (_, i) => ({
        id: `event-${i}`,
        title: `Job ${i}`,
        start: new Date(2024, 0, 1 + (i % 28), 9 + (i % 8), 0),
        end: new Date(2024, 0, 1 + (i % 28), 10 + (i % 8), 0),
        jobNumber: `JOB-${String(i).padStart(3, '0')}`,
        status: JobStatus.SCHEDULED,
      }));

      const { container } = render(
        <Calendar defaultDate={DEFAULT_DATE} events={manyEvents} />
      );
      expect(container).toBeInTheDocument();

      // Calendar should render without performance issues
      expect(document.querySelector('.rbc-calendar')).toBeInTheDocument();
    });
  });

  describe('Edge Cases', () => {
    it('should handle empty events array', () => {
      render(<Calendar defaultDate={DEFAULT_DATE} events={[]} />);
      expect(document.querySelector('.rbc-calendar')).toBeInTheDocument();
    });

    it('should handle events with missing optional fields', () => {
      const minimalEvent: JobEvent = {
        id: '1',
        title: 'Minimal Job',
        start: new Date(2024, 0, 15, 9, 0),
        end: new Date(2024, 0, 15, 12, 0),
        jobNumber: 'JOB-MIN',
        status: JobStatus.SCHEDULED,
      };

      render(<Calendar defaultDate={DEFAULT_DATE} events={[minimalEvent]} />);
      expect(screen.getByText('Minimal Job')).toBeInTheDocument();
    });

    it('should handle overlapping events', () => {
      const overlappingEvents: JobEvent[] = [
        {
          id: '1',
          title: 'Event 1',
          start: new Date(2024, 0, 15, 9, 0),
          end: new Date(2024, 0, 15, 12, 0),
          jobNumber: 'JOB-001',
          status: JobStatus.SCHEDULED,
        },
        {
          id: '2',
          title: 'Event 2',
          start: new Date(2024, 0, 15, 10, 0),
          end: new Date(2024, 0, 15, 13, 0),
          jobNumber: 'JOB-002',
          status: JobStatus.IN_PROGRESS,
        },
      ];

      render(
        <Calendar defaultDate={DEFAULT_DATE} events={overlappingEvents} />
      );
      expect(screen.getByText('Event 1')).toBeInTheDocument();
      expect(screen.getByText('Event 2')).toBeInTheDocument();
    });
  });
});

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Calendar } from '../../../src/components/calendar/Calendar';
import {
  JobEvent,
  JobStatus,
  JobPriority,
} from '../../../src/components/calendar/types';

// All fixtures live in January 2024; render the calendar on that month.
const DEFAULT_DATE = new Date(2024, 0, 15);

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

const getCalendarRoot = (container: HTMLElement) =>
  container.querySelector('.rbc-calendar');

describe('Calendar Component', () => {
  const mockOnEventClick = vi.fn();
  const mockOnNavigate = vi.fn();
  const mockOnViewChange = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Rendering', () => {
    it('should render calendar without errors', () => {
      const { container } = render(
        <Calendar events={mockEvents} defaultDate={DEFAULT_DATE} />
      );
      expect(getCalendarRoot(container)).toBeInTheDocument();
    });

    it('should render all events in the visible month', () => {
      render(<Calendar events={mockEvents} defaultDate={DEFAULT_DATE} />);
      mockEvents.forEach((event) => {
        expect(screen.getByText(event.title)).toBeInTheDocument();
      });
    });

    it('should render loading skeleton when isLoading is true', () => {
      const { container } = render(
        <Calendar events={mockEvents} isLoading={true} />
      );
      expect(screen.getByTestId('calendar-skeleton')).toBeInTheDocument();
      expect(getCalendarRoot(container)).not.toBeInTheDocument();
    });

    it('should apply custom className', () => {
      const { container } = render(
        <Calendar events={mockEvents} className="custom-class" />
      );
      expect(container.firstChild).toHaveClass('custom-class');
    });
  });

  describe('Event Interactions', () => {
    it('should call onEventClick with the event and click position', async () => {
      const user = userEvent.setup();
      render(
        <Calendar
          events={mockEvents}
          defaultDate={DEFAULT_DATE}
          onEventClick={mockOnEventClick}
        />
      );

      await user.click(screen.getByText(mockEvents[0].title));

      expect(mockOnEventClick).toHaveBeenCalledWith(
        expect.objectContaining({
          event: expect.objectContaining({ id: mockEvents[0].id }),
          position: expect.objectContaining({
            x: expect.any(Number),
            y: expect.any(Number),
          }),
        })
      );
    });

    // Drag-and-drop, resize and slot selection are intentionally disabled
    // until conflict detection lands (TODO T027).
    it('should not make slots selectable while drag-and-drop is disabled', async () => {
      const user = userEvent.setup();
      const { container } = render(
        <Calendar events={mockEvents} defaultDate={DEFAULT_DATE} />
      );

      const dayCell = container.querySelector('.rbc-day-bg');
      expect(dayCell).not.toBeNull();
      await user.click(dayCell as Element);

      expect(container.querySelector('.rbc-slot-selection')).toBeNull();
    });
  });

  describe('Navigation', () => {
    it('should call onNavigate when navigating to different date', async () => {
      const user = userEvent.setup();
      render(<Calendar events={mockEvents} onNavigate={mockOnNavigate} />);

      await user.click(screen.getByLabelText('Next'));

      expect(mockOnNavigate).toHaveBeenCalledWith(expect.any(Date));
    });

    it('should navigate to today when Today button is clicked', async () => {
      const user = userEvent.setup();
      render(<Calendar events={mockEvents} onNavigate={mockOnNavigate} />);

      await user.click(screen.getByText('Today'));

      expect(mockOnNavigate).toHaveBeenCalled();
    });
  });

  describe('View Changes', () => {
    it('should call onViewChange when changing view', async () => {
      const user = userEvent.setup();
      render(<Calendar events={mockEvents} onViewChange={mockOnViewChange} />);

      await user.click(screen.getByText('Week'));

      expect(mockOnViewChange).toHaveBeenCalledWith('week');
    });

    it.each(['month', 'week', 'day', 'agenda'] as const)(
      'should render the %s view',
      (view) => {
        const { container } = render(
          <Calendar
            events={mockEvents}
            defaultView={view}
            defaultDate={DEFAULT_DATE}
          />
        );
        expect(getCalendarRoot(container)).toBeInTheDocument();
        expect(
          screen.getByText(view.charAt(0).toUpperCase() + view.slice(1))
        ).toHaveAttribute('aria-pressed', 'true');
      }
    );
  });

  describe('Error Handling', () => {
    const brokenEvents = (): JobEvent[] => [
      {
        ...mockEvents[0],
        get start(): Date {
          throw new Error('Test error');
        },
      },
    ];

    it('should render error boundary when rendering fails', () => {
      const consoleError = vi
        .spyOn(console, 'error')
        .mockImplementation(() => undefined);

      render(<Calendar events={brokenEvents()} defaultDate={DEFAULT_DATE} />);

      expect(screen.getByText('Calendar Error')).toBeInTheDocument();
      expect(screen.getByText(/Test error/i)).toBeInTheDocument();
      expect(screen.getByText('Reload Calendar')).toBeInTheDocument();

      consoleError.mockRestore();
    });
  });

  describe('Default Props', () => {
    it('should use default view when not specified', () => {
      render(<Calendar events={mockEvents} />);
      expect(screen.getByText('Month')).toHaveAttribute('aria-pressed', 'true');
    });

    it('should use default date when not specified', () => {
      const { container } = render(<Calendar events={mockEvents} />);
      const monthYear = new Date().toLocaleDateString('en-US', {
        month: 'long',
        year: 'numeric',
      });
      expect(container.textContent).toContain(monthYear);
    });
  });

  describe('Accessibility', () => {
    it('should have proper ARIA labels', () => {
      render(<Calendar events={mockEvents} />);

      expect(screen.getByLabelText('Previous')).toBeInTheDocument();
      expect(screen.getByLabelText('Next')).toBeInTheDocument();
      expect(screen.getByLabelText('Today')).toBeInTheDocument();
    });
  });

  describe('Performance', () => {
    it('should handle large number of events', () => {
      const largeEventSet: JobEvent[] = Array.from(
        { length: 1000 },
        (_, i) => ({
          id: `event-${i}`,
          title: `Event ${i}`,
          start: new Date(2024, 0, 1 + (i % 28), 9, 0),
          end: new Date(2024, 0, 1 + (i % 28), 10, 0),
          jobNumber: `JOB-${i}`,
          status: JobStatus.SCHEDULED,
        })
      );

      const { container } = render(
        <Calendar events={largeEventSet} defaultDate={DEFAULT_DATE} />
      );
      expect(getCalendarRoot(container)).toBeInTheDocument();
    });
  });
});

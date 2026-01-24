import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Calendar } from '../../../src/components/calendar/Calendar';
import {
  JobEvent,
  JobStatus,
  JobPriority,
} from '../../../src/components/calendar/types';

// Mock events for testing
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

describe('Calendar Component', () => {
  const mockOnEventClick = vi.fn();
  const mockOnEventDrop = vi.fn().mockResolvedValue(undefined);
  const mockOnEventResize = vi.fn().mockResolvedValue(undefined);
  const mockOnSelectSlot = vi.fn();
  const mockOnNavigate = vi.fn();
  const mockOnViewChange = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Rendering', () => {
    it('should render calendar without errors', () => {
      render(<Calendar events={mockEvents} />);
      expect(screen.getByRole('application')).toBeInTheDocument();
    });

    it('should render all events', () => {
      render(<Calendar events={mockEvents} />);
      mockEvents.forEach((event) => {
        expect(screen.getByText(event.title)).toBeInTheDocument();
      });
    });

    it('should render loading skeleton when isLoading is true', () => {
      render(<Calendar events={mockEvents} isLoading={true} />);
      expect(screen.getByTestId('calendar-skeleton')).toBeInTheDocument();
    });

    it('should apply custom className', () => {
      const { container } = render(
        <Calendar events={mockEvents} className="custom-class" />
      );
      expect(container.firstChild).toHaveClass('custom-class');
    });
  });

  describe('Event Interactions', () => {
    it('should call onEventClick when event is clicked', async () => {
      const user = userEvent.setup();
      render(<Calendar events={mockEvents} onEventClick={mockOnEventClick} />);

      const event = screen.getByText(mockEvents[0].title);
      await user.click(event);

      expect(mockOnEventClick).toHaveBeenCalledWith(
        expect.objectContaining({ id: mockEvents[0].id })
      );
    });

    it('should handle event drag and drop', async () => {
      render(<Calendar events={mockEvents} onEventDrop={mockOnEventDrop} />);

      // Simulate drag and drop
      const event = screen.getByText(mockEvents[0].title);
      fireEvent.dragStart(event);
      fireEvent.drop(event);

      await waitFor(() => {
        expect(mockOnEventDrop).toHaveBeenCalled();
      });
    });

    it('should handle event resize', async () => {
      render(
        <Calendar events={mockEvents} onEventResize={mockOnEventResize} />
      );

      // Simulate resize
      const event = screen.getByText(mockEvents[0].title);
      const resizeHandle = event.querySelector('.rbc-addons-dnd-resize-anchor');

      if (resizeHandle) {
        fireEvent.mouseDown(resizeHandle);
        fireEvent.mouseMove(resizeHandle, { clientY: 100 });
        fireEvent.mouseUp(resizeHandle);

        await waitFor(() => {
          expect(mockOnEventResize).toHaveBeenCalled();
        });
      }
    });
  });

  describe('Slot Selection', () => {
    it('should call onSelectSlot when empty slot is selected', async () => {
      const user = userEvent.setup();
      render(<Calendar events={mockEvents} onSelectSlot={mockOnSelectSlot} />);

      // Find and click an empty day cell
      const dayCell = screen.getAllByRole('cell')[10]; // Arbitrary empty cell
      await user.click(dayCell);

      expect(mockOnSelectSlot).toHaveBeenCalledWith(
        expect.objectContaining({
          start: expect.any(Date),
          end: expect.any(Date),
          slots: expect.any(Array),
        })
      );
    });
  });

  describe('Navigation', () => {
    it('should call onNavigate when navigating to different date', async () => {
      const user = userEvent.setup();
      render(<Calendar events={mockEvents} onNavigate={mockOnNavigate} />);

      const nextButton = screen.getByLabelText('Next');
      await user.click(nextButton);

      expect(mockOnNavigate).toHaveBeenCalledWith(expect.any(Date));
    });

    it('should navigate to today when Today button is clicked', async () => {
      const user = userEvent.setup();
      render(<Calendar events={mockEvents} onNavigate={mockOnNavigate} />);

      const todayButton = screen.getByText('Today');
      await user.click(todayButton);

      expect(mockOnNavigate).toHaveBeenCalled();
    });
  });

  describe('View Changes', () => {
    it('should call onViewChange when changing view', async () => {
      const user = userEvent.setup();
      render(<Calendar events={mockEvents} onViewChange={mockOnViewChange} />);

      const weekButton = screen.getByText('Week');
      await user.click(weekButton);

      expect(mockOnViewChange).toHaveBeenCalledWith('week');
    });

    it('should render different views correctly', async () => {
      const { rerender } = render(
        <Calendar events={mockEvents} defaultView="month" />
      );
      expect(screen.getByRole('application')).toBeInTheDocument();

      rerender(<Calendar events={mockEvents} defaultView="week" />);
      expect(screen.getByRole('application')).toBeInTheDocument();

      rerender(<Calendar events={mockEvents} defaultView="day" />);
      expect(screen.getByRole('application')).toBeInTheDocument();

      rerender(<Calendar events={mockEvents} defaultView="agenda" />);
      expect(screen.getByRole('application')).toBeInTheDocument();
    });
  });

  describe('Error Handling', () => {
    it('should render error boundary when error occurs', () => {
      const ThrowError = () => {
        throw new Error('Test error');
      };

      // Suppress console.error for this test
      const consoleError = vi
        .spyOn(console, 'error')
        .mockImplementation(() => {});

      render(
        <Calendar events={mockEvents}>
          <ThrowError />
        </Calendar>
      );

      expect(screen.getByText('Calendar Error')).toBeInTheDocument();
      expect(screen.getByText(/Test error/i)).toBeInTheDocument();

      consoleError.mockRestore();
    });

    it('should reset error boundary when reload button is clicked', async () => {
      const user = userEvent.setup();
      const ThrowError = () => {
        throw new Error('Test error');
      };

      const consoleError = vi
        .spyOn(console, 'error')
        .mockImplementation(() => {});

      render(
        <Calendar events={mockEvents}>
          <ThrowError />
        </Calendar>
      );

      const reloadButton = screen.getByText('Reload Calendar');
      await user.click(reloadButton);

      // Error should be cleared
      expect(screen.queryByText('Calendar Error')).not.toBeInTheDocument();

      consoleError.mockRestore();
    });
  });

  describe('Default Props', () => {
    it('should use default view when not specified', () => {
      render(<Calendar events={mockEvents} />);
      // Default view is 'month'
      expect(screen.getByText('Month')).toHaveAttribute('aria-pressed', 'true');
    });

    it('should use default date when not specified', () => {
      const { container } = render(<Calendar events={mockEvents} />);
      const today = new Date();
      const monthYear = today.toLocaleDateString('en-US', {
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

    it('should have keyboard navigation', async () => {
      const user = userEvent.setup();
      render(<Calendar events={mockEvents} onEventClick={mockOnEventClick} />);

      const event = screen.getByText(mockEvents[0].title);
      event.focus();
      await user.keyboard('{Enter}');

      expect(mockOnEventClick).toHaveBeenCalled();
    });
  });

  describe('Performance', () => {
    it('should handle large number of events', () => {
      const largeEventSet: JobEvent[] = Array.from(
        { length: 1000 },
        (_, i) => ({
          id: `event-${i}`,
          title: `Event ${i}`,
          start: new Date(2024, 0, 15 + (i % 30), 9, 0),
          end: new Date(2024, 0, 15 + (i % 30), 10, 0),
          jobNumber: `JOB-${i}`,
          status: JobStatus.SCHEDULED,
        })
      );

      const { container } = render(<Calendar events={largeEventSet} />);
      expect(container).toBeInTheDocument();
    });
  });
});

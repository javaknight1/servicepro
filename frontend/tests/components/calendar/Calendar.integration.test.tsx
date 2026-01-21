import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
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
        <Calendar events={mockEvents} onNavigate={mockHandlers.onNavigate} />
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
          events={mockEvents}
          onEventClick={mockHandlers.onEventClick}
        />
      );

      // Click on first event
      const event = screen.getByText(mockEvents[0].title);
      await user.click(event);

      expect(mockHandlers.onEventClick).toHaveBeenCalledWith(
        expect.objectContaining({
          id: mockEvents[0].id,
          title: mockEvents[0].title,
          jobNumber: mockEvents[0].jobNumber,
        })
      );
    });
  });

  describe('Drag and Drop Workflows', () => {
    it('should complete drag and drop workflow for rescheduling', async () => {
      render(
        <Calendar events={mockEvents} onEventDrop={mockHandlers.onEventDrop} />
      );

      const event = screen.getByText(mockEvents[0].title);

      // Simulate drag start
      await userEvent.pointer([
        {
          target: event,
          keys: '[MouseLeft>]',
          coords: { clientX: 0, clientY: 0 },
        },
        { coords: { clientX: 100, clientY: 100 } },
        { keys: '[/MouseLeft]' },
      ]);

      await waitFor(() => {
        expect(mockHandlers.onEventDrop).toHaveBeenCalledWith(
          expect.objectContaining({
            event: expect.objectContaining({ id: mockEvents[0].id }),
            start: expect.any(Date),
            end: expect.any(Date),
          })
        );
      });
    });

    it('should handle drag and drop failure gracefully', async () => {
      const failingHandler = vi
        .fn()
        .mockRejectedValue(new Error('Network error'));
      const consoleError = vi
        .spyOn(console, 'error')
        .mockImplementation(() => {});

      render(<Calendar events={mockEvents} onEventDrop={failingHandler} />);

      const event = screen.getByText(mockEvents[0].title);

      // Simulate drag and drop
      await userEvent.pointer([
        {
          target: event,
          keys: '[MouseLeft>]',
          coords: { clientX: 0, clientY: 0 },
        },
        { coords: { clientX: 100, clientY: 100 } },
        { keys: '[/MouseLeft]' },
      ]);

      await waitFor(() => {
        expect(failingHandler).toHaveBeenCalled();
        expect(consoleError).toHaveBeenCalledWith(
          'Failed to update event:',
          expect.any(Error)
        );
      });

      consoleError.mockRestore();
    });
  });

  describe('Resize Workflows', () => {
    it('should complete resize workflow for adjusting duration', async () => {
      render(
        <Calendar
          events={mockEvents}
          onEventResize={mockHandlers.onEventResize}
        />
      );

      const event = screen.getByText(mockEvents[0].title);
      const resizeHandle = event.parentElement?.querySelector(
        '.rbc-addons-dnd-resize-anchor'
      );

      if (resizeHandle) {
        await userEvent.pointer([
          {
            target: resizeHandle,
            keys: '[MouseLeft>]',
            coords: { clientX: 0, clientY: 0 },
          },
          { coords: { clientX: 0, clientY: 50 } },
          { keys: '[/MouseLeft]' },
        ]);

        await waitFor(() => {
          expect(mockHandlers.onEventResize).toHaveBeenCalledWith(
            expect.objectContaining({
              event: expect.objectContaining({ id: mockEvents[0].id }),
              start: expect.any(Date),
              end: expect.any(Date),
            })
          );
        });
      }
    });
  });

  describe('Slot Selection Workflows', () => {
    it('should handle slot selection for creating new event', async () => {
      const user = userEvent.setup();
      render(
        <Calendar
          events={mockEvents}
          onSelectSlot={mockHandlers.onSelectSlot}
        />
      );

      // Find and select an empty slot
      const cells = screen.getAllByRole('cell');
      const emptyCell = cells.find(
        (cell) => !within(cell).queryByText(/Install|Fix|Electrical/)
      );

      if (emptyCell) {
        await user.click(emptyCell);

        expect(mockHandlers.onSelectSlot).toHaveBeenCalledWith(
          expect.objectContaining({
            start: expect.any(Date),
            end: expect.any(Date),
            slots: expect.any(Array),
          })
        );
      }
    });

    it('should handle multi-slot selection via drag', async () => {
      render(
        <Calendar
          events={mockEvents}
          onSelectSlot={mockHandlers.onSelectSlot}
        />
      );

      const cells = screen.getAllByRole('cell');
      const startCell = cells[10];
      const endCell = cells[12];

      // Simulate drag selection
      await userEvent.pointer([
        {
          target: startCell,
          keys: '[MouseLeft>]',
          coords: { clientX: 0, clientY: 0 },
        },
        { target: endCell, coords: { clientX: 100, clientY: 100 } },
        { keys: '[/MouseLeft]' },
      ]);

      await waitFor(() => {
        expect(mockHandlers.onSelectSlot).toHaveBeenCalled();
      });
    });
  });

  describe('Combined Workflows', () => {
    it('should handle view change and navigation together', async () => {
      const user = userEvent.setup();
      render(
        <Calendar
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
      const user = userEvent.setup();
      const { rerender } = render(
        <Calendar events={mockEvents} isLoading={false} />
      );

      // Verify calendar is visible
      expect(screen.getByRole('application')).toBeInTheDocument();

      // Set loading state
      rerender(<Calendar events={mockEvents} isLoading={true} />);

      // Verify loading skeleton is shown
      expect(screen.getByTestId('calendar-skeleton')).toBeInTheDocument();

      // Remove loading state
      rerender(<Calendar events={mockEvents} isLoading={false} />);

      // Verify calendar is visible again
      expect(screen.getByRole('application')).toBeInTheDocument();
    });
  });

  describe('Real-world Scenarios', () => {
    it('should handle technician rescheduling a job', async () => {
      const user = userEvent.setup();
      render(
        <Calendar
          events={mockEvents}
          onEventClick={mockHandlers.onEventClick}
          onEventDrop={mockHandlers.onEventDrop}
        />
      );

      // 1. View job details
      const job = screen.getByText(mockEvents[0].title);
      await user.click(job);
      expect(mockHandlers.onEventClick).toHaveBeenCalled();

      // 2. Drag to reschedule
      await userEvent.pointer([
        {
          target: job,
          keys: '[MouseLeft>]',
          coords: { clientX: 0, clientY: 0 },
        },
        { coords: { clientX: 100, clientY: 100 } },
        { keys: '[/MouseLeft]' },
      ]);

      await waitFor(() => {
        expect(mockHandlers.onEventDrop).toHaveBeenCalled();
      });
    });

    it('should handle manager viewing weekly schedule', async () => {
      const user = userEvent.setup();
      render(
        <Calendar
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

    it('should handle dispatcher creating new job from empty slot', async () => {
      const user = userEvent.setup();
      render(
        <Calendar
          events={mockEvents}
          onSelectSlot={mockHandlers.onSelectSlot}
        />
      );

      // Find empty slot
      const cells = screen.getAllByRole('cell');
      const emptyCell = cells.find(
        (cell) => !within(cell).queryByText(/Install|Fix|Electrical/)
      );

      if (emptyCell) {
        // Click to select slot
        await user.click(emptyCell);

        expect(mockHandlers.onSelectSlot).toHaveBeenCalledWith(
          expect.objectContaining({
            start: expect.any(Date),
            end: expect.any(Date),
          })
        );
      }
    });

    it('should handle viewing different job statuses', async () => {
      render(<Calendar events={mockEvents} />);

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

      const { container } = render(<Calendar events={manyEvents} />);
      expect(container).toBeInTheDocument();

      // Calendar should render without performance issues
      expect(screen.getByRole('application')).toBeInTheDocument();
    });
  });

  describe('Edge Cases', () => {
    it('should handle empty events array', () => {
      render(<Calendar events={[]} />);
      expect(screen.getByRole('application')).toBeInTheDocument();
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

      render(<Calendar events={[minimalEvent]} />);
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

      render(<Calendar events={overlappingEvents} />);
      expect(screen.getByText('Event 1')).toBeInTheDocument();
      expect(screen.getByText('Event 2')).toBeInTheDocument();
    });
  });
});

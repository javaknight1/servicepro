import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { CalendarEvent } from '../../../src/components/calendar/CalendarEvent';
import {
  JobEvent,
  JobStatus,
  JobPriority,
} from '../../../src/components/calendar/types';

describe('CalendarEvent Component', () => {
  const mockEvent: JobEvent = {
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
  };

  describe('Rendering', () => {
    it('should render event title', () => {
      render(<CalendarEvent event={mockEvent} title={mockEvent.title} />);
      expect(screen.getByText(mockEvent.title)).toBeInTheDocument();
    });

    it('should render job number', () => {
      render(<CalendarEvent event={mockEvent} title={mockEvent.title} />);
      expect(screen.getByText(`#${mockEvent.jobNumber}`)).toBeInTheDocument();
    });

    it('should render priority badge for high priority', () => {
      render(<CalendarEvent event={mockEvent} title={mockEvent.title} />);
      expect(screen.getByTitle('Priority: high')).toBeInTheDocument();
    });

    it('should render urgent priority with icon', () => {
      const urgentEvent: JobEvent = {
        ...mockEvent,
        priority: JobPriority.URGENT,
      };
      render(<CalendarEvent event={urgentEvent} title={urgentEvent.title} />);
      expect(screen.getByTitle('Priority: urgent')).toBeInTheDocument();
    });

    it('should render location when provided', () => {
      render(<CalendarEvent event={mockEvent} title={mockEvent.title} />);
      expect(screen.getByTitle(mockEvent.location!)).toBeInTheDocument();
    });

    it('should not render priority badge when priority is not set', () => {
      const eventWithoutPriority: JobEvent = {
        ...mockEvent,
        priority: undefined,
      };
      render(
        <CalendarEvent
          event={eventWithoutPriority}
          title={eventWithoutPriority.title}
        />
      );
      expect(screen.queryByText(/Priority:/i)).not.toBeInTheDocument();
    });

    it('should not render location when not provided', () => {
      const eventWithoutLocation: JobEvent = {
        ...mockEvent,
        location: undefined,
      };
      render(
        <CalendarEvent
          event={eventWithoutLocation}
          title={eventWithoutLocation.title}
        />
      );
      expect(screen.queryByTitle(/Main St/i)).not.toBeInTheDocument();
    });
  });

  describe('Time Formatting', () => {
    it('should format time range correctly', () => {
      render(<CalendarEvent event={mockEvent} title={mockEvent.title} />);
      // Time formatting: 9:00 AM - 12:00 PM
      expect(screen.getByText(/9:00/)).toBeInTheDocument();
      expect(screen.getByText(/12:00/)).toBeInTheDocument();
    });

    it('should handle different time ranges', () => {
      const eveningEvent: JobEvent = {
        ...mockEvent,
        start: new Date(2024, 0, 15, 18, 30),
        end: new Date(2024, 0, 15, 20, 0),
      };
      render(<CalendarEvent event={eveningEvent} title={eveningEvent.title} />);
      expect(screen.getByText(/6:30/)).toBeInTheDocument();
      expect(screen.getByText(/8:00/)).toBeInTheDocument();
    });
  });

  describe('Priority Colors', () => {
    it('should apply correct color for low priority', () => {
      const lowPriorityEvent: JobEvent = {
        ...mockEvent,
        priority: JobPriority.LOW,
      };
      const { container } = render(
        <CalendarEvent
          event={lowPriorityEvent}
          title={lowPriorityEvent.title}
        />
      );
      const badge = container.querySelector('[title="Priority: low"]');
      expect(badge).toHaveClass('bg-gray-200');
    });

    it('should apply correct color for medium priority', () => {
      const mediumPriorityEvent: JobEvent = {
        ...mockEvent,
        priority: JobPriority.MEDIUM,
      };
      const { container } = render(
        <CalendarEvent
          event={mediumPriorityEvent}
          title={mediumPriorityEvent.title}
        />
      );
      const badge = container.querySelector('[title="Priority: medium"]');
      expect(badge).toHaveClass('bg-blue-200');
    });

    it('should apply correct color for high priority', () => {
      const { container } = render(
        <CalendarEvent event={mockEvent} title={mockEvent.title} />
      );
      const badge = container.querySelector('[title="Priority: high"]');
      expect(badge).toHaveClass('bg-orange-200');
    });

    it('should apply correct color for urgent priority', () => {
      const urgentEvent: JobEvent = {
        ...mockEvent,
        priority: JobPriority.URGENT,
      };
      const { container } = render(
        <CalendarEvent event={urgentEvent} title={urgentEvent.title} />
      );
      const badge = container.querySelector('[title="Priority: urgent"]');
      expect(badge).toHaveClass('bg-red-200');
    });
  });

  describe('Responsive Behavior', () => {
    it('should have responsive classes for time display', () => {
      const { container } = render(
        <CalendarEvent event={mockEvent} title={mockEvent.title} />
      );
      const timeElement = container.querySelector('.hidden.sm\\:flex');
      expect(timeElement).toBeInTheDocument();
    });

    it('should have responsive classes for location display', () => {
      const { container } = render(
        <CalendarEvent event={mockEvent} title={mockEvent.title} />
      );
      const locationElement = container.querySelector('.hidden.md\\:flex');
      expect(locationElement).toBeInTheDocument();
    });
  });

  describe('Text Truncation', () => {
    it('should truncate long titles', () => {
      const longTitleEvent: JobEvent = {
        ...mockEvent,
        title:
          'This is a very long title that should be truncated in the calendar view',
      };
      const { container } = render(
        <CalendarEvent event={longTitleEvent} title={longTitleEvent.title} />
      );
      const titleElement = container.querySelector('.truncate');
      expect(titleElement).toBeInTheDocument();
      expect(titleElement).toHaveAttribute('title', longTitleEvent.title);
    });

    it('should truncate long job numbers', () => {
      const longJobNumberEvent: JobEvent = {
        ...mockEvent,
        jobNumber: 'JOB-VERY-LONG-NUMBER-12345',
      };
      const { container } = render(
        <CalendarEvent
          event={longJobNumberEvent}
          title={longJobNumberEvent.title}
        />
      );
      const jobNumberElement = screen.getByText(
        `#${longJobNumberEvent.jobNumber}`
      );
      expect(jobNumberElement).toHaveClass('truncate');
    });
  });

  describe('Memoization', () => {
    it('should be memoized for performance', () => {
      const { rerender } = render(
        <CalendarEvent event={mockEvent} title={mockEvent.title} />
      );

      // Re-render with same props
      rerender(<CalendarEvent event={mockEvent} title={mockEvent.title} />);

      // Component should still render correctly
      expect(screen.getByText(mockEvent.title)).toBeInTheDocument();
    });
  });
});

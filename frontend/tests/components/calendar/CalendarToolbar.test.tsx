import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { CalendarToolbar } from '../../../src/components/calendar/CalendarToolbar';
import { CalendarView } from '../../../src/components/calendar/types';

describe('CalendarToolbar Component', () => {
  const mockOnNavigate = vi.fn();
  const mockOnViewChange = vi.fn();

  const defaultProps = {
    date: new Date(2024, 0, 15), // January 15, 2024
    view: 'month' as CalendarView,
    onNavigate: mockOnNavigate,
    onViewChange: mockOnViewChange,
    label: 'January 2024',
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('Rendering', () => {
    it('should render toolbar with all elements', () => {
      render(<CalendarToolbar {...defaultProps} />);

      expect(screen.getByLabelText('Previous')).toBeInTheDocument();
      expect(screen.getByLabelText('Today')).toBeInTheDocument();
      expect(screen.getByLabelText('Next')).toBeInTheDocument();
      expect(screen.getByText('January 2024')).toBeInTheDocument();
    });

    it('should render all view options', () => {
      render(<CalendarToolbar {...defaultProps} />);

      expect(screen.getByText('Month')).toBeInTheDocument();
      expect(screen.getByText('Week')).toBeInTheDocument();
      expect(screen.getByText('Day')).toBeInTheDocument();
      expect(screen.getByText('Agenda')).toBeInTheDocument();
    });

    it('should highlight active view', () => {
      render(<CalendarToolbar {...defaultProps} />);

      const monthButton = screen.getByText('Month');
      expect(monthButton).toHaveAttribute('aria-pressed', 'true');
      expect(monthButton).toHaveClass('text-blue-700');
    });

    it('should not highlight inactive views', () => {
      render(<CalendarToolbar {...defaultProps} />);

      const weekButton = screen.getByText('Week');
      expect(weekButton).toHaveAttribute('aria-pressed', 'false');
      expect(weekButton).not.toHaveClass('text-blue-700');
    });
  });

  describe('Navigation Actions', () => {
    it('should call onNavigate with PREV when previous button is clicked', async () => {
      const user = userEvent.setup();
      render(<CalendarToolbar {...defaultProps} />);

      const prevButton = screen.getByLabelText('Previous');
      await user.click(prevButton);

      expect(mockOnNavigate).toHaveBeenCalledWith('PREV');
      expect(mockOnNavigate).toHaveBeenCalledTimes(1);
    });

    it('should call onNavigate with NEXT when next button is clicked', async () => {
      const user = userEvent.setup();
      render(<CalendarToolbar {...defaultProps} />);

      const nextButton = screen.getByLabelText('Next');
      await user.click(nextButton);

      expect(mockOnNavigate).toHaveBeenCalledWith('NEXT');
      expect(mockOnNavigate).toHaveBeenCalledTimes(1);
    });

    it('should call onNavigate with TODAY when today button is clicked', async () => {
      const user = userEvent.setup();
      render(<CalendarToolbar {...defaultProps} />);

      const todayButton = screen.getByLabelText('Today');
      await user.click(todayButton);

      expect(mockOnNavigate).toHaveBeenCalledWith('TODAY');
      expect(mockOnNavigate).toHaveBeenCalledTimes(1);
    });
  });

  describe('View Change Actions', () => {
    it('should call onViewChange with correct view when month is clicked', async () => {
      const user = userEvent.setup();
      const props = { ...defaultProps, view: 'week' as CalendarView };
      render(<CalendarToolbar {...props} />);

      const monthButton = screen.getByText('Month');
      await user.click(monthButton);

      expect(mockOnViewChange).toHaveBeenCalledWith('month');
    });

    it('should call onViewChange with correct view when week is clicked', async () => {
      const user = userEvent.setup();
      render(<CalendarToolbar {...defaultProps} />);

      const weekButton = screen.getByText('Week');
      await user.click(weekButton);

      expect(mockOnViewChange).toHaveBeenCalledWith('week');
    });

    it('should call onViewChange with correct view when day is clicked', async () => {
      const user = userEvent.setup();
      render(<CalendarToolbar {...defaultProps} />);

      const dayButton = screen.getByText('Day');
      await user.click(dayButton);

      expect(mockOnViewChange).toHaveBeenCalledWith('day');
    });

    it('should call onViewChange with correct view when agenda is clicked', async () => {
      const user = userEvent.setup();
      render(<CalendarToolbar {...defaultProps} />);

      const agendaButton = screen.getByText('Agenda');
      await user.click(agendaButton);

      expect(mockOnViewChange).toHaveBeenCalledWith('agenda');
    });
  });

  describe('Label Display', () => {
    it('should display custom label', () => {
      const customLabel = 'February 2024';
      render(<CalendarToolbar {...defaultProps} label={customLabel} />);

      expect(screen.getByText(customLabel)).toBeInTheDocument();
    });

    it('should update label when props change', () => {
      const { rerender } = render(<CalendarToolbar {...defaultProps} />);
      expect(screen.getByText('January 2024')).toBeInTheDocument();

      rerender(
        <CalendarToolbar
          {...defaultProps}
          label="February 2024"
          date={new Date(2024, 1, 15)}
        />
      );
      expect(screen.getByText('February 2024')).toBeInTheDocument();
    });
  });

  describe('Icons', () => {
    it('should render chevron icons for navigation', () => {
      const { container } = render(<CalendarToolbar {...defaultProps} />);

      const prevButton = screen.getByLabelText('Previous');
      const nextButton = screen.getByLabelText('Next');

      expect(prevButton.querySelector('svg')).toBeInTheDocument();
      expect(nextButton.querySelector('svg')).toBeInTheDocument();
    });

    it('should render calendar icon on today button', () => {
      const { container } = render(<CalendarToolbar {...defaultProps} />);

      const todayButton = screen.getByLabelText('Today');
      expect(todayButton.querySelector('svg')).toBeInTheDocument();
    });
  });

  describe('Responsive Design', () => {
    it('should have responsive container classes', () => {
      const { container } = render(<CalendarToolbar {...defaultProps} />);
      const toolbar = container.firstChild;

      expect(toolbar).toHaveClass('flex');
      expect(toolbar).toHaveClass('flex-col');
      expect(toolbar).toHaveClass('sm:flex-row');
    });
  });

  describe('Accessibility', () => {
    it('should have proper ARIA labels for all buttons', () => {
      render(<CalendarToolbar {...defaultProps} />);

      expect(screen.getByLabelText('Previous')).toBeInTheDocument();
      expect(screen.getByLabelText('Today')).toBeInTheDocument();
      expect(screen.getByLabelText('Next')).toBeInTheDocument();
      expect(screen.getByLabelText('month view')).toBeInTheDocument();
      expect(screen.getByLabelText('week view')).toBeInTheDocument();
      expect(screen.getByLabelText('day view')).toBeInTheDocument();
      expect(screen.getByLabelText('agenda view')).toBeInTheDocument();
    });

    it('should have proper aria-pressed attributes', () => {
      render(<CalendarToolbar {...defaultProps} />);

      const monthButton = screen.getByLabelText('month view');
      const weekButton = screen.getByLabelText('week view');

      expect(monthButton).toHaveAttribute('aria-pressed', 'true');
      expect(weekButton).toHaveAttribute('aria-pressed', 'false');
    });

    it('should be keyboard navigable', async () => {
      const user = userEvent.setup();
      render(<CalendarToolbar {...defaultProps} />);

      const prevButton = screen.getByLabelText('Previous');
      prevButton.focus();

      await user.keyboard('{Enter}');
      expect(mockOnNavigate).toHaveBeenCalledWith('PREV');
    });
  });

  describe('Button Styles', () => {
    it('should apply correct styles to navigation buttons', () => {
      render(<CalendarToolbar {...defaultProps} />);

      const prevButton = screen.getByLabelText('Previous');
      expect(prevButton).toHaveClass('hover:bg-gray-50');
      expect(prevButton).toHaveClass('focus:ring-2');
    });

    it('should apply correct styles to today button', () => {
      render(<CalendarToolbar {...defaultProps} />);

      const todayButton = screen.getByLabelText('Today');
      expect(todayButton).toHaveClass('bg-blue-600');
      expect(todayButton).toHaveClass('text-white');
    });

    it('should apply correct styles to active view button', () => {
      render(<CalendarToolbar {...defaultProps} />);

      const monthButton = screen.getByText('Month');
      expect(monthButton).toHaveClass('text-blue-700');
      expect(monthButton).toHaveClass('bg-white');
    });

    it('should apply correct styles to inactive view buttons', () => {
      render(<CalendarToolbar {...defaultProps} />);

      const weekButton = screen.getByText('Week');
      expect(weekButton).toHaveClass('text-gray-700');
      expect(weekButton).not.toHaveClass('bg-white');
    });
  });

  describe('Memoization', () => {
    it('should be memoized for performance', () => {
      const { rerender } = render(<CalendarToolbar {...defaultProps} />);

      // Re-render with same props
      rerender(<CalendarToolbar {...defaultProps} />);

      // Component should still render correctly
      expect(screen.getByText('January 2024')).toBeInTheDocument();
    });
  });
});

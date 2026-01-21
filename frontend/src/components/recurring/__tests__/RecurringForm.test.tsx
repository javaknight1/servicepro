import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { RecurringForm } from '../RecurringForm';
import { RecurrenceType, RecurringPatternResponse } from '../types';

describe('RecurringForm', () => {
  const mockOnSubmit = vi.fn();
  const mockOnCancel = vi.fn();

  beforeEach(() => {
    vi.clearAllMocks();
  });

  // ==========================================================================
  // Rendering Tests
  // ==========================================================================
  describe('Rendering', () => {
    it('renders all basic form fields', () => {
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/description/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/location/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/recurrence type/i)).toBeInTheDocument();
    });

    it('renders time fields', () => {
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      expect(screen.getByLabelText(/start date/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/start time/i)).toBeInTheDocument();
      expect(screen.getByLabelText(/end time/i)).toBeInTheDocument();
    });

    it('renders end condition options', () => {
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      expect(screen.getByLabelText(/never ends/i)).toBeInTheDocument();
      expect(screen.getByText(/ends on/i)).toBeInTheDocument();
      expect(screen.getByText(/ends after/i)).toBeInTheDocument();
    });

    it('renders submit button', () => {
      render(<RecurringForm onSubmit={mockOnSubmit} />);
      expect(
        screen.getByRole('button', { name: /create pattern/i })
      ).toBeInTheDocument();
    });

    it('renders cancel button when onCancel provided', () => {
      render(<RecurringForm onSubmit={mockOnSubmit} onCancel={mockOnCancel} />);
      expect(
        screen.getByRole('button', { name: /cancel/i })
      ).toBeInTheDocument();
    });

    it('does not render cancel button when onCancel not provided', () => {
      render(<RecurringForm onSubmit={mockOnSubmit} />);
      expect(
        screen.queryByRole('button', { name: /cancel/i })
      ).not.toBeInTheDocument();
    });
  });

  // ==========================================================================
  // Initial Data Tests
  // ==========================================================================
  describe('Initial Data', () => {
    it('populates form with initial data', () => {
      const initialData = {
        id: '1',
        title: 'Weekly Maintenance',
        description: 'Regular maintenance',
        recurrenceType: RecurrenceType.WEEKLY,
        recurrenceRule: '',
        startDate: '2024-01-01',
        interval: 2,
        daysOfWeek: [1, 3, 5],
        timeStart: '10:00',
        timeEnd: '14:00',
        duration: 240,
        assignedTechIds: [],
        location: '123 Main St',
        isActive: true,
        createdAt: '2024-01-01',
        updatedAt: '2024-01-01',
      } as RecurringPatternResponse;

      render(
        <RecurringForm onSubmit={mockOnSubmit} initialData={initialData} />
      );

      expect(
        screen.getByDisplayValue('Weekly Maintenance')
      ).toBeInTheDocument();
      expect(
        screen.getByDisplayValue('Regular maintenance')
      ).toBeInTheDocument();
      expect(screen.getByDisplayValue('123 Main St')).toBeInTheDocument();
    });

    it('shows "Update Pattern" button when editing', () => {
      const initialData = {
        id: '1',
        title: 'Test',
        description: '',
        recurrenceType: RecurrenceType.WEEKLY,
        recurrenceRule: '',
        startDate: '2024-01-01',
        interval: 1,
        timeStart: '09:00',
        timeEnd: '17:00',
        duration: 480,
        assignedTechIds: [],
        location: '',
        isActive: true,
        createdAt: '2024-01-01',
        updatedAt: '2024-01-01',
      } as RecurringPatternResponse;

      render(
        <RecurringForm onSubmit={mockOnSubmit} initialData={initialData} />
      );
      expect(
        screen.getByRole('button', { name: /update pattern/i })
      ).toBeInTheDocument();
    });
  });

  // ==========================================================================
  // Title Validation Tests
  // ==========================================================================
  describe('Title Validation', () => {
    it('shows error when title is empty', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(screen.getByText(/title is required/i)).toBeInTheDocument();
      expect(mockOnSubmit).not.toHaveBeenCalled();
    });

    it('shows error when title is only whitespace', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/title/i), '   ');
      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(screen.getByText(/title is required/i)).toBeInTheDocument();
    });

    it('clears error when valid title is entered', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      // First submit with empty title
      await user.click(screen.getByRole('button', { name: /create pattern/i }));
      expect(screen.getByText(/title is required/i)).toBeInTheDocument();

      // Then enter valid title
      await user.type(screen.getByLabelText(/title/i), 'Valid Title');

      expect(screen.queryByText(/title is required/i)).not.toBeInTheDocument();
    });
  });

  // ==========================================================================
  // Weekly Recurrence Validation Tests
  // ==========================================================================
  describe('Weekly Recurrence Validation', () => {
    it('shows error when no days selected for weekly recurrence', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      // Fill required fields
      await user.type(screen.getByLabelText(/title/i), 'Test Pattern');

      // Default is weekly, so just submit
      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(screen.getByText(/select at least one day/i)).toBeInTheDocument();
    });

    it('allows submission when days are selected', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/title/i), 'Test Pattern');

      // Select Monday
      await user.click(screen.getByRole('button', { name: 'Mon' }));

      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(
        screen.queryByText(/select at least one day/i)
      ).not.toBeInTheDocument();
      expect(mockOnSubmit).toHaveBeenCalled();
    });

    it('toggles days correctly', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      const monButton = screen.getByRole('button', { name: 'Mon' });
      const wedButton = screen.getByRole('button', { name: 'Wed' });

      // Initially not selected
      expect(monButton).toHaveClass('bg-gray-100');

      // Select Monday
      await user.click(monButton);
      expect(monButton).toHaveClass('bg-blue-600');

      // Select Wednesday
      await user.click(wedButton);
      expect(wedButton).toHaveClass('bg-blue-600');

      // Deselect Monday
      await user.click(monButton);
      expect(monButton).toHaveClass('bg-gray-100');
      expect(wedButton).toHaveClass('bg-blue-600');
    });
  });

  // ==========================================================================
  // Monthly Recurrence Validation Tests
  // ==========================================================================
  describe('Monthly Recurrence Validation', () => {
    it('shows error when day of month not set for monthly', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/title/i), 'Test Pattern');

      // Change to monthly
      await user.selectOptions(
        screen.getByLabelText(/recurrence type/i),
        RecurrenceType.MONTHLY
      );

      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(screen.getByText(/day of month is required/i)).toBeInTheDocument();
    });

    it('allows submission when day of month is set', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/title/i), 'Test Pattern');
      await user.selectOptions(
        screen.getByLabelText(/recurrence type/i),
        RecurrenceType.MONTHLY
      );

      // Set day of month
      const dayInput = screen.getByRole('spinbutton', {
        name: /day of month/i,
      });
      await user.clear(dayInput);
      await user.type(dayInput, '15');

      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(
        screen.queryByText(/day of month is required/i)
      ).not.toBeInTheDocument();
      expect(mockOnSubmit).toHaveBeenCalled();
    });
  });

  // ==========================================================================
  // Yearly Recurrence Validation Tests
  // ==========================================================================
  describe('Yearly Recurrence Validation', () => {
    it('shows error when month not set for yearly', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/title/i), 'Test Pattern');
      await user.selectOptions(
        screen.getByLabelText(/recurrence type/i),
        RecurrenceType.YEARLY
      );

      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(screen.getByText(/month is required/i)).toBeInTheDocument();
    });

    it('shows error when day not set for yearly', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/title/i), 'Test Pattern');
      await user.selectOptions(
        screen.getByLabelText(/recurrence type/i),
        RecurrenceType.YEARLY
      );

      // Set only month
      await user.selectOptions(
        screen.getByRole('combobox', { name: /month/i }),
        '6'
      );

      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(screen.getByText(/day is required/i)).toBeInTheDocument();
    });

    it('allows submission when month and day are set', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/title/i), 'Annual Review');
      await user.selectOptions(
        screen.getByLabelText(/recurrence type/i),
        RecurrenceType.YEARLY
      );

      // Set month and day
      await user.selectOptions(
        screen.getByRole('combobox', { name: /month/i }),
        '6'
      );
      const dayInput = screen.getByRole('spinbutton', { name: /day$/i });
      await user.type(dayInput, '15');

      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(mockOnSubmit).toHaveBeenCalled();
    });
  });

  // ==========================================================================
  // Daily Recurrence Tests
  // ==========================================================================
  describe('Daily Recurrence', () => {
    it('does not show days of week for daily recurrence', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.selectOptions(
        screen.getByLabelText(/recurrence type/i),
        RecurrenceType.DAILY
      );

      expect(
        screen.queryByRole('button', { name: 'Mon' })
      ).not.toBeInTheDocument();
    });

    it('allows submission with just title for daily', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/title/i), 'Daily Task');
      await user.selectOptions(
        screen.getByLabelText(/recurrence type/i),
        RecurrenceType.DAILY
      );

      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(mockOnSubmit).toHaveBeenCalled();
    });
  });

  // ==========================================================================
  // Time Validation Tests
  // ==========================================================================
  describe('Time Validation', () => {
    it('shows error when end time is before start time', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/title/i), 'Test Pattern');
      await user.selectOptions(
        screen.getByLabelText(/recurrence type/i),
        RecurrenceType.DAILY
      );

      // Set invalid times
      const startTimeInput = screen.getByLabelText(/start time/i);
      const endTimeInput = screen.getByLabelText(/end time/i);

      await user.clear(startTimeInput);
      await user.type(startTimeInput, '14:00');

      await user.clear(endTimeInput);
      await user.type(endTimeInput, '10:00');

      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(
        screen.getByText(/end time must be after start time/i)
      ).toBeInTheDocument();
    });

    it('shows error when end time equals start time', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/title/i), 'Test Pattern');
      await user.selectOptions(
        screen.getByLabelText(/recurrence type/i),
        RecurrenceType.DAILY
      );

      const startTimeInput = screen.getByLabelText(/start time/i);
      const endTimeInput = screen.getByLabelText(/end time/i);

      await user.clear(startTimeInput);
      await user.type(startTimeInput, '10:00');

      await user.clear(endTimeInput);
      await user.type(endTimeInput, '10:00');

      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(
        screen.getByText(/end time must be after start time/i)
      ).toBeInTheDocument();
    });

    it('accepts valid time range', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/title/i), 'Test Pattern');
      await user.selectOptions(
        screen.getByLabelText(/recurrence type/i),
        RecurrenceType.DAILY
      );

      const startTimeInput = screen.getByLabelText(/start time/i);
      const endTimeInput = screen.getByLabelText(/end time/i);

      await user.clear(startTimeInput);
      await user.type(startTimeInput, '09:00');

      await user.clear(endTimeInput);
      await user.type(endTimeInput, '17:00');

      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(
        screen.queryByText(/end time must be after start time/i)
      ).not.toBeInTheDocument();
      expect(mockOnSubmit).toHaveBeenCalled();
    });
  });

  // ==========================================================================
  // End Date Validation Tests
  // ==========================================================================
  describe('End Date Validation', () => {
    it('shows error when end date is before start date', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/title/i), 'Test Pattern');
      await user.selectOptions(
        screen.getByLabelText(/recurrence type/i),
        RecurrenceType.DAILY
      );

      // Select "Ends on" option
      await user.click(screen.getByLabelText(/ends on/i));

      // Set dates - end before start
      const startDateInput = screen.getByLabelText(/start date/i);
      await user.clear(startDateInput);
      await user.type(startDateInput, '2024-06-01');

      // Find the end date input (appears after selecting "Ends on")
      const endDateInputs = screen
        .getAllByRole('textbox')
        .filter((input) => (input as HTMLInputElement).type === 'date');
      const endDateInput = endDateInputs[endDateInputs.length - 1];
      await user.clear(endDateInput);
      await user.type(endDateInput, '2024-01-01');

      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(
        screen.getByText(/end date must be after start date/i)
      ).toBeInTheDocument();
    });
  });

  // ==========================================================================
  // End Condition Tests
  // ==========================================================================
  describe('End Conditions', () => {
    it('defaults to never ends', () => {
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      expect(screen.getByLabelText(/never ends/i)).toBeChecked();
    });

    it('clears end date when switching to never', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      // Select ends on
      await user.click(screen.getByLabelText(/ends on/i));

      // Switch back to never
      await user.click(screen.getByLabelText(/never ends/i));

      expect(screen.getByLabelText(/never ends/i)).toBeChecked();
    });

    it('clears occurrences when switching to date mode', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      // Select ends after
      await user.click(screen.getByLabelText(/ends after/i));

      // Enter occurrence count
      const occurrenceInput = screen.getByRole('spinbutton');
      await user.type(occurrenceInput, '10');

      // Switch to date mode
      await user.click(screen.getByLabelText(/ends on/i));

      expect(screen.getByLabelText(/ends on/i)).toBeChecked();
    });

    it('disables end date input when not in date mode', () => {
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      // The end date input should be disabled when "Never ends" is selected
      const dateInputs = screen
        .getAllByRole('textbox')
        .filter((input) => (input as HTMLInputElement).type === 'date');
      // Last date input should be the end date
      const endDateInput = dateInputs[dateInputs.length - 1];
      expect(endDateInput).toBeDisabled();
    });

    it('enables end date input in date mode', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.click(screen.getByLabelText(/ends on/i));

      const dateInputs = screen
        .getAllByRole('textbox')
        .filter((input) => (input as HTMLInputElement).type === 'date');
      const endDateInput = dateInputs[dateInputs.length - 1];
      expect(endDateInput).not.toBeDisabled();
    });
  });

  // ==========================================================================
  // Form Submission Tests
  // ==========================================================================
  describe('Form Submission', () => {
    it('submits form with correct data', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/title/i), 'Weekly Meeting');
      await user.type(screen.getByLabelText(/description/i), 'Team sync');
      await user.type(screen.getByLabelText(/location/i), 'Conference Room');

      // Select days
      await user.click(screen.getByRole('button', { name: 'Mon' }));
      await user.click(screen.getByRole('button', { name: 'Wed' }));
      await user.click(screen.getByRole('button', { name: 'Fri' }));

      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(mockOnSubmit).toHaveBeenCalledWith(
        expect.objectContaining({
          title: 'Weekly Meeting',
          description: 'Team sync',
          location: 'Conference Room',
          recurrenceType: RecurrenceType.WEEKLY,
          daysOfWeek: [1, 3, 5],
        })
      );
    });

    it('prevents default form submission', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/title/i), 'Test');
      await user.selectOptions(
        screen.getByLabelText(/recurrence type/i),
        RecurrenceType.DAILY
      );

      const form = screen
        .getByRole('button', { name: /create pattern/i })
        .closest('form');
      const submitEvent = vi.fn((e) => e.preventDefault());
      form?.addEventListener('submit', submitEvent);

      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(mockOnSubmit).toHaveBeenCalled();
    });
  });

  // ==========================================================================
  // Cancel Button Tests
  // ==========================================================================
  describe('Cancel Button', () => {
    it('calls onCancel when clicked', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} onCancel={mockOnCancel} />);

      await user.click(screen.getByRole('button', { name: /cancel/i }));

      expect(mockOnCancel).toHaveBeenCalledTimes(1);
    });

    it('is disabled when loading', () => {
      render(
        <RecurringForm
          onSubmit={mockOnSubmit}
          onCancel={mockOnCancel}
          isLoading={true}
        />
      );

      expect(screen.getByRole('button', { name: /cancel/i })).toBeDisabled();
    });
  });

  // ==========================================================================
  // Loading State Tests
  // ==========================================================================
  describe('Loading State', () => {
    it('disables submit button when loading', () => {
      render(<RecurringForm onSubmit={mockOnSubmit} isLoading={true} />);
      expect(
        screen.getByRole('button', { name: /create pattern/i })
      ).toBeDisabled();
    });

    it('shows loading spinner when loading', () => {
      const { container } = render(
        <RecurringForm onSubmit={mockOnSubmit} isLoading={true} />
      );

      expect(container.querySelector('.animate-spin')).toBeInTheDocument();
    });
  });

  // ==========================================================================
  // Interval Field Tests
  // ==========================================================================
  describe('Interval Field', () => {
    it('displays correct unit label based on recurrence type', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      // Weekly (default)
      expect(screen.getByText('week(s)')).toBeInTheDocument();

      // Daily
      await user.selectOptions(
        screen.getByLabelText(/recurrence type/i),
        RecurrenceType.DAILY
      );
      expect(screen.getByText('day(s)')).toBeInTheDocument();

      // Monthly
      await user.selectOptions(
        screen.getByLabelText(/recurrence type/i),
        RecurrenceType.MONTHLY
      );
      expect(screen.getByText('month(s)')).toBeInTheDocument();

      // Yearly
      await user.selectOptions(
        screen.getByLabelText(/recurrence type/i),
        RecurrenceType.YEARLY
      );
      expect(screen.getByText('year(s)')).toBeInTheDocument();
    });

    it('accepts custom interval values', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByLabelText(/title/i), 'Bi-weekly Task');
      await user.click(screen.getByRole('button', { name: 'Mon' }));

      const intervalInput = screen.getByRole('spinbutton', { name: '' });
      await user.clear(intervalInput);
      await user.type(intervalInput, '2');

      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(mockOnSubmit).toHaveBeenCalledWith(
        expect.objectContaining({
          interval: 2,
        })
      );
    });
  });

  // ==========================================================================
  // Recurrence Type Switch Tests
  // ==========================================================================
  describe('Recurrence Type Switching', () => {
    const recurrenceTypes = [
      { type: RecurrenceType.DAILY, fieldVisible: false, label: 'day' },
      {
        type: RecurrenceType.WEEKLY,
        fieldVisible: true,
        label: 'days of week',
      },
      {
        type: RecurrenceType.MONTHLY,
        fieldVisible: true,
        label: 'day of month',
      },
      {
        type: RecurrenceType.YEARLY,
        fieldVisible: true,
        label: 'month selector',
      },
    ];

    recurrenceTypes.forEach(({ type, fieldVisible, label }) => {
      it(`shows correct fields for ${type} recurrence`, async () => {
        const user = userEvent.setup();
        render(<RecurringForm onSubmit={mockOnSubmit} />);

        await user.selectOptions(
          screen.getByLabelText(/recurrence type/i),
          type
        );

        if (type === RecurrenceType.WEEKLY) {
          expect(
            screen.getByRole('button', { name: 'Mon' })
          ).toBeInTheDocument();
        } else if (type === RecurrenceType.MONTHLY) {
          expect(screen.getByLabelText(/day of month/i)).toBeInTheDocument();
        } else if (type === RecurrenceType.YEARLY) {
          expect(
            screen.getByRole('combobox', { name: /month/i })
          ).toBeInTheDocument();
        }
      });
    });
  });
});

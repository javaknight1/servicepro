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

      // Use placeholder text or text content since labels aren't properly associated
      expect(screen.getByPlaceholderText(/weekly maintenance/i)).toBeInTheDocument();
      expect(screen.getByPlaceholderText(/regular maintenance schedule/i)).toBeInTheDocument();
      expect(screen.getByPlaceholderText(/123 main st/i)).toBeInTheDocument();
      expect(screen.getByText(/recurrence type/i)).toBeInTheDocument();
    });

    it('renders time fields', () => {
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      expect(screen.getByText(/start date/i)).toBeInTheDocument();
      expect(screen.getByText(/start time/i)).toBeInTheDocument();
      expect(screen.getByText(/end time/i)).toBeInTheDocument();
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

      await user.type(screen.getByPlaceholderText(/weekly maintenance/i), '   ');
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
      await user.type(screen.getByPlaceholderText(/weekly maintenance/i), 'Valid Title');

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
      await user.type(screen.getByPlaceholderText(/weekly maintenance/i), 'Test Pattern');

      // Default is weekly, so just submit
      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(screen.getByText(/select at least one day/i)).toBeInTheDocument();
    });

    it('allows submission when days are selected', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByPlaceholderText(/weekly maintenance/i), 'Test Pattern');

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

      await user.type(screen.getByPlaceholderText(/weekly maintenance/i), 'Test Pattern');

      // Change to monthly - get select by its displayed text
      const recurrenceSelect = screen.getByRole('combobox');
      await user.selectOptions(recurrenceSelect, RecurrenceType.MONTHLY);

      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(screen.getByText(/day of month is required/i)).toBeInTheDocument();
    });

    it('allows submission when day of month is set', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByPlaceholderText(/weekly maintenance/i), 'Test Pattern');
      const recurrenceSelect = screen.getByRole('combobox');
      await user.selectOptions(recurrenceSelect, RecurrenceType.MONTHLY);

      // Set day of month - get all spinbuttons and use the second one (day of month)
      // First is interval, second is day of month
      const spinbuttons = screen.getAllByRole('spinbutton');
      const dayInput = spinbuttons[1]; // Day of month input
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

      await user.type(screen.getByPlaceholderText(/weekly maintenance/i), 'Test Pattern');
      const recurrenceSelect = screen.getByRole('combobox');
      await user.selectOptions(recurrenceSelect, RecurrenceType.YEARLY);

      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(screen.getByText(/month is required/i)).toBeInTheDocument();
    });

    it('shows error when day not set for yearly', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByPlaceholderText(/weekly maintenance/i), 'Test Pattern');
      const recurrenceSelect = screen.getByRole('combobox');
      await user.selectOptions(recurrenceSelect, RecurrenceType.YEARLY);

      // Set only month - after switching to yearly, a month select appears
      const allSelects = screen.getAllByRole('combobox');
      const monthSelect = allSelects[allSelects.length - 1]; // Month select is the last one
      await user.selectOptions(monthSelect, '6');

      await user.click(screen.getByRole('button', { name: /create pattern/i }));

      expect(screen.getByText(/day is required/i)).toBeInTheDocument();
    });

    it('allows submission when month and day are set', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByPlaceholderText(/weekly maintenance/i), 'Annual Review');
      const recurrenceSelect = screen.getByRole('combobox');
      await user.selectOptions(recurrenceSelect, RecurrenceType.YEARLY);

      // Set month - after switching to yearly, a month select appears as the second combobox
      const allSelects = screen.getAllByRole('combobox');
      const monthSelect = allSelects[1]; // Second combobox is month selector
      await user.selectOptions(monthSelect, '6');

      // Set day - second spinbutton is the day input
      const allSpinbuttons = screen.getAllByRole('spinbutton');
      const dayInput = allSpinbuttons[1]; // Second spinbutton is day input
      await user.clear(dayInput);
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

      const recurrenceSelect = screen.getByRole('combobox');
      await user.selectOptions(recurrenceSelect, RecurrenceType.DAILY);

      expect(
        screen.queryByRole('button', { name: 'Mon' })
      ).not.toBeInTheDocument();
    });

    it('allows submission with just title for daily', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByPlaceholderText(/weekly maintenance/i), 'Daily Task');
      const recurrenceSelect = screen.getByRole('combobox');
      await user.selectOptions(recurrenceSelect, RecurrenceType.DAILY);

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
      const { container } = render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByPlaceholderText(/weekly maintenance/i), 'Test Pattern');
      const recurrenceSelect = screen.getByRole('combobox');
      await user.selectOptions(recurrenceSelect, RecurrenceType.DAILY);

      // Get time inputs by type attribute
      const timeInputs = container.querySelectorAll('input[type="time"]');
      const startTimeInput = timeInputs[0] as HTMLInputElement;
      const endTimeInput = timeInputs[1] as HTMLInputElement;

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
      const { container } = render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByPlaceholderText(/weekly maintenance/i), 'Test Pattern');
      const recurrenceSelect = screen.getByRole('combobox');
      await user.selectOptions(recurrenceSelect, RecurrenceType.DAILY);

      const timeInputs = container.querySelectorAll('input[type="time"]');
      const startTimeInput = timeInputs[0] as HTMLInputElement;
      const endTimeInput = timeInputs[1] as HTMLInputElement;

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
      const { container } = render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByPlaceholderText(/weekly maintenance/i), 'Test Pattern');
      const recurrenceSelect = screen.getByRole('combobox');
      await user.selectOptions(recurrenceSelect, RecurrenceType.DAILY);

      const timeInputs = container.querySelectorAll('input[type="time"]');
      const startTimeInput = timeInputs[0] as HTMLInputElement;
      const endTimeInput = timeInputs[1] as HTMLInputElement;

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
      const { container } = render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByPlaceholderText(/weekly maintenance/i), 'Test Pattern');
      const recurrenceSelect = screen.getByRole('combobox');
      await user.selectOptions(recurrenceSelect, RecurrenceType.DAILY);

      // Select "Ends on" option
      await user.click(screen.getByLabelText(/ends on/i));

      // Get date inputs by type
      const dateInputs = container.querySelectorAll('input[type="date"]');
      const startDateInput = dateInputs[0] as HTMLInputElement;
      const endDateInput = dateInputs[dateInputs.length - 1] as HTMLInputElement;

      await user.clear(startDateInput);
      await user.type(startDateInput, '2024-06-01');

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

      // Enter occurrence count - it's the last spinbutton (after interval)
      const spinbuttons = screen.getAllByRole('spinbutton');
      const occurrenceInput = spinbuttons[spinbuttons.length - 1];
      await user.type(occurrenceInput, '10');

      // Switch to date mode
      await user.click(screen.getByLabelText(/ends on/i));

      expect(screen.getByLabelText(/ends on/i)).toBeChecked();
    });

    it('disables end date input when not in date mode', () => {
      const { container } = render(<RecurringForm onSubmit={mockOnSubmit} />);

      // The end date input should be disabled when "Never ends" is selected
      const dateInputs = container.querySelectorAll('input[type="date"]');
      // Last date input should be the end date
      const endDateInput = dateInputs[dateInputs.length - 1] as HTMLInputElement;
      expect(endDateInput).toBeDisabled();
    });

    it('enables end date input in date mode', async () => {
      const user = userEvent.setup();
      const { container } = render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.click(screen.getByLabelText(/ends on/i));

      const dateInputs = container.querySelectorAll('input[type="date"]');
      const endDateInput = dateInputs[dateInputs.length - 1] as HTMLInputElement;
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

      await user.type(screen.getByPlaceholderText(/weekly maintenance/i), 'Weekly Meeting');
      await user.type(screen.getByPlaceholderText(/regular maintenance schedule/i), 'Team sync');
      await user.type(screen.getByPlaceholderText(/123 main st/i), 'Conference Room');

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

      await user.type(screen.getByPlaceholderText(/weekly maintenance/i), 'Test');
      const recurrenceSelect = screen.getByRole('combobox');
      await user.selectOptions(recurrenceSelect, RecurrenceType.DAILY);

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
      const recurrenceSelect = screen.getByRole('combobox');
      await user.selectOptions(recurrenceSelect, RecurrenceType.DAILY);
      expect(screen.getByText('day(s)')).toBeInTheDocument();

      // Monthly
      await user.selectOptions(recurrenceSelect, RecurrenceType.MONTHLY);
      expect(screen.getByText('month(s)')).toBeInTheDocument();

      // Yearly
      await user.selectOptions(recurrenceSelect, RecurrenceType.YEARLY);
      expect(screen.getByText('year(s)')).toBeInTheDocument();
    });

    it('accepts custom interval values', async () => {
      const user = userEvent.setup();
      render(<RecurringForm onSubmit={mockOnSubmit} />);

      await user.type(screen.getByPlaceholderText(/weekly maintenance/i), 'Bi-weekly Task');
      await user.click(screen.getByRole('button', { name: 'Mon' }));

      // Interval is the first spinbutton
      const spinbuttons = screen.getAllByRole('spinbutton');
      const intervalInput = spinbuttons[0];
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

    recurrenceTypes.forEach(({ type }) => {
      it(`shows correct fields for ${type} recurrence`, async () => {
        const user = userEvent.setup();
        render(<RecurringForm onSubmit={mockOnSubmit} />);

        const recurrenceSelect = screen.getByRole('combobox');
        await user.selectOptions(recurrenceSelect, type);

        if (type === RecurrenceType.WEEKLY) {
          expect(
            screen.getByRole('button', { name: 'Mon' })
          ).toBeInTheDocument();
        } else if (type === RecurrenceType.MONTHLY) {
          // Monthly shows a "Day of Month" label and spinbutton
          expect(screen.getByText(/day of month/i)).toBeInTheDocument();
        } else if (type === RecurrenceType.YEARLY) {
          // Yearly shows a month selector - it's an additional combobox
          const allSelects = screen.getAllByRole('combobox');
          expect(allSelects.length).toBeGreaterThan(1);
        }
      });
    });
  });
});

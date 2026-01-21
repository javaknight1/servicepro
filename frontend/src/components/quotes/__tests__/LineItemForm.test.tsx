import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import LineItemForm from '../LineItemForm';

describe('LineItemForm', () => {
  const mockOnAdd = vi.fn();
  const mockOnCancel = vi.fn();

  beforeEach(() => {
    mockOnAdd.mockClear();
    mockOnCancel.mockClear();
  });

  it('should render all form fields', () => {
    render(<LineItemForm onAdd={mockOnAdd} />);

    expect(screen.getByLabelText(/description/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/quantity/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/unit price/i)).toBeInTheDocument();
    expect(screen.getByText(/total/i)).toBeInTheDocument();
  });

  it('should call onAdd with valid data when form is submitted', async () => {
    const user = userEvent.setup();
    render(<LineItemForm onAdd={mockOnAdd} />);

    await user.type(screen.getByLabelText(/description/i), 'Test Service');
    await user.type(screen.getByLabelText(/quantity/i), '2');
    await user.type(screen.getByLabelText(/unit price/i), '50');

    await user.click(screen.getByRole('button', { name: /add/i }));

    await waitFor(() => {
      expect(mockOnAdd).toHaveBeenCalledWith({
        description: 'Test Service',
        quantity: '2',
        unit_price: '50',
      });
    });
  });

  it('should show validation error for empty description', async () => {
    const user = userEvent.setup();
    render(<LineItemForm onAdd={mockOnAdd} />);

    await user.type(screen.getByLabelText(/quantity/i), '2');
    await user.type(screen.getByLabelText(/unit price/i), '50');
    await user.click(screen.getByRole('button', { name: /add/i }));

    await waitFor(() => {
      expect(screen.getByText(/description is required/i)).toBeInTheDocument();
    });

    expect(mockOnAdd).not.toHaveBeenCalled();
  });

  it('should show validation error for negative quantity', async () => {
    const user = userEvent.setup();
    render(<LineItemForm onAdd={mockOnAdd} />);

    await user.type(screen.getByLabelText(/description/i), 'Test Service');
    await user.type(screen.getByLabelText(/quantity/i), '-1');
    await user.type(screen.getByLabelText(/unit price/i), '50');
    await user.click(screen.getByRole('button', { name: /add/i }));

    await waitFor(() => {
      expect(screen.getByText(/must be greater than 0/i)).toBeInTheDocument();
    });

    expect(mockOnAdd).not.toHaveBeenCalled();
  });

  it('should show validation error for negative unit price', async () => {
    const user = userEvent.setup();
    render(<LineItemForm onAdd={mockOnAdd} />);

    await user.type(screen.getByLabelText(/description/i), 'Test Service');
    await user.type(screen.getByLabelText(/quantity/i), '2');
    await user.type(screen.getByLabelText(/unit price/i), '-50');
    await user.click(screen.getByRole('button', { name: /add/i }));

    await waitFor(() => {
      expect(screen.getByText(/cannot be negative/i)).toBeInTheDocument();
    });

    expect(mockOnAdd).not.toHaveBeenCalled();
  });

  it('should calculate and display total preview', async () => {
    const user = userEvent.setup();
    render(<LineItemForm onAdd={mockOnAdd} />);

    await user.type(screen.getByLabelText(/quantity/i), '2');
    await user.type(screen.getByLabelText(/unit price/i), '50');

    expect(screen.getByText('$100.00')).toBeInTheDocument();
  });

  it('should clear form after successful submission', async () => {
    const user = userEvent.setup();
    render(<LineItemForm onAdd={mockOnAdd} />);

    const descriptionInput = screen.getByLabelText(
      /description/i
    ) as HTMLInputElement;
    const quantityInput = screen.getByLabelText(
      /quantity/i
    ) as HTMLInputElement;
    const priceInput = screen.getByLabelText(/unit price/i) as HTMLInputElement;

    await user.type(descriptionInput, 'Test Service');
    await user.type(quantityInput, '2');
    await user.type(priceInput, '50');

    await user.click(screen.getByRole('button', { name: /add/i }));

    await waitFor(() => {
      expect(descriptionInput.value).toBe('');
      expect(quantityInput.value).toBe('');
      expect(priceInput.value).toBe('');
    });
  });

  it('should populate form with initial data', () => {
    const initialData = {
      description: 'Initial Service',
      quantity: 3,
      unit_price: 75,
    };

    render(<LineItemForm onAdd={mockOnAdd} initialData={initialData} />);

    expect(screen.getByLabelText(/description/i)).toHaveValue(
      'Initial Service'
    );
    expect(screen.getByLabelText(/quantity/i)).toHaveValue(3);
    expect(screen.getByLabelText(/unit price/i)).toHaveValue(75);
  });

  it('should call onCancel when cancel button is clicked', async () => {
    const user = userEvent.setup();
    render(
      <LineItemForm onAdd={mockOnAdd} onCancel={mockOnCancel} showCancel />
    );

    await user.click(screen.getByRole('button', { name: /cancel/i }));

    expect(mockOnCancel).toHaveBeenCalled();
  });

  it('should clear form when clear button is clicked', async () => {
    const user = userEvent.setup();
    render(<LineItemForm onAdd={mockOnAdd} />);

    await user.type(screen.getByLabelText(/description/i), 'Test Service');
    await user.type(screen.getByLabelText(/quantity/i), '2');
    await user.type(screen.getByLabelText(/unit price/i), '50');

    await user.click(screen.getByText(/clear form/i));

    expect(screen.getByLabelText(/description/i)).toHaveValue('');
    expect(screen.getByLabelText(/quantity/i)).toHaveValue(null);
    expect(screen.getByLabelText(/unit price/i)).toHaveValue(null);
  });

  it('should handle Enter key to submit', async () => {
    const user = userEvent.setup();
    render(<LineItemForm onAdd={mockOnAdd} />);

    await user.type(screen.getByLabelText(/description/i), 'Test Service');
    await user.type(screen.getByLabelText(/quantity/i), '2');
    await user.type(screen.getByLabelText(/unit price/i), '50');

    const form = screen.getByLabelText(/description/i).closest('form');
    if (form) {
      fireEvent.submit(form);
    }

    await waitFor(() => {
      expect(mockOnAdd).toHaveBeenCalled();
    });
  });

  it('should handle Escape key to clear', async () => {
    const user = userEvent.setup();
    render(<LineItemForm onAdd={mockOnAdd} />);

    await user.type(screen.getByLabelText(/description/i), 'Test Service');

    fireEvent.keyDown(screen.getByLabelText(/description/i), { key: 'Escape' });

    await waitFor(() => {
      expect(screen.getByLabelText(/description/i)).toHaveValue('');
    });
  });

  it('should show error only after field is touched', async () => {
    const user = userEvent.setup();
    render(<LineItemForm onAdd={mockOnAdd} />);

    const descriptionInput = screen.getByLabelText(/description/i);

    // Error should not be visible initially
    expect(
      screen.queryByText(/description is required/i)
    ).not.toBeInTheDocument();

    // Focus and blur without entering value
    await user.click(descriptionInput);
    await user.tab();

    // Error should now be visible
    await waitFor(() => {
      expect(screen.getByText(/description is required/i)).toBeInTheDocument();
    });
  });

  it('should validate description length', async () => {
    const user = userEvent.setup();
    render(<LineItemForm onAdd={mockOnAdd} />);

    const longDescription = 'a'.repeat(501);
    await user.type(screen.getByLabelText(/description/i), longDescription);
    await user.click(screen.getByRole('button', { name: /add/i }));

    await waitFor(() => {
      expect(screen.getByText(/description too long/i)).toBeInTheDocument();
    });
  });

  it('should handle decimal quantities', async () => {
    const user = userEvent.setup();
    render(<LineItemForm onAdd={mockOnAdd} />);

    await user.type(screen.getByLabelText(/description/i), 'Test Service');
    await user.type(screen.getByLabelText(/quantity/i), '2.5');
    await user.type(screen.getByLabelText(/unit price/i), '50.50');

    await user.click(screen.getByRole('button', { name: /add/i }));

    await waitFor(() => {
      expect(mockOnAdd).toHaveBeenCalledWith({
        description: 'Test Service',
        quantity: '2.5',
        unit_price: '50.50',
      });
    });
  });
});

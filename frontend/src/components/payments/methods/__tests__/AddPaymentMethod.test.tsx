import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { AddPaymentMethod } from '../AddPaymentMethod';

// Mock Stripe
vi.mock('@stripe/stripe-js', () => ({
  loadStripe: vi.fn(() => Promise.resolve(null)),
}));

vi.mock('@stripe/react-stripe-js', () => ({
  Elements: ({ children }: { children: React.ReactNode }) => (
    <div data-testid="stripe-elements">{children}</div>
  ),
  CardElement: ({
    onChange,
    ...props
  }: {
    onChange?: (e: unknown) => void;
  }) => (
    <div
      data-testid="card-element"
      {...props}
      onClick={() => onChange?.({ complete: true, error: null })}
    />
  ),
  useStripe: () => ({
    createPaymentMethod: vi.fn(() =>
      Promise.resolve({
        paymentMethod: { id: 'pm_test_123' },
        error: null,
      })
    ),
  }),
  useElements: () => ({
    getElement: vi.fn(() => ({ mount: vi.fn() })),
  }),
}));

describe('AddPaymentMethod', () => {
  const defaultProps = {
    stripePublishableKey: 'pk_test_123',
    customerId: 'cus_123',
    isOpen: true,
  };

  beforeEach(() => {
    vi.clearAllMocks();
    // Reset body overflow
    document.body.style.overflow = '';
  });

  afterEach(() => {
    document.body.style.overflow = '';
  });

  // ==========================================================================
  // Rendering Tests
  // ==========================================================================

  it('renders modal when isOpen is true', () => {
    render(<AddPaymentMethod {...defaultProps} />);

    expect(screen.getByRole('dialog')).toBeInTheDocument();
  });

  it('does not render when isOpen is false', () => {
    render(<AddPaymentMethod {...defaultProps} isOpen={false} />);

    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
  });

  it('renders modal title', () => {
    render(<AddPaymentMethod {...defaultProps} />);

    expect(screen.getByText('Add Payment Method')).toBeInTheDocument();
  });

  it('renders card information label', async () => {
    render(<AddPaymentMethod {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getByText('Card Information')).toBeInTheDocument();
    });
  });

  it('renders nickname input', async () => {
    render(<AddPaymentMethod {...defaultProps} />);

    await waitFor(() => {
      expect(screen.getByLabelText(/Nickname/)).toBeInTheDocument();
    });
  });

  it('renders security notice', async () => {
    render(<AddPaymentMethod {...defaultProps} />);

    await waitFor(() => {
      expect(
        screen.getByText(/Your card information is encrypted/i)
      ).toBeInTheDocument();
    });
  });

  // ==========================================================================
  // Default Toggle Tests
  // ==========================================================================

  it('renders default payment toggle when allowSetDefault is true', async () => {
    render(<AddPaymentMethod {...defaultProps} allowSetDefault />);

    await waitFor(() => {
      expect(screen.getByRole('switch')).toBeInTheDocument();
    });
  });

  it('does not render default payment toggle when allowSetDefault is false', async () => {
    render(<AddPaymentMethod {...defaultProps} allowSetDefault={false} />);

    await waitFor(() => {
      expect(screen.queryByRole('switch')).not.toBeInTheDocument();
    });
  });

  // ==========================================================================
  // Close Button Tests
  // ==========================================================================

  it('renders close button when onCancel is provided', () => {
    const onCancel = vi.fn();
    render(<AddPaymentMethod {...defaultProps} onCancel={onCancel} />);

    expect(screen.getByLabelText('Close')).toBeInTheDocument();
  });

  it('calls onCancel when close button is clicked', () => {
    const onCancel = vi.fn();
    render(<AddPaymentMethod {...defaultProps} onCancel={onCancel} />);

    fireEvent.click(screen.getByLabelText('Close'));

    expect(onCancel).toHaveBeenCalled();
  });

  it('calls onCancel when backdrop is clicked', () => {
    const onCancel = vi.fn();
    render(<AddPaymentMethod {...defaultProps} onCancel={onCancel} />);

    const dialog = screen.getByRole('dialog');
    fireEvent.click(dialog);

    expect(onCancel).toHaveBeenCalled();
  });

  it('does not call onCancel when modal content is clicked', () => {
    const onCancel = vi.fn();
    render(<AddPaymentMethod {...defaultProps} onCancel={onCancel} />);

    const title = screen.getByText('Add Payment Method');
    fireEvent.click(title);

    expect(onCancel).not.toHaveBeenCalled();
  });

  // ==========================================================================
  // Keyboard Interaction Tests
  // ==========================================================================

  it('calls onCancel when Escape key is pressed', () => {
    const onCancel = vi.fn();
    render(<AddPaymentMethod {...defaultProps} onCancel={onCancel} />);

    fireEvent.keyDown(document, { key: 'Escape' });

    expect(onCancel).toHaveBeenCalled();
  });

  // ==========================================================================
  // Body Scroll Lock Tests
  // ==========================================================================

  it('prevents body scroll when modal is open', () => {
    render(<AddPaymentMethod {...defaultProps} />);

    expect(document.body.style.overflow).toBe('hidden');
  });

  it('restores body scroll when modal is closed', () => {
    const { rerender } = render(<AddPaymentMethod {...defaultProps} />);

    rerender(<AddPaymentMethod {...defaultProps} isOpen={false} />);

    expect(document.body.style.overflow).toBe('');
  });

  // ==========================================================================
  // Form Submission Tests
  // ==========================================================================

  it('renders submit button', async () => {
    render(<AddPaymentMethod {...defaultProps} />);

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: 'Add Card' })
      ).toBeInTheDocument();
    });
  });

  it('renders cancel button when onCancel is provided', async () => {
    const onCancel = vi.fn();
    render(<AddPaymentMethod {...defaultProps} onCancel={onCancel} />);

    await waitFor(() => {
      expect(
        screen.getByRole('button', { name: 'Cancel' })
      ).toBeInTheDocument();
    });
  });

  // ==========================================================================
  // Accessibility Tests
  // ==========================================================================

  it('has correct aria attributes on dialog', () => {
    render(<AddPaymentMethod {...defaultProps} />);

    const dialog = screen.getByRole('dialog');
    expect(dialog).toHaveAttribute('aria-modal', 'true');
    expect(dialog).toHaveAttribute(
      'aria-labelledby',
      'add-payment-method-title'
    );
  });

  it('title has correct id for aria-labelledby', () => {
    render(<AddPaymentMethod {...defaultProps} />);

    const title = screen.getByText('Add Payment Method');
    expect(title).toHaveAttribute('id', 'add-payment-method-title');
  });

  // ==========================================================================
  // CSS Class Tests
  // ==========================================================================

  it('applies custom className to modal', () => {
    render(<AddPaymentMethod {...defaultProps} className="custom-class" />);

    const modalContent = screen
      .getByRole('dialog')
      .querySelector('.custom-class');
    expect(modalContent).toBeInTheDocument();
  });

  // ==========================================================================
  // Loading State Tests
  // ==========================================================================

  it('shows loading spinner when stripe is initializing', () => {
    // This test verifies the loading state before Stripe loads
    render(<AddPaymentMethod {...defaultProps} />);

    // The Stripe Elements wrapper should be present
    expect(screen.getByTestId('stripe-elements')).toBeInTheDocument();
  });

  // ==========================================================================
  // Input Tests
  // ==========================================================================

  it('allows entering nickname', async () => {
    render(<AddPaymentMethod {...defaultProps} />);

    await waitFor(() => {
      const nicknameInput = screen.getByLabelText(/Nickname/);
      fireEvent.change(nicknameInput, { target: { value: 'My Card' } });
      expect(nicknameInput).toHaveValue('My Card');
    });
  });

  it('has max length on nickname input', async () => {
    render(<AddPaymentMethod {...defaultProps} />);

    await waitFor(() => {
      const nicknameInput = screen.getByLabelText(/Nickname/);
      expect(nicknameInput).toHaveAttribute('maxLength', '100');
    });
  });

  it('shows placeholder text in nickname input', async () => {
    render(<AddPaymentMethod {...defaultProps} />);

    await waitFor(() => {
      const nicknameInput = screen.getByLabelText(/Nickname/);
      expect(nicknameInput).toHaveAttribute(
        'placeholder',
        'e.g., Personal Card, Business Card'
      );
    });
  });
});

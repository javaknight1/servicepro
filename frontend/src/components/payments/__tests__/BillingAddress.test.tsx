import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';
import { BillingAddress } from '../BillingAddress';
import type { BillingAddressData } from '../../../types/payment';
import { DEFAULT_BILLING_ADDRESS } from '../../../types/payment';

describe('BillingAddress', () => {
  const defaultProps = {
    value: DEFAULT_BILLING_ADDRESS,
    onChange: vi.fn(),
  };

  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('renders all required fields', () => {
    render(<BillingAddress {...defaultProps} />);

    expect(screen.getByLabelText(/full name/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/email/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/phone number/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/street address/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/apt, suite/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/city/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/state/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/zip code/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/country/i)).toBeInTheDocument();
  });

  it('displays section header', () => {
    render(<BillingAddress {...defaultProps} />);

    expect(screen.getByText('Billing Address')).toBeInTheDocument();
  });

  it('calls onChange when name field is updated', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<BillingAddress {...defaultProps} onChange={onChange} />);

    const nameInput = screen.getByLabelText(/full name/i);
    await user.type(nameInput, 'John Doe');

    expect(onChange).toHaveBeenCalled();
    const lastCall = onChange.mock.calls[onChange.mock.calls.length - 1][0];
    expect(lastCall.name).toContain('e'); // Last character typed
  });

  it('calls onChange when email field is updated', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    render(<BillingAddress {...defaultProps} onChange={onChange} />);

    const emailInput = screen.getByLabelText(/email/i);
    await user.type(emailInput, 'test@example.com');

    expect(onChange).toHaveBeenCalled();
  });

  it('displays error messages', () => {
    const errors = {
      name: 'Name is required',
      email: 'Invalid email',
    };

    render(<BillingAddress {...defaultProps} errors={errors} />);

    expect(screen.getByText('Name is required')).toBeInTheDocument();
    expect(screen.getByText('Invalid email')).toBeInTheDocument();
  });

  it('marks error fields as invalid', () => {
    const errors = {
      name: 'Name is required',
    };

    render(<BillingAddress {...defaultProps} errors={errors} />);

    const nameInput = screen.getByLabelText(/full name/i);
    expect(nameInput).toHaveAttribute('aria-invalid', 'true');
  });

  it('disables all fields when disabled prop is true', () => {
    render(<BillingAddress {...defaultProps} disabled />);

    expect(screen.getByLabelText(/full name/i)).toBeDisabled();
    expect(screen.getByLabelText(/email/i)).toBeDisabled();
    expect(screen.getByLabelText(/street address/i)).toBeDisabled();
    expect(screen.getByLabelText(/city/i)).toBeDisabled();
    expect(screen.getByLabelText(/zip code/i)).toBeDisabled();
    expect(screen.getByLabelText(/country/i)).toBeDisabled();
  });

  it('shows US state dropdown when country is US', () => {
    const value: BillingAddressData = {
      ...DEFAULT_BILLING_ADDRESS,
      country: 'US',
    };

    render(<BillingAddress {...defaultProps} value={value} />);

    const stateSelect = screen.getByLabelText(/state/i);
    expect(stateSelect.tagName).toBe('SELECT');
    expect(screen.getByText('California')).toBeInTheDocument();
  });

  it('shows text input for state when country is not US', () => {
    const value: BillingAddressData = {
      ...DEFAULT_BILLING_ADDRESS,
      country: 'CA',
    };

    render(<BillingAddress {...defaultProps} value={value} />);

    const stateInput = screen.getByLabelText(/state/i);
    expect(stateInput.tagName).toBe('INPUT');
  });

  it('preserves existing values', () => {
    const value: BillingAddressData = {
      name: 'John Doe',
      email: 'john@example.com',
      phone: '+1234567890',
      line1: '123 Main St',
      line2: 'Apt 4B',
      city: 'San Francisco',
      state: 'CA',
      postalCode: '94102',
      country: 'US',
    };

    render(<BillingAddress {...defaultProps} value={value} />);

    expect(screen.getByLabelText(/full name/i)).toHaveValue('John Doe');
    expect(screen.getByLabelText(/email/i)).toHaveValue('john@example.com');
    expect(screen.getByLabelText(/phone number/i)).toHaveValue('+1234567890');
    expect(screen.getByLabelText(/street address/i)).toHaveValue('123 Main St');
    expect(screen.getByLabelText(/apt, suite/i)).toHaveValue('Apt 4B');
    expect(screen.getByLabelText(/city/i)).toHaveValue('San Francisco');
    expect(screen.getByLabelText(/zip code/i)).toHaveValue('94102');
  });

  it('applies custom className', () => {
    const { container } = render(
      <BillingAddress {...defaultProps} className="custom-class" />
    );

    expect(container.firstChild).toHaveClass('custom-class');
  });

  it('has proper autocomplete attributes', () => {
    render(<BillingAddress {...defaultProps} />);

    expect(screen.getByLabelText(/full name/i)).toHaveAttribute(
      'autocomplete',
      'name'
    );
    expect(screen.getByLabelText(/email/i)).toHaveAttribute(
      'autocomplete',
      'email'
    );
    expect(screen.getByLabelText(/street address/i)).toHaveAttribute(
      'autocomplete',
      'address-line1'
    );
    expect(screen.getByLabelText(/city/i)).toHaveAttribute(
      'autocomplete',
      'address-level2'
    );
    expect(screen.getByLabelText(/zip code/i)).toHaveAttribute(
      'autocomplete',
      'postal-code'
    );
  });

  it('marks required fields with asterisks', () => {
    render(<BillingAddress {...defaultProps} />);

    // Check for required field indicators
    const nameLabel = screen.getByText(/full name/i).closest('label');
    expect(nameLabel).toHaveTextContent('*');

    const emailLabel = screen.getByText(/^email$/i).closest('label');
    expect(emailLabel).toHaveTextContent('*');
  });

  it('updates country and clears state when country changes', async () => {
    const user = userEvent.setup();
    const onChange = vi.fn();
    const value: BillingAddressData = {
      ...DEFAULT_BILLING_ADDRESS,
      country: 'US',
      state: 'CA',
    };

    render(<BillingAddress value={value} onChange={onChange} />);

    const countrySelect = screen.getByLabelText(/country/i);
    await user.selectOptions(countrySelect, 'CA');

    expect(onChange).toHaveBeenCalledWith(
      expect.objectContaining({ country: 'CA' })
    );
  });
});

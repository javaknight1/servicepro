import { render, screen, fireEvent } from '@testing-library/react';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { DefaultPaymentToggle } from '../DefaultPaymentToggle';

describe('DefaultPaymentToggle', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  // ==========================================================================
  // Rendering Tests
  // ==========================================================================

  it('renders toggle button', () => {
    render(<DefaultPaymentToggle isDefault={false} onChange={vi.fn()} />);

    expect(screen.getByRole('switch')).toBeInTheDocument();
  });

  it('renders label text when not default', () => {
    render(<DefaultPaymentToggle isDefault={false} onChange={vi.fn()} />);

    expect(screen.getByText('Set as default')).toBeInTheDocument();
  });

  it('renders label text when default', () => {
    render(<DefaultPaymentToggle isDefault={true} onChange={vi.fn()} />);

    expect(screen.getByText('Default payment method')).toBeInTheDocument();
  });

  // ==========================================================================
  // State Tests
  // ==========================================================================

  it('has correct aria-checked when isDefault is true', () => {
    render(<DefaultPaymentToggle isDefault={true} onChange={vi.fn()} />);

    expect(screen.getByRole('switch')).toHaveAttribute('aria-checked', 'true');
  });

  it('has correct aria-checked when isDefault is false', () => {
    render(<DefaultPaymentToggle isDefault={false} onChange={vi.fn()} />);

    expect(screen.getByRole('switch')).toHaveAttribute('aria-checked', 'false');
  });

  // ==========================================================================
  // Interaction Tests
  // ==========================================================================

  it('calls onChange with true when clicked and currently false', () => {
    const onChange = vi.fn();
    render(<DefaultPaymentToggle isDefault={false} onChange={onChange} />);

    fireEvent.click(screen.getByRole('switch'));

    expect(onChange).toHaveBeenCalledWith(true);
  });

  it('calls onChange with false when clicked and currently true', () => {
    const onChange = vi.fn();
    render(<DefaultPaymentToggle isDefault={true} onChange={onChange} />);

    fireEvent.click(screen.getByRole('switch'));

    expect(onChange).toHaveBeenCalledWith(false);
  });

  it('calls onChange when label is clicked', () => {
    const onChange = vi.fn();
    render(<DefaultPaymentToggle isDefault={false} onChange={onChange} />);

    fireEvent.click(screen.getByText('Set as default'));

    expect(onChange).toHaveBeenCalledWith(true);
  });

  it('calls onChange when Enter key is pressed', () => {
    const onChange = vi.fn();
    render(<DefaultPaymentToggle isDefault={false} onChange={onChange} />);

    fireEvent.keyDown(screen.getByRole('switch'), { key: 'Enter' });

    expect(onChange).toHaveBeenCalledWith(true);
  });

  it('calls onChange when Space key is pressed', () => {
    const onChange = vi.fn();
    render(<DefaultPaymentToggle isDefault={false} onChange={onChange} />);

    fireEvent.keyDown(screen.getByRole('switch'), { key: ' ' });

    expect(onChange).toHaveBeenCalledWith(true);
  });

  // ==========================================================================
  // Disabled State Tests
  // ==========================================================================

  it('does not call onChange when disabled', () => {
    const onChange = vi.fn();
    render(
      <DefaultPaymentToggle isDefault={false} onChange={onChange} disabled />
    );

    fireEvent.click(screen.getByRole('switch'));

    expect(onChange).not.toHaveBeenCalled();
  });

  it('has disabled attribute when disabled', () => {
    render(
      <DefaultPaymentToggle isDefault={false} onChange={vi.fn()} disabled />
    );

    expect(screen.getByRole('switch')).toBeDisabled();
  });

  // ==========================================================================
  // Size Variant Tests
  // ==========================================================================

  it('renders small size correctly', () => {
    render(
      <DefaultPaymentToggle isDefault={false} onChange={vi.fn()} size="sm" />
    );

    const toggle = screen.getByRole('switch');
    expect(toggle).toHaveClass('h-5', 'w-9');
  });

  it('renders medium size correctly', () => {
    render(
      <DefaultPaymentToggle isDefault={false} onChange={vi.fn()} size="md" />
    );

    const toggle = screen.getByRole('switch');
    expect(toggle).toHaveClass('h-6', 'w-11');
  });

  it('renders large size correctly', () => {
    render(
      <DefaultPaymentToggle isDefault={false} onChange={vi.fn()} size="lg" />
    );

    const toggle = screen.getByRole('switch');
    expect(toggle).toHaveClass('h-7', 'w-14');
  });

  // ==========================================================================
  // Accessibility Tests
  // ==========================================================================

  it('has correct aria-label when not default', () => {
    render(<DefaultPaymentToggle isDefault={false} onChange={vi.fn()} />);

    expect(screen.getByRole('switch')).toHaveAttribute(
      'aria-label',
      'Set as default payment method'
    );
  });

  it('has correct aria-label when default', () => {
    render(<DefaultPaymentToggle isDefault={true} onChange={vi.fn()} />);

    expect(screen.getByRole('switch')).toHaveAttribute(
      'aria-label',
      'Remove as default payment method'
    );
  });

  it('associates label with toggle via htmlFor', () => {
    render(<DefaultPaymentToggle isDefault={false} onChange={vi.fn()} />);

    const toggle = screen.getByRole('switch');
    const label = screen.getByText('Set as default');

    expect(label).toHaveAttribute('for', toggle.id);
  });

  // ==========================================================================
  // CSS Class Tests
  // ==========================================================================

  it('applies custom className', () => {
    render(
      <DefaultPaymentToggle
        isDefault={false}
        onChange={vi.fn()}
        className="custom-class"
      />
    );

    const container = screen.getByRole('switch').parentElement;
    expect(container).toHaveClass('custom-class');
  });

  // ==========================================================================
  // Visual State Tests
  // ==========================================================================

  it('has correct background when on', () => {
    render(<DefaultPaymentToggle isDefault={true} onChange={vi.fn()} />);

    const toggle = screen.getByRole('switch');
    expect(toggle).toHaveClass('bg-primary-600');
  });

  it('has correct background when off', () => {
    render(<DefaultPaymentToggle isDefault={false} onChange={vi.fn()} />);

    const toggle = screen.getByRole('switch');
    expect(toggle).toHaveClass('bg-neutral-200');
  });
});

import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Input } from './Input';

describe('Input Component', () => {
  // ==========================================================================
  // Rendering Tests
  // ==========================================================================
  describe('Rendering', () => {
    it('renders input element', () => {
      render(<Input />);
      expect(screen.getByRole('textbox')).toBeInTheDocument();
    });

    it('renders with label', () => {
      render(<Input label="Email" />);

      expect(screen.getByLabelText('Email')).toBeInTheDocument();
      expect(screen.getByText('Email')).toBeInTheDocument();
    });

    it('generates id from label', () => {
      render(<Input label="First Name" />);
      expect(screen.getByRole('textbox')).toHaveAttribute('id', 'first-name');
    });

    it('uses provided id over generated one', () => {
      render(<Input label="Email" id="custom-id" />);
      expect(screen.getByRole('textbox')).toHaveAttribute('id', 'custom-id');
    });

    it('renders with forwardRef', () => {
      const ref = vi.fn();
      render(<Input ref={ref} />);
      expect(ref).toHaveBeenCalled();
    });
  });

  // ==========================================================================
  // Error State Tests
  // ==========================================================================
  describe('Error State', () => {
    it('displays error message', () => {
      render(<Input error="This field is required" />);
      expect(screen.getByText('This field is required')).toBeInTheDocument();
    });

    it('applies error styles to input', () => {
      render(<Input error="Error" />);
      const input = screen.getByRole('textbox');

      expect(input).toHaveClass('border-error-500');
      expect(input).toHaveClass('focus:ring-error-500');
    });

    it('error message has correct styling', () => {
      render(<Input error="Error message" />);
      const errorElement = screen.getByText('Error message');

      expect(errorElement).toHaveClass('text-error-600');
      expect(errorElement).toHaveClass('text-sm');
    });

    it('does not show helper text when error is present', () => {
      render(<Input error="Error" helperText="Helper" />);

      expect(screen.getByText('Error')).toBeInTheDocument();
      expect(screen.queryByText('Helper')).not.toBeInTheDocument();
    });
  });

  // ==========================================================================
  // Helper Text Tests
  // ==========================================================================
  describe('Helper Text', () => {
    it('displays helper text', () => {
      render(<Input helperText="Enter your email address" />);
      expect(screen.getByText('Enter your email address')).toBeInTheDocument();
    });

    it('helper text has correct styling', () => {
      render(<Input helperText="Helper text" />);
      const helperElement = screen.getByText('Helper text');

      expect(helperElement).toHaveClass('text-neutral-500');
      expect(helperElement).toHaveClass('text-sm');
    });

    it('shows helper text when no error', () => {
      render(<Input helperText="Helper" />);
      expect(screen.getByText('Helper')).toBeInTheDocument();
    });
  });

  // ==========================================================================
  // Full Width Tests
  // ==========================================================================
  describe('Full Width', () => {
    it('applies full width to container', () => {
      const { container } = render(<Input fullWidth label="Test" />);
      expect(container.firstChild).toHaveClass('w-full');
    });

    it('applies full width to input', () => {
      render(<Input fullWidth />);
      expect(screen.getByRole('textbox')).toHaveClass('w-full');
    });

    it('does not apply full width by default', () => {
      render(<Input />);
      expect(screen.getByRole('textbox')).not.toHaveClass('w-full');
    });
  });

  // ==========================================================================
  // User Interaction Tests
  // ==========================================================================
  describe('User Interactions', () => {
    it('allows user to type', async () => {
      const user = userEvent.setup();
      render(<Input />);

      const input = screen.getByRole('textbox');
      await user.type(input, 'Hello World');

      expect(input).toHaveValue('Hello World');
    });

    it('calls onChange handler', async () => {
      const handleChange = vi.fn();
      const user = userEvent.setup();
      render(<Input onChange={handleChange} />);

      await user.type(screen.getByRole('textbox'), 'a');
      expect(handleChange).toHaveBeenCalled();
    });

    it('calls onBlur handler', async () => {
      const handleBlur = vi.fn();
      const user = userEvent.setup();
      render(<Input onBlur={handleBlur} />);

      const input = screen.getByRole('textbox');
      await user.click(input);
      await user.tab();

      expect(handleBlur).toHaveBeenCalled();
    });

    it('calls onFocus handler', async () => {
      const handleFocus = vi.fn();
      const user = userEvent.setup();
      render(<Input onFocus={handleFocus} />);

      await user.click(screen.getByRole('textbox'));
      expect(handleFocus).toHaveBeenCalled();
    });

    it('can be focused with keyboard', async () => {
      const user = userEvent.setup();
      render(<Input label="Test" />);

      await user.tab();
      expect(screen.getByRole('textbox')).toHaveFocus();
    });
  });

  // ==========================================================================
  // Input Type Tests - Table Driven
  // ==========================================================================
  describe('Input Types', () => {
    it('renders text input type', () => {
      render(<Input type="text" />);
      const input = screen.getByRole('textbox');
      expect(input).toHaveAttribute('type', 'text');
    });

    it('renders email input type', () => {
      render(<Input type="email" />);
      const input = screen.getByRole('textbox');
      expect(input).toHaveAttribute('type', 'email');
    });

    it('renders password input type', () => {
      render(<Input type="password" data-testid="password-input" />);
      const input = screen.getByTestId('password-input');
      expect(input).toHaveAttribute('type', 'password');
    });

    it('renders tel input type', () => {
      render(<Input type="tel" />);
      const input = screen.getByRole('textbox');
      expect(input).toHaveAttribute('type', 'tel');
    });

    it('renders url input type', () => {
      render(<Input type="url" />);
      const input = screen.getByRole('textbox');
      expect(input).toHaveAttribute('type', 'url');
    });

    it('renders search input type', () => {
      render(<Input type="search" />);
      const input = screen.getByRole('searchbox');
      expect(input).toHaveAttribute('type', 'search');
    });
  });

  // ==========================================================================
  // Disabled State Tests
  // ==========================================================================
  describe('Disabled State', () => {
    it('applies disabled attribute', () => {
      render(<Input disabled />);
      expect(screen.getByRole('textbox')).toBeDisabled();
    });

    it('applies disabled styles', () => {
      render(<Input disabled />);
      const input = screen.getByRole('textbox');

      expect(input).toHaveClass('disabled:bg-neutral-100');
      expect(input).toHaveClass('disabled:cursor-not-allowed');
    });

    it('prevents user interaction when disabled', async () => {
      const handleChange = vi.fn();
      const user = userEvent.setup();
      render(<Input disabled onChange={handleChange} />);

      // Attempting to type in disabled input
      await user.click(screen.getByRole('textbox'));
      expect(handleChange).not.toHaveBeenCalled();
    });
  });

  // ==========================================================================
  // Placeholder Tests
  // ==========================================================================
  describe('Placeholder', () => {
    it('displays placeholder text', () => {
      render(<Input placeholder="Enter your name" />);
      expect(
        screen.getByPlaceholderText('Enter your name')
      ).toBeInTheDocument();
    });

    it('placeholder has correct styling', () => {
      render(<Input placeholder="Test" />);
      expect(screen.getByRole('textbox')).toHaveClass(
        'placeholder:text-neutral-400'
      );
    });
  });

  // ==========================================================================
  // Custom Props Tests
  // ==========================================================================
  describe('Custom Props', () => {
    it('applies custom className to input', () => {
      render(<Input className="custom-class" />);
      expect(screen.getByRole('textbox')).toHaveClass('custom-class');
    });

    it('passes through additional HTML attributes', () => {
      render(
        <Input
          name="email"
          autoComplete="email"
          maxLength={50}
          data-testid="email-input"
        />
      );

      const input = screen.getByRole('textbox');
      expect(input).toHaveAttribute('name', 'email');
      expect(input).toHaveAttribute('autocomplete', 'email');
      expect(input).toHaveAttribute('maxlength', '50');
      expect(input).toHaveAttribute('data-testid', 'email-input');
    });

    it('supports required attribute', () => {
      render(<Input required />);
      expect(screen.getByRole('textbox')).toBeRequired();
    });

    it('supports readonly attribute', () => {
      render(<Input readOnly value="Read only" />);
      expect(screen.getByRole('textbox')).toHaveAttribute('readonly');
    });
  });

  // ==========================================================================
  // Controlled vs Uncontrolled Tests
  // ==========================================================================
  describe('Controlled vs Uncontrolled', () => {
    it('works as controlled input', async () => {
      const handleChange = vi.fn();
      const { rerender } = render(
        <Input value="initial" onChange={handleChange} />
      );

      expect(screen.getByRole('textbox')).toHaveValue('initial');

      rerender(<Input value="updated" onChange={handleChange} />);
      expect(screen.getByRole('textbox')).toHaveValue('updated');
    });

    it('works as uncontrolled input with defaultValue', () => {
      render(<Input defaultValue="default" />);
      expect(screen.getByRole('textbox')).toHaveValue('default');
    });
  });

  // ==========================================================================
  // Accessibility Tests
  // ==========================================================================
  describe('Accessibility', () => {
    it('associates label with input using htmlFor', () => {
      render(<Input label="Username" />);
      const label = screen.getByText('Username');
      const input = screen.getByRole('textbox');

      expect(label).toHaveAttribute('for', 'username');
      expect(input).toHaveAttribute('id', 'username');
    });

    it('supports aria-describedby for error messages', () => {
      render(<Input aria-describedby="error-msg" error="Error" />);
      expect(screen.getByRole('textbox')).toHaveAttribute(
        'aria-describedby',
        'error-msg'
      );
    });

    it('supports aria-invalid', () => {
      render(<Input aria-invalid="true" />);
      expect(screen.getByRole('textbox')).toHaveAttribute(
        'aria-invalid',
        'true'
      );
    });

    it('supports aria-required', () => {
      render(<Input aria-required="true" />);
      expect(screen.getByRole('textbox')).toHaveAttribute(
        'aria-required',
        'true'
      );
    });
  });

  // ==========================================================================
  // Focus Styles Tests
  // ==========================================================================
  describe('Focus Styles', () => {
    it('has default focus styles', () => {
      render(<Input />);
      const input = screen.getByRole('textbox');

      expect(input).toHaveClass('focus:outline-none');
      expect(input).toHaveClass('focus:ring-2');
      expect(input).toHaveClass('focus:ring-primary-500');
    });

    it('has error focus styles when error present', () => {
      render(<Input error="Error" />);
      const input = screen.getByRole('textbox');

      expect(input).toHaveClass('focus:ring-error-500');
      expect(input).toHaveClass('focus:border-error-500');
    });
  });
});

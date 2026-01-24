import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Button } from './Button';

describe('Button Component', () => {
  // ==========================================================================
  // Rendering Tests
  // ==========================================================================
  describe('Rendering', () => {
    it('renders children correctly', () => {
      render(<Button>Click me</Button>);
      expect(screen.getByRole('button')).toHaveTextContent('Click me');
    });

    it('renders with default props', () => {
      render(<Button>Default Button</Button>);
      const button = screen.getByRole('button');

      expect(button).toBeInTheDocument();
      expect(button).not.toBeDisabled();
      expect(button).toHaveClass('bg-primary-600'); // primary variant
      expect(button).toHaveClass('px-4'); // md size
    });

    it('renders with forwardRef', () => {
      const ref = vi.fn();
      render(<Button ref={ref}>Button</Button>);
      expect(ref).toHaveBeenCalled();
    });
  });

  // ==========================================================================
  // Variant Tests - Table Driven
  // ==========================================================================
  describe('Variants', () => {
    const variants = [
      { variant: 'primary' as const, expectedClass: 'bg-primary-600' },
      { variant: 'secondary' as const, expectedClass: 'bg-secondary-600' },
      { variant: 'outline' as const, expectedClass: 'border-primary-600' },
      { variant: 'ghost' as const, expectedClass: 'text-neutral-700' },
      { variant: 'danger' as const, expectedClass: 'bg-error-600' },
    ];

    variants.forEach(({ variant, expectedClass }) => {
      it(`renders ${variant} variant with correct styles`, () => {
        render(<Button variant={variant}>{variant} Button</Button>);
        expect(screen.getByRole('button')).toHaveClass(expectedClass);
      });
    });
  });

  // ==========================================================================
  // Size Tests - Table Driven
  // ==========================================================================
  describe('Sizes', () => {
    const sizes = [
      { size: 'sm' as const, expectedClass: 'px-3 py-1.5 text-sm' },
      { size: 'md' as const, expectedClass: 'px-4 py-2 text-base' },
      { size: 'lg' as const, expectedClass: 'px-6 py-3 text-lg' },
    ];

    sizes.forEach(({ size, expectedClass }) => {
      it(`renders ${size} size with correct styles`, () => {
        render(<Button size={size}>{size} Button</Button>);
        const button = screen.getByRole('button');
        expectedClass.split(' ').forEach((cls) => {
          expect(button).toHaveClass(cls);
        });
      });
    });
  });

  // ==========================================================================
  // Loading State Tests
  // ==========================================================================
  describe('Loading State', () => {
    it('shows loading spinner when isLoading is true', () => {
      render(<Button isLoading>Loading</Button>);
      const button = screen.getByRole('button');

      // Check for spinner SVG
      expect(button.querySelector('svg')).toBeInTheDocument();
      expect(button.querySelector('svg')).toHaveClass('animate-spin');
    });

    it('disables button when loading', () => {
      render(<Button isLoading>Loading</Button>);
      expect(screen.getByRole('button')).toBeDisabled();
    });

    it('does not show spinner when not loading', () => {
      render(<Button>Not Loading</Button>);
      expect(
        screen.getByRole('button').querySelector('svg')
      ).not.toBeInTheDocument();
    });

    it('still shows children when loading', () => {
      render(<Button isLoading>Loading Text</Button>);
      expect(screen.getByRole('button')).toHaveTextContent('Loading Text');
    });
  });

  // ==========================================================================
  // Disabled State Tests
  // ==========================================================================
  describe('Disabled State', () => {
    it('applies disabled styles when disabled', () => {
      render(<Button disabled>Disabled</Button>);
      const button = screen.getByRole('button');

      expect(button).toBeDisabled();
      expect(button).toHaveClass('disabled:opacity-50');
      expect(button).toHaveClass('disabled:cursor-not-allowed');
    });

    it('does not trigger onClick when disabled', async () => {
      const handleClick = vi.fn();
      render(
        <Button disabled onClick={handleClick}>
          Disabled
        </Button>
      );

      await userEvent.click(screen.getByRole('button'));
      expect(handleClick).not.toHaveBeenCalled();
    });
  });

  // ==========================================================================
  // Full Width Tests
  // ==========================================================================
  describe('Full Width', () => {
    it('applies full width styles when fullWidth is true', () => {
      render(<Button fullWidth>Full Width</Button>);
      expect(screen.getByRole('button')).toHaveClass('w-full');
    });

    it('does not apply full width by default', () => {
      render(<Button>Normal Width</Button>);
      expect(screen.getByRole('button')).not.toHaveClass('w-full');
    });
  });

  // ==========================================================================
  // User Interaction Tests
  // ==========================================================================
  describe('User Interactions', () => {
    it('calls onClick when clicked', async () => {
      const handleClick = vi.fn();
      render(<Button onClick={handleClick}>Click me</Button>);

      await userEvent.click(screen.getByRole('button'));
      expect(handleClick).toHaveBeenCalledTimes(1);
    });

    it('calls onClick with event object', async () => {
      const handleClick = vi.fn();
      render(<Button onClick={handleClick}>Click me</Button>);

      await userEvent.click(screen.getByRole('button'));
      expect(handleClick).toHaveBeenCalledWith(expect.any(Object));
    });

    it('can be focused with keyboard', async () => {
      render(<Button>Focusable</Button>);
      const button = screen.getByRole('button');

      await userEvent.tab();
      expect(button).toHaveFocus();
    });

    it('triggers onClick on Enter key', async () => {
      const handleClick = vi.fn();
      render(<Button onClick={handleClick}>Press Enter</Button>);

      const button = screen.getByRole('button');
      button.focus();
      await userEvent.keyboard('{Enter}');

      expect(handleClick).toHaveBeenCalled();
    });

    it('triggers onClick on Space key', async () => {
      const handleClick = vi.fn();
      render(<Button onClick={handleClick}>Press Space</Button>);

      const button = screen.getByRole('button');
      button.focus();
      await userEvent.keyboard(' ');

      expect(handleClick).toHaveBeenCalled();
    });
  });

  // ==========================================================================
  // Custom Class and Props Tests
  // ==========================================================================
  describe('Custom Props', () => {
    it('applies custom className', () => {
      render(<Button className="custom-class">Custom</Button>);
      expect(screen.getByRole('button')).toHaveClass('custom-class');
    });

    it('passes through additional HTML attributes', () => {
      render(
        <Button type="submit" data-testid="submit-btn" aria-label="Submit form">
          Submit
        </Button>
      );

      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('type', 'submit');
      expect(button).toHaveAttribute('data-testid', 'submit-btn');
      expect(button).toHaveAttribute('aria-label', 'Submit form');
    });

    it('supports form attribute for form association', () => {
      render(<Button form="my-form">Submit</Button>);
      expect(screen.getByRole('button')).toHaveAttribute('form', 'my-form');
    });
  });

  // ==========================================================================
  // Accessibility Tests
  // ==========================================================================
  describe('Accessibility', () => {
    it('has correct role', () => {
      render(<Button>Accessible</Button>);
      expect(screen.getByRole('button')).toBeInTheDocument();
    });

    it('supports aria-disabled', () => {
      render(<Button aria-disabled="true">Aria Disabled</Button>);
      expect(screen.getByRole('button')).toHaveAttribute(
        'aria-disabled',
        'true'
      );
    });

    it('supports aria-pressed for toggle buttons', () => {
      render(<Button aria-pressed="true">Toggle</Button>);
      expect(screen.getByRole('button')).toHaveAttribute(
        'aria-pressed',
        'true'
      );
    });

    it('supports aria-expanded', () => {
      render(<Button aria-expanded="false">Expand</Button>);
      expect(screen.getByRole('button')).toHaveAttribute(
        'aria-expanded',
        'false'
      );
    });
  });

  // ==========================================================================
  // Combination Tests
  // ==========================================================================
  describe('Prop Combinations', () => {
    it('handles multiple props together', () => {
      render(
        <Button variant="danger" size="lg" fullWidth className="custom-class">
          Combined
        </Button>
      );

      const button = screen.getByRole('button');
      expect(button).toHaveClass('bg-error-600'); // danger
      expect(button).toHaveClass('px-6'); // lg
      expect(button).toHaveClass('w-full'); // fullWidth
      expect(button).toHaveClass('custom-class'); // custom
    });

    it('disables both when disabled and loading', () => {
      render(
        <Button disabled isLoading>
          Both Disabled
        </Button>
      );

      expect(screen.getByRole('button')).toBeDisabled();
    });
  });
});

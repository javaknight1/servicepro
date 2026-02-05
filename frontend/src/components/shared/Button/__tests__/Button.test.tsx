import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';
import { Button } from '../Button';

describe('Button', () => {
  describe('rendering', () => {
    it('should render children', () => {
      render(<Button>Click me</Button>);

      expect(
        screen.getByRole('button', { name: 'Click me' })
      ).toBeInTheDocument();
    });

    it('should apply base styles', () => {
      const { container } = render(<Button>Button</Button>);

      expect(container.firstChild).toHaveClass(
        'inline-flex',
        'items-center',
        'justify-center',
        'font-medium',
        'rounded-lg'
      );
    });
  });

  describe('variants', () => {
    it('should apply primary variant by default', () => {
      const { container } = render(<Button>Primary</Button>);

      expect(container.firstChild).toHaveClass('bg-primary-600', 'text-white');
    });

    it('should apply secondary variant', () => {
      const { container } = render(
        <Button variant="secondary">Secondary</Button>
      );

      expect(container.firstChild).toHaveClass(
        'bg-secondary-600',
        'text-white'
      );
    });

    it('should apply outline variant', () => {
      const { container } = render(<Button variant="outline">Outline</Button>);

      expect(container.firstChild).toHaveClass(
        'border-2',
        'border-primary-600',
        'text-primary-600'
      );
    });

    it('should apply ghost variant', () => {
      const { container } = render(<Button variant="ghost">Ghost</Button>);

      expect(container.firstChild).toHaveClass('text-neutral-700');
    });

    it('should apply danger variant', () => {
      const { container } = render(<Button variant="danger">Danger</Button>);

      expect(container.firstChild).toHaveClass('bg-error-600', 'text-white');
    });
  });

  describe('sizes', () => {
    it('should apply md size by default', () => {
      const { container } = render(<Button>Medium</Button>);

      expect(container.firstChild).toHaveClass('px-4', 'py-2', 'text-base');
    });

    it('should apply sm size', () => {
      const { container } = render(<Button size="sm">Small</Button>);

      expect(container.firstChild).toHaveClass('px-3', 'py-1.5', 'text-sm');
    });

    it('should apply lg size', () => {
      const { container } = render(<Button size="lg">Large</Button>);

      expect(container.firstChild).toHaveClass('px-6', 'py-3', 'text-lg');
    });
  });

  describe('fullWidth', () => {
    it('should not be full width by default', () => {
      const { container } = render(<Button>Button</Button>);

      expect(container.firstChild).not.toHaveClass('w-full');
    });

    it('should apply full width when prop is true', () => {
      const { container } = render(<Button fullWidth>Full Width</Button>);

      expect(container.firstChild).toHaveClass('w-full');
    });
  });

  describe('loading state', () => {
    it('should not show loading spinner by default', () => {
      const { container } = render(<Button>Button</Button>);

      expect(container.querySelector('svg')).not.toBeInTheDocument();
    });

    it('should show loading spinner when isLoading is true', () => {
      const { container } = render(<Button isLoading>Loading</Button>);

      expect(container.querySelector('svg')).toBeInTheDocument();
      expect(container.querySelector('svg')).toHaveClass('animate-spin');
    });

    it('should disable button when isLoading is true', () => {
      render(<Button isLoading>Loading</Button>);

      expect(screen.getByRole('button')).toBeDisabled();
    });

    it('should still render children when loading', () => {
      render(<Button isLoading>Loading Text</Button>);

      expect(screen.getByText('Loading Text')).toBeInTheDocument();
    });
  });

  describe('disabled state', () => {
    it('should be enabled by default', () => {
      render(<Button>Button</Button>);

      expect(screen.getByRole('button')).not.toBeDisabled();
    });

    it('should be disabled when disabled prop is true', () => {
      render(<Button disabled>Disabled</Button>);

      expect(screen.getByRole('button')).toBeDisabled();
    });

    it('should apply disabled styles', () => {
      const { container } = render(<Button disabled>Disabled</Button>);

      expect(container.firstChild).toHaveClass(
        'disabled:opacity-50',
        'disabled:cursor-not-allowed'
      );
    });
  });

  describe('click handling', () => {
    it('should call onClick when clicked', () => {
      const handleClick = vi.fn();
      render(<Button onClick={handleClick}>Click me</Button>);

      fireEvent.click(screen.getByRole('button'));

      expect(handleClick).toHaveBeenCalledTimes(1);
    });

    it('should not call onClick when disabled', () => {
      const handleClick = vi.fn();
      render(
        <Button disabled onClick={handleClick}>
          Disabled
        </Button>
      );

      fireEvent.click(screen.getByRole('button'));

      expect(handleClick).not.toHaveBeenCalled();
    });
  });

  describe('custom className', () => {
    it('should apply custom className', () => {
      const { container } = render(
        <Button className="custom-button-class">Button</Button>
      );

      expect(container.firstChild).toHaveClass('custom-button-class');
    });
  });

  describe('ref forwarding', () => {
    it('should forward ref to button element', () => {
      const ref = React.createRef<HTMLButtonElement>();
      render(<Button ref={ref}>Button</Button>);

      expect(ref.current).toBeInstanceOf(HTMLButtonElement);
    });
  });

  describe('additional props', () => {
    it('should pass type prop', () => {
      render(<Button type="submit">Submit</Button>);

      expect(screen.getByRole('button')).toHaveAttribute('type', 'submit');
    });

    it('should pass data attributes', () => {
      render(<Button data-testid="custom-button">Button</Button>);

      expect(screen.getByTestId('custom-button')).toBeInTheDocument();
    });

    it('should pass aria attributes', () => {
      render(<Button aria-label="Custom label">Button</Button>);

      expect(screen.getByLabelText('Custom label')).toBeInTheDocument();
    });
  });
});

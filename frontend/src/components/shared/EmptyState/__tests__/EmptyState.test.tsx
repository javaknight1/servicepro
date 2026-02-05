import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';
import { EmptyState } from '../EmptyState';

describe('EmptyState', () => {
  describe('rendering', () => {
    it('should render title', () => {
      render(<EmptyState title="No items found" />);

      expect(screen.getByText('No items found')).toBeInTheDocument();
    });

    it('should render title with proper heading element', () => {
      render(<EmptyState title="Empty State Title" />);

      const heading = screen.getByRole('heading', { level: 3 });
      expect(heading).toHaveTextContent('Empty State Title');
    });
  });

  describe('description', () => {
    it('should render description when provided', () => {
      render(
        <EmptyState
          title="No items"
          description="There are no items to display at this time."
        />
      );

      expect(
        screen.getByText('There are no items to display at this time.')
      ).toBeInTheDocument();
    });

    it('should not render description when not provided', () => {
      const { container } = render(<EmptyState title="No items" />);

      // Only the title should exist, not a description paragraph
      const paragraphs = container.querySelectorAll('p');
      expect(paragraphs.length).toBe(0);
    });
  });

  describe('icon', () => {
    it('should render icon when provided', () => {
      render(
        <EmptyState
          title="No data"
          icon={<span data-testid="empty-icon">Icon</span>}
        />
      );

      expect(screen.getByTestId('empty-icon')).toBeInTheDocument();
    });

    it('should render icon in styled container', () => {
      const { container } = render(
        <EmptyState
          title="No data"
          icon={<span data-testid="empty-icon">Icon</span>}
        />
      );

      const iconContainer = container.querySelector('.bg-neutral-100');
      expect(iconContainer).toBeInTheDocument();
      expect(iconContainer).toHaveClass('rounded-full');
    });

    it('should not render icon container when not provided', () => {
      const { container } = render(<EmptyState title="No data" />);

      expect(
        container.querySelector('.bg-neutral-100')
      ).not.toBeInTheDocument();
    });
  });

  describe('action button', () => {
    it('should render action button when provided', () => {
      const handleClick = vi.fn();

      render(
        <EmptyState
          title="No items"
          action={{
            label: 'Add Item',
            onClick: handleClick,
          }}
        />
      );

      expect(
        screen.getByRole('button', { name: 'Add Item' })
      ).toBeInTheDocument();
    });

    it('should call onClick when action button is clicked', () => {
      const handleClick = vi.fn();

      render(
        <EmptyState
          title="No items"
          action={{
            label: 'Create New',
            onClick: handleClick,
          }}
        />
      );

      const button = screen.getByRole('button', { name: 'Create New' });
      fireEvent.click(button);

      expect(handleClick).toHaveBeenCalledTimes(1);
    });

    it('should not render button when action is not provided', () => {
      render(<EmptyState title="No items" />);

      expect(screen.queryByRole('button')).not.toBeInTheDocument();
    });
  });

  describe('custom className', () => {
    it('should apply custom className', () => {
      const { container } = render(
        <EmptyState title="Custom Styled" className="my-custom-class" />
      );

      expect(container.firstChild).toHaveClass('my-custom-class');
    });
  });

  describe('styling', () => {
    it('should have centered flex layout', () => {
      const { container } = render(<EmptyState title="Centered" />);

      expect(container.firstChild).toHaveClass(
        'flex',
        'flex-col',
        'items-center',
        'justify-center',
        'text-center'
      );
    });

    it('should have proper padding', () => {
      const { container } = render(<EmptyState title="Padded" />);

      expect(container.firstChild).toHaveClass('py-12', 'px-4');
    });
  });

  describe('complete empty state', () => {
    it('should render all elements when all props provided', () => {
      const handleClick = vi.fn();

      render(
        <EmptyState
          icon={<span data-testid="complete-icon">Icon</span>}
          title="No Results"
          description="Try adjusting your search criteria"
          action={{
            label: 'Clear Filters',
            onClick: handleClick,
          }}
          className="complete-empty-state"
        />
      );

      expect(screen.getByTestId('complete-icon')).toBeInTheDocument();
      expect(screen.getByText('No Results')).toBeInTheDocument();
      expect(
        screen.getByText('Try adjusting your search criteria')
      ).toBeInTheDocument();
      expect(
        screen.getByRole('button', { name: 'Clear Filters' })
      ).toBeInTheDocument();
    });
  });
});

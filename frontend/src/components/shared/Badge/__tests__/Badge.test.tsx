import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Badge, getStatusBadgeVariant } from '../Badge';

describe('Badge', () => {
  describe('rendering', () => {
    it('should render children', () => {
      render(<Badge>Test Badge</Badge>);
      expect(screen.getByText('Test Badge')).toBeInTheDocument();
    });

    it('should render with default variant', () => {
      render(<Badge>Default</Badge>);
      const badge = screen.getByText('Default');
      expect(badge).toHaveClass('bg-neutral-100');
    });

    it('should render with default size', () => {
      render(<Badge>Default Size</Badge>);
      const badge = screen.getByText('Default Size');
      expect(badge).toHaveClass('px-2.5', 'py-1', 'text-sm');
    });
  });

  describe('variants', () => {
    it('should render success variant', () => {
      render(<Badge variant="success">Success</Badge>);
      const badge = screen.getByText('Success');
      expect(badge).toHaveClass('bg-success-100');
      expect(badge).toHaveClass('text-success-700');
    });

    it('should render warning variant', () => {
      render(<Badge variant="warning">Warning</Badge>);
      const badge = screen.getByText('Warning');
      expect(badge).toHaveClass('bg-warning-100');
      expect(badge).toHaveClass('text-warning-700');
    });

    it('should render error variant', () => {
      render(<Badge variant="error">Error</Badge>);
      const badge = screen.getByText('Error');
      expect(badge).toHaveClass('bg-error-100');
      expect(badge).toHaveClass('text-error-700');
    });

    it('should render info variant', () => {
      render(<Badge variant="info">Info</Badge>);
      const badge = screen.getByText('Info');
      expect(badge).toHaveClass('bg-primary-100');
      expect(badge).toHaveClass('text-primary-700');
    });

    it('should render neutral variant', () => {
      render(<Badge variant="neutral">Neutral</Badge>);
      const badge = screen.getByText('Neutral');
      expect(badge).toHaveClass('bg-neutral-100');
      expect(badge).toHaveClass('text-neutral-700');
    });
  });

  describe('sizes', () => {
    it('should render small size', () => {
      render(<Badge size="sm">Small</Badge>);
      const badge = screen.getByText('Small');
      expect(badge).toHaveClass('px-2', 'py-0.5', 'text-xs');
    });

    it('should render medium size', () => {
      render(<Badge size="md">Medium</Badge>);
      const badge = screen.getByText('Medium');
      expect(badge).toHaveClass('px-2.5', 'py-1', 'text-sm');
    });
  });

  describe('custom className', () => {
    it('should apply custom className', () => {
      render(<Badge className="custom-class">Custom</Badge>);
      const badge = screen.getByText('Custom');
      expect(badge).toHaveClass('custom-class');
    });
  });
});

describe('getStatusBadgeVariant', () => {
  describe('generic statuses', () => {
    it('should return success for active status', () => {
      expect(getStatusBadgeVariant('active')).toBe('success');
    });

    it('should return neutral for inactive status', () => {
      expect(getStatusBadgeVariant('inactive')).toBe('neutral');
    });

    it('should return warning for pending status', () => {
      expect(getStatusBadgeVariant('pending')).toBe('warning');
    });

    it('should return success for completed status', () => {
      expect(getStatusBadgeVariant('completed')).toBe('success');
    });

    it('should return error for cancelled status', () => {
      expect(getStatusBadgeVariant('cancelled')).toBe('error');
    });
  });

  describe('quote statuses', () => {
    it('should return neutral for draft status', () => {
      expect(getStatusBadgeVariant('draft')).toBe('neutral');
    });

    it('should return info for sent status', () => {
      expect(getStatusBadgeVariant('sent')).toBe('info');
    });

    it('should return info for viewed status', () => {
      expect(getStatusBadgeVariant('viewed')).toBe('info');
    });

    it('should return success for accepted status', () => {
      expect(getStatusBadgeVariant('accepted')).toBe('success');
    });

    it('should return error for declined status', () => {
      expect(getStatusBadgeVariant('declined')).toBe('error');
    });

    it('should return warning for expired status', () => {
      expect(getStatusBadgeVariant('expired')).toBe('warning');
    });
  });

  describe('job statuses', () => {
    it('should return info for scheduled status', () => {
      expect(getStatusBadgeVariant('scheduled')).toBe('info');
    });

    it('should return warning for in_progress status', () => {
      expect(getStatusBadgeVariant('in_progress')).toBe('warning');
    });
  });

  describe('invoice statuses', () => {
    it('should return success for paid status', () => {
      expect(getStatusBadgeVariant('paid')).toBe('success');
    });

    it('should return warning for partially_paid status', () => {
      expect(getStatusBadgeVariant('partially_paid')).toBe('warning');
    });

    it('should return error for overdue status', () => {
      expect(getStatusBadgeVariant('overdue')).toBe('error');
    });
  });

  describe('customer statuses', () => {
    it('should return info for prospect status', () => {
      expect(getStatusBadgeVariant('prospect')).toBe('info');
    });
  });

  describe('priorities', () => {
    it('should return neutral for low priority', () => {
      expect(getStatusBadgeVariant('low')).toBe('neutral');
    });

    it('should return info for medium priority', () => {
      expect(getStatusBadgeVariant('medium')).toBe('info');
    });

    it('should return warning for high priority', () => {
      expect(getStatusBadgeVariant('high')).toBe('warning');
    });

    it('should return error for urgent priority', () => {
      expect(getStatusBadgeVariant('urgent')).toBe('error');
    });
  });

  describe('edge cases', () => {
    it('should be case-insensitive', () => {
      expect(getStatusBadgeVariant('ACTIVE')).toBe('success');
      expect(getStatusBadgeVariant('Active')).toBe('success');
    });

    it('should return neutral for unknown status', () => {
      expect(getStatusBadgeVariant('unknown')).toBe('neutral');
      expect(getStatusBadgeVariant('random')).toBe('neutral');
    });
  });
});

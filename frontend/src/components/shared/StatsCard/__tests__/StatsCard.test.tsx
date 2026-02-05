import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import React from 'react';
import { StatsCard } from '../StatsCard';

describe('StatsCard', () => {
  describe('rendering', () => {
    it('should render label and value', () => {
      render(<StatsCard label="Total Sales" value="$10,000" />);

      expect(screen.getByText('Total Sales')).toBeInTheDocument();
      expect(screen.getByText('$10,000')).toBeInTheDocument();
    });

    it('should render numeric value', () => {
      render(<StatsCard label="Count" value={42} />);

      expect(screen.getByText('42')).toBeInTheDocument();
    });
  });

  describe('icon', () => {
    it('should render icon when provided', () => {
      render(
        <StatsCard
          label="Sales"
          value="$5,000"
          icon={<span data-testid="test-icon">Icon</span>}
        />
      );

      expect(screen.getByTestId('test-icon')).toBeInTheDocument();
    });

    it('should not render icon container when not provided', () => {
      const { container } = render(<StatsCard label="Sales" value="$5,000" />);

      expect(
        container.querySelector('.bg-primary-100')
      ).not.toBeInTheDocument();
    });
  });

  describe('change indicator', () => {
    it('should render change when provided', () => {
      render(<StatsCard label="Revenue" value="$20,000" change="+15%" />);

      expect(screen.getByText('+15%')).toBeInTheDocument();
    });

    it('should not render change when not provided', () => {
      const { container } = render(
        <StatsCard label="Revenue" value="$20,000" />
      );

      // Check that no change indicator exists
      expect(
        container.querySelector('.text-success-600')
      ).not.toBeInTheDocument();
      expect(
        container.querySelector('.text-error-600')
      ).not.toBeInTheDocument();
    });
  });

  describe('changeType', () => {
    it('should apply positive change type styling', () => {
      const { container } = render(
        <StatsCard
          label="Profit"
          value="$5,000"
          change="+25%"
          changeType="positive"
        />
      );

      const changeContainer = container.querySelector('.text-success-600');
      expect(changeContainer).toBeInTheDocument();
    });

    it('should apply negative change type styling', () => {
      const { container } = render(
        <StatsCard
          label="Loss"
          value="$1,000"
          change="-10%"
          changeType="negative"
        />
      );

      const changeContainer = container.querySelector('.text-error-600');
      expect(changeContainer).toBeInTheDocument();
    });

    it('should apply neutral change type styling by default', () => {
      const { container } = render(
        <StatsCard label="Balance" value="$3,000" change="0%" />
      );

      const changeContainer = container.querySelector('.text-neutral-600');
      expect(changeContainer).toBeInTheDocument();
    });

    it('should render TrendingUp icon for positive change', () => {
      const { container } = render(
        <StatsCard
          label="Growth"
          value="150"
          change="+50%"
          changeType="positive"
        />
      );

      // Lucide icons render as SVG
      const svg = container.querySelector('svg');
      expect(svg).toBeInTheDocument();
    });

    it('should render TrendingDown icon for negative change', () => {
      const { container } = render(
        <StatsCard
          label="Decline"
          value="50"
          change="-20%"
          changeType="negative"
        />
      );

      const svg = container.querySelector('svg');
      expect(svg).toBeInTheDocument();
    });

    it('should render Minus icon for neutral change', () => {
      const { container } = render(
        <StatsCard
          label="Static"
          value="100"
          change="0%"
          changeType="neutral"
        />
      );

      const svg = container.querySelector('svg');
      expect(svg).toBeInTheDocument();
    });
  });

  describe('custom className', () => {
    it('should apply custom className', () => {
      const { container } = render(
        <StatsCard label="Test" value="123" className="custom-stats-class" />
      );

      expect(container.firstChild).toHaveClass('custom-stats-class');
    });
  });

  describe('styling', () => {
    it('should have default card styling', () => {
      const { container } = render(<StatsCard label="Default" value="100" />);

      expect(container.firstChild).toHaveClass(
        'bg-white',
        'rounded-xl',
        'border',
        'border-neutral-200',
        'p-6',
        'shadow-sm'
      );
    });
  });
});

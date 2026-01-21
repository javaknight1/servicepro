import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from './Card';

describe('Card Component', () => {
  // ==========================================================================
  // Rendering Tests
  // ==========================================================================
  describe('Rendering', () => {
    it('renders children correctly', () => {
      render(<Card>Card content</Card>);
      expect(screen.getByText('Card content')).toBeInTheDocument();
    });

    it('renders with default props', () => {
      const { container } = render(<Card>Default Card</Card>);
      const card = container.firstChild;

      expect(card).toHaveClass('rounded-lg');
      expect(card).toHaveClass('bg-white');
      expect(card).toHaveClass('border');
      expect(card).toHaveClass('p-6'); // default md padding
    });

    it('renders with forwardRef', () => {
      const ref = vi.fn();
      render(<Card ref={ref}>Ref Card</Card>);
      expect(ref).toHaveBeenCalled();
    });
  });

  // ==========================================================================
  // Variant Tests - Table Driven
  // ==========================================================================
  describe('Variants', () => {
    const variants = [
      { variant: 'default' as const, expectedClass: 'border-neutral-200' },
      { variant: 'bordered' as const, expectedClass: 'border-2' },
      { variant: 'elevated' as const, expectedClass: 'shadow-medium' },
    ];

    variants.forEach(({ variant, expectedClass }) => {
      it(`renders ${variant} variant with correct styles`, () => {
        const { container } = render(
          <Card variant={variant}>Variant Card</Card>
        );
        expect(container.firstChild).toHaveClass(expectedClass);
      });
    });
  });

  // ==========================================================================
  // Padding Tests - Table Driven
  // ==========================================================================
  describe('Padding', () => {
    const paddings = [
      { padding: 'none' as const, expectedClass: null, notExpected: 'p-4' },
      { padding: 'sm' as const, expectedClass: 'p-4', notExpected: 'p-6' },
      { padding: 'md' as const, expectedClass: 'p-6', notExpected: 'p-8' },
      { padding: 'lg' as const, expectedClass: 'p-8', notExpected: 'p-4' },
    ];

    paddings.forEach(({ padding, expectedClass, notExpected }) => {
      it(`renders ${padding} padding with correct styles`, () => {
        const { container } = render(
          <Card padding={padding}>Padded Card</Card>
        );
        const card = container.firstChild;

        if (expectedClass) {
          expect(card).toHaveClass(expectedClass);
        }
        expect(card).not.toHaveClass(notExpected);
      });
    });
  });

  // ==========================================================================
  // Custom Props Tests
  // ==========================================================================
  describe('Custom Props', () => {
    it('applies custom className', () => {
      const { container } = render(
        <Card className="custom-class">Custom Card</Card>
      );
      expect(container.firstChild).toHaveClass('custom-class');
    });

    it('merges custom className with default classes', () => {
      const { container } = render(
        <Card className="custom-class">Custom Card</Card>
      );
      const card = container.firstChild;

      expect(card).toHaveClass('custom-class');
      expect(card).toHaveClass('rounded-lg');
      expect(card).toHaveClass('bg-white');
    });

    it('passes through additional HTML attributes', () => {
      const { container } = render(
        <Card data-testid="card-test" aria-label="Test Card">
          Card
        </Card>
      );
      const card = container.firstChild;

      expect(card).toHaveAttribute('data-testid', 'card-test');
      expect(card).toHaveAttribute('aria-label', 'Test Card');
    });

    it('supports onClick handler', async () => {
      const handleClick = vi.fn();
      const { container } = render(
        <Card onClick={handleClick}>Clickable Card</Card>
      );

      (container.firstChild as HTMLElement).click();
      expect(handleClick).toHaveBeenCalledTimes(1);
    });
  });

  // ==========================================================================
  // Combination Tests
  // ==========================================================================
  describe('Prop Combinations', () => {
    it('handles multiple props together', () => {
      const { container } = render(
        <Card
          variant="elevated"
          padding="lg"
          className="custom-class"
          data-testid="combo-card"
        >
          Combined Props
        </Card>
      );
      const card = container.firstChild;

      expect(card).toHaveClass('shadow-medium');
      expect(card).toHaveClass('p-8');
      expect(card).toHaveClass('custom-class');
      expect(card).toHaveAttribute('data-testid', 'combo-card');
    });
  });
});

// =============================================================================
// CardHeader Tests
// =============================================================================
describe('CardHeader Component', () => {
  it('renders children correctly', () => {
    render(<CardHeader>Header Content</CardHeader>);
    expect(screen.getByText('Header Content')).toBeInTheDocument();
  });

  it('applies default styles', () => {
    const { container } = render(<CardHeader>Header</CardHeader>);
    const header = container.firstChild;

    expect(header).toHaveClass('flex');
    expect(header).toHaveClass('flex-col');
    expect(header).toHaveClass('space-y-1.5');
  });

  it('applies custom className', () => {
    const { container } = render(
      <CardHeader className="custom-header">Header</CardHeader>
    );
    expect(container.firstChild).toHaveClass('custom-header');
  });

  it('renders with forwardRef', () => {
    const ref = vi.fn();
    render(<CardHeader ref={ref}>Header</CardHeader>);
    expect(ref).toHaveBeenCalled();
  });
});

// =============================================================================
// CardTitle Tests
// =============================================================================
describe('CardTitle Component', () => {
  it('renders as h3 element', () => {
    render(<CardTitle>Title</CardTitle>);
    expect(screen.getByRole('heading', { level: 3 })).toBeInTheDocument();
  });

  it('renders children correctly', () => {
    render(<CardTitle>Card Title</CardTitle>);
    expect(screen.getByText('Card Title')).toBeInTheDocument();
  });

  it('applies default styles', () => {
    const { container } = render(<CardTitle>Title</CardTitle>);
    const title = container.firstChild;

    expect(title).toHaveClass('text-2xl');
    expect(title).toHaveClass('font-semibold');
    expect(title).toHaveClass('leading-none');
    expect(title).toHaveClass('tracking-tight');
  });

  it('applies custom className', () => {
    const { container } = render(
      <CardTitle className="custom-title">Title</CardTitle>
    );
    expect(container.firstChild).toHaveClass('custom-title');
  });

  it('renders with forwardRef', () => {
    const ref = vi.fn();
    render(<CardTitle ref={ref}>Title</CardTitle>);
    expect(ref).toHaveBeenCalled();
  });
});

// =============================================================================
// CardDescription Tests
// =============================================================================
describe('CardDescription Component', () => {
  it('renders as p element', () => {
    const { container } = render(
      <CardDescription>Description</CardDescription>
    );
    expect(container.querySelector('p')).toBeInTheDocument();
  });

  it('renders children correctly', () => {
    render(<CardDescription>Card Description</CardDescription>);
    expect(screen.getByText('Card Description')).toBeInTheDocument();
  });

  it('applies default styles', () => {
    const { container } = render(
      <CardDescription>Description</CardDescription>
    );
    const description = container.firstChild;

    expect(description).toHaveClass('text-sm');
    expect(description).toHaveClass('text-neutral-500');
  });

  it('applies custom className', () => {
    const { container } = render(
      <CardDescription className="custom-desc">Description</CardDescription>
    );
    expect(container.firstChild).toHaveClass('custom-desc');
  });

  it('renders with forwardRef', () => {
    const ref = vi.fn();
    render(<CardDescription ref={ref}>Description</CardDescription>);
    expect(ref).toHaveBeenCalled();
  });
});

// =============================================================================
// CardContent Tests
// =============================================================================
describe('CardContent Component', () => {
  it('renders children correctly', () => {
    render(<CardContent>Content</CardContent>);
    expect(screen.getByText('Content')).toBeInTheDocument();
  });

  it('applies default styles', () => {
    const { container } = render(<CardContent>Content</CardContent>);
    expect(container.firstChild).toHaveClass('pt-0');
  });

  it('applies custom className', () => {
    const { container } = render(
      <CardContent className="custom-content">Content</CardContent>
    );
    expect(container.firstChild).toHaveClass('custom-content');
  });

  it('renders with forwardRef', () => {
    const ref = vi.fn();
    render(<CardContent ref={ref}>Content</CardContent>);
    expect(ref).toHaveBeenCalled();
  });
});

// =============================================================================
// CardFooter Tests
// =============================================================================
describe('CardFooter Component', () => {
  it('renders children correctly', () => {
    render(<CardFooter>Footer</CardFooter>);
    expect(screen.getByText('Footer')).toBeInTheDocument();
  });

  it('applies default styles', () => {
    const { container } = render(<CardFooter>Footer</CardFooter>);
    const footer = container.firstChild;

    expect(footer).toHaveClass('flex');
    expect(footer).toHaveClass('items-center');
    expect(footer).toHaveClass('pt-0');
  });

  it('applies custom className', () => {
    const { container } = render(
      <CardFooter className="custom-footer">Footer</CardFooter>
    );
    expect(container.firstChild).toHaveClass('custom-footer');
  });

  it('renders with forwardRef', () => {
    const ref = vi.fn();
    render(<CardFooter ref={ref}>Footer</CardFooter>);
    expect(ref).toHaveBeenCalled();
  });
});

// =============================================================================
// Integration Tests - Full Card Composition
// =============================================================================
describe('Card Composition', () => {
  it('renders a complete card with all sub-components', () => {
    render(
      <Card>
        <CardHeader>
          <CardTitle>Welcome</CardTitle>
          <CardDescription>This is a description</CardDescription>
        </CardHeader>
        <CardContent>
          <p>Main content goes here</p>
        </CardContent>
        <CardFooter>
          <button>Action</button>
        </CardFooter>
      </Card>
    );

    expect(
      screen.getByRole('heading', { name: 'Welcome' })
    ).toBeInTheDocument();
    expect(screen.getByText('This is a description')).toBeInTheDocument();
    expect(screen.getByText('Main content goes here')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Action' })).toBeInTheDocument();
  });

  it('nests components correctly', () => {
    const { container } = render(
      <Card data-testid="outer-card">
        <CardHeader data-testid="header">
          <CardTitle data-testid="title">Title</CardTitle>
        </CardHeader>
        <CardContent data-testid="content">Content</CardContent>
      </Card>
    );

    const card = container.firstChild;
    expect(card).toContainElement(screen.getByTestId('header'));
    expect(card).toContainElement(screen.getByTestId('title'));
    expect(card).toContainElement(screen.getByTestId('content'));
  });
});

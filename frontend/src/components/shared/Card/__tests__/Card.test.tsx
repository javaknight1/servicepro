import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import React from 'react';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from '../Card';

describe('Card', () => {
  describe('Card component', () => {
    it('should render children', () => {
      render(<Card>Card Content</Card>);

      expect(screen.getByText('Card Content')).toBeInTheDocument();
    });

    it('should apply default variant classes', () => {
      const { container } = render(<Card>Content</Card>);

      expect(container.firstChild).toHaveClass(
        'bg-white',
        'border',
        'border-neutral-200'
      );
    });

    it('should apply bordered variant classes', () => {
      const { container } = render(<Card variant="bordered">Content</Card>);

      expect(container.firstChild).toHaveClass(
        'bg-white',
        'border-2',
        'border-neutral-300'
      );
    });

    it('should apply elevated variant classes', () => {
      const { container } = render(<Card variant="elevated">Content</Card>);

      expect(container.firstChild).toHaveClass('bg-white', 'shadow-medium');
    });

    it('should apply default md padding', () => {
      const { container } = render(<Card>Content</Card>);

      expect(container.firstChild).toHaveClass('p-6');
    });

    it('should apply none padding', () => {
      const { container } = render(<Card padding="none">Content</Card>);

      expect(container.firstChild).not.toHaveClass('p-4', 'p-6', 'p-8');
    });

    it('should apply sm padding', () => {
      const { container } = render(<Card padding="sm">Content</Card>);

      expect(container.firstChild).toHaveClass('p-4');
    });

    it('should apply lg padding', () => {
      const { container } = render(<Card padding="lg">Content</Card>);

      expect(container.firstChild).toHaveClass('p-8');
    });

    it('should apply custom className', () => {
      const { container } = render(
        <Card className="custom-class">Content</Card>
      );

      expect(container.firstChild).toHaveClass('custom-class');
    });

    it('should have rounded-lg class', () => {
      const { container } = render(<Card>Content</Card>);

      expect(container.firstChild).toHaveClass('rounded-lg');
    });

    it('should pass additional props', () => {
      render(<Card data-testid="test-card">Content</Card>);

      expect(screen.getByTestId('test-card')).toBeInTheDocument();
    });

    it('should forward ref', () => {
      const ref = React.createRef<HTMLDivElement>();
      render(<Card ref={ref}>Content</Card>);

      expect(ref.current).toBeInstanceOf(HTMLDivElement);
    });
  });

  describe('CardHeader', () => {
    it('should render children', () => {
      render(<CardHeader>Header Content</CardHeader>);

      expect(screen.getByText('Header Content')).toBeInTheDocument();
    });

    it('should apply flex classes', () => {
      const { container } = render(<CardHeader>Header</CardHeader>);

      expect(container.firstChild).toHaveClass(
        'flex',
        'flex-col',
        'space-y-1.5'
      );
    });

    it('should apply custom className', () => {
      const { container } = render(
        <CardHeader className="custom-header">Header</CardHeader>
      );

      expect(container.firstChild).toHaveClass('custom-header');
    });

    it('should forward ref', () => {
      const ref = React.createRef<HTMLDivElement>();
      render(<CardHeader ref={ref}>Header</CardHeader>);

      expect(ref.current).toBeInstanceOf(HTMLDivElement);
    });
  });

  describe('CardTitle', () => {
    it('should render children', () => {
      render(<CardTitle>Title Text</CardTitle>);

      expect(screen.getByText('Title Text')).toBeInTheDocument();
    });

    it('should render as h3 element', () => {
      render(<CardTitle>Title</CardTitle>);

      expect(screen.getByRole('heading', { level: 3 })).toBeInTheDocument();
    });

    it('should apply title styling classes', () => {
      const { container } = render(<CardTitle>Title</CardTitle>);

      expect(container.firstChild).toHaveClass(
        'text-2xl',
        'font-semibold',
        'leading-none',
        'tracking-tight'
      );
    });

    it('should apply custom className', () => {
      const { container } = render(
        <CardTitle className="custom-title">Title</CardTitle>
      );

      expect(container.firstChild).toHaveClass('custom-title');
    });

    it('should forward ref', () => {
      const ref = React.createRef<HTMLHeadingElement>();
      render(<CardTitle ref={ref}>Title</CardTitle>);

      expect(ref.current).toBeInstanceOf(HTMLHeadingElement);
    });
  });

  describe('CardDescription', () => {
    it('should render children', () => {
      render(<CardDescription>Description text</CardDescription>);

      expect(screen.getByText('Description text')).toBeInTheDocument();
    });

    it('should apply description styling classes', () => {
      const { container } = render(<CardDescription>Desc</CardDescription>);

      expect(container.firstChild).toHaveClass('text-sm', 'text-neutral-500');
    });

    it('should apply custom className', () => {
      const { container } = render(
        <CardDescription className="custom-desc">Desc</CardDescription>
      );

      expect(container.firstChild).toHaveClass('custom-desc');
    });

    it('should forward ref', () => {
      const ref = React.createRef<HTMLParagraphElement>();
      render(<CardDescription ref={ref}>Desc</CardDescription>);

      expect(ref.current).toBeInstanceOf(HTMLParagraphElement);
    });
  });

  describe('CardContent', () => {
    it('should render children', () => {
      render(<CardContent>Main content here</CardContent>);

      expect(screen.getByText('Main content here')).toBeInTheDocument();
    });

    it('should apply content styling class', () => {
      const { container } = render(<CardContent>Content</CardContent>);

      expect(container.firstChild).toHaveClass('pt-0');
    });

    it('should apply custom className', () => {
      const { container } = render(
        <CardContent className="custom-content">Content</CardContent>
      );

      expect(container.firstChild).toHaveClass('custom-content');
    });

    it('should forward ref', () => {
      const ref = React.createRef<HTMLDivElement>();
      render(<CardContent ref={ref}>Content</CardContent>);

      expect(ref.current).toBeInstanceOf(HTMLDivElement);
    });
  });

  describe('CardFooter', () => {
    it('should render children', () => {
      render(<CardFooter>Footer content</CardFooter>);

      expect(screen.getByText('Footer content')).toBeInTheDocument();
    });

    it('should apply footer styling classes', () => {
      const { container } = render(<CardFooter>Footer</CardFooter>);

      expect(container.firstChild).toHaveClass('flex', 'items-center', 'pt-0');
    });

    it('should apply custom className', () => {
      const { container } = render(
        <CardFooter className="custom-footer">Footer</CardFooter>
      );

      expect(container.firstChild).toHaveClass('custom-footer');
    });

    it('should forward ref', () => {
      const ref = React.createRef<HTMLDivElement>();
      render(<CardFooter ref={ref}>Footer</CardFooter>);

      expect(ref.current).toBeInstanceOf(HTMLDivElement);
    });
  });

  describe('composed Card', () => {
    it('should render a complete card structure', () => {
      render(
        <Card>
          <CardHeader>
            <CardTitle>Card Title</CardTitle>
            <CardDescription>Card description text</CardDescription>
          </CardHeader>
          <CardContent>
            <p>Main card content</p>
          </CardContent>
          <CardFooter>
            <button>Action</button>
          </CardFooter>
        </Card>
      );

      expect(screen.getByText('Card Title')).toBeInTheDocument();
      expect(screen.getByText('Card description text')).toBeInTheDocument();
      expect(screen.getByText('Main card content')).toBeInTheDocument();
      expect(
        screen.getByRole('button', { name: 'Action' })
      ).toBeInTheDocument();
    });
  });
});

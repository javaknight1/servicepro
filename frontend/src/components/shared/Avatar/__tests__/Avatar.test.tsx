import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { Avatar } from '../Avatar';

describe('Avatar', () => {
  describe('rendering with initials', () => {
    it('should render initials when no profile picture', () => {
      render(
        <Avatar email="test@example.com" firstName="John" lastName="Doe" />
      );

      expect(screen.getByText('JD')).toBeInTheDocument();
    });

    it('should render first two letters when only first name provided', () => {
      render(<Avatar email="test@example.com" firstName="John" />);

      expect(screen.getByText('JO')).toBeInTheDocument();
    });

    it('should render first two letters of email when no name provided', () => {
      render(<Avatar email="test@example.com" />);

      expect(screen.getByText('TE')).toBeInTheDocument();
    });
  });

  describe('rendering with profile picture', () => {
    it('should render img when profilePictureUrl is provided', () => {
      render(
        <Avatar
          email="test@example.com"
          firstName="John"
          lastName="Doe"
          profilePictureUrl="https://example.com/avatar.jpg"
        />
      );

      const img = screen.getByRole('img');
      expect(img).toHaveAttribute('src', 'https://example.com/avatar.jpg');
    });

    it('should have correct alt text', () => {
      render(
        <Avatar
          email="test@example.com"
          firstName="John"
          lastName="Doe"
          profilePictureUrl="https://example.com/avatar.jpg"
        />
      );

      const img = screen.getByRole('img');
      expect(img).toHaveAttribute('alt', "John Doe's avatar");
    });
  });

  describe('sizes', () => {
    it('should render xs size', () => {
      const { container } = render(
        <Avatar email="test@example.com" firstName="A" size="xs" />
      );

      expect(container.firstChild).toHaveClass('h-6', 'w-6', 'text-xs');
    });

    it('should render sm size', () => {
      const { container } = render(
        <Avatar email="test@example.com" firstName="A" size="sm" />
      );

      expect(container.firstChild).toHaveClass('h-8', 'w-8', 'text-sm');
    });

    it('should render md size (default)', () => {
      const { container } = render(
        <Avatar email="test@example.com" firstName="A" />
      );

      expect(container.firstChild).toHaveClass('h-10', 'w-10', 'text-sm');
    });

    it('should render lg size', () => {
      const { container } = render(
        <Avatar email="test@example.com" firstName="A" size="lg" />
      );

      expect(container.firstChild).toHaveClass('h-12', 'w-12', 'text-base');
    });

    it('should render xl size', () => {
      const { container } = render(
        <Avatar email="test@example.com" firstName="A" size="xl" />
      );

      expect(container.firstChild).toHaveClass('h-16', 'w-16', 'text-lg');
    });
  });

  describe('custom className', () => {
    it('should apply custom className to initials avatar', () => {
      const { container } = render(
        <Avatar
          email="test@example.com"
          firstName="A"
          className="custom-class"
        />
      );

      expect(container.firstChild).toHaveClass('custom-class');
    });

    it('should apply custom className to image avatar', () => {
      const { container } = render(
        <Avatar
          email="test@example.com"
          firstName="A"
          profilePictureUrl="https://example.com/avatar.jpg"
          className="custom-img-class"
        />
      );

      expect(container.firstChild).toHaveClass('custom-img-class');
    });
  });

  describe('title attribute', () => {
    it('should have title with display name', () => {
      render(
        <Avatar email="test@example.com" firstName="John" lastName="Doe" />
      );

      expect(screen.getByTitle('John Doe')).toBeInTheDocument();
    });

    it('should use email when no name provided', () => {
      render(<Avatar email="test@example.com" />);

      expect(screen.getByTitle('test@example.com')).toBeInTheDocument();
    });
  });

  describe('styling', () => {
    it('should have rounded-full class', () => {
      const { container } = render(
        <Avatar email="test@example.com" firstName="A" />
      );

      expect(container.firstChild).toHaveClass('rounded-full');
    });

    it('should have object-cover class for images', () => {
      render(
        <Avatar
          email="test@example.com"
          profilePictureUrl="https://example.com/avatar.jpg"
        />
      );

      const img = screen.getByRole('img');
      expect(img).toHaveClass('object-cover');
    });
  });

  describe('null values', () => {
    it('should handle null firstName', () => {
      render(
        <Avatar email="test@example.com" firstName={null} lastName="Doe" />
      );

      // When only lastName is provided, uses first 2 letters
      expect(screen.getByText('DO')).toBeInTheDocument();
    });

    it('should handle null lastName', () => {
      render(
        <Avatar email="test@example.com" firstName="John" lastName={null} />
      );

      // When only firstName is provided, uses first 2 letters
      expect(screen.getByText('JO')).toBeInTheDocument();
    });

    it('should handle null profilePictureUrl', () => {
      render(
        <Avatar
          email="test@example.com"
          firstName="John"
          lastName="Doe"
          profilePictureUrl={null}
        />
      );

      // Should show initials, not an image
      expect(screen.queryByRole('img')).not.toBeInTheDocument();
      expect(screen.getByText('JD')).toBeInTheDocument();
    });
  });
});

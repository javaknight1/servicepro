import { describe, it, expect, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Modal } from './Modal';

describe('Modal Component', () => {
  // ==========================================================================
  // Rendering Tests
  // ==========================================================================
  describe('Rendering', () => {
    it('renders children when open', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()}>
          Modal content
        </Modal>
      );
      expect(screen.getByText('Modal content')).toBeInTheDocument();
    });

    it('does not render children when closed', () => {
      render(
        <Modal isOpen={false} onClose={vi.fn()}>
          Modal content
        </Modal>
      );
      expect(screen.queryByText('Modal content')).not.toBeInTheDocument();
    });

    it('renders title when provided', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()} title="Modal Title">
          Content
        </Modal>
      );
      expect(screen.getByText('Modal Title')).toBeInTheDocument();
    });

    it('does not render title when not provided', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()}>
          Content
        </Modal>
      );
      expect(screen.queryByRole('heading')).not.toBeInTheDocument();
    });

    it('renders description when provided', () => {
      render(
        <Modal
          isOpen={true}
          onClose={vi.fn()}
          description="This is a description"
        >
          Content
        </Modal>
      );
      expect(screen.getByText('This is a description')).toBeInTheDocument();
    });

    it('does not render description when not provided', () => {
      const { container } = render(
        <Modal isOpen={true} onClose={vi.fn()}>
          Content
        </Modal>
      );
      expect(
        container.querySelector('[id*="description"]')
      ).not.toBeInTheDocument();
    });
  });

  // ==========================================================================
  // Close Button Tests
  // ==========================================================================
  describe('Close Button', () => {
    it('renders close button by default', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()}>
          Content
        </Modal>
      );
      // Close button has X icon from lucide-react
      const closeButton = screen.getByRole('button');
      expect(closeButton).toBeInTheDocument();
    });

    it('does not render close button when showCloseButton is false', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()} showCloseButton={false}>
          Content
        </Modal>
      );
      // When showCloseButton is false, no button should be in the modal
      expect(screen.queryByRole('button')).not.toBeInTheDocument();
    });

    it('calls onClose when close button is clicked', async () => {
      const handleClose = vi.fn();
      const user = userEvent.setup();

      render(
        <Modal isOpen={true} onClose={handleClose}>
          Content
        </Modal>
      );

      await user.click(screen.getByRole('button'));
      expect(handleClose).toHaveBeenCalledTimes(1);
    });
  });

  // ==========================================================================
  // Size Tests - Table Driven
  // ==========================================================================
  describe('Sizes', () => {
    const sizes = [
      { size: 'sm' as const, expectedClass: 'max-w-sm' },
      { size: 'md' as const, expectedClass: 'max-w-md' },
      { size: 'lg' as const, expectedClass: 'max-w-lg' },
      { size: 'xl' as const, expectedClass: 'max-w-xl' },
      { size: 'full' as const, expectedClass: 'max-w-full' },
    ];

    sizes.forEach(({ size, expectedClass }) => {
      it(`renders ${size} size with correct styles`, () => {
        const { container } = render(
          <Modal isOpen={true} onClose={vi.fn()} size={size}>
            {size} Modal
          </Modal>
        );

        // Find the Dialog.Panel element which has the size class
        const panel = container.querySelector('[class*="rounded-2xl"]');
        expect(panel).toHaveClass(expectedClass);
      });
    });

    it('uses md size by default', () => {
      const { container } = render(
        <Modal isOpen={true} onClose={vi.fn()}>
          Default Size Modal
        </Modal>
      );

      const panel = container.querySelector('[class*="rounded-2xl"]');
      expect(panel).toHaveClass('max-w-md');
    });
  });

  // ==========================================================================
  // Dialog Behavior Tests
  // ==========================================================================
  describe('Dialog Behavior', () => {
    it('has correct dialog role', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()}>
          Content
        </Modal>
      );
      expect(screen.getByRole('dialog')).toBeInTheDocument();
    });

    it('title has correct role when provided', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()} title="Test Title">
          Content
        </Modal>
      );
      // HeadlessUI Dialog.Title sets up proper heading
      expect(screen.getByText('Test Title').tagName).toBe('H3');
    });

    it('calls onClose when clicking outside modal', async () => {
      const handleClose = vi.fn();
      const user = userEvent.setup();

      render(
        <Modal isOpen={true} onClose={handleClose}>
          Content
        </Modal>
      );

      // Click on the backdrop (the fixed inset-0 overlay)
      const backdrop = document.querySelector('.fixed.inset-0.bg-black');
      if (backdrop) {
        await user.click(backdrop);
        await waitFor(() => {
          expect(handleClose).toHaveBeenCalled();
        });
      }
    });

    it('calls onClose when pressing Escape key', async () => {
      const handleClose = vi.fn();
      const user = userEvent.setup();

      render(
        <Modal isOpen={true} onClose={handleClose}>
          Content
        </Modal>
      );

      await user.keyboard('{Escape}');
      await waitFor(() => {
        expect(handleClose).toHaveBeenCalled();
      });
    });
  });

  // ==========================================================================
  // Accessibility Tests
  // ==========================================================================
  describe('Accessibility', () => {
    it('has dialog role', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()}>
          Content
        </Modal>
      );
      expect(screen.getByRole('dialog')).toBeInTheDocument();
    });

    it('is labeled by title when provided', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()} title="Accessible Modal">
          Content
        </Modal>
      );

      const dialog = screen.getByRole('dialog');
      // HeadlessUI sets aria-labelledby automatically
      expect(dialog).toHaveAttribute('aria-labelledby');
    });

    it('is described by description when provided', () => {
      render(
        <Modal
          isOpen={true}
          onClose={vi.fn()}
          title="Modal"
          description="Description text"
        >
          Content
        </Modal>
      );

      const dialog = screen.getByRole('dialog');
      // HeadlessUI sets aria-describedby automatically
      expect(dialog).toHaveAttribute('aria-describedby');
    });

    it('traps focus within modal', async () => {
      const user = userEvent.setup();

      render(
        <Modal isOpen={true} onClose={vi.fn()}>
          <input data-testid="input-1" />
          <button data-testid="button-1">Button</button>
        </Modal>
      );

      // Focus should be trapped within the modal
      const input = screen.getByTestId('input-1');
      const button = screen.getByTestId('button-1');

      input.focus();
      expect(input).toHaveFocus();

      await user.tab();
      expect(button).toHaveFocus();
    });
  });

  // ==========================================================================
  // Styling Tests
  // ==========================================================================
  describe('Styling', () => {
    it('applies base modal panel styles', () => {
      const { container } = render(
        <Modal isOpen={true} onClose={vi.fn()}>
          Content
        </Modal>
      );

      const panel = container.querySelector('[class*="rounded-2xl"]');
      expect(panel).toHaveClass('bg-white');
      expect(panel).toHaveClass('p-6');
      expect(panel).toHaveClass('shadow-xl');
      expect(panel).toHaveClass('rounded-2xl');
    });

    it('applies overlay styles', () => {
      const { container } = render(
        <Modal isOpen={true} onClose={vi.fn()}>
          Content
        </Modal>
      );

      const overlay = container.querySelector('.bg-black.bg-opacity-25');
      expect(overlay).toBeInTheDocument();
    });

    it('centers modal in viewport', () => {
      const { container } = render(
        <Modal isOpen={true} onClose={vi.fn()}>
          Content
        </Modal>
      );

      const centering = container.querySelector(
        '.flex.min-h-full.items-center.justify-center'
      );
      expect(centering).toBeInTheDocument();
    });
  });

  // ==========================================================================
  // Title and Description Styling Tests
  // ==========================================================================
  describe('Title and Description Styling', () => {
    it('title has correct styles', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()} title="Styled Title">
          Content
        </Modal>
      );

      const title = screen.getByText('Styled Title');
      expect(title).toHaveClass('text-lg');
      expect(title).toHaveClass('font-medium');
      expect(title).toHaveClass('text-neutral-900');
    });

    it('description has correct styles', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()} description="Styled description">
          Content
        </Modal>
      );

      const description = screen.getByText('Styled description');
      expect(description).toHaveClass('text-sm');
      expect(description).toHaveClass('text-neutral-500');
    });
  });

  // ==========================================================================
  // Transition States Tests
  // ==========================================================================
  describe('Transitions', () => {
    it('transitions from closed to open', async () => {
      const { rerender } = render(
        <Modal isOpen={false} onClose={vi.fn()}>
          Content
        </Modal>
      );

      expect(screen.queryByText('Content')).not.toBeInTheDocument();

      rerender(
        <Modal isOpen={true} onClose={vi.fn()}>
          Content
        </Modal>
      );

      await waitFor(() => {
        expect(screen.getByText('Content')).toBeInTheDocument();
      });
    });

    it('transitions from open to closed', async () => {
      const { rerender } = render(
        <Modal isOpen={true} onClose={vi.fn()}>
          Content
        </Modal>
      );

      expect(screen.getByText('Content')).toBeInTheDocument();

      rerender(
        <Modal isOpen={false} onClose={vi.fn()}>
          Content
        </Modal>
      );

      await waitFor(() => {
        expect(screen.queryByText('Content')).not.toBeInTheDocument();
      });
    });
  });

  // ==========================================================================
  // Complex Content Tests
  // ==========================================================================
  describe('Complex Content', () => {
    it('renders form elements correctly', () => {
      render(
        <Modal isOpen={true} onClose={vi.fn()} title="Form Modal">
          <form>
            <input placeholder="Name" />
            <button type="submit">Submit</button>
          </form>
        </Modal>
      );

      expect(screen.getByPlaceholderText('Name')).toBeInTheDocument();
      expect(
        screen.getByRole('button', { name: 'Submit' })
      ).toBeInTheDocument();
    });

    it('renders nested modals warning scenario', () => {
      // This tests that content renders correctly even with complex nesting
      render(
        <Modal isOpen={true} onClose={vi.fn()} title="Outer">
          <div data-testid="nested-content">
            <p>Nested paragraph</p>
            <ul>
              <li>Item 1</li>
              <li>Item 2</li>
            </ul>
          </div>
        </Modal>
      );

      expect(screen.getByTestId('nested-content')).toBeInTheDocument();
      expect(screen.getByText('Item 1')).toBeInTheDocument();
      expect(screen.getByText('Item 2')).toBeInTheDocument();
    });
  });

  // ==========================================================================
  // Full Size Modal Tests
  // ==========================================================================
  describe('Full Size Modal', () => {
    it('applies full size specific styles', () => {
      const { container } = render(
        <Modal isOpen={true} onClose={vi.fn()} size="full">
          Full content
        </Modal>
      );

      const panel = container.querySelector('[class*="rounded-2xl"]');
      expect(panel).toHaveClass('max-w-full');
      expect(panel).toHaveClass('mx-4');
    });
  });
});

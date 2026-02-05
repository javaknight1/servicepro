import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import React from 'react';
import {
  DropdownMenu,
  DropdownMenuItem,
  DropdownMenuDivider,
} from '../DropdownMenu';

// Mock createPortal
vi.mock('react-dom', async () => {
  const actual = await vi.importActual('react-dom');
  return {
    ...actual,
    createPortal: (children: React.ReactNode) => children,
  };
});

describe('DropdownMenu', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('rendering', () => {
    it('should render trigger button', () => {
      render(<DropdownMenu>{() => <div>Menu Content</div>}</DropdownMenu>);

      expect(screen.getByRole('button')).toBeInTheDocument();
    });

    it('should render with custom button label', () => {
      render(
        <DropdownMenu buttonLabel="Custom Actions">
          {() => <div>Menu Content</div>}
        </DropdownMenu>
      );

      expect(screen.getByLabelText('Custom Actions')).toBeInTheDocument();
    });

    it('should not show menu content initially', () => {
      render(
        <DropdownMenu>
          {() => <div data-testid="menu-content">Menu Content</div>}
        </DropdownMenu>
      );

      expect(screen.queryByTestId('menu-content')).not.toBeInTheDocument();
    });
  });

  describe('opening and closing', () => {
    it('should open menu when button is clicked', async () => {
      render(
        <DropdownMenu>
          {() => <div data-testid="menu-content">Menu Content</div>}
        </DropdownMenu>
      );

      fireEvent.click(screen.getByRole('button'));

      await waitFor(() => {
        expect(screen.getByTestId('menu-content')).toBeInTheDocument();
      });
    });

    it('should close menu when button is clicked again', async () => {
      render(
        <DropdownMenu>
          {() => <div data-testid="menu-content">Menu Content</div>}
        </DropdownMenu>
      );

      const button = screen.getByRole('button');

      // Open
      fireEvent.click(button);
      await waitFor(() => {
        expect(screen.getByTestId('menu-content')).toBeInTheDocument();
      });

      // Close
      fireEvent.click(button);
      await waitFor(() => {
        expect(screen.queryByTestId('menu-content')).not.toBeInTheDocument();
      });
    });

    it('should close menu when close callback is called', async () => {
      render(
        <DropdownMenu>
          {(close) => (
            <button data-testid="close-btn" onClick={close}>
              Close
            </button>
          )}
        </DropdownMenu>
      );

      // Open menu
      fireEvent.click(screen.getByRole('button', { name: 'Actions' }));

      await waitFor(() => {
        expect(screen.getByTestId('close-btn')).toBeInTheDocument();
      });

      // Click close button
      fireEvent.click(screen.getByTestId('close-btn'));

      await waitFor(() => {
        expect(screen.queryByTestId('close-btn')).not.toBeInTheDocument();
      });
    });

    it('should close menu when clicking outside', async () => {
      render(
        <div>
          <div data-testid="outside">Outside</div>
          <DropdownMenu>
            {() => <div data-testid="menu-content">Menu Content</div>}
          </DropdownMenu>
        </div>
      );

      // Open menu
      fireEvent.click(screen.getByRole('button'));

      await waitFor(() => {
        expect(screen.getByTestId('menu-content')).toBeInTheDocument();
      });

      // Click outside
      fireEvent.mouseDown(screen.getByTestId('outside'));

      await waitFor(() => {
        expect(screen.queryByTestId('menu-content')).not.toBeInTheDocument();
      });
    });
  });

  describe('accessibility', () => {
    it('should have aria-expanded attribute', () => {
      render(<DropdownMenu>{() => <div>Menu</div>}</DropdownMenu>);

      const button = screen.getByRole('button');
      expect(button).toHaveAttribute('aria-expanded', 'false');

      fireEvent.click(button);
      expect(button).toHaveAttribute('aria-expanded', 'true');
    });

    it('should have aria-haspopup attribute', () => {
      render(<DropdownMenu>{() => <div>Menu</div>}</DropdownMenu>);

      expect(screen.getByRole('button')).toHaveAttribute(
        'aria-haspopup',
        'menu'
      );
    });

    it('should have role menu on dropdown', async () => {
      render(<DropdownMenu>{() => <div>Menu Content</div>}</DropdownMenu>);

      fireEvent.click(screen.getByRole('button'));

      await waitFor(() => {
        expect(screen.getByRole('menu')).toBeInTheDocument();
      });
    });
  });

  describe('event propagation', () => {
    it('should stop propagation on button click', () => {
      const parentClickHandler = vi.fn();

      render(
        <div onClick={parentClickHandler}>
          <DropdownMenu>{() => <div>Menu</div>}</DropdownMenu>
        </div>
      );

      fireEvent.click(screen.getByRole('button'));

      expect(parentClickHandler).not.toHaveBeenCalled();
    });
  });
});

describe('DropdownMenuItem', () => {
  describe('rendering', () => {
    it('should render label and icon', () => {
      const handleClick = vi.fn();

      render(
        <DropdownMenuItem
          onClick={handleClick}
          icon={<span data-testid="item-icon">Icon</span>}
          label="Edit"
        />
      );

      expect(screen.getByTestId('item-icon')).toBeInTheDocument();
      expect(screen.getByText('Edit')).toBeInTheDocument();
    });

    it('should have menuitem role', () => {
      render(
        <DropdownMenuItem
          onClick={vi.fn()}
          icon={<span>Icon</span>}
          label="Action"
        />
      );

      expect(screen.getByRole('menuitem')).toBeInTheDocument();
    });
  });

  describe('variants', () => {
    it('should apply default variant styling', () => {
      render(
        <DropdownMenuItem
          onClick={vi.fn()}
          icon={<span>Icon</span>}
          label="Default"
        />
      );

      const button = screen.getByRole('menuitem');
      expect(button).toHaveClass('text-neutral-700');
    });

    it('should apply success variant styling', () => {
      render(
        <DropdownMenuItem
          onClick={vi.fn()}
          icon={<span>Icon</span>}
          label="Success"
          variant="success"
        />
      );

      const button = screen.getByRole('menuitem');
      expect(button).toHaveClass('text-green-700');
    });

    it('should apply danger variant styling', () => {
      render(
        <DropdownMenuItem
          onClick={vi.fn()}
          icon={<span>Icon</span>}
          label="Delete"
          variant="danger"
        />
      );

      const button = screen.getByRole('menuitem');
      expect(button).toHaveClass('text-red-700');
    });
  });

  describe('disabled state', () => {
    it('should be disabled when disabled prop is true', () => {
      render(
        <DropdownMenuItem
          onClick={vi.fn()}
          icon={<span>Icon</span>}
          label="Disabled"
          disabled
        />
      );

      expect(screen.getByRole('menuitem')).toBeDisabled();
    });

    it('should apply disabled styling', () => {
      render(
        <DropdownMenuItem
          onClick={vi.fn()}
          icon={<span>Icon</span>}
          label="Disabled"
          disabled
        />
      );

      expect(screen.getByRole('menuitem')).toHaveClass(
        'text-neutral-400',
        'cursor-not-allowed'
      );
    });

    it('should not call onClick when disabled', () => {
      const handleClick = vi.fn();

      render(
        <DropdownMenuItem
          onClick={handleClick}
          icon={<span>Icon</span>}
          label="Disabled"
          disabled
        />
      );

      fireEvent.click(screen.getByRole('menuitem'));
      expect(handleClick).not.toHaveBeenCalled();
    });
  });

  describe('loading state', () => {
    it('should be disabled when loading', () => {
      render(
        <DropdownMenuItem
          onClick={vi.fn()}
          icon={<span>Icon</span>}
          label="Loading"
          isLoading
        />
      );

      expect(screen.getByRole('menuitem')).toBeDisabled();
    });

    it('should show loading spinner when loading', () => {
      const { container } = render(
        <DropdownMenuItem
          onClick={vi.fn()}
          icon={<span data-testid="icon">Icon</span>}
          label="Loading"
          isLoading
        />
      );

      // Icon should not be visible when loading
      expect(screen.queryByTestId('icon')).not.toBeInTheDocument();
      // Loading spinner should be visible
      expect(container.querySelector('.animate-spin')).toBeInTheDocument();
    });
  });

  describe('click handler', () => {
    it('should call onClick when clicked', () => {
      const handleClick = vi.fn();

      render(
        <DropdownMenuItem
          onClick={handleClick}
          icon={<span>Icon</span>}
          label="Click Me"
        />
      );

      fireEvent.click(screen.getByRole('menuitem'));
      expect(handleClick).toHaveBeenCalledTimes(1);
    });
  });
});

describe('DropdownMenuDivider', () => {
  it('should render a divider', () => {
    const { container } = render(<DropdownMenuDivider />);

    const divider = container.firstChild;
    expect(divider).toHaveClass('border-t', 'border-neutral-200', 'my-1');
  });
});

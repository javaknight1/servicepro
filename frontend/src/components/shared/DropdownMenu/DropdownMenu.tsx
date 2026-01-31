import { useState, useRef, useEffect, useCallback, ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { MoreVertical } from 'lucide-react';

interface DropdownMenuProps {
  children: (close: () => void) => ReactNode;
  buttonLabel?: string;
}

export function DropdownMenu({
  children,
  buttonLabel = 'Actions',
}: DropdownMenuProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [position, setPosition] = useState({ top: 0, left: 0 });
  const buttonRef = useRef<HTMLButtonElement>(null);
  const menuRef = useRef<HTMLDivElement>(null);

  const updatePosition = useCallback(() => {
    if (buttonRef.current) {
      const rect = buttonRef.current.getBoundingClientRect();
      const menuHeight = menuRef.current?.offsetHeight || 200;
      const viewportHeight = window.innerHeight;

      // Check if menu would overflow below the viewport
      const spaceBelow = viewportHeight - rect.bottom;
      const spaceAbove = rect.top;

      let top: number;
      if (spaceBelow < menuHeight && spaceAbove > spaceBelow) {
        // Open upward
        top = rect.top - menuHeight + window.scrollY;
      } else {
        // Open downward
        top = rect.bottom + window.scrollY;
      }

      // Position menu to align with the right edge of the button
      const left = rect.right - 192 + window.scrollX; // 192px = w-48 (menu width)

      setPosition({
        top: Math.max(8, top),
        left: Math.max(8, left),
      });
    }
  }, []);

  useEffect(() => {
    if (isOpen) {
      updatePosition();
      window.addEventListener('scroll', updatePosition, true);
      window.addEventListener('resize', updatePosition);
      return () => {
        window.removeEventListener('scroll', updatePosition, true);
        window.removeEventListener('resize', updatePosition);
      };
    }
  }, [isOpen, updatePosition]);

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        menuRef.current &&
        !menuRef.current.contains(event.target as Node) &&
        buttonRef.current &&
        !buttonRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    }

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
      return () =>
        document.removeEventListener('mousedown', handleClickOutside);
    }
  }, [isOpen]);

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    setIsOpen(!isOpen);
  };

  const close = () => setIsOpen(false);

  return (
    <>
      <button
        ref={buttonRef}
        onClick={handleClick}
        className="p-1 rounded-md hover:bg-neutral-100 focus:outline-none focus:ring-2 focus:ring-primary-500"
        aria-label={buttonLabel}
        aria-expanded={isOpen}
        aria-haspopup="menu"
      >
        <MoreVertical className="h-5 w-5 text-neutral-500" />
      </button>

      {isOpen &&
        createPortal(
          <div
            ref={menuRef}
            role="menu"
            className="fixed w-48 bg-white rounded-md shadow-lg border border-neutral-200 z-50"
            style={{
              top: position.top,
              left: position.left,
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <div className="py-1">{children(close)}</div>
          </div>,
          document.body
        )}
    </>
  );
}

interface DropdownMenuItemProps {
  onClick: (e: React.MouseEvent) => void;
  disabled?: boolean;
  icon: ReactNode;
  label: string;
  variant?: 'default' | 'success' | 'danger';
  isLoading?: boolean;
}

export function DropdownMenuItem({
  onClick,
  disabled = false,
  icon,
  label,
  variant = 'default',
  isLoading = false,
}: DropdownMenuItemProps) {
  const variantClasses = {
    default: 'text-neutral-700 hover:bg-neutral-100',
    success: 'text-green-700 hover:bg-green-50',
    danger: 'text-red-700 hover:bg-red-50',
  };

  return (
    <button
      role="menuitem"
      onClick={onClick}
      disabled={disabled || isLoading}
      className={`w-full flex items-center gap-2 px-4 py-2 text-sm ${
        disabled
          ? 'text-neutral-400 cursor-not-allowed'
          : variantClasses[variant]
      }`}
    >
      {isLoading ? (
        <span className="h-4 w-4 animate-spin rounded-full border-2 border-current border-t-transparent" />
      ) : (
        icon
      )}
      {label}
    </button>
  );
}

export function DropdownMenuDivider() {
  return <div className="border-t border-neutral-200 my-1" />;
}

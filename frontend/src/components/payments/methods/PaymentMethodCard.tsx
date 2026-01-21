import { useState } from 'react';
import type { PaymentMethodCardProps, CardBrand } from '../../../types/payment';
import { formatCardBrand } from '../../../types/payment';
import { cn } from '../../../utils/cn';

// Card brand icons as SVG components
const CardBrandIcon = ({ brand }: { brand: CardBrand }) => {
  const iconClass = 'w-8 h-5';

  switch (brand) {
    case 'visa':
      return (
        <svg
          className={iconClass}
          viewBox="0 0 32 20"
          fill="none"
          aria-label="Visa"
        >
          <rect width="32" height="20" rx="2" fill="#1A1F71" />
          <path d="M13.2 13.5L14.4 6.5H16.1L14.9 13.5H13.2Z" fill="white" />
          <path
            d="M20.8 6.7C20.4 6.5 19.8 6.4 19.1 6.4C17.4 6.4 16.2 7.3 16.2 8.5C16.2 9.4 17 9.9 17.6 10.2C18.2 10.5 18.4 10.7 18.4 11C18.4 11.4 17.9 11.6 17.5 11.6C16.9 11.6 16.5 11.5 15.9 11.2L15.7 11.1L15.4 13C15.9 13.2 16.7 13.4 17.5 13.4C19.3 13.4 20.5 12.5 20.5 11.2C20.5 10.5 20 9.9 19.1 9.5C18.5 9.2 18.2 9 18.2 8.7C18.2 8.4 18.5 8.1 19.2 8.1C19.7 8.1 20.1 8.2 20.4 8.3L20.6 8.4L20.8 6.7Z"
            fill="white"
          />
          <path
            d="M24.1 6.5H22.7C22.3 6.5 22 6.6 21.8 7L19.3 13.5H21.1L21.5 12.3H23.6L23.8 13.5H25.4L24.1 6.5ZM22 10.7L22.7 8.7L23.1 10.7H22Z"
            fill="white"
          />
          <path
            d="M12 6.5L10.3 11.2L10.1 10.2C9.7 9 8.6 7.6 7.3 6.9L8.9 13.5H10.7L13.8 6.5H12Z"
            fill="white"
          />
          <path
            d="M8.5 6.5H5.7L5.6 6.7C7.7 7.2 9.1 8.5 9.6 10.1L9 7C8.9 6.6 8.6 6.5 8.2 6.5H8.5Z"
            fill="#F9A51A"
          />
        </svg>
      );
    case 'mastercard':
      return (
        <svg
          className={iconClass}
          viewBox="0 0 32 20"
          fill="none"
          aria-label="Mastercard"
        >
          <rect width="32" height="20" rx="2" fill="#F5F5F5" />
          <circle cx="12" cy="10" r="6" fill="#EB001B" />
          <circle cx="20" cy="10" r="6" fill="#F79E1B" />
          <path
            d="M16 5.5C17.5 6.7 18.5 8.2 18.5 10C18.5 11.8 17.5 13.3 16 14.5C14.5 13.3 13.5 11.8 13.5 10C13.5 8.2 14.5 6.7 16 5.5Z"
            fill="#FF5F00"
          />
        </svg>
      );
    case 'amex':
      return (
        <svg
          className={iconClass}
          viewBox="0 0 32 20"
          fill="none"
          aria-label="American Express"
        >
          <rect width="32" height="20" rx="2" fill="#006FCF" />
          <path
            d="M5 12.5V7.5H8.5L9 8.5L9.5 7.5H27V12C27 12 26.5 12.5 26 12.5H17.5L17 11.5V12.5H14.5V11C14.5 11 14 11.5 13 11.5H12.5V12.5H9L8.5 11.5L8 12.5H5Z"
            fill="white"
          />
          <path
            d="M5.5 8L4 11.5H5.5L5.8 10.8H7.2L7.5 11.5H9L7.5 8H5.5ZM6 9.5L6.5 8.5L7 9.5H6Z"
            fill="#006FCF"
          />
          <path
            d="M9 11.5V8H11L12 10L13 8H15V11.5H13.5V9.5L12.5 11.5H11.5L10.5 9.5V11.5H9Z"
            fill="#006FCF"
          />
          <path
            d="M15.5 11.5V8H19.5V9H17V9.5H19.5V10.5H17V11H19.5V11.5H15.5Z"
            fill="#006FCF"
          />
          <path
            d="M20 11.5V8H22C23.5 8 24 8.5 24 9.25C24 10 23.5 10.5 22.5 10.5L24 11.5H22.5L21.5 10.5V11.5H20ZM21.5 10H22C22.5 10 22.5 9.5 22 9.5H21.5V10Z"
            fill="#006FCF"
          />
          <path d="M24 8V11.5H25.5V8H24Z" fill="#006FCF" />
          <path
            d="M26 10C26 8.9 26.9 8 28 8C28.5 8 28.9 8.2 29.2 8.5L28.5 9.2C28.3 9 28.2 8.9 28 8.9C27.4 8.9 27 9.4 27 10C27 10.6 27.4 11.1 28 11.1C28.2 11.1 28.4 11 28.5 10.8L29.2 11.5C28.9 11.8 28.5 12 28 12C26.9 12 26 11.1 26 10Z"
            fill="#006FCF"
          />
        </svg>
      );
    case 'discover':
      return (
        <svg
          className={iconClass}
          viewBox="0 0 32 20"
          fill="none"
          aria-label="Discover"
        >
          <rect width="32" height="20" rx="2" fill="#F5F5F5" />
          <path
            d="M16 15C19.866 15 23 12.09 23 8.5C23 4.91 19.866 2 16 2V15Z"
            fill="#F47216"
          />
          <text x="4" y="12" fontSize="5" fill="#1A1F71" fontWeight="bold">
            DISCOVER
          </text>
        </svg>
      );
    default:
      return (
        <svg
          className={iconClass}
          viewBox="0 0 32 20"
          fill="none"
          aria-label="Card"
        >
          <rect width="32" height="20" rx="2" fill="#E5E7EB" />
          <rect x="4" y="6" width="8" height="4" rx="1" fill="#9CA3AF" />
          <rect x="4" y="12" width="12" height="2" rx="0.5" fill="#9CA3AF" />
        </svg>
      );
  }
};

// Bank icon
const BankIcon = () => (
  <svg
    className="w-8 h-5"
    viewBox="0 0 32 20"
    fill="none"
    aria-label="Bank Account"
  >
    <rect width="32" height="20" rx="2" fill="#E5E7EB" />
    <path
      d="M16 4L8 8V9H24V8L16 4ZM10 10V14H12V10H10ZM14 10V14H18V10H14ZM20 10V14H22V10H20ZM8 15V16H24V15H8Z"
      fill="#6B7280"
    />
  </svg>
);

export function PaymentMethodCard({
  paymentMethod,
  isSelected = false,
  onSelect,
  onEdit,
  onDelete,
  onSetDefault,
  disabled = false,
  className,
}: PaymentMethodCardProps) {
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);

  const isCard = paymentMethod.type === 'card';
  const isBank =
    paymentMethod.type === 'bank_account' ||
    paymentMethod.type === 'us_bank_account';

  const handleSelect = () => {
    if (!disabled && onSelect) {
      onSelect(paymentMethod);
    }
  };

  const handleEdit = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onEdit) {
      onEdit(paymentMethod);
    }
  };

  const handleDeleteClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    setShowDeleteConfirm(true);
  };

  const handleDeleteConfirm = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onDelete) {
      setIsDeleting(true);
      try {
        onDelete(paymentMethod);
      } finally {
        setIsDeleting(false);
        setShowDeleteConfirm(false);
      }
    }
  };

  const handleDeleteCancel = (e: React.MouseEvent) => {
    e.stopPropagation();
    setShowDeleteConfirm(false);
  };

  const handleSetDefault = (e: React.MouseEvent) => {
    e.stopPropagation();
    if (onSetDefault && !paymentMethod.isDefault) {
      onSetDefault(paymentMethod);
    }
  };

  // Format expiry date
  const formatExpiry = () => {
    if (!paymentMethod.cardExpMonth || !paymentMethod.cardExpYear) return null;
    const month = paymentMethod.cardExpMonth.toString().padStart(2, '0');
    const year = paymentMethod.cardExpYear.toString().slice(-2);
    return `${month}/${year}`;
  };

  return (
    <div
      role="button"
      tabIndex={disabled ? -1 : 0}
      aria-selected={isSelected}
      aria-disabled={disabled}
      aria-label={`${paymentMethod.displayName}${
        paymentMethod.isDefault ? ', default payment method' : ''
      }`}
      onClick={handleSelect}
      onKeyDown={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          handleSelect();
        }
      }}
      className={cn(
        'relative rounded-lg border p-4 transition-all',
        'focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2',
        isSelected
          ? 'border-primary-500 bg-primary-50 ring-1 ring-primary-500'
          : 'border-neutral-200 bg-white hover:border-neutral-300',
        disabled && 'cursor-not-allowed opacity-50',
        !disabled && 'cursor-pointer',
        className
      )}
    >
      {/* Default Badge */}
      {paymentMethod.isDefault && (
        <span className="absolute right-2 top-2 rounded-full bg-primary-100 px-2 py-0.5 text-xs font-medium text-primary-700">
          Default
        </span>
      )}

      <div className="flex items-start gap-3">
        {/* Card/Bank Icon */}
        <div className="flex-shrink-0 pt-0.5">
          {isCard && paymentMethod.cardBrand ? (
            <CardBrandIcon brand={paymentMethod.cardBrand} />
          ) : isBank ? (
            <BankIcon />
          ) : (
            <CardBrandIcon brand="unknown" />
          )}
        </div>

        {/* Details */}
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-2">
            <span className="font-medium text-neutral-900 truncate">
              {paymentMethod.nickname || paymentMethod.displayName}
            </span>
          </div>

          {/* Card details */}
          {isCard && (
            <div className="mt-1 text-sm text-neutral-600">
              {paymentMethod.cardBrand && (
                <span>{formatCardBrand(paymentMethod.cardBrand)}</span>
              )}{' '}
              {paymentMethod.cardLast4 && (
                <span>ending in {paymentMethod.cardLast4}</span>
              )}
              {formatExpiry() && (
                <span className="ml-2 text-neutral-500">
                  Exp: {formatExpiry()}
                </span>
              )}
            </div>
          )}

          {/* Bank details */}
          {isBank && (
            <div className="mt-1 text-sm text-neutral-600">
              {paymentMethod.bankName && <span>{paymentMethod.bankName}</span>}{' '}
              {paymentMethod.bankLast4 && (
                <span>ending in {paymentMethod.bankLast4}</span>
              )}
            </div>
          )}

          {/* Status warnings */}
          {paymentMethod.isExpired && (
            <div className="mt-2 flex items-center gap-1 text-sm text-red-600">
              <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
                <path
                  fillRule="evenodd"
                  d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                  clipRule="evenodd"
                />
              </svg>
              Card expired
            </div>
          )}

          {!paymentMethod.isExpired && paymentMethod.isExpiringSoon && (
            <div className="mt-2 flex items-center gap-1 text-sm text-amber-600">
              <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
                <path
                  fillRule="evenodd"
                  d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                  clipRule="evenodd"
                />
              </svg>
              Expiring soon
            </div>
          )}

          {!paymentMethod.isActive && (
            <div className="mt-2 flex items-center gap-1 text-sm text-red-600">
              <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 20 20">
                <path
                  fillRule="evenodd"
                  d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                  clipRule="evenodd"
                />
              </svg>
              Inactive
            </div>
          )}
        </div>

        {/* Actions */}
        {(onEdit || onDelete || onSetDefault) && !showDeleteConfirm && (
          <div className="flex items-center gap-1 flex-shrink-0">
            {onSetDefault && !paymentMethod.isDefault && (
              <button
                type="button"
                onClick={handleSetDefault}
                disabled={disabled}
                className="rounded p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600 focus:outline-none focus:ring-2 focus:ring-primary-500"
                aria-label="Set as default"
                title="Set as default"
              >
                <svg
                  className="h-4 w-4"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M11.049 2.927c.3-.921 1.603-.921 1.902 0l1.519 4.674a1 1 0 00.95.69h4.915c.969 0 1.371 1.24.588 1.81l-3.976 2.888a1 1 0 00-.363 1.118l1.518 4.674c.3.922-.755 1.688-1.538 1.118l-3.976-2.888a1 1 0 00-1.176 0l-3.976 2.888c-.783.57-1.838-.197-1.538-1.118l1.518-4.674a1 1 0 00-.363-1.118l-3.976-2.888c-.784-.57-.38-1.81.588-1.81h4.914a1 1 0 00.951-.69l1.519-4.674z"
                  />
                </svg>
              </button>
            )}

            {onEdit && (
              <button
                type="button"
                onClick={handleEdit}
                disabled={disabled}
                className="rounded p-1.5 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600 focus:outline-none focus:ring-2 focus:ring-primary-500"
                aria-label="Edit payment method"
                title="Edit"
              >
                <svg
                  className="h-4 w-4"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"
                  />
                </svg>
              </button>
            )}

            {onDelete && (
              <button
                type="button"
                onClick={handleDeleteClick}
                disabled={disabled}
                className="rounded p-1.5 text-neutral-400 hover:bg-red-50 hover:text-red-600 focus:outline-none focus:ring-2 focus:ring-red-500"
                aria-label="Delete payment method"
                title="Delete"
              >
                <svg
                  className="h-4 w-4"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                  />
                </svg>
              </button>
            )}
          </div>
        )}

        {/* Delete confirmation */}
        {showDeleteConfirm && (
          <div className="flex items-center gap-2 flex-shrink-0">
            <span className="text-sm text-neutral-600">Delete?</span>
            <button
              type="button"
              onClick={handleDeleteConfirm}
              disabled={isDeleting}
              className="rounded bg-red-600 px-2 py-1 text-xs font-medium text-white hover:bg-red-700 focus:outline-none focus:ring-2 focus:ring-red-500"
            >
              {isDeleting ? 'Deleting...' : 'Yes'}
            </button>
            <button
              type="button"
              onClick={handleDeleteCancel}
              disabled={isDeleting}
              className="rounded bg-neutral-200 px-2 py-1 text-xs font-medium text-neutral-700 hover:bg-neutral-300 focus:outline-none focus:ring-2 focus:ring-neutral-500"
            >
              No
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

import { useCallback, useId } from 'react';
import { cn } from '../../utils/cn';
import type {
  BillingAddressProps,
  BillingAddressData,
} from '../../types/payment';
import { COUNTRIES, US_STATES } from '../../types/payment';
import { MapPin } from 'lucide-react';

// =============================================================================
// Billing Address Component
// =============================================================================

export function BillingAddress({
  value,
  errors = {},
  onChange,
  disabled = false,
  className,
}: BillingAddressProps) {
  const baseId = useId();

  const handleChange = useCallback(
    (field: keyof BillingAddressData) =>
      (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
        onChange({
          ...value,
          [field]: e.target.value,
        });
      },
    [onChange, value]
  );

  const inputClasses = (hasError: boolean) =>
    cn(
      'w-full px-3 py-2 border rounded-lg text-neutral-900 placeholder:text-neutral-400',
      'focus:outline-none focus:ring-2 focus:ring-offset-0 transition-all',
      'disabled:bg-neutral-100 disabled:cursor-not-allowed',
      hasError
        ? 'border-error-500 focus:ring-error-500 focus:border-error-500'
        : 'border-neutral-300 focus:ring-primary-500 focus:border-primary-500'
    );

  const labelClasses = 'block mb-1.5 text-sm font-medium text-neutral-700';

  const showStateDropdown = value.country === 'US';

  return (
    <div className={cn('space-y-4', className)}>
      {/* Section Header */}
      <div className="flex items-center gap-2 text-neutral-700">
        <MapPin className="h-4 w-4" aria-hidden="true" />
        <h3 className="text-sm font-medium">Billing Address</h3>
      </div>

      {/* Name and Email Row */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {/* Full Name */}
        <div>
          <label htmlFor={`${baseId}-name`} className={labelClasses}>
            Full Name <span className="text-error-500">*</span>
          </label>
          <input
            id={`${baseId}-name`}
            type="text"
            value={value.name}
            onChange={handleChange('name')}
            disabled={disabled}
            placeholder="John Doe"
            autoComplete="name"
            aria-required="true"
            aria-invalid={!!errors.name}
            aria-describedby={errors.name ? `${baseId}-name-error` : undefined}
            className={inputClasses(!!errors.name)}
          />
          {errors.name && (
            <p
              id={`${baseId}-name-error`}
              className="mt-1.5 text-sm text-error-600"
              role="alert"
            >
              {errors.name}
            </p>
          )}
        </div>

        {/* Email */}
        <div>
          <label htmlFor={`${baseId}-email`} className={labelClasses}>
            Email <span className="text-error-500">*</span>
          </label>
          <input
            id={`${baseId}-email`}
            type="email"
            value={value.email}
            onChange={handleChange('email')}
            disabled={disabled}
            placeholder="john@example.com"
            autoComplete="email"
            aria-required="true"
            aria-invalid={!!errors.email}
            aria-describedby={
              errors.email ? `${baseId}-email-error` : undefined
            }
            className={inputClasses(!!errors.email)}
          />
          {errors.email && (
            <p
              id={`${baseId}-email-error`}
              className="mt-1.5 text-sm text-error-600"
              role="alert"
            >
              {errors.email}
            </p>
          )}
        </div>
      </div>

      {/* Phone (Optional) */}
      <div>
        <label htmlFor={`${baseId}-phone`} className={labelClasses}>
          Phone Number <span className="text-neutral-400">(optional)</span>
        </label>
        <input
          id={`${baseId}-phone`}
          type="tel"
          value={value.phone || ''}
          onChange={handleChange('phone')}
          disabled={disabled}
          placeholder="+1 (555) 123-4567"
          autoComplete="tel"
          aria-invalid={!!errors.phone}
          aria-describedby={errors.phone ? `${baseId}-phone-error` : undefined}
          className={inputClasses(!!errors.phone)}
        />
        {errors.phone && (
          <p
            id={`${baseId}-phone-error`}
            className="mt-1.5 text-sm text-error-600"
            role="alert"
          >
            {errors.phone}
          </p>
        )}
      </div>

      {/* Address Line 1 */}
      <div>
        <label htmlFor={`${baseId}-line1`} className={labelClasses}>
          Street Address <span className="text-error-500">*</span>
        </label>
        <input
          id={`${baseId}-line1`}
          type="text"
          value={value.line1}
          onChange={handleChange('line1')}
          disabled={disabled}
          placeholder="123 Main Street"
          autoComplete="address-line1"
          aria-required="true"
          aria-invalid={!!errors.line1}
          aria-describedby={errors.line1 ? `${baseId}-line1-error` : undefined}
          className={inputClasses(!!errors.line1)}
        />
        {errors.line1 && (
          <p
            id={`${baseId}-line1-error`}
            className="mt-1.5 text-sm text-error-600"
            role="alert"
          >
            {errors.line1}
          </p>
        )}
      </div>

      {/* Address Line 2 */}
      <div>
        <label htmlFor={`${baseId}-line2`} className={labelClasses}>
          Apt, Suite, etc. <span className="text-neutral-400">(optional)</span>
        </label>
        <input
          id={`${baseId}-line2`}
          type="text"
          value={value.line2 || ''}
          onChange={handleChange('line2')}
          disabled={disabled}
          placeholder="Apt 4B"
          autoComplete="address-line2"
          aria-invalid={!!errors.line2}
          aria-describedby={errors.line2 ? `${baseId}-line2-error` : undefined}
          className={inputClasses(!!errors.line2)}
        />
        {errors.line2 && (
          <p
            id={`${baseId}-line2-error`}
            className="mt-1.5 text-sm text-error-600"
            role="alert"
          >
            {errors.line2}
          </p>
        )}
      </div>

      {/* City, State, Postal Code Row */}
      <div className="grid grid-cols-2 gap-4 sm:grid-cols-3">
        {/* City */}
        <div className="col-span-2 sm:col-span-1">
          <label htmlFor={`${baseId}-city`} className={labelClasses}>
            City <span className="text-error-500">*</span>
          </label>
          <input
            id={`${baseId}-city`}
            type="text"
            value={value.city}
            onChange={handleChange('city')}
            disabled={disabled}
            placeholder="San Francisco"
            autoComplete="address-level2"
            aria-required="true"
            aria-invalid={!!errors.city}
            aria-describedby={errors.city ? `${baseId}-city-error` : undefined}
            className={inputClasses(!!errors.city)}
          />
          {errors.city && (
            <p
              id={`${baseId}-city-error`}
              className="mt-1.5 text-sm text-error-600"
              role="alert"
            >
              {errors.city}
            </p>
          )}
        </div>

        {/* State/Province */}
        <div>
          <label htmlFor={`${baseId}-state`} className={labelClasses}>
            State <span className="text-error-500">*</span>
          </label>
          {showStateDropdown ? (
            <select
              id={`${baseId}-state`}
              value={value.state}
              onChange={handleChange('state')}
              disabled={disabled}
              autoComplete="address-level1"
              aria-required="true"
              aria-invalid={!!errors.state}
              aria-describedby={
                errors.state ? `${baseId}-state-error` : undefined
              }
              className={inputClasses(!!errors.state)}
            >
              <option value="">Select state</option>
              {US_STATES.map((state) => (
                <option key={state.code} value={state.code}>
                  {state.name}
                </option>
              ))}
            </select>
          ) : (
            <input
              id={`${baseId}-state`}
              type="text"
              value={value.state}
              onChange={handleChange('state')}
              disabled={disabled}
              placeholder="Province/Region"
              autoComplete="address-level1"
              aria-required="true"
              aria-invalid={!!errors.state}
              aria-describedby={
                errors.state ? `${baseId}-state-error` : undefined
              }
              className={inputClasses(!!errors.state)}
            />
          )}
          {errors.state && (
            <p
              id={`${baseId}-state-error`}
              className="mt-1.5 text-sm text-error-600"
              role="alert"
            >
              {errors.state}
            </p>
          )}
        </div>

        {/* Postal Code */}
        <div>
          <label htmlFor={`${baseId}-postal`} className={labelClasses}>
            ZIP Code <span className="text-error-500">*</span>
          </label>
          <input
            id={`${baseId}-postal`}
            type="text"
            value={value.postalCode}
            onChange={handleChange('postalCode')}
            disabled={disabled}
            placeholder="94102"
            autoComplete="postal-code"
            aria-required="true"
            aria-invalid={!!errors.postalCode}
            aria-describedby={
              errors.postalCode ? `${baseId}-postal-error` : undefined
            }
            className={inputClasses(!!errors.postalCode)}
          />
          {errors.postalCode && (
            <p
              id={`${baseId}-postal-error`}
              className="mt-1.5 text-sm text-error-600"
              role="alert"
            >
              {errors.postalCode}
            </p>
          )}
        </div>
      </div>

      {/* Country */}
      <div>
        <label htmlFor={`${baseId}-country`} className={labelClasses}>
          Country <span className="text-error-500">*</span>
        </label>
        <select
          id={`${baseId}-country`}
          value={value.country}
          onChange={handleChange('country')}
          disabled={disabled}
          autoComplete="country"
          aria-required="true"
          aria-invalid={!!errors.country}
          aria-describedby={
            errors.country ? `${baseId}-country-error` : undefined
          }
          className={inputClasses(!!errors.country)}
        >
          {COUNTRIES.map((country) => (
            <option key={country.code} value={country.code}>
              {country.name}
            </option>
          ))}
        </select>
        {errors.country && (
          <p
            id={`${baseId}-country-error`}
            className="mt-1.5 text-sm text-error-600"
            role="alert"
          >
            {errors.country}
          </p>
        )}
      </div>
    </div>
  );
}

BillingAddress.displayName = 'BillingAddress';

export default BillingAddress;

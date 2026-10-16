import type { TaxRate } from '@services/taxRateService';
import { taxRateToPercent } from '@/utils/money';

/** The fields the picker shows for a configured rate. */
export type TaxRateOption = Pick<TaxRate, 'id' | 'name' | 'rate'>;

/** The picker's value: a configured rate, or a typed percent when
 * `taxRateId` is '' ("Custom rate"). */
export interface TaxRateSelection {
  taxRateId: string;
  /** Percent ("8.25"); '' leaves the rate to the tenant default. */
  percent: string;
}

export interface TaxRatePickerProps extends TaxRateSelection {
  /** Configured rates to pick from; empty shows only the typed-rate input. */
  taxRates: TaxRateOption[];
  /** The tenant's default rate, marked in the list. */
  defaultTaxRateId?: string;
  /** Placeholder for the typed-rate input. */
  placeholder?: string;
  /** Hint under the field. */
  hint?: string;
  onChange: (next: TaxRateSelection) => void;
}

const fieldClass =
  'w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500';

/** "Name (8.25%)" for a configured rate. */
function optionLabel(rate: TaxRateOption, isDefault: boolean): string {
  const label = `${rate.name} (${taxRateToPercent(rate.rate)}%)`;
  return isDefault ? `${label} · default` : label;
}

/**
 * Tax rate field for the invoice and quote forms (T237): a select of the
 * configured tax rates with a "Custom rate" option that reveals a percent
 * input. With no configured rates it is just the percent input.
 */
export function TaxRatePicker({
  taxRates,
  taxRateId,
  percent,
  defaultTaxRateId,
  placeholder,
  hint,
  onChange,
}: TaxRatePickerProps) {
  const percentInput = (id: string, label: string) => (
    <>
      <label
        htmlFor={id}
        className="block text-sm font-medium text-neutral-700 mb-1"
      >
        {label}
      </label>
      <input
        type="number"
        id={id}
        name="tax_rate"
        value={percent}
        onChange={(e) => onChange({ taxRateId: '', percent: e.target.value })}
        min="0"
        max="100"
        step="0.01"
        placeholder={placeholder}
        className={fieldClass}
      />
    </>
  );

  if (taxRates.length === 0) {
    return (
      <div>
        {percentInput('tax_rate', 'Tax Rate (%)')}
        {hint && <p className="mt-1 text-xs text-neutral-500">{hint}</p>}
      </div>
    );
  }

  const handleSelect = (id: string) => {
    const rate = taxRates.find((r) => r.id === id);
    onChange(
      rate
        ? { taxRateId: rate.id, percent: taxRateToPercent(rate.rate) }
        : { taxRateId: '', percent: percent === '' ? '0' : percent }
    );
  };

  return (
    <div className="space-y-2">
      <div>
        <label
          htmlFor="tax_rate_id"
          className="block text-sm font-medium text-neutral-700 mb-1"
        >
          Tax Rate
        </label>
        <select
          id="tax_rate_id"
          name="tax_rate_id"
          value={taxRateId}
          onChange={(e) => handleSelect(e.target.value)}
          className={fieldClass}
        >
          {taxRates.map((rate) => (
            <option key={rate.id} value={rate.id}>
              {optionLabel(rate, rate.id === defaultTaxRateId)}
            </option>
          ))}
          <option value="">Custom rate</option>
        </select>
      </div>
      {taxRateId === '' && (
        <div>{percentInput('tax_rate', 'Custom rate (%)')}</div>
      )}
      {hint && <p className="text-xs text-neutral-500">{hint}</p>}
    </div>
  );
}

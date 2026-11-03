import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import {
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Input,
} from '@components/shared';
import { useTenantStore } from '@store';
import {
  paymentTermService,
  paymentTermsQueryKey,
} from '@services/paymentTermService';
import { taxRateService, taxRatesQueryKey } from '@services/taxRateService';
import { getFieldErrors, getTenantErrorMessage } from '@/utils/error';
import { taxRateToPercent } from '@/utils/money';
import {
  CURRENCY_OPTIONS,
  DEFAULT_PREFIXES,
  MAX_NUMBER_PREFIX_LENGTH,
  mergeSettings,
  prefixError,
} from '@/utils/tenantSettings';
import type { Tenant, TenantSettings } from '@/types/tenant';

const selectClass =
  'w-full px-3 py-2 border rounded-lg text-sm focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-primary-500 disabled:bg-neutral-100';

interface FormState {
  currency: string;
  invoicePrefix: string;
  quotePrefix: string;
  jobPrefix: string;
  defaultPaymentTermId: string;
  defaultTaxRateId: string;
  laborRate: string;
}

/** An amount like 85 or 85.50 (the API allows 0-10000, 2 decimals). */
const LABOR_RATE_PATTERN = /^\d{1,5}(\.\d{1,2})?$/;

function formFrom(settings: TenantSettings | undefined): FormState {
  return {
    currency: settings?.currency || 'USD',
    invoicePrefix: settings?.invoicePrefix ?? '',
    quotePrefix: settings?.quotePrefix ?? '',
    jobPrefix: settings?.jobPrefix ?? '',
    defaultPaymentTermId: settings?.defaultPaymentTermId ?? '',
    defaultTaxRateId: settings?.defaultTaxRateId ?? '',
    laborRate: settings?.laborRate ?? '',
  };
}

type PrefixKey = 'invoicePrefix' | 'quotePrefix' | 'jobPrefix';

const PREFIX_FIELDS: { key: PrefixKey; label: string }[] = [
  { key: 'invoicePrefix', label: 'Invoice prefix' },
  { key: 'quotePrefix', label: 'Quote prefix' },
  { key: 'jobPrefix', label: 'Job prefix' },
];

function FieldError({ message }: { message?: string }) {
  if (!message) return null;
  return <p className="mt-1.5 text-sm text-error-600">{message}</p>;
}

/**
 * Settings → Billing defaults (T235): currency, document number prefixes,
 * and the payment term and tax rate new invoices and quotes start with.
 * The API validates on save; its field errors show under each control.
 */
export function BillingDefaultsCard({
  tenant,
  canEdit,
}: {
  tenant: Tenant;
  canEdit: boolean;
}) {
  const { updateTenant } = useTenantStore();
  const [form, setForm] = useState<FormState>(() => formFrom(tenant.settings));
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    setForm(formFrom(tenant.settings));
  }, [tenant.settings]);

  const termsQuery = useQuery({
    queryKey: paymentTermsQueryKey,
    queryFn: () => paymentTermService.listPaymentTerms(),
  });
  const ratesQuery = useQuery({
    queryKey: taxRatesQueryKey,
    queryFn: () => taxRateService.listTaxRates(false),
  });
  const terms = termsQuery.data ?? [];
  // Inactive rates can't become the default (the API rejects them), but the
  // current default stays listed so the select shows it.
  const rates = (ratesQuery.data ?? []).filter(
    (r) => r.is_active || r.id === form.defaultTaxRateId
  );

  const set = <K extends keyof FormState>(key: K, value: FormState[K]) => {
    setForm((prev) => ({ ...prev, [key]: value }));
    setFieldErrors((prev) => {
      if (!(key in prev)) return prev;
      const next = { ...prev };
      delete next[key];
      return next;
    });
    setSaved(false);
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const local: Record<string, string> = {};
    for (const { key } of PREFIX_FIELDS) {
      const problem = prefixError(form[key]);
      if (problem) local[key] = problem;
    }
    const laborRate = form.laborRate.trim();
    if (laborRate && !LABOR_RATE_PATTERN.test(laborRate)) {
      local.laborRate = 'Enter an hourly amount like 85 or 85.50';
    }
    if (Object.keys(local).length > 0) {
      setFieldErrors(local);
      return;
    }

    setSaving(true);
    setError(null);
    setSaved(false);
    try {
      await updateTenant(tenant.id, {
        settings: mergeSettings(tenant, {
          currency: form.currency,
          invoicePrefix: form.invoicePrefix.trim() || undefined,
          quotePrefix: form.quotePrefix.trim() || undefined,
          jobPrefix: form.jobPrefix.trim() || undefined,
          defaultPaymentTermId: form.defaultPaymentTermId || undefined,
          defaultTaxRateId: form.defaultTaxRateId || undefined,
          laborRate: laborRate || undefined,
        }),
      });
      setFieldErrors({});
      setSaved(true);
    } catch (err: unknown) {
      const fields = getFieldErrors(err);
      setFieldErrors(fields);
      setError(
        Object.keys(fields).length > 0
          ? 'Some settings need fixing.'
          : getTenantErrorMessage(err, 'Failed to save billing defaults')
      );
    } finally {
      setSaving(false);
    }
  };

  const borderFor = (key: string) =>
    fieldErrors[key] ? 'border-error-500' : 'border-neutral-300';

  return (
    <Card variant="elevated" padding="lg">
      <CardHeader>
        <CardTitle>Billing defaults</CardTitle>
        <CardDescription>
          What new invoices and quotes start with. You can change them on any
          document.
        </CardDescription>
      </CardHeader>
      <CardContent>
        <form onSubmit={handleSubmit} className="mt-6 space-y-6" noValidate>
          {!canEdit && (
            <p className="text-sm text-neutral-600">
              Only organization admins can change these settings.
            </p>
          )}
          {error && (
            <div
              role="alert"
              className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700"
            >
              {error}
            </div>
          )}
          {saved && (
            <div className="rounded-lg border border-green-200 bg-green-50 p-3 text-sm text-green-700">
              Billing defaults saved
            </div>
          )}

          <div>
            <label
              htmlFor="billing-currency"
              className="mb-1.5 block text-sm font-medium text-neutral-700"
            >
              Currency
            </label>
            <select
              id="billing-currency"
              value={form.currency}
              onChange={(e) => set('currency', e.target.value)}
              disabled={!canEdit}
              className={`${selectClass} max-w-sm ${borderFor('currency')}`}
            >
              {CURRENCY_OPTIONS.map((c) => (
                <option key={c.value} value={c.value}>
                  {c.label}
                </option>
              ))}
            </select>
            <FieldError message={fieldErrors.currency} />
            {!fieldErrors.currency && (
              <p className="mt-1.5 text-sm text-neutral-500">
                New invoices and quotes are billed in this currency.
              </p>
            )}
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
            {PREFIX_FIELDS.map(({ key, label }) => (
              <Input
                key={key}
                id={`billing-${key}`}
                label={label}
                value={form[key]}
                onChange={(e) => set(key, e.target.value)}
                placeholder={DEFAULT_PREFIXES[key]}
                maxLength={MAX_NUMBER_PREFIX_LENGTH + 5}
                disabled={!canEdit}
                error={fieldErrors[key]}
                helperText={`Next number looks like ${
                  form[key].trim() || DEFAULT_PREFIXES[key]
                }000123`}
                fullWidth
              />
            ))}
          </div>

          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
            <div>
              <label
                htmlFor="billing-term"
                className="mb-1.5 block text-sm font-medium text-neutral-700"
              >
                Default payment terms
              </label>
              <select
                id="billing-term"
                value={form.defaultPaymentTermId}
                onChange={(e) => set('defaultPaymentTermId', e.target.value)}
                disabled={!canEdit}
                className={`${selectClass} ${borderFor('defaultPaymentTermId')}`}
              >
                <option value="">No default (set on each invoice)</option>
                {terms.map((t) => (
                  <option key={t.id} value={t.id}>
                    {t.name}
                  </option>
                ))}
              </select>
              <FieldError message={fieldErrors.defaultPaymentTermId} />
            </div>

            <div>
              <label
                htmlFor="billing-tax"
                className="mb-1.5 block text-sm font-medium text-neutral-700"
              >
                Default tax rate
              </label>
              <select
                id="billing-tax"
                value={form.defaultTaxRateId}
                onChange={(e) => set('defaultTaxRateId', e.target.value)}
                disabled={!canEdit}
                className={`${selectClass} ${borderFor('defaultTaxRateId')}`}
              >
                <option value="">No sales tax by default</option>
                {rates.map((r) => (
                  <option key={r.id} value={r.id}>
                    {r.name} · {taxRateToPercent(r.rate)}%
                  </option>
                ))}
              </select>
              <FieldError message={fieldErrors.defaultTaxRateId} />
              {!fieldErrors.defaultTaxRateId && (
                <p className="mt-1.5 text-sm text-neutral-500">
                  Manage rates under Tax rates.
                </p>
              )}
            </div>
          </div>

          <div className="max-w-sm">
            <Input
              id="billing-laborRate"
              label="Labor rate (per hour)"
              inputMode="decimal"
              value={form.laborRate}
              onChange={(e) => set('laborRate', e.target.value)}
              placeholder="85.00"
              disabled={!canEdit}
              error={fieldErrors.laborRate}
              helperText="Invoices generated from a job bill its time on site at this rate. Leave empty to bill the job's amount."
              fullWidth
            />
          </div>

          {canEdit && (
            <div className="flex justify-end space-x-3 border-t border-neutral-200 pt-4">
              <Button
                type="button"
                variant="outline"
                onClick={() => {
                  setForm(formFrom(tenant.settings));
                  setFieldErrors({});
                  setError(null);
                }}
              >
                Reset
              </Button>
              <Button type="submit" variant="primary" disabled={saving}>
                {saving ? 'Saving...' : 'Save billing defaults'}
              </Button>
            </div>
          )}
        </form>
      </CardContent>
    </Card>
  );
}

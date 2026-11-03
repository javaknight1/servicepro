import { useEffect, useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import {
  paymentTermService,
  paymentTermsQueryKey,
} from '@services/paymentTermService';
import { taxRateService, taxRatesQueryKey } from '@services/taxRateService';
import { getFieldErrors, getTenantErrorMessage } from '@/utils/error';
import { percentToTaxRate, taxRateToPercent } from '@/utils/money';
import {
  DEFAULT_PREFIXES,
  MAX_NUMBER_PREFIX_LENGTH,
  prefixError,
} from '@/utils/tenantSettings';
import type { Tenant } from '@/types/tenant';
import { ChoiceCard, FieldError, Hint, StepFooter, ToggleRow } from './parts';
import { fieldBorder, fieldClass, type StepProps } from './steps';

/** Tax select values besides an existing rate's ID. */
const NO_TAX = '';
const NEW_RATE = '__new__';

/** Onboarding 2 — how you bill (mockup 03): the defaults new customers and
 * invoices start with. */
export function BillingStep({
  tenant,
  canManageTaxRates,
  onContinue,
  onSkip,
  onBack,
  busy,
}: StepProps & { tenant: Tenant; canManageTaxRates: boolean }) {
  const queryClient = useQueryClient();
  const s = tenant.settings ?? {};
  // A fresh tenant (never saved this step) starts on consolidated monthly,
  // the mockup's default for the D7 buyer.
  const [consolidated, setConsolidated] = useState(
    s.defaultConsolidatedBilling ?? (s.onboardingStep ?? 0) <= 2
  );
  const [termId, setTermId] = useState(s.defaultPaymentTermId ?? '');
  const [taxChoice, setTaxChoice] = useState(s.defaultTaxRateId ?? NO_TAX);
  const [newRateName, setNewRateName] = useState('Sales tax');
  const [newRatePercent, setNewRatePercent] = useState('');
  const [prefix, setPrefix] = useState(s.invoicePrefix ?? '');
  const [poRequired, setPoRequired] = useState(s.defaultPoRequired ?? false);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);
  const [creatingRate, setCreatingRate] = useState(false);

  const termsQuery = useQuery({
    queryKey: paymentTermsQueryKey,
    queryFn: () => paymentTermService.listPaymentTerms(),
  });
  const ratesQuery = useQuery({
    queryKey: taxRatesQueryKey,
    queryFn: () => taxRateService.listTaxRates(false),
  });
  const terms = termsQuery.data ?? [];
  const rates = (ratesQuery.data ?? []).filter((r) => r.is_active);

  // Preselect Net 30 when nothing is set yet.
  const loadedTerms = termsQuery.data;
  useEffect(() => {
    if (termId || !loadedTerms) return;
    const net30 = loadedTerms.find((t) => t.days_until_due === 30);
    if (net30) setTermId(net30.id);
  }, [loadedTerms, termId]);

  const selectedTerm = terms.find((t) => t.id === termId);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const local: Record<string, string> = {};
    const pErr = prefixError(prefix);
    if (pErr) local.invoicePrefix = pErr;
    const pct = Number(newRatePercent);
    if (taxChoice === NEW_RATE) {
      if (!newRateName.trim()) local.newRateName = 'Name the rate';
      if (newRatePercent.trim() === '' || Number.isNaN(pct)) {
        local.newRatePercent = 'Enter a percentage';
      } else if (pct < 0 || pct > 100) {
        local.newRatePercent = 'Use a percentage between 0 and 100';
      }
    }
    setErrors(local);
    setError(null);
    if (Object.keys(local).length > 0) return;

    let taxRateId = taxChoice;
    try {
      if (taxChoice === NEW_RATE) {
        setCreatingRate(true);
        const created = await taxRateService.createTaxRate({
          name: newRateName.trim(),
          rate: percentToTaxRate(newRatePercent),
          is_active: true,
        });
        taxRateId = created.id;
        setTaxChoice(created.id);
        void queryClient.invalidateQueries({ queryKey: ['tax-rates'] });
      }
      await onContinue({
        settings: {
          defaultConsolidatedBilling: consolidated,
          defaultPaymentTermId: termId || undefined,
          defaultTaxRateId: taxRateId || undefined,
          invoicePrefix: prefix.trim() || undefined,
          defaultPoRequired: poRequired,
        },
      });
    } catch (err: unknown) {
      const fields = getFieldErrors(err);
      setErrors(fields);
      setError(
        Object.keys(fields).length > 0
          ? null
          : getTenantErrorMessage(err, 'Could not save how you bill')
      );
    } finally {
      setCreatingRate(false);
    }
  };

  const shownPrefix = prefix.trim() || DEFAULT_PREFIXES.invoicePrefix;

  return (
    <form onSubmit={handleSubmit} noValidate>
      <h1 className="text-3xl font-bold tracking-tight text-neutral-900 md:text-4xl">
        How you bill
      </h1>
      <p className="mt-2 text-lg text-neutral-600">
        These become the defaults for every new customer. You can override any
        of them per customer or per invoice.
      </p>

      {error && (
        <div
          role="alert"
          className="mt-6 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700"
        >
          {error}
        </div>
      )}

      <div className="mt-8 space-y-7">
        <fieldset>
          <legend className="mb-2 font-medium text-neutral-800">
            Invoice style
          </legend>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <ChoiceCard
              name="invoice-style"
              value="consolidated"
              checked={consolidated}
              onChange={() => setConsolidated(true)}
              title="Consolidated monthly"
              badge="Default"
              description="One invoice per customer each month, a line per site"
            />
            <ChoiceCard
              name="invoice-style"
              value="per-visit"
              checked={!consolidated}
              onChange={() => setConsolidated(false)}
              title="Per visit"
              description="Invoice each completed visit or job"
            />
          </div>
        </fieldset>

        <div className="grid grid-cols-1 gap-6 sm:grid-cols-3">
          <div>
            <label
              htmlFor="ob-term"
              className="mb-2 block font-medium text-neutral-800"
            >
              Default payment terms
            </label>
            <select
              id="ob-term"
              value={termId}
              onChange={(e) => setTermId(e.target.value)}
              className={`${fieldClass} ${fieldBorder(errors.defaultPaymentTermId)}`}
            >
              <option value="">No default</option>
              {terms.map((t) => (
                <option key={t.id} value={t.id}>
                  {t.name}
                </option>
              ))}
            </select>
            <FieldError message={errors.defaultPaymentTermId} />
            {!errors.defaultPaymentTermId && selectedTerm && (
              <Hint>
                {selectedTerm.days_until_due === 0
                  ? 'Due on the invoice date.'
                  : `Due ${selectedTerm.days_until_due} days after the invoice date.`}
              </Hint>
            )}
          </div>

          <div>
            <label
              htmlFor="ob-tax"
              className="mb-2 block font-medium text-neutral-800"
            >
              Sales tax
            </label>
            <select
              id="ob-tax"
              value={taxChoice}
              onChange={(e) => setTaxChoice(e.target.value)}
              className={`${fieldClass} ${fieldBorder(errors.defaultTaxRateId)}`}
            >
              <option value={NO_TAX}>No sales tax</option>
              {rates.map((r) => (
                <option key={r.id} value={r.id}>
                  {r.name} · {taxRateToPercent(r.rate)}%
                </option>
              ))}
              {canManageTaxRates && (
                <option value={NEW_RATE}>Add a flat rate…</option>
              )}
            </select>
            <FieldError message={errors.defaultTaxRateId} />
            {!errors.defaultTaxRateId && (
              <Hint>One flat rate for every invoice.</Hint>
            )}
          </div>

          <div>
            <label
              htmlFor="ob-prefix"
              className="mb-2 block font-medium text-neutral-800"
            >
              Invoice numbering
            </label>
            <input
              id="ob-prefix"
              value={prefix}
              onChange={(e) => setPrefix(e.target.value)}
              placeholder={DEFAULT_PREFIXES.invoicePrefix}
              maxLength={MAX_NUMBER_PREFIX_LENGTH + 5}
              className={`${fieldClass} ${fieldBorder(errors.invoicePrefix)}`}
            />
            <FieldError message={errors.invoicePrefix} />
            {!errors.invoicePrefix && (
              <Hint>
                Prefix {shownPrefix}, numbers like {shownPrefix}000001.
              </Hint>
            )}
          </div>
        </div>

        {taxChoice === NEW_RATE && (
          <div className="grid grid-cols-1 gap-6 rounded-xl border border-neutral-200 bg-white p-4 sm:grid-cols-2">
            <div>
              <label
                htmlFor="ob-rate-name"
                className="mb-2 block font-medium text-neutral-800"
              >
                Rate name
              </label>
              <input
                id="ob-rate-name"
                value={newRateName}
                onChange={(e) => setNewRateName(e.target.value)}
                maxLength={100}
                className={`${fieldClass} ${fieldBorder(errors.newRateName)}`}
              />
              <FieldError message={errors.newRateName} />
            </div>
            <div>
              <label
                htmlFor="ob-rate-percent"
                className="mb-2 block font-medium text-neutral-800"
              >
                Rate (%)
              </label>
              <input
                id="ob-rate-percent"
                type="number"
                inputMode="decimal"
                min={0}
                max={100}
                step="0.001"
                value={newRatePercent}
                onChange={(e) => setNewRatePercent(e.target.value)}
                placeholder="8"
                className={`${fieldClass} ${fieldBorder(errors.newRatePercent)}`}
              />
              <FieldError message={errors.newRatePercent} />
            </div>
          </div>
        )}

        <div className="divide-y divide-neutral-200 rounded-xl border border-neutral-200 bg-white">
          <ToggleRow
            id="ob-po"
            title="Require PO numbers"
            description="New customers start with “PO required”: their invoices can't be sent without one."
            checked={poRequired}
            onChange={setPoRequired}
          />
        </div>
      </div>

      <StepFooter
        onBack={onBack}
        onSkip={() => void onSkip()}
        primaryLabel="Continue"
        busy={busy || creatingRate}
      />
    </form>
  );
}

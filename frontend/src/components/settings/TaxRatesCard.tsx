import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Plus } from 'lucide-react';
import {
  Badge,
  Button,
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
  Input,
  Modal,
} from '@components/shared';
import {
  taxRateService,
  taxRatesQueryKey,
  type TaxRate,
  type TaxRateInput,
} from '@services/taxRateService';
import { getErrorMessage, getFieldErrors } from '@/utils/error';
import { percentToTaxRate, taxRateToPercent } from '@/utils/money';

interface Draft {
  /** Rate being edited, or null for a new one. */
  id: string | null;
  name: string;
  percent: string;
  region: string;
}

const EMPTY_DRAFT: Draft = { id: null, name: '', percent: '', region: '' };

/** Client-side checks; the API validates again. */
function draftErrors(d: Draft): Record<string, string> {
  const errors: Record<string, string> = {};
  if (!d.name.trim()) errors.name = 'Enter a name';
  const pct = Number(d.percent);
  if (d.percent.trim() === '' || Number.isNaN(pct)) {
    errors.rate = 'Enter a percentage';
  } else if (pct < 0 || pct > 100) {
    errors.rate = 'Use a percentage between 0 and 100';
  }
  return errors;
}

/**
 * Settings → Tax rates (T235 on the T237 API): the tenant's own rates plus
 * the built-in global ones (read-only). With tax_rates.manage the owner can
 * add, edit, deactivate and reactivate their own rates; deactivating keeps
 * invoices that already use a rate intact.
 */
export function TaxRatesCard({ canManage }: { canManage: boolean }) {
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState<Draft | null>(null);
  const [fieldErrors, setFieldErrors] = useState<Record<string, string>>({});
  const [formError, setFormError] = useState<string | null>(null);
  const [listError, setListError] = useState<string | null>(null);

  const ratesQuery = useQuery({
    queryKey: taxRatesQueryKey,
    queryFn: () => taxRateService.listTaxRates(false),
  });
  const rates = ratesQuery.data ?? [];

  const refresh = () =>
    queryClient.invalidateQueries({ queryKey: ['tax-rates'] });

  const save = useMutation({
    mutationFn: ({ id, input }: { id: string | null; input: TaxRateInput }) =>
      id
        ? taxRateService.updateTaxRate(id, input)
        : taxRateService.createTaxRate({ ...input, is_active: true }),
    onSuccess: () => {
      setDraft(null);
      void refresh();
    },
    onError: (err: unknown) => {
      const fields = getFieldErrors(err);
      setFieldErrors(fields);
      setFormError(
        Object.keys(fields).length > 0
          ? null
          : getErrorMessage(err, 'Failed to save the tax rate')
      );
    },
  });

  const toggleActive = useMutation({
    mutationFn: (rate: TaxRate) =>
      taxRateService.updateTaxRate(rate.id, { is_active: !rate.is_active }),
    onSuccess: () => {
      setListError(null);
      void refresh();
    },
    onError: (err: unknown) =>
      setListError(getErrorMessage(err, 'Failed to update the tax rate')),
  });

  const openDraft = (next: Draft) => {
    setDraft(next);
    setFieldErrors({});
    setFormError(null);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!draft) return;
    const errors = draftErrors(draft);
    setFieldErrors(errors);
    if (Object.keys(errors).length > 0) return;
    save.mutate({
      id: draft.id,
      input: {
        name: draft.name.trim(),
        rate: percentToTaxRate(draft.percent),
        region: draft.region.trim(),
      },
    });
  };

  return (
    <Card variant="elevated" padding="lg">
      <CardHeader>
        <div className="flex items-center justify-between gap-4">
          <div>
            <CardTitle>Tax rates</CardTitle>
            <CardDescription>
              Sales tax rates you can put on invoices and quotes.
            </CardDescription>
          </div>
          {canManage && (
            <Button
              variant="primary"
              size="sm"
              onClick={() => openDraft(EMPTY_DRAFT)}
            >
              <Plus className="mr-1 h-4 w-4" />
              Add tax rate
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent>
        {!canManage && (
          <p className="mt-4 text-sm text-neutral-600">
            Only organization admins can change tax rates.
          </p>
        )}
        {listError && (
          <div
            role="alert"
            className="mt-4 rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700"
          >
            {listError}
          </div>
        )}
        {ratesQuery.isLoading ? (
          <div className="py-8 text-center text-neutral-500">
            Loading tax rates...
          </div>
        ) : ratesQuery.isError ? (
          <div role="alert" className="py-8 text-center text-neutral-600">
            {getErrorMessage(ratesQuery.error, 'Failed to load tax rates')}
          </div>
        ) : rates.length === 0 ? (
          <div className="py-8 text-center text-neutral-500">
            No tax rates yet.
          </div>
        ) : (
          <ul className="mt-4 divide-y divide-neutral-200">
            {rates.map((rate) => {
              const builtIn = !rate.tenant_id;
              return (
                <li
                  key={rate.id}
                  className="flex flex-wrap items-center justify-between gap-3 py-3"
                >
                  <div className="min-w-0">
                    <p className="text-sm font-medium text-neutral-900">
                      {rate.name}{' '}
                      <span className="font-normal text-neutral-600">
                        · {taxRateToPercent(rate.rate)}%
                      </span>
                    </p>
                    <div className="mt-1 flex flex-wrap items-center gap-2">
                      {rate.region && (
                        <span className="text-xs text-neutral-500">
                          {rate.region}
                        </span>
                      )}
                      {builtIn && (
                        <Badge variant="neutral" size="sm">
                          Built-in
                        </Badge>
                      )}
                      {!rate.is_active && (
                        <Badge variant="warning" size="sm">
                          Inactive
                        </Badge>
                      )}
                    </div>
                  </div>
                  {canManage && !builtIn && (
                    <div className="flex items-center gap-2">
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label={`Edit ${rate.name}`}
                        onClick={() =>
                          openDraft({
                            id: rate.id,
                            name: rate.name,
                            percent: taxRateToPercent(rate.rate),
                            region: rate.region ?? '',
                          })
                        }
                      >
                        Edit
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        aria-label={`${rate.is_active ? 'Deactivate' : 'Activate'} ${rate.name}`}
                        disabled={toggleActive.isPending}
                        onClick={() => toggleActive.mutate(rate)}
                      >
                        {rate.is_active ? 'Deactivate' : 'Activate'}
                      </Button>
                    </div>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </CardContent>

      <Modal
        isOpen={draft !== null}
        onClose={() => setDraft(null)}
        title={draft?.id ? 'Edit tax rate' : 'Add tax rate'}
      >
        {draft && (
          <form onSubmit={handleSubmit} className="space-y-4" noValidate>
            {formError && (
              <div
                role="alert"
                className="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700"
              >
                {formError}
              </div>
            )}
            <Input
              id="tax-rate-name"
              label="Name"
              value={draft.name}
              onChange={(e) => setDraft({ ...draft, name: e.target.value })}
              placeholder="Ohio sales tax"
              maxLength={100}
              error={fieldErrors.name}
              fullWidth
            />
            <Input
              id="tax-rate-percent"
              label="Rate (%)"
              type="number"
              inputMode="decimal"
              min="0"
              max="100"
              step="0.001"
              value={draft.percent}
              onChange={(e) => setDraft({ ...draft, percent: e.target.value })}
              placeholder="8"
              error={fieldErrors.rate}
              fullWidth
            />
            <Input
              id="tax-rate-region"
              label="Region (optional)"
              value={draft.region}
              onChange={(e) => setDraft({ ...draft, region: e.target.value })}
              placeholder="Franklin County, OH"
              maxLength={100}
              error={fieldErrors.region}
              fullWidth
            />
            <div className="flex justify-end space-x-3 pt-2">
              <Button
                type="button"
                variant="outline"
                onClick={() => setDraft(null)}
              >
                Cancel
              </Button>
              <Button type="submit" variant="primary" disabled={save.isPending}>
                {save.isPending ? 'Saving...' : 'Save tax rate'}
              </Button>
            </div>
          </form>
        )}
      </Modal>
    </Card>
  );
}

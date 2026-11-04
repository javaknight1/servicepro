import { useEffect, useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { AlertCircle, CheckCircle2, RefreshCw } from 'lucide-react';
import { Badge, Button, Card, ErrorState, Modal } from '@components/shared';
import {
  quickbooksApi,
  quickbooksOptionsQueryKey,
  quickbooksStatusQueryKey,
} from '@services/quickbooksService';
import { getErrorMessage } from '@/utils/error';
import { redirectTo } from '@/utils/redirect';
import { timeAgo } from '@/utils/timeAgo';
import type {
  QuickBooksRef,
  QuickBooksStatus,
  QuickBooksSyncRecord,
} from '@/types/quickbooks';

const selectClass =
  'w-full px-3 py-2 border border-neutral-300 rounded-lg text-sm bg-white focus:outline-hidden focus:ring-2 focus:ring-primary-500 focus:border-primary-500 disabled:bg-neutral-100';

/** Why the Intuit callback sent the user back with ?quickbooks=error. */
const CALLBACK_ERRORS: Record<string, string> = {
  cancelled: 'QuickBooks wasn’t connected: the connection was cancelled.',
  expired:
    'That QuickBooks link expired or was already used. Connect again to start over.',
  unavailable: 'QuickBooks isn’t available yet.',
  failed: 'QuickBooks couldn’t be connected. Please try again.',
};

function QuickBooksLogo() {
  return (
    <span
      aria-hidden
      className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl border border-neutral-200 bg-white"
    >
      <span className="flex h-9 w-9 items-center justify-center rounded-full bg-[#2ca01c] text-sm font-bold text-white">
        qb
      </span>
    </span>
  );
}

function Stat({
  label,
  value,
  tone,
}: {
  label: string;
  value: number;
  tone?: 'success' | 'error';
}) {
  const color =
    tone === 'error'
      ? 'text-error-600'
      : tone === 'success'
        ? 'text-success-600'
        : 'text-neutral-900';
  return (
    <div className="px-5 py-4">
      <div className="text-sm text-neutral-600">{label}</div>
      <div className={`mt-1 text-3xl font-semibold ${color}`}>{value}</div>
    </div>
  );
}

function RefSelect({
  id,
  label,
  value,
  options,
  current,
  emptyLabel,
  onChange,
  disabled,
}: {
  id: string;
  label: string;
  value: string;
  options: QuickBooksRef[] | undefined;
  current?: QuickBooksRef;
  emptyLabel: string;
  onChange: (v: string) => void;
  disabled: boolean;
}) {
  // Keep the saved choice listed while options load (or if it was removed).
  const list = [...(options ?? [])];
  if (current && !list.some((o) => o.id === current.id)) list.unshift(current);
  return (
    <div className="grid grid-cols-1 items-center gap-2 sm:grid-cols-[150px_minmax(0,1fr)]">
      <label htmlFor={id} className="text-sm text-neutral-600">
        {label}
      </label>
      <select
        id={id}
        className={selectClass}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
      >
        <option value="">{emptyLabel}</option>
        {list.map((o) => (
          <option key={o.id} value={o.id}>
            {o.name}
          </option>
        ))}
      </select>
    </div>
  );
}

function RecordRow({
  record,
  onRetry,
  retrying,
}: {
  record: QuickBooksSyncRecord;
  onRetry?: () => void;
  retrying?: boolean;
}) {
  const failed = record.status === 'error';
  return (
    <li className="flex items-start gap-3 px-5 py-3">
      {failed ? (
        <AlertCircle className="mt-0.5 h-5 w-5 shrink-0 text-error-600" />
      ) : (
        <CheckCircle2 className="mt-0.5 h-5 w-5 shrink-0 text-success-600" />
      )}
      <div className="min-w-0 flex-1">
        <p className="text-sm text-neutral-900">
          {record.label}
          {!failed && ' sent to QuickBooks'}
        </p>
        {failed && record.last_error && (
          <p className="mt-0.5 text-sm text-error-700">{record.last_error}</p>
        )}
      </div>
      {onRetry ? (
        <Button
          variant="outline"
          size="sm"
          onClick={onRetry}
          isLoading={retrying}
        >
          Retry
        </Button>
      ) : (
        <span className="shrink-0 text-sm text-neutral-500">
          {timeAgo(record.last_synced_at ?? record.updated_at)}
        </span>
      )}
    </li>
  );
}

function ComingSoonCard() {
  return (
    <Card variant="elevated" padding="lg">
      <div className="flex flex-wrap items-start gap-4">
        <QuickBooksLogo />
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <h2 className="text-lg font-semibold text-neutral-900">
              QuickBooks Online
            </h2>
            <Badge variant="neutral" size="sm">
              Coming soon
            </Badge>
          </div>
          <p className="mt-1 text-sm text-neutral-600">
            Send customers, invoices and payments to QuickBooks automatically.
            We&apos;ll let you know when it&apos;s ready.
          </p>
        </div>
        <Button variant="outline" disabled>
          Connect QuickBooks
        </Button>
      </div>
    </Card>
  );
}

/**
 * Settings → Integrations → QuickBooks Online (T192, mockup 14): connect
 * through Intuit, map the income account / sales tax / deposit account,
 * see what synced and what failed (with Retry), sync now, disconnect.
 * When the server has no Intuit app yet it shows "Coming soon" (D11).
 */
export function QuickBooksCard({
  tenantId,
  canManage,
  callback,
  onCallbackSeen,
}: {
  tenantId: string;
  canManage: boolean;
  /** ?quickbooks=connected|error and ?reason= from the Intuit callback. */
  callback?: { result: string | null; reason: string | null };
  onCallbackSeen?: () => void;
}) {
  const queryClient = useQueryClient();
  const [actionError, setActionError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [confirmDisconnect, setConfirmDisconnect] = useState(false);
  const [form, setForm] = useState({
    income_account_id: '',
    tax_code_id: '',
    deposit_account_id: '',
  });

  const statusQuery = useQuery({
    queryKey: quickbooksStatusQueryKey(tenantId),
    queryFn: async () => (await quickbooksApi.getStatus(tenantId)).data,
    enabled: canManage,
  });
  const status = statusQuery.data;
  const connected = status?.status === 'connected';

  const optionsQuery = useQuery({
    queryKey: quickbooksOptionsQueryKey(tenantId),
    queryFn: async () => (await quickbooksApi.getOptions(tenantId)).data,
    enabled: canManage && connected,
    staleTime: 5 * 60_000,
    retry: false,
  });

  useEffect(() => {
    if (!status) return;
    setForm({
      income_account_id: status.income_account?.id ?? '',
      tax_code_id: status.tax_code?.id ?? '',
      deposit_account_id: status.deposit_account?.id ?? '',
    });
  }, [status]);

  // The Intuit callback's outcome, shown once.
  const result = callback?.result ?? null;
  const reason = callback?.reason ?? null;
  useEffect(() => {
    if (!result) return;
    if (result === 'connected') {
      setNotice(
        'QuickBooks is connected. New invoices and payments will sync.'
      );
    } else {
      setActionError(CALLBACK_ERRORS[reason ?? ''] ?? CALLBACK_ERRORS.failed);
    }
    onCallbackSeen?.();
  }, [result, reason, onCallbackSeen]);

  const setStatus = (next: QuickBooksStatus) =>
    queryClient.setQueryData(quickbooksStatusQueryKey(tenantId), next);
  const fail = (fallback: string) => (error: unknown) =>
    setActionError(getErrorMessage(error, fallback));

  const connect = useMutation({
    mutationFn: async () => (await quickbooksApi.connect(tenantId)).data,
    onSuccess: (data) => redirectTo(data.url),
    onError: fail('Couldn’t start connecting QuickBooks. Please try again.'),
  });
  const disconnect = useMutation({
    mutationFn: async () => (await quickbooksApi.disconnect(tenantId)).data,
    onSuccess: (data) => {
      setConfirmDisconnect(false);
      setStatus(data);
      setNotice('QuickBooks is disconnected. Nothing was removed from it.');
    },
    onError: fail('Couldn’t disconnect QuickBooks.'),
  });
  const save = useMutation({
    mutationFn: async () =>
      (await quickbooksApi.updateSettings(tenantId, form)).data,
    onSuccess: (data) => {
      setStatus(data);
      setNotice('QuickBooks settings saved.');
    },
    onError: fail('Couldn’t save the QuickBooks settings.'),
  });
  const syncNow = useMutation({
    mutationFn: async () => (await quickbooksApi.syncNow(tenantId)).data,
    onSuccess: (r) => {
      setNotice(
        r.synced === 0 && r.failed === 0
          ? 'Everything is already in QuickBooks.'
          : `Sent ${r.synced} record${r.synced === 1 ? '' : 's'} to QuickBooks${
              r.failed ? `; ${r.failed} failed (see below)` : ''
            }${r.more ? '. The rest follows with the hourly sync.' : '.'}`
      );
      void statusQuery.refetch();
    },
    onError: fail('QuickBooks sync failed. Please try again.'),
  });
  const retry = useMutation({
    mutationFn: async (recordId: string) =>
      (await quickbooksApi.retry(tenantId, recordId)).data,
    onSuccess: (data) => {
      setStatus(data);
      setNotice('Queued again. It goes with the next sync, or Sync now.');
    },
    onError: fail('Couldn’t retry that record.'),
  });

  const run = (fn: () => void) => () => {
    setActionError(null);
    setNotice(null);
    fn();
  };

  if (!canManage) {
    return (
      <Card variant="elevated" padding="lg">
        <p className="text-neutral-600">
          Only owners and admins can connect QuickBooks and see its sync.
        </p>
      </Card>
    );
  }
  if (statusQuery.isError) {
    return (
      <ErrorState
        title="Couldn't load QuickBooks"
        message={getErrorMessage(statusQuery.error)}
        onRetry={() => statusQuery.refetch()}
      />
    );
  }
  if (!status) {
    return (
      <Card variant="elevated" padding="lg">
        <p className="text-neutral-500">Loading QuickBooks…</p>
      </Card>
    );
  }
  if (!status.configured) return <ComingSoonCard />;

  const messages = (
    <>
      {notice && (
        <div
          role="status"
          className="rounded-lg border border-success-200 bg-success-50 p-3 text-sm text-success-800"
        >
          {notice}
        </div>
      )}
      {actionError && (
        <div
          role="alert"
          className="rounded-lg border border-error-200 bg-error-50 p-3 text-sm text-error-700"
        >
          {actionError}
        </div>
      )}
    </>
  );

  if (status.status !== 'connected') {
    const reauth = status.status === 'reauth_required';
    return (
      <div className="space-y-4">
        {messages}
        <Card variant="elevated" padding="lg">
          <div className="flex flex-wrap items-start gap-4">
            <QuickBooksLogo />
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <h2 className="text-lg font-semibold text-neutral-900">
                  QuickBooks Online
                </h2>
                <Badge variant={reauth ? 'warning' : 'neutral'} size="sm">
                  {reauth ? 'Reconnect needed' : 'Not connected'}
                </Badge>
              </div>
              <p className="mt-1 text-sm text-neutral-600">
                {reauth
                  ? status.last_sync_error ||
                    'The QuickBooks authorization expired. Reconnect to resume syncing.'
                  : 'Send customers, invoices and payments to QuickBooks automatically: one-way, ServicePro to QuickBooks. Invoices issued from the day you connect are synced.'}
              </p>
            </div>
            <Button
              onClick={run(() => connect.mutate())}
              isLoading={connect.isPending}
            >
              {reauth ? 'Reconnect QuickBooks' : 'Connect QuickBooks'}
            </Button>
          </div>
        </Card>
      </div>
    );
  }

  const options = optionsQuery.data;
  const dirty =
    form.income_account_id !== (status.income_account?.id ?? '') ||
    form.tax_code_id !== (status.tax_code?.id ?? '') ||
    form.deposit_account_id !== (status.deposit_account?.id ?? '');

  return (
    <div className="space-y-4">
      {messages}
      <Card variant="elevated" padding="none" className="overflow-hidden">
        <div className="flex flex-wrap items-center gap-4 border-b border-neutral-200 p-5">
          <QuickBooksLogo />
          <div className="min-w-0 flex-1">
            <div className="flex flex-wrap items-center gap-2">
              <h2 className="text-lg font-semibold text-neutral-900">
                QuickBooks Online
              </h2>
              <Badge variant="success" size="sm">
                Connected
              </Badge>
              {status.environment === 'sandbox' && (
                <Badge variant="warning" size="sm">
                  Sandbox
                </Badge>
              )}
            </div>
            <p className="mt-1 text-sm text-neutral-600">
              {status.company_name ? `${status.company_name} · ` : ''}one-way
              sync, ServicePro to QuickBooks
            </p>
          </div>
          <span className="flex items-center gap-1.5 text-sm text-neutral-500">
            <RefreshCw className="h-4 w-4" />
            Last sync {timeAgo(status.last_sync_at)}
          </span>
          <Button
            variant="outline"
            onClick={run(() => syncNow.mutate())}
            isLoading={syncNow.isPending}
          >
            Sync now
          </Button>
        </div>

        <div className="grid grid-cols-2 divide-neutral-200 border-b border-neutral-200 sm:grid-cols-4 sm:divide-x">
          <Stat label="Customers synced" value={status.counts.customers} />
          <Stat label="Invoices synced" value={status.counts.invoices} />
          <Stat label="Payments synced" value={status.counts.payments} />
          <Stat
            label="Sync errors"
            value={status.counts.errors}
            tone={status.counts.errors > 0 ? 'error' : 'success'}
          />
        </div>

        <form
          className="space-y-4 p-5"
          onSubmit={(e) => {
            e.preventDefault();
            run(() => save.mutate())();
          }}
        >
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <RefSelect
              id="qb-income-account"
              label="Income account"
              value={form.income_account_id}
              options={options?.income_accounts}
              current={status.income_account}
              emptyLabel="QuickBooks “Services” item"
              onChange={(v) => setForm({ ...form, income_account_id: v })}
              disabled={save.isPending}
            />
            <RefSelect
              id="qb-tax-code"
              label="Sales tax"
              value={form.tax_code_id}
              options={options?.tax_codes}
              current={status.tax_code}
              emptyLabel="Tax as on the invoice"
              onChange={(v) => setForm({ ...form, tax_code_id: v })}
              disabled={save.isPending}
            />
            <RefSelect
              id="qb-deposit-account"
              label="Payment deposits"
              value={form.deposit_account_id}
              options={options?.deposit_accounts}
              current={status.deposit_account}
              emptyLabel="Undeposited Funds"
              onChange={(v) => setForm({ ...form, deposit_account_id: v })}
              disabled={save.isPending}
            />
          </div>
          {optionsQuery.isError && (
            <p className="text-sm text-warning-700">
              Couldn&apos;t load accounts from QuickBooks:{' '}
              {getErrorMessage(optionsQuery.error)}
            </p>
          )}
          <div className="flex flex-wrap items-center justify-between gap-3">
            <p className="text-xs text-neutral-500">
              Invoices issued from {status.sync_from ?? 'the connection day'} on
              are synced, with ServicePro&apos;s tax total. Payments are applied
              to their invoice.
            </p>
            <Button type="submit" disabled={!dirty} isLoading={save.isPending}>
              Save mapping
            </Button>
          </div>
        </form>
      </Card>

      {status.errors.length > 0 && (
        <Card variant="elevated" padding="none">
          <div className="border-b border-neutral-200 px-5 py-4">
            <h3 className="font-semibold text-neutral-900">
              Records that didn&apos;t sync
            </h3>
            <p className="mt-0.5 text-sm text-neutral-600">
              Fix the cause (often the mapping above), then retry.
            </p>
          </div>
          <ul className="divide-y divide-neutral-100">
            {status.errors.map((r) => (
              <RecordRow
                key={r.id}
                record={r}
                onRetry={run(() => retry.mutate(r.id))}
                retrying={retry.isPending && retry.variables === r.id}
              />
            ))}
          </ul>
        </Card>
      )}

      <Card variant="elevated" padding="none">
        <div className="border-b border-neutral-200 px-5 py-4">
          <h3 className="font-semibold text-neutral-900">
            Recent sync activity
          </h3>
        </div>
        {status.recent.length === 0 ? (
          <p className="px-5 py-4 text-sm text-neutral-500">
            Nothing synced yet. Sent invoices and recorded payments go to
            QuickBooks within the hour, or use Sync now.
          </p>
        ) : (
          <ul className="divide-y divide-neutral-100">
            {status.recent.map((r) => (
              <RecordRow key={r.id} record={r} />
            ))}
          </ul>
        )}
      </Card>

      <div className="flex justify-end">
        <Button variant="ghost" onClick={() => setConfirmDisconnect(true)}>
          Disconnect QuickBooks
        </Button>
      </div>

      <Modal
        isOpen={confirmDisconnect}
        onClose={() => setConfirmDisconnect(false)}
        title="Disconnect QuickBooks?"
        description="Syncing stops. Nothing already in QuickBooks is changed or removed; reconnecting the same company carries on where it stopped."
        size="sm"
      >
        <div className="flex justify-end gap-3">
          <Button variant="outline" onClick={() => setConfirmDisconnect(false)}>
            Cancel
          </Button>
          <Button
            variant="danger"
            onClick={run(() => disconnect.mutate())}
            isLoading={disconnect.isPending}
          >
            Disconnect
          </Button>
        </div>
      </Modal>
    </div>
  );
}

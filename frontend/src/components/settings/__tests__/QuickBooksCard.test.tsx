import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AxiosError } from 'axios';
import type { AxiosResponse, InternalAxiosRequestConfig } from 'axios';
import type {
  QuickBooksConnectResponse,
  QuickBooksOptions,
  QuickBooksSettingsRequest,
  QuickBooksStatus,
  QuickBooksSyncResult,
} from '@/types/quickbooks';

const api = vi.hoisted(() => ({
  getStatus: vi.fn(),
  connect: vi.fn(),
  disconnect: vi.fn(),
  getOptions: vi.fn(),
  updateSettings: vi.fn(),
  syncNow: vi.fn(),
  retry: vi.fn(),
}));
vi.mock('@services/quickbooksService', () => ({
  quickbooksApi: api,
  quickbooksStatusQueryKey: (id: string) => ['quickbooks-status', id],
  quickbooksOptionsQueryKey: (id: string) => ['quickbooks-options', id],
}));

const redirectTo = vi.fn<(url: string) => void>();
vi.mock('@/utils/redirect', () => ({
  redirectTo: (url: string) => redirectTo(url),
}));

import { QuickBooksCard } from '../QuickBooksCard';
import { timeAgo } from '@/utils/timeAgo';

function ok<T>(data: T): AxiosResponse<T> {
  return {
    data,
    status: 200,
    statusText: 'OK',
    headers: {},
    config: { headers: {} } as InternalAxiosRequestConfig,
  };
}

function apiError(status: number, error: string, message: string): AxiosError {
  const err = new AxiosError(message);
  err.response = {
    data: { error, message },
    status,
    statusText: '',
    headers: {},
    config: { headers: {} } as InternalAxiosRequestConfig,
  };
  return err;
}

function qbStatus(overrides: Partial<QuickBooksStatus> = {}): QuickBooksStatus {
  return {
    configured: true,
    status: 'disconnected',
    counts: { customers: 0, invoices: 0, payments: 0, pending: 0, errors: 0 },
    errors: [],
    recent: [],
    ...overrides,
  };
}

const connected = qbStatus({
  status: 'connected',
  environment: 'sandbox',
  company_name: 'Brightline Facility Services LLC',
  sync_from: '2026-11-04',
  last_sync_at: new Date(Date.now() - 2 * 60_000).toISOString(),
  income_account: { id: '79', name: 'Cleaning Services' },
  counts: { customers: 23, invoices: 118, payments: 96, pending: 0, errors: 1 },
  errors: [
    {
      id: 'rec-1',
      entity: 'invoice',
      local_id: 'inv-1',
      label: 'Invoice INV-9 · Acme · $108.00',
      status: 'error',
      attempts: 2,
      last_error: 'choose an income account in Settings',
      updated_at: '2026-11-04T10:00:00Z',
    },
  ],
  recent: [
    {
      id: 'rec-2',
      entity: 'invoice',
      local_id: 'inv-2',
      label: 'Invoice INV-1042 · Harborview Property Management · $9,882.00',
      status: 'synced',
      attempts: 0,
      last_synced_at: '2026-11-01T14:14:00Z',
      updated_at: '2026-11-01T14:14:00Z',
    },
  ],
});

const options: QuickBooksOptions = {
  income_accounts: [
    { id: '79', name: 'Cleaning Services' },
    { id: '80', name: 'Maintenance Income' },
  ],
  tax_codes: [{ id: '3', name: 'OH 8.0%' }],
  deposit_accounts: [{ id: '35', name: 'Checking' }],
};

function renderCard(props: Partial<Parameters<typeof QuickBooksCard>[0]> = {}) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <QuickBooksCard tenantId="tenant-1" canManage {...props} />
    </QueryClientProvider>
  );
}

beforeEach(() => {
  Object.values(api).forEach((fn) => fn.mockReset());
  redirectTo.mockReset();
  api.getOptions.mockResolvedValue(ok(options));
});

describe('QuickBooksCard', () => {
  it('shows Coming soon while the server has no Intuit app', async () => {
    api.getStatus.mockResolvedValue(ok(qbStatus({ configured: false })));
    renderCard();
    expect(await screen.findByText('Coming soon')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Connect QuickBooks' })
    ).toBeDisabled();
  });

  it('hides everything from people who may not manage integrations', () => {
    renderCard({ canManage: false });
    expect(screen.getByText(/Only owners and admins/)).toBeInTheDocument();
    expect(api.getStatus).not.toHaveBeenCalled();
  });

  it('connect sends the browser to Intuit', async () => {
    api.getStatus.mockResolvedValue(ok(qbStatus()));
    api.connect.mockResolvedValue(
      ok<QuickBooksConnectResponse>({
        url: 'https://appcenter.intuit.com/connect/oauth2?state=x',
      })
    );
    renderCard();
    const user = userEvent.setup();
    await user.click(
      await screen.findByRole('button', { name: 'Connect QuickBooks' })
    );
    await waitFor(() =>
      expect(redirectTo).toHaveBeenCalledWith(
        'https://appcenter.intuit.com/connect/oauth2?state=x'
      )
    );
  });

  it('asks to reconnect when the authorization expired', async () => {
    api.getStatus.mockResolvedValue(
      ok(
        qbStatus({
          status: 'reauth_required',
          last_sync_error: 'QuickBooks authorization expired. Reconnect.',
        })
      )
    );
    renderCard();
    expect(await screen.findByText('Reconnect needed')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: 'Reconnect QuickBooks' })
    ).toBeEnabled();
  });

  it('shows the callback outcome once', async () => {
    api.getStatus.mockResolvedValue(ok(qbStatus()));
    const seen = vi.fn();
    renderCard({
      callback: { result: 'error', reason: 'cancelled' },
      onCallbackSeen: seen,
    });
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'the connection was cancelled'
    );
    expect(seen).toHaveBeenCalledTimes(1);
  });

  it('shows counts, errors with retry and recent activity when connected', async () => {
    api.getStatus.mockResolvedValue(ok(connected));
    api.retry.mockResolvedValue(
      ok({
        ...connected,
        errors: [],
        counts: { ...connected.counts, errors: 0 },
      })
    );
    renderCard();
    const user = userEvent.setup();

    expect(await screen.findByText('Connected')).toBeInTheDocument();
    expect(screen.getByText('Sandbox')).toBeInTheDocument();
    expect(
      screen.getByText(/Brightline Facility Services LLC · one-way/)
    ).toBeInTheDocument();
    expect(screen.getByText('Last sync 2 min ago')).toBeInTheDocument();
    expect(screen.getByText('118')).toBeInTheDocument();
    expect(screen.getByText('96')).toBeInTheDocument();
    expect(
      screen.getByText(/Invoice INV-1042 · Harborview Property Management/)
    ).toBeInTheDocument();

    const errorRow = screen
      .getByText('Invoice INV-9 · Acme · $108.00')
      .closest('li') as HTMLElement;
    expect(
      within(errorRow).getByText('choose an income account in Settings')
    ).toBeInTheDocument();
    await user.click(within(errorRow).getByRole('button', { name: 'Retry' }));
    await waitFor(() =>
      expect(api.retry).toHaveBeenCalledWith('tenant-1', 'rec-1')
    );
    expect(await screen.findByRole('status')).toHaveTextContent('Queued again');
    expect(
      screen.queryByText('Invoice INV-9 · Acme · $108.00')
    ).not.toBeInTheDocument();
  });

  it('saves the mapping', async () => {
    api.getStatus.mockResolvedValue(ok(connected));
    api.updateSettings.mockResolvedValue(
      ok({
        ...connected,
        income_account: { id: '80', name: 'Maintenance Income' },
        tax_code: { id: '3', name: 'OH 8.0%' },
      })
    );
    renderCard();
    const user = userEvent.setup();

    const save = await screen.findByRole('button', { name: 'Save mapping' });
    expect(save).toBeDisabled();
    await waitFor(() =>
      expect(
        screen.getByRole('option', { name: 'Maintenance Income' })
      ).toBeInTheDocument()
    );
    await user.selectOptions(screen.getByLabelText('Income account'), '80');
    await user.selectOptions(screen.getByLabelText('Sales tax'), '3');
    await user.click(save);
    await waitFor(() =>
      expect(api.updateSettings).toHaveBeenCalledWith('tenant-1', {
        income_account_id: '80',
        tax_code_id: '3',
        deposit_account_id: '',
      } satisfies QuickBooksSettingsRequest)
    );
    expect(await screen.findByRole('status')).toHaveTextContent(
      'QuickBooks settings saved.'
    );
  });

  it('sync now reports what was sent; API errors are shown', async () => {
    api.getStatus.mockResolvedValue(ok(connected));
    api.syncNow
      .mockResolvedValueOnce(
        ok<QuickBooksSyncResult>({
          enqueued: 3,
          synced: 3,
          failed: 0,
          skipped: 0,
          more: false,
        })
      )
      .mockRejectedValueOnce(
        apiError(
          409,
          'quickbooks_reauth_required',
          'QuickBooks needs to be reconnected'
        )
      );
    renderCard();
    const user = userEvent.setup();
    const button = await screen.findByRole('button', { name: 'Sync now' });
    await user.click(button);
    expect(await screen.findByRole('status')).toHaveTextContent(
      'Sent 3 records to QuickBooks.'
    );
    await user.click(button);
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'QuickBooks needs to be reconnected'
    );
  });

  it('disconnect asks first', async () => {
    api.getStatus.mockResolvedValue(ok(connected));
    api.disconnect.mockResolvedValue(ok(qbStatus()));
    renderCard();
    const user = userEvent.setup();
    await user.click(
      await screen.findByRole('button', { name: 'Disconnect QuickBooks' })
    );
    expect(api.disconnect).not.toHaveBeenCalled();
    await user.click(screen.getByRole('button', { name: 'Disconnect' }));
    await waitFor(() =>
      expect(api.disconnect).toHaveBeenCalledWith('tenant-1')
    );
    expect(
      await screen.findByRole('button', { name: 'Connect QuickBooks' })
    ).toBeInTheDocument();
  });
});

describe('timeAgo', () => {
  const now = Date.parse('2026-11-04T12:00:00Z');
  it('formats relative times', () => {
    expect(timeAgo(undefined, now)).toBe('never');
    expect(timeAgo('2026-11-04T11:59:50Z', now)).toBe('just now');
    expect(timeAgo('2026-11-04T11:58:00Z', now)).toBe('2 min ago');
    expect(timeAgo('2026-11-04T09:00:00Z', now)).toBe('3 h ago');
    expect(timeAgo('2026-11-01T09:00:00Z', now)).toBe('Nov 1');
  });
});

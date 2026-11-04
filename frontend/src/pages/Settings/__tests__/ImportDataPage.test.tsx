import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router';
import type { ReactNode } from 'react';
import type { ImportJob, ImportPreview } from '@services/importService';
import { ImportDataPage } from '../ImportDataPage';

vi.mock('@components/layout', () => ({
  DashboardLayout: ({ children }: { children: ReactNode }) => (
    <div>{children}</div>
  ),
}));

const api = vi.hoisted(() => ({
  preview: vi.fn(),
  start: vi.fn(),
  run: vi.fn(),
  get: vi.fn(),
}));

vi.mock('@services/importService', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('@services/importService')>();
  return { ...actual, importService: api };
});

const fields = [
  { key: 'full_name', label: 'Full name', required: false, group: 'name' },
  { key: 'email', label: 'Email', required: true },
  { key: 'phone', label: 'Phone', required: false },
];

function preview(over: Partial<ImportPreview> = {}): ImportPreview {
  return {
    source: 'jobber',
    entity: 'customers',
    filename: 'clients.csv',
    headers: ['Display Name', 'E-mails', 'Main Phone #s'],
    fields,
    mapping: { full_name: 'Display Name', email: 'E-mails' },
    missing: [],
    total_rows: 3,
    valid_rows: 2,
    skip_rows: 0,
    error_rows: 1,
    rows: [
      { row: 2, values: { first_name: 'Dana', email: 'dana@x.io' } },
      {
        row: 3,
        values: { first_name: 'Lee' },
        errors: [{ row: 3, field: 'email', message: 'email is required' }],
      },
    ],
    errors: [{ row: 3, field: 'email', message: 'email is required' }],
    ...over,
  };
}

function job(over: Partial<ImportJob> = {}): ImportJob {
  return {
    job_id: 'job-1',
    status: 'pending',
    source: 'jobber',
    entity: 'customers',
    filename: 'clients.csv',
    total_rows: 3,
    processed_rows: 1,
    successful_rows: 1,
    failed_rows: 0,
    skipped_rows: 0,
    progress: 33,
    created_at: '2026-11-04T00:00:00Z',
    ...over,
  };
}

function renderPage(path = '/settings/import?source=jobber') {
  return render(
    <MemoryRouter initialEntries={[path]}>
      <ImportDataPage />
    </MemoryRouter>
  );
}

const csv = () =>
  new File(['Display Name,E-mails\nDana,dana@x.io\n'], 'clients.csv', {
    type: 'text/csv',
  });

describe('ImportDataPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('previews a Jobber file, imports it and follows it to the end', async () => {
    api.preview.mockResolvedValue(preview());
    api.start.mockResolvedValue(job());
    api.run.mockResolvedValue(
      job({
        status: 'completed',
        processed_rows: 3,
        successful_rows: 2,
        failed_rows: 1,
        errors: [{ row_number: 3, field: 'email', error: 'email is required' }],
      })
    );
    const user = userEvent.setup();
    renderPage();

    expect(screen.getByRole('button', { name: /Jobber$/ })).toHaveAttribute(
      'aria-pressed',
      'true'
    );
    const file = csv();
    await user.upload(screen.getByLabelText('Choose file'), file);

    expect(api.preview).toHaveBeenCalledWith(
      'customers',
      file,
      'jobber',
      undefined
    );
    expect(await screen.findByTestId('import-summary')).toHaveTextContent(
      '3 rows: 2 ready, 1 with problems'
    );
    expect(screen.getByText('Row 3: email is required')).toBeInTheDocument();
    // Jobber files map automatically: no mapping editor.
    expect(screen.queryByLabelText('Column for Email')).not.toBeInTheDocument();

    await user.click(
      screen.getByRole('button', { name: 'Import 2 customers' })
    );
    expect(api.start).toHaveBeenCalledWith('customers', file, 'jobber', {
      full_name: 'Display Name',
      email: 'E-mails',
    });
    expect(await screen.findByText('Import finished')).toBeInTheDocument();
    expect(api.run).toHaveBeenCalledWith('customers', 'job-1');
    expect(screen.getByTestId('import-result')).toHaveTextContent(
      '2 imported, 0 already there or skipped, 1 failed (of 3)'
    );
    expect(
      screen.getByRole('link', { name: 'View customers' })
    ).toHaveAttribute('href', '/customers');
  });

  it('lets a generic CSV map its columns before importing', async () => {
    api.preview.mockResolvedValueOnce(
      preview({
        source: 'generic',
        mapping: { full_name: 'Display Name' },
        missing: ['Email'],
        valid_rows: 0,
        error_rows: 0,
        errors: [],
      })
    );
    api.preview.mockResolvedValueOnce(preview({ source: 'generic' }));
    const user = userEvent.setup();
    renderPage('/settings/import?source=generic');

    const file = csv();
    await user.upload(screen.getByLabelText('Choose file'), file);
    expect(await screen.findByText('Still needed: Email')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: /Import 0 customers/ })
    ).toBeDisabled();

    await user.selectOptions(
      screen.getByLabelText('Column for Email'),
      'E-mails'
    );
    await waitFor(() =>
      expect(api.preview).toHaveBeenLastCalledWith(
        'customers',
        file,
        'generic',
        {
          full_name: 'Display Name',
          email: 'E-mails',
        }
      )
    );
    expect(
      await screen.findByRole('button', { name: 'Import 2 customers' })
    ).toBeEnabled();
  });

  it('imports the chosen entity and shows API errors', async () => {
    api.preview.mockRejectedValue({
      isAxiosError: true,
      response: {
        status: 413,
        data: {
          error: 'file_too_large',
          message: 'file exceeds the maximum of 50000 rows',
        },
      },
    });
    const user = userEvent.setup();
    renderPage('/settings/import?source=housecall_pro');

    await user.click(screen.getByRole('tab', { name: '4. Open invoices' }));
    expect(screen.getByText(/jobs data report/)).toBeInTheDocument();
    await user.upload(screen.getByLabelText('Choose file'), csv());
    expect(api.preview.mock.calls[0][0]).toBe('invoices');
    expect(api.preview.mock.calls[0][2]).toBe('housecall_pro');
    expect(await screen.findByRole('alert')).toBeInTheDocument();
  });

  it('refuses a file over 10 MB without uploading it', async () => {
    const user = userEvent.setup();
    renderPage();
    const big = new File(['x'], 'big.csv', { type: 'text/csv' });
    Object.defineProperty(big, 'size', { value: 11 * 1024 * 1024 });
    await user.upload(screen.getByLabelText('Choose file'), big);
    expect(await screen.findByRole('alert')).toHaveTextContent(
      'larger than 10 MB'
    );
    expect(api.preview).not.toHaveBeenCalled();
  });
});

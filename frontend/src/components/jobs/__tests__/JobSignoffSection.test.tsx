import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AxiosError, type AxiosResponse } from 'axios';

vi.mock('@services/jobSignoffService', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@services/jobSignoffService')>()),
  jobSignoffService: {
    get: vi.fn(),
    capture: vi.fn(),
    emailReport: vi.fn(),
    reportLink: vi.fn(),
  },
}));

import {
  jobSignoffService,
  type JobSignoff,
} from '@services/jobSignoffService';
import { JobSignoffSection } from '../JobSignoffSection';
import { signoffTime } from '../jobSignoff';

const signoff: JobSignoff = {
  id: 'so-1',
  job_id: 'job-1',
  signer_name: 'Ray Burke',
  signer_title: 'Site manager',
  signed_at: '2026-11-03T00:58:00Z', // Nov 2, 6:58 PM in Chicago
  captured_by: 'tech-1',
  captured_by_name: 'Tasha Green',
  signature_url: 'https://store/sig.png',
  signature_path: '/api/v1/jobs/job-1/signoff/signature',
  report_path: '/api/v1/jobs/job-1/signoff/report',
  report_recipients: [],
  report_email_count: 0,
  suggested_recipients: ['karen@harborview.test'],
  locked: false,
};

const emailed: JobSignoff = {
  ...signoff,
  report_emailed_at: '2026-11-03T00:59:00Z',
  report_recipients: ['karen@harborview.test'],
  report_email_count: 1,
  locked: true,
};

const renderSection = (
  props: Partial<Parameters<typeof JobSignoffSection>[0]> = {}
) =>
  render(
    <QueryClientProvider
      client={
        new QueryClient({ defaultOptions: { queries: { retry: false } } })
      }
    >
      <JobSignoffSection
        jobId="job-1"
        jobStatus="completed"
        canEdit
        timeZone="America/Chicago"
        siteName="Elm Street Plaza"
        {...props}
      />
    </QueryClientProvider>
  );

describe('JobSignoffSection (T176)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('shows the signature, signer and emailed report (mockup 09) and resends', async () => {
    vi.mocked(jobSignoffService.get).mockResolvedValue({
      signoff: emailed,
      recipients: ['karen@harborview.test'],
    });
    vi.mocked(jobSignoffService.emailReport).mockResolvedValue({
      sent: true,
      signoff: { ...emailed, report_email_count: 2 },
    });
    renderSection();
    const user = userEvent.setup();

    expect(
      await screen.findByRole('img', { name: 'Signature of Ray Burke' })
    ).toHaveAttribute('src', 'https://store/sig.png');
    expect(screen.getByText('Ray Burke, Site manager')).toBeInTheDocument();
    expect(
      screen.getByText("Signed on Tasha Green's phone at Mon, Nov 2 at 6:58 PM")
    ).toBeInTheDocument();
    expect(
      screen.getByText('Service report emailed to karen@harborview.test')
    ).toBeInTheDocument();
    // Locked once emailed: no re-signing.
    expect(
      screen.queryByRole('button', { name: /sign/i })
    ).not.toBeInTheDocument();

    await user.click(screen.getByRole('button', { name: 'Resend report' }));
    expect(jobSignoffService.emailReport).toHaveBeenCalledWith('job-1', true);
    expect(await screen.findByRole('status')).toHaveTextContent(
      'Service report sent to karen@harborview.test.'
    );
    expect(screen.getByText(/sent 2 times/)).toBeInTheDocument();
  });

  it('offers to email the report after a completed visit on the phone', async () => {
    vi.mocked(jobSignoffService.get).mockResolvedValue({
      signoff,
      recipients: ['karen@harborview.test', 'ray@harborview.test'],
    });
    vi.mocked(jobSignoffService.emailReport).mockResolvedValue({
      sent: true,
      signoff: {
        ...emailed,
        report_recipients: ['karen@harborview.test', 'ray@harborview.test'],
      },
    });
    renderSection({ variant: 'phone' });
    const user = userEvent.setup();

    expect(
      await screen.findByText('Visit complete. Email the service report?')
    ).toBeInTheDocument();
    expect(
      screen.getByText('It goes to karen@harborview.test, ray@harborview.test.')
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Email report' }));
    expect(jobSignoffService.emailReport).toHaveBeenCalledWith('job-1', false);
    expect(
      await screen.findByText(
        'Service report emailed to karen@harborview.test, ray@harborview.test'
      )
    ).toBeInTheDocument();
  });

  it('says when a repeated send sent nothing, and shows send errors', async () => {
    vi.mocked(jobSignoffService.get).mockResolvedValue({
      signoff,
      recipients: ['karen@harborview.test'],
    });
    vi.mocked(jobSignoffService.emailReport)
      .mockRejectedValueOnce(
        new AxiosError('Bad Gateway', '502', undefined, undefined, {
          status: 502,
          data: {
            error: 'email_failed',
            message: "The service report couldn't be emailed. Try again.",
          },
        } as AxiosResponse)
      )
      .mockResolvedValueOnce({ sent: false, signoff: emailed });
    renderSection();
    const user = userEvent.setup();

    await user.click(
      await screen.findByRole('button', { name: 'Email report' })
    );
    expect(await screen.findByRole('alert')).toHaveTextContent(
      "The service report couldn't be emailed. Try again."
    );
    await user.click(screen.getByRole('button', { name: 'Email report' }));
    expect(await screen.findByRole('status')).toHaveTextContent(
      'The service report had already been emailed.'
    );
  });

  it('explains when there is nobody to email', async () => {
    vi.mocked(jobSignoffService.get).mockResolvedValue({
      signoff: { ...signoff, suggested_recipients: [] },
      recipients: [],
    });
    renderSection();
    expect(
      await screen.findByText('No email address for the report')
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: 'Email report' })
    ).not.toBeInTheDocument();
  });

  it('opens the report PDF in a new tab', async () => {
    vi.mocked(jobSignoffService.get).mockResolvedValue({
      signoff,
      recipients: ['karen@harborview.test'],
    });
    vi.mocked(jobSignoffService.reportLink).mockResolvedValue(
      'https://store/report.pdf'
    );
    const tab = { opener: {}, location: { href: '' }, close: vi.fn() };
    const open = vi
      .spyOn(window, 'open')
      .mockReturnValue(tab as unknown as Window);
    renderSection();
    const user = userEvent.setup();

    await user.click(
      await screen.findByRole('button', { name: 'Download PDF' })
    );
    expect(open).toHaveBeenCalledWith('', '_blank');
    await waitFor(() =>
      expect(tab.location.href).toBe('https://store/report.pdf')
    );
    expect(tab.opener).toBeNull();
  });

  it('lets the office get a signature on an unsigned visit', async () => {
    vi.mocked(jobSignoffService.get).mockResolvedValue({
      signoff: null,
      recipients: ['karen@harborview.test'],
    });
    renderSection({ jobStatus: 'in_progress' });
    const user = userEvent.setup();
    expect(await screen.findByText(/Not signed yet/)).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: 'Get signature' }));
    expect(
      screen.getByRole('dialog', { name: 'Customer sign-off' })
    ).toBeInTheDocument();
    expect(screen.getByText('Elm Street Plaza')).toBeInTheDocument();
  });

  it('renders nothing where sign-off is not available (404) and on an unsigned phone visit', async () => {
    vi.mocked(jobSignoffService.get).mockRejectedValue(
      new AxiosError('Not Found', '404', undefined, undefined, {
        status: 404,
        data: { error: 'not_found', message: 'Not found' },
      } as AxiosResponse)
    );
    const { container, unmount } = renderSection();
    await waitFor(() => expect(jobSignoffService.get).toHaveBeenCalled());
    await waitFor(() => expect(container).toBeEmptyDOMElement());
    unmount();

    vi.mocked(jobSignoffService.get).mockResolvedValue({
      signoff: null,
      recipients: [],
    });
    const phone = renderSection({ variant: 'phone' });
    await waitFor(() => expect(phone.container).toBeEmptyDOMElement());
  });
});

describe('signoffTime', () => {
  it('shows the time alone today and the date otherwise', () => {
    const now = new Date('2026-11-03T02:00:00Z'); // Nov 2, 8 PM Chicago
    expect(signoffTime('2026-11-03T00:58:00Z', 'America/Chicago', now)).toBe(
      '6:58 PM'
    );
    expect(signoffTime('2026-11-01T00:58:00Z', 'America/Chicago', now)).toBe(
      'Sat, Oct 31 at 7:58 PM'
    );
  });
});

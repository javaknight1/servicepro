import { useState } from 'react';
import { useNavigate } from 'react-router';
import { jobService } from '@services/jobService';
import { trackEvent, AnalyticsEvents } from '@services/analytics';
import { getApiErrorInfo, getErrorMessage } from '@/utils/error';

/**
 * "Generate invoice" for a completed job (T029): POST /jobs/{id}/invoice,
 * then open the draft invoice. When the job already has one (409
 * already_invoiced, e.g. a double click or a consolidated invoice), open
 * that invoice instead; other refusals come back as `error`.
 */
export function useInvoiceJob() {
  const navigate = useNavigate();
  const [pendingJobId, setPendingJobId] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const invoiceJob = async (jobId: string) => {
    setPendingJobId(jobId);
    setError(null);
    try {
      const invoice = await jobService.invoiceJob(jobId);
      trackEvent(AnalyticsEvents.INVOICE_CREATED, {
        line_items: invoice.lines?.length ?? 0,
        source: 'job',
      });
      navigate(`/invoices/${invoice.id}`);
    } catch (err: unknown) {
      const { code, message, resourceId } = getApiErrorInfo(err);
      if (code === 'already_invoiced' && resourceId) {
        navigate(`/invoices/${resourceId}`);
        return;
      }
      setError(message || getErrorMessage(err, 'Failed to create the invoice'));
    } finally {
      setPendingJobId(null);
    }
  };

  return { invoiceJob, pendingJobId, error };
}

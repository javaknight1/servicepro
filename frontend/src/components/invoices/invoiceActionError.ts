import { getApiErrorInfo, getErrorMessage } from '@/utils/error';

const GENERIC_API_MESSAGE = 'The request could not be processed';

function sentence(text: string): string {
  const trimmed = text.trim();
  const capitalized = trimmed.charAt(0).toUpperCase() + trimmed.slice(1);
  return /[.!?]$/.test(capitalized) ? capitalized : `${capitalized}.`;
}

/**
 * Message for a failed invoice money action (payment, refund, cancel).
 *
 * The invoice API answers business-rule failures with a human sentence in
 * `error` (e.g. "payment amount exceeds outstanding balance"):
 * - 400: the request breaks a rule (overpayment, invoice not open for
 *   payments, refund larger than the amount paid, invalid amount);
 * - 409: the invoice is closed or its status changed (e.g. someone else just
 *   paid or cancelled it) — reload and look again.
 */
export function getInvoiceActionError(error: unknown, action: string): string {
  const { status, code, message } = getApiErrorInfo(error);
  const detail = message || code;
  const useful = detail && detail !== GENERIC_API_MESSAGE ? detail : undefined;

  switch (status) {
    case 400:
      return useful
        ? sentence(useful)
        : `${action} failed: check the amount and try again.`;
    case 409:
      return `${useful ? sentence(useful) : 'This invoice is closed or its status changed.'} Reload the invoice to see its current state.`;
    case 403:
      return "You don't have permission to do this.";
    case 404:
      return 'This invoice no longer exists.';
    default:
      return useful
        ? sentence(useful)
        : getErrorMessage(error, `${action} failed. Please try again.`);
  }
}

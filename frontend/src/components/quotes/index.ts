/**
 * Quote Components
 *
 * Export all quote-related components for easy importing
 */

// Components
export { LineItemList } from './LineItemList';
export { LineItemRow } from './LineItemRow';
export { LineItemForm } from './LineItemForm';
export { TotalCalculator } from './TotalCalculator';
export { QuoteList } from './QuoteList';
export { QuoteDetail } from './QuoteDetail';
export { QuoteStatusBadge } from './QuoteStatusBadge';
export { QuoteStatusTimeline } from './QuoteStatusTimeline';
export { QuoteStatusActions } from './QuoteStatusActions';
export { QuoteActionsMenu } from './QuoteActionsMenu';

// Hooks
export {
  useQuoteCalculations,
  useLineItemCalculation,
  useTaxRateLookup,
  useQuoteTotals,
  useAutoSave,
  useManualSave,
  formatLastSaved,
  type AutoSaveOptions,
  type AutoSaveStatus,
} from './hooks';

// Utilities
export {
  QuotePDFDocument,
  generateQuotePDF,
  downloadQuotePDF,
  PDFDownloadButton,
  previewQuotePDF,
  emailQuotePDF,
} from './utils';

// Validation
export {
  quoteFormSchema,
  lineItemSchema,
  customerSchema,
  termsSchema,
  getDefaultQuoteValues,
  validationMessages,
  type QuoteFormData,
  type LineItemFormData,
  type CustomerFormData,
  type TermsFormData,
} from './validation';

// Re-export types from types/quote
export type {
  LineItem,
  LineItemValidationErrors,
  Quote,
  QuoteRequest,
  QuoteStatus,
} from '../../types/quote';

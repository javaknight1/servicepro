/**
 * Quote Hooks
 * Export all custom hooks for quote management
 */

export {
  useQuoteCalculations,
  useLineItemCalculation,
  useTaxRateLookup,
  useQuoteTotals,
} from './useQuoteCalculations';

export {
  useAutoSave,
  useManualSave,
  formatLastSaved,
  type AutoSaveOptions,
  type AutoSaveStatus,
} from './useAutoSave';

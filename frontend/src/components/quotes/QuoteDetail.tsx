import React, { useEffect, useState, useCallback } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  quoteFormSchema,
  QuoteFormData,
  getDefaultQuoteValues,
} from './validation';
import { useQuoteCalculations } from './hooks/useQuoteCalculations';
import { useAutoSave, formatLastSaved } from './hooks/useAutoSave';
import { LineItemList } from './LineItemList';
import { TotalCalculator } from './TotalCalculator';
import { downloadQuotePDF, previewQuotePDF } from './utils/pdfGenerator';
import { quoteService } from '../../services/quoteService';
import { Quote, LineItem, QuoteStatus } from '../../types/quote';

interface QuoteDetailProps {
  quoteId?: string;
  onSave?: (quote: Quote) => void;
  onCancel?: () => void;
  enableAutoSave?: boolean;
}

export const QuoteDetail: React.FC<QuoteDetailProps> = ({
  quoteId,
  onSave,
  onCancel,
  enableAutoSave = true,
}) => {
  const queryClient = useQueryClient();
  const [_showPDFPreview, _setShowPDFPreview] = useState(false);

  // Fetch existing quote if editing
  const { data: existingQuote, isLoading: isLoadingQuote } = useQuery({
    queryKey: ['quote', quoteId],
    queryFn: () => quoteService.getQuote(quoteId!),
    enabled: !!quoteId,
  });

  // Initialize form with React Hook Form
  const {
    register,
    control,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isDirty },
    reset,
  } = useForm<QuoteFormData>({
    resolver: zodResolver(quoteFormSchema),
    defaultValues: getDefaultQuoteValues(),
  });

  // Load existing quote data into form
  useEffect(() => {
    if (existingQuote) {
      // Cast to unknown first to handle type differences between Quote and QuoteFormData
      reset({
        ...existingQuote,
        tax_exemption_type: 'none',
      } as unknown as QuoteFormData);
    }
  }, [existingQuote, reset]);

  // Real-time calculations
  const {
    subtotal: _subtotal,
    taxAmount: _taxAmount,
    total: _total,
  } = useQuoteCalculations(watch, setValue);

  // Save mutation
  const saveMutation = useMutation({
    mutationFn: async (data: QuoteFormData) => {
      if (quoteId) {
        return await quoteService.updateQuote(
          quoteId,
          data as unknown as Partial<Quote>
        );
      } else {
        return await quoteService.createQuote(
          data as unknown as Partial<Quote>
        );
      }
    },
    onSuccess: (savedQuote) => {
      queryClient.invalidateQueries({ queryKey: ['quotes'] });
      queryClient.invalidateQueries({ queryKey: ['quote', quoteId] });
      onSave?.(savedQuote);
    },
  });

  // Auto-save functionality
  const autoSaveStatus = useAutoSave(
    watch,
    async (data) => {
      if (quoteId && isDirty) {
        await saveMutation.mutateAsync(data);
      }
    },
    {
      enabled: enableAutoSave && !!quoteId,
      delay: 2000,
    }
  );

  // Manual save handler
  const onSubmit = async (data: QuoteFormData) => {
    try {
      await saveMutation.mutateAsync(data);
    } catch (error) {
      console.error('Failed to save quote:', error);
    }
  };

  // Line items management
  const items = watch('items') || [];
  const _handleItemsChange = useCallback(
    (updatedItems: LineItem[]) => {
      setValue('items', updatedItems, {
        shouldValidate: true,
        shouldDirty: true,
      });
    },
    [setValue]
  );

  // PDF handlers
  const handleDownloadPDF = async () => {
    const formData = watch();
    const quoteData = {
      ...formData,
      id: quoteId || 'draft',
      quote_number: existingQuote?.quote_number || 'DRAFT',
      created_at: existingQuote?.created_at || new Date().toISOString(),
      updated_at: new Date().toISOString(),
      is_expired: false,
      created_by: '',
    } as unknown as Quote;

    await downloadQuotePDF(quoteData);
  };

  const handlePreviewPDF = async () => {
    const formData = watch();
    const quoteData = {
      ...formData,
      id: quoteId || 'draft',
      quote_number: existingQuote?.quote_number || 'DRAFT',
      created_at: existingQuote?.created_at || new Date().toISOString(),
      updated_at: new Date().toISOString(),
      is_expired: false,
      created_by: '',
    } as unknown as Quote;

    await previewQuotePDF(quoteData);
  };

  // Send quote to customer
  const sendMutation = useMutation({
    mutationFn: () => quoteService.sendQuote(quoteId!),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['quote', quoteId] });
    },
  });

  if (isLoadingQuote) {
    return (
      <div className="flex h-64 items-center justify-center">
        <div className="h-8 w-8 animate-spin rounded-full border-4 border-blue-600 border-t-transparent"></div>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      {/* Header with actions */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">
            {quoteId
              ? `Edit Quote ${existingQuote?.quote_number}`
              : 'New Quote'}
          </h2>
          {enableAutoSave && quoteId && (
            <p className="mt-1 text-sm text-gray-500">
              {autoSaveStatus.isSaving ? (
                <span className="flex items-center gap-2">
                  <span className="h-4 w-4 animate-spin rounded-full border-2 border-blue-600 border-t-transparent"></span>
                  Saving...
                </span>
              ) : (
                <span>
                  Last saved: {formatLastSaved(autoSaveStatus.lastSaved)}
                </span>
              )}
            </p>
          )}
        </div>

        <div className="flex gap-2">
          {quoteId && (
            <>
              <button
                type="button"
                onClick={handlePreviewPDF}
                className="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
              >
                Preview PDF
              </button>
              <button
                type="button"
                onClick={handleDownloadPDF}
                className="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
              >
                Download PDF
              </button>
              {existingQuote?.status === QuoteStatus.DRAFT && (
                <button
                  type="button"
                  onClick={() => sendMutation.mutate()}
                  disabled={sendMutation.isPending}
                  className="rounded-md bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50"
                >
                  {sendMutation.isPending ? 'Sending...' : 'Send to Customer'}
                </button>
              )}
            </>
          )}
          {onCancel && (
            <button
              type="button"
              onClick={onCancel}
              className="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
            >
              Cancel
            </button>
          )}
          <button
            type="submit"
            disabled={saveMutation.isPending}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {saveMutation.isPending
              ? 'Saving...'
              : quoteId
                ? 'Save'
                : 'Create Quote'}
          </button>
        </div>
      </div>

      {/* Error display */}
      {(saveMutation.error || autoSaveStatus.error) && (
        <div className="rounded-md bg-red-50 p-4">
          <div className="flex">
            <div className="flex-shrink-0">
              <svg
                className="h-5 w-5 text-red-400"
                viewBox="0 0 20 20"
                fill="currentColor"
              >
                <path
                  fillRule="evenodd"
                  d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                  clipRule="evenodd"
                />
              </svg>
            </div>
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">
                {saveMutation.error
                  ? 'Failed to save quote'
                  : 'Auto-save failed'}
              </h3>
              <p className="mt-2 text-sm text-red-700">
                {saveMutation.error instanceof Error
                  ? saveMutation.error.message
                  : autoSaveStatus.error?.message}
              </p>
            </div>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Main content - 2 columns */}
        <div className="lg:col-span-2 space-y-6">
          {/* Customer Information Section */}
          <div className="rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
            <h3 className="mb-4 text-lg font-medium text-gray-900">
              Customer Information
            </h3>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-700">
                  Customer Name *
                </label>
                <input
                  {...register('customer_name')}
                  type="text"
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                />
                {errors.customer_name && (
                  <p className="mt-1 text-sm text-red-600">
                    {errors.customer_name.message}
                  </p>
                )}
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Email *
                </label>
                <input
                  {...register('customer_email')}
                  type="email"
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                />
                {errors.customer_email && (
                  <p className="mt-1 text-sm text-red-600">
                    {errors.customer_email.message}
                  </p>
                )}
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Phone
                </label>
                <input
                  {...register('customer_phone')}
                  type="tel"
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                />
                {errors.customer_phone && (
                  <p className="mt-1 text-sm text-red-600">
                    {errors.customer_phone.message}
                  </p>
                )}
              </div>

              <div className="md:col-span-2">
                <label className="block text-sm font-medium text-gray-700">
                  Address
                </label>
                <input
                  {...register('customer_address')}
                  type="text"
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  City
                </label>
                <input
                  {...register('customer_city')}
                  type="text"
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  State
                </label>
                <input
                  {...register('customer_state')}
                  type="text"
                  maxLength={2}
                  placeholder="CA"
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                />
                {errors.customer_state && (
                  <p className="mt-1 text-sm text-red-600">
                    {errors.customer_state.message}
                  </p>
                )}
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  ZIP Code
                </label>
                <input
                  {...register('customer_zip')}
                  type="text"
                  placeholder="12345"
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                />
                {errors.customer_zip && (
                  <p className="mt-1 text-sm text-red-600">
                    {errors.customer_zip.message}
                  </p>
                )}
              </div>
            </div>
          </div>

          {/* Line Items Section */}
          <div className="rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
            <h3 className="mb-4 text-lg font-medium text-gray-900">
              Line Items *
            </h3>
            <Controller
              name="items"
              control={control}
              render={({ field }) => (
                <LineItemList
                  items={field.value || []}
                  onChange={field.onChange}
                  taxRate={parseFloat(watch('tax_rate') || '0')}
                />
              )}
            />
            {errors.items && (
              <p className="mt-2 text-sm text-red-600">
                {errors.items.message}
              </p>
            )}
          </div>

          {/* Terms Section */}
          <div className="rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
            <h3 className="mb-4 text-lg font-medium text-gray-900">
              Terms & Conditions
            </h3>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Valid Until *
                </label>
                <input
                  {...register('valid_until')}
                  type="date"
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                />
                {errors.valid_until && (
                  <p className="mt-1 text-sm text-red-600">
                    {errors.valid_until.message}
                  </p>
                )}
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Payment Terms
                </label>
                <input
                  {...register('payment_terms')}
                  type="text"
                  placeholder="Net 30"
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Delivery Timeline
                </label>
                <input
                  {...register('delivery_timeline')}
                  type="text"
                  placeholder="2-3 weeks"
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Warranty Information
                </label>
                <textarea
                  {...register('warranty_info')}
                  rows={3}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                />
              </div>
            </div>
          </div>

          {/* Notes Section */}
          <div className="rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
            <h3 className="mb-4 text-lg font-medium text-gray-900">Notes</h3>
            <textarea
              {...register('notes')}
              rows={4}
              placeholder="Additional notes or special instructions..."
              className="block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
            />
            {errors.notes && (
              <p className="mt-1 text-sm text-red-600">
                {errors.notes.message}
              </p>
            )}
          </div>
        </div>

        {/* Sidebar - 1 column */}
        <div className="space-y-6">
          {/* Tax Information */}
          <div className="rounded-lg border border-gray-200 bg-white p-6 shadow-sm">
            <h3 className="mb-4 text-lg font-medium text-gray-900">
              Tax Information
            </h3>

            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Tax ZIP Code
                </label>
                <input
                  {...register('tax_zip_code')}
                  type="text"
                  placeholder="12345"
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Tax Rate (decimal)
                </label>
                <input
                  {...register('tax_rate')}
                  type="text"
                  placeholder="0.0825"
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                />
                <p className="mt-1 text-xs text-gray-500">
                  Example: 0.0825 for 8.25%
                </p>
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Tax Exemption
                </label>
                <select
                  {...register('tax_exemption_type')}
                  className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                >
                  <option value="none">None</option>
                  <option value="nonprofit">Non-Profit</option>
                  <option value="government">Government</option>
                  <option value="resale">Resale</option>
                  <option value="manufacture">Manufacturing</option>
                  <option value="educational">Educational</option>
                </select>
              </div>

              {watch('tax_exemption_type') !== 'none' && (
                <div>
                  <label className="block text-sm font-medium text-gray-700">
                    Exemption Number
                  </label>
                  <input
                    {...register('tax_exemption_number')}
                    type="text"
                    className="mt-1 block w-full rounded-md border-gray-300 shadow-sm focus:border-blue-500 focus:ring-blue-500"
                  />
                  {errors.tax_exemption_number && (
                    <p className="mt-1 text-sm text-red-600">
                      {errors.tax_exemption_number.message}
                    </p>
                  )}
                </div>
              )}
            </div>
          </div>

          {/* Totals */}
          <div className="sticky top-6">
            <TotalCalculator
              items={items}
              taxRate={parseFloat(watch('tax_rate') || '0')}
            />
          </div>
        </div>
      </div>
    </form>
  );
};

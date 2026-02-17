import { useState, useEffect } from 'react';
import { useNavigate, useParams, useSearchParams } from 'react-router-dom';
import { DashboardLayout } from '@components/layout';
import { Button } from '@components/shared';
import { quoteService } from '@services/quoteService';
import { getErrorMessage } from '@/utils/error';
import { trackEvent, AnalyticsEvents } from '@services/analytics';
import { customerService, Customer } from '@services/customerService';
import type { Quote, LineItemFormData } from '@app-types/quote';
import {
  ArrowLeft,
  Save,
  Trash2,
  Loader2,
  Plus,
  X,
  Send,
  Download,
  Check,
  XCircle,
} from 'lucide-react';

interface QuoteFormData {
  customer_id: string;
  valid_until: string;
  tax_rate: string;
  notes: string;
  terms: string;
}

const initialFormData: QuoteFormData = {
  customer_id: '',
  valid_until: '',
  tax_rate: '0',
  notes: '',
  terms: '',
};

const emptyLineItem: LineItemFormData = {
  description: '',
  quantity: 1,
  unit_price: 0,
};

export function QuoteDetailPage() {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const cloneFromId = searchParams.get('clone');
  const isNew = !id || id === 'new';
  const isCloning = isNew && !!cloneFromId;

  const [formData, setFormData] = useState<QuoteFormData>(initialFormData);
  const [lineItems, setLineItems] = useState<LineItemFormData[]>([
    { ...emptyLineItem },
  ]);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [isLoading, setIsLoading] = useState(!isNew);
  const [isLoadingCustomers, setIsLoadingCustomers] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [isSendingQuote, setIsSendingQuote] = useState(false);
  const [isAcceptingQuote, setIsAcceptingQuote] = useState(false);
  const [isDecliningQuote, setIsDecliningQuote] = useState(false);
  const [isDownloadingPDF, setIsDownloadingPDF] = useState(false);
  const [quoteStatus, setQuoteStatus] = useState<string>('draft');
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    loadCustomers();
    if (!isNew && id) {
      loadQuote(id);
    } else if (isCloning && cloneFromId) {
      // Load source quote for cloning
      loadQuoteForCloning(cloneFromId);
    } else {
      // Set default valid_until to 30 days from now
      const defaultDate = new Date();
      defaultDate.setDate(defaultDate.getDate() + 30);
      setFormData((prev) => ({
        ...prev,
        valid_until: defaultDate.toISOString().split('T')[0],
      }));
    }
  }, [id, isNew, isCloning, cloneFromId]);

  const loadCustomers = async () => {
    setIsLoadingCustomers(true);
    try {
      const response = await customerService.getCustomers({ page_size: 100 });
      setCustomers(response.customers);
    } catch (err) {
      console.error('Failed to load customers:', err);
    } finally {
      setIsLoadingCustomers(false);
    }
  };

  const loadQuote = async (quoteId: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const quote = await quoteService.getQuote(quoteId);
      // Convert tax_rate from decimal (0.07) back to percentage (7) for display
      const taxRatePercent = quote.tax_rate
        ? (quote.tax_rate * 100).toString()
        : '0';
      setFormData({
        customer_id: quote.customer_id || '',
        valid_until: quote.valid_until ? quote.valid_until.split('T')[0] : '',
        tax_rate: taxRatePercent,
        notes: quote.notes || '',
        terms: quote.terms || '',
      });
      if (quote.items && quote.items.length > 0) {
        setLineItems(
          quote.items.map((item) => ({
            description: item.description,
            quantity: item.quantity,
            unit_price: item.unit_price,
          }))
        );
      }
      setQuoteStatus(quote.status || 'draft');
    } catch (err) {
      console.error('Failed to load quote:', err);
      setError('Failed to load quote details');
    } finally {
      setIsLoading(false);
    }
  };

  const loadQuoteForCloning = async (sourceQuoteId: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const quote = await quoteService.getQuote(sourceQuoteId);
      // Convert tax_rate from decimal (0.07) back to percentage (7) for display
      const taxRatePercent = quote.tax_rate
        ? (quote.tax_rate * 100).toString()
        : '0';
      // Set default valid_until to 30 days from now for the cloned quote
      const defaultDate = new Date();
      defaultDate.setDate(defaultDate.getDate() + 30);
      setFormData({
        customer_id: quote.customer_id || '',
        valid_until: defaultDate.toISOString().split('T')[0],
        tax_rate: taxRatePercent,
        notes: quote.notes || '',
        terms: quote.terms || '',
      });
      if (quote.items && quote.items.length > 0) {
        setLineItems(
          quote.items.map((item) => ({
            description: item.description,
            quantity: item.quantity,
            unit_price: item.unit_price,
          }))
        );
      }
      // New cloned quote starts as draft
      setQuoteStatus('draft');
    } catch (err) {
      console.error('Failed to load quote for cloning:', err);
      setError('Failed to load source quote for cloning');
    } finally {
      setIsLoading(false);
    }
  };

  const handleChange = (
    e: React.ChangeEvent<
      HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement
    >
  ) => {
    const { name, value } = e.target;
    setFormData((prev) => ({ ...prev, [name]: value }));
  };

  const handleLineItemChange = (
    index: number,
    field: keyof LineItemFormData,
    value: string | number
  ) => {
    setLineItems((prev) => {
      const updated = [...prev];
      updated[index] = { ...updated[index], [field]: value };
      return updated;
    });
  };

  const addLineItem = () => {
    setLineItems((prev) => [...prev, { ...emptyLineItem }]);
  };

  const removeLineItem = (index: number) => {
    if (lineItems.length > 1) {
      setLineItems((prev) => prev.filter((_, i) => i !== index));
    }
  };

  const calculateSubtotal = () => {
    return lineItems.reduce((sum, item) => {
      const qty =
        typeof item.quantity === 'string'
          ? parseFloat(item.quantity) || 0
          : item.quantity;
      const price =
        typeof item.unit_price === 'string'
          ? parseFloat(item.unit_price) || 0
          : item.unit_price;
      return sum + qty * price;
    }, 0);
  };

  const calculateTax = () => {
    const taxRate = parseFloat(formData.tax_rate) || 0;
    return calculateSubtotal() * (taxRate / 100);
  };

  const calculateTotal = () => {
    return calculateSubtotal() + calculateTax();
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSaving(true);
    setError(null);

    // Convert date string to RFC3339 format (backend expects time.Time)
    const validUntilDate = formData.valid_until
      ? new Date(formData.valid_until + 'T23:59:59Z').toISOString()
      : undefined;

    // Convert tax rate from percentage (e.g., 7) to decimal (e.g., 0.07)
    const taxRateDecimal = (parseFloat(formData.tax_rate) || 0) / 100;

    const submitData = {
      customer_id: formData.customer_id,
      valid_until: validUntilDate,
      tax_rate: taxRateDecimal,
      notes: formData.notes || undefined,
      terms: formData.terms || undefined,
      items: lineItems.map((item, index) => ({
        description: item.description,
        quantity:
          typeof item.quantity === 'string'
            ? parseFloat(item.quantity) || 0
            : item.quantity,
        unit_price:
          typeof item.unit_price === 'string'
            ? parseFloat(item.unit_price) || 0
            : item.unit_price,
        sort_order: index,
      })),
    };

    try {
      if (isNew) {
        await quoteService.createQuote(submitData as unknown as Partial<Quote>);
        trackEvent(AnalyticsEvents.QUOTE_CREATED, {
          total: calculateTotal(),
          line_items: lineItems.length,
        });
      } else if (id) {
        await quoteService.updateQuote(
          id,
          submitData as unknown as Partial<Quote>
        );
      }
      navigate('/quotes');
    } catch (err) {
      console.error('Failed to save quote:', err);
      setError(getErrorMessage(err, 'Failed to save quote'));
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!id || isNew) return;
    if (!window.confirm('Are you sure you want to delete this quote?')) return;

    setIsDeleting(true);
    setError(null);
    try {
      await quoteService.deleteQuote(id);
      navigate('/quotes');
    } catch (err) {
      console.error('Failed to delete quote:', err);
      setError(getErrorMessage(err, 'Failed to delete quote'));
    } finally {
      setIsDeleting(false);
    }
  };

  const handleSendQuote = async () => {
    if (!id || isNew || quoteStatus !== 'draft') return;

    setIsSendingQuote(true);
    setError(null);
    try {
      await quoteService.sendQuote(id);
      setQuoteStatus('sent');
      trackEvent(AnalyticsEvents.QUOTE_SENT, { total: calculateTotal() });
    } catch (err: unknown) {
      console.error('Failed to send quote:', err);
      const error = err as { response?: { data?: { message?: string } } };
      setError(error?.response?.data?.message || 'Failed to send quote');
    } finally {
      setIsSendingQuote(false);
    }
  };

  const handleDownloadPDF = async () => {
    if (!id || isNew) return;

    setIsDownloadingPDF(true);
    try {
      await quoteService.downloadQuotePDF(id);
      trackEvent(AnalyticsEvents.EXPORT_DOWNLOADED, {
        format: 'pdf',
        type: 'quote',
      });
    } catch (err) {
      console.error('Failed to download PDF:', err);
    } finally {
      setIsDownloadingPDF(false);
    }
  };

  const handleAcceptQuote = async () => {
    if (!id || isNew) return;

    setIsAcceptingQuote(true);
    setError(null);
    try {
      await quoteService.acceptQuote(id);
      setQuoteStatus('accepted');
      trackEvent(AnalyticsEvents.QUOTE_ACCEPTED, { total: calculateTotal() });
    } catch (err: unknown) {
      console.error('Failed to accept quote:', err);
      const error = err as { response?: { data?: { message?: string } } };
      setError(error?.response?.data?.message || 'Failed to accept quote');
    } finally {
      setIsAcceptingQuote(false);
    }
  };

  const handleDeclineQuote = async () => {
    if (!id || isNew) return;

    setIsDecliningQuote(true);
    setError(null);
    try {
      await quoteService.rejectQuote(id);
      setQuoteStatus('declined');
      trackEvent(AnalyticsEvents.QUOTE_REJECTED);
    } catch (err: unknown) {
      console.error('Failed to decline quote:', err);
      const error = err as { response?: { data?: { message?: string } } };
      setError(error?.response?.data?.message || 'Failed to decline quote');
    } finally {
      setIsDecliningQuote(false);
    }
  };

  const canAcceptDecline = quoteStatus === 'sent' || quoteStatus === 'viewed';

  if (isLoading) {
    return (
      <DashboardLayout>
        <div className="flex items-center justify-center min-h-96">
          <Loader2 className="h-8 w-8 animate-spin text-primary-500" />
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout>
      <div className="p-6 lg:p-8 max-w-4xl mx-auto">
        <div className="flex items-center justify-between mb-6">
          <div className="flex items-center gap-4">
            <Button
              variant="ghost"
              onClick={() => navigate('/quotes')}
              className="flex items-center gap-2"
            >
              <ArrowLeft className="h-4 w-4" />
              Back
            </Button>
            <h1 className="text-2xl font-bold text-neutral-900">
              {isCloning
                ? 'Clone Quote'
                : isNew
                  ? 'Create Quote'
                  : 'Edit Quote'}
            </h1>
          </div>
          {!isNew && (
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="secondary"
                onClick={handleSendQuote}
                disabled={quoteStatus !== 'draft' || isSendingQuote}
                className="flex items-center gap-2"
              >
                {isSendingQuote ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Send className="h-4 w-4" />
                )}
                Send Quote
              </Button>
              <Button
                type="button"
                variant="secondary"
                onClick={handleDownloadPDF}
                disabled={isDownloadingPDF}
                className="flex items-center gap-2"
              >
                {isDownloadingPDF ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Download className="h-4 w-4" />
                )}
                Download PDF
              </Button>
              {canAcceptDecline && (
                <>
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={handleAcceptQuote}
                    disabled={isAcceptingQuote}
                    className="flex items-center gap-2 text-green-700 border-green-300 hover:bg-green-50"
                  >
                    {isAcceptingQuote ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <Check className="h-4 w-4" />
                    )}
                    Accept
                  </Button>
                  <Button
                    type="button"
                    variant="secondary"
                    onClick={handleDeclineQuote}
                    disabled={isDecliningQuote}
                    className="flex items-center gap-2 text-red-700 border-red-300 hover:bg-red-50"
                  >
                    {isDecliningQuote ? (
                      <Loader2 className="h-4 w-4 animate-spin" />
                    ) : (
                      <XCircle className="h-4 w-4" />
                    )}
                    Decline
                  </Button>
                </>
              )}
            </div>
          )}
        </div>

        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="bg-white rounded-lg border border-neutral-200 p-6">
            <h2 className="text-lg font-semibold text-neutral-900 mb-4">
              Quote Details
            </h2>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label
                  htmlFor="customer_id"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Customer *
                </label>
                <select
                  id="customer_id"
                  name="customer_id"
                  value={formData.customer_id}
                  onChange={handleChange}
                  required
                  disabled={isLoadingCustomers}
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                >
                  <option value="">Select a customer</option>
                  {customers.map((customer) => (
                    <option key={customer.id} value={customer.id}>
                      {customer.first_name} {customer.last_name}
                      {customer.company_name
                        ? ` (${customer.company_name})`
                        : ''}
                    </option>
                  ))}
                </select>
              </div>

              <div>
                <label
                  htmlFor="valid_until"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Valid Until *
                </label>
                <input
                  type="date"
                  id="valid_until"
                  name="valid_until"
                  value={formData.valid_until}
                  onChange={handleChange}
                  required
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>

              <div>
                <label
                  htmlFor="tax_rate"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Tax Rate (%)
                </label>
                <input
                  type="number"
                  id="tax_rate"
                  name="tax_rate"
                  value={formData.tax_rate}
                  onChange={handleChange}
                  min="0"
                  max="100"
                  step="0.01"
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>
            </div>
          </div>

          <div className="bg-white rounded-lg border border-neutral-200 p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-neutral-900">
                Line Items
              </h2>
              <Button
                type="button"
                variant="secondary"
                onClick={addLineItem}
                className="flex items-center gap-2"
              >
                <Plus className="h-4 w-4" />
                Add Item
              </Button>
            </div>

            <div className="space-y-3">
              {lineItems.map((item, index) => (
                <div
                  key={index}
                  className="flex gap-3 items-start p-3 bg-neutral-50 rounded-lg"
                >
                  <div className="flex-1">
                    <input
                      type="text"
                      value={item.description}
                      onChange={(e) =>
                        handleLineItemChange(
                          index,
                          'description',
                          e.target.value
                        )
                      }
                      placeholder="Description"
                      required
                      className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                    />
                  </div>
                  <div className="w-24">
                    <input
                      type="number"
                      value={item.quantity}
                      onChange={(e) =>
                        handleLineItemChange(index, 'quantity', e.target.value)
                      }
                      placeholder="Qty"
                      min="0"
                      step="1"
                      required
                      className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                    />
                  </div>
                  <div className="w-32">
                    <input
                      type="number"
                      value={item.unit_price}
                      onChange={(e) =>
                        handleLineItemChange(
                          index,
                          'unit_price',
                          e.target.value
                        )
                      }
                      placeholder="Price"
                      min="0"
                      step="0.01"
                      required
                      className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                    />
                  </div>
                  <div className="w-24 text-right py-2 font-medium">
                    $
                    {(
                      (typeof item.quantity === 'string'
                        ? parseFloat(item.quantity) || 0
                        : item.quantity) *
                      (typeof item.unit_price === 'string'
                        ? parseFloat(item.unit_price) || 0
                        : item.unit_price)
                    ).toFixed(2)}
                  </div>
                  <Button
                    type="button"
                    variant="ghost"
                    onClick={() => removeLineItem(index)}
                    disabled={lineItems.length === 1}
                    className="text-red-500 hover:text-red-700"
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              ))}
            </div>

            <div className="mt-4 pt-4 border-t border-neutral-200">
              <div className="flex justify-end space-y-1">
                <div className="w-64 space-y-2">
                  <div className="flex justify-between">
                    <span className="text-neutral-600">Subtotal:</span>
                    <span className="font-medium">
                      ${calculateSubtotal().toFixed(2)}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-neutral-600">
                      Tax ({formData.tax_rate}%):
                    </span>
                    <span className="font-medium">
                      ${calculateTax().toFixed(2)}
                    </span>
                  </div>
                  <div className="flex justify-between text-lg font-semibold pt-2 border-t">
                    <span>Total:</span>
                    <span>${calculateTotal().toFixed(2)}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div className="bg-white rounded-lg border border-neutral-200 p-6">
            <h2 className="text-lg font-semibold text-neutral-900 mb-4">
              Notes & Terms
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label
                  htmlFor="notes"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Notes
                </label>
                <textarea
                  id="notes"
                  name="notes"
                  value={formData.notes}
                  onChange={handleChange}
                  rows={3}
                  placeholder="Additional notes for the customer..."
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>
              <div>
                <label
                  htmlFor="terms"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Terms & Conditions
                </label>
                <textarea
                  id="terms"
                  name="terms"
                  value={formData.terms}
                  onChange={handleChange}
                  rows={3}
                  placeholder="Payment terms, conditions..."
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>
            </div>
          </div>

          <div className="flex items-center justify-between pt-4">
            <div>
              {!isNew && (
                <Button
                  type="button"
                  variant="danger"
                  onClick={handleDelete}
                  disabled={isDeleting}
                  className="flex items-center gap-2"
                >
                  {isDeleting ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Trash2 className="h-4 w-4" />
                  )}
                  Delete Quote
                </Button>
              )}
            </div>
            <div className="flex items-center gap-3">
              <Button
                type="button"
                variant="secondary"
                onClick={() => navigate('/quotes')}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                variant="primary"
                disabled={isSaving}
                className="flex items-center gap-2"
              >
                {isSaving ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Save className="h-4 w-4" />
                )}
                {isNew ? 'Create Quote' : 'Save Changes'}
              </Button>
            </div>
          </div>
        </form>
      </div>
    </DashboardLayout>
  );
}

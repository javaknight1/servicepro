import { useState, useEffect, useCallback } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { DashboardLayout } from '@components/layout';
import { Button, Badge, getStatusBadgeVariant } from '@components/shared';
import {
  invoiceService,
  Invoice,
  InvoiceStatus,
  type CreateInvoiceRequest,
  type InvoiceLineRequest,
  type UpdateInvoiceRequest,
} from '@services/invoiceService';
import {
  paymentReminderService,
  type PaymentReminderResponse,
} from '@services/paymentReminderService';
import { getErrorMessage } from '@/utils/error';
import {
  CancelInvoiceModal,
  InvoicePaymentHistory,
  RecordPaymentModal,
  RefundModal,
} from '@components/invoices';
import { getInvoiceStatusLabel } from '@app-types';
import { useTenantStore } from '@store';
import { useTenantPermissions } from '@/utils/tenantPermissions';
import {
  formatDollars,
  lineTotal,
  toAmount,
  percentToTaxRate,
  roundToCents,
  taxRateToPercent,
} from '@/utils/money';
import { trackEvent, AnalyticsEvents } from '@services/analytics';
import type { Customer } from '@services/customerService';
import { fetchAllCustomers } from '@services/customerLookup';
import { paymentTermService } from '@services/paymentTermService';
import {
  ArrowLeft,
  Save,
  Trash2,
  Loader2,
  Plus,
  X,
  Send,
  Mail,
  Download,
  FileText,
  Bell,
  DollarSign,
  Ban,
  Undo2,
} from 'lucide-react';

interface InvoiceFormData {
  customer_id: string;
  issue_date: string;
  due_date: string;
  /** Percent ("8.25"); sent as a direct rate fraction. */
  tax_rate: string;
  notes: string;
  terms_and_conditions: string;
}

interface LineItemFormData {
  description: string;
  quantity: number | string;
  unit_price: number | string;
  taxable: boolean;
}

/**
 * The rate an invoice actually uses: the server stores it on every line
 * (`lines[].tax_rate`, also when it came from a direct rate); `tax_rate` on
 * the invoice is the configured tax-rate row for `tax_rate_id`, if any.
 */
/** "YYYY-MM-DD" plus `days`, as "YYYY-MM-DD". */
function addDays(date: string, days: number): string {
  const d = new Date(date + 'T00:00:00Z');
  d.setUTCDate(d.getUTCDate() + days);
  return d.toISOString().split('T')[0];
}

function storedTaxRate(invoice: Invoice): string | number | undefined {
  return invoice.lines?.[0]?.tax_rate ?? invoice.tax_rate?.rate;
}

const initialFormData: InvoiceFormData = {
  customer_id: '',
  issue_date: '',
  due_date: '',
  tax_rate: '0',
  notes: '',
  terms_and_conditions: '',
};

const emptyLineItem: LineItemFormData = {
  description: '',
  quantity: 1,
  unit_price: 0,
  taxable: true,
};

export function InvoiceDetailPage() {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const isNew = !id || id === 'new';

  const [formData, setFormData] = useState<InvoiceFormData>(initialFormData);
  const [lineItems, setLineItems] = useState<LineItemFormData[]>([
    { ...emptyLineItem },
  ]);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [isLoading, setIsLoading] = useState(!isNew);
  const [isLoadingCustomers, setIsLoadingCustomers] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [isSending, setIsSending] = useState(false);
  const [isDownloadingInvoice, setIsDownloadingInvoice] = useState(false);
  const [isDownloadingReceipt, setIsDownloadingReceipt] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [invoiceStatus, setInvoiceStatus] = useState<InvoiceStatus>(
    InvoiceStatus.DRAFT
  );
  const [invoiceNumber, setInvoiceNumber] = useState<string>('');
  const [showSendConfirmation, setShowSendConfirmation] = useState(false);
  const [customerEmail, setCustomerEmail] = useState<string>('');
  const [reminders, setReminders] = useState<PaymentReminderResponse[]>([]);
  const [remindersLoading, setRemindersLoading] = useState(false);
  const [isSendingReminder, setIsSendingReminder] = useState(false);
  const [showReminderConfirmation, setShowReminderConfirmation] =
    useState(false);
  const [customerName, setCustomerName] = useState<string>('');
  const [invoice, setInvoice] = useState<Invoice | null>(null);
  // The tax percent as loaded, so an unchanged rate isn't re-sent on save
  const [loadedTaxRate, setLoadedTaxRate] = useState<string | null>(null);
  const [showPaymentModal, setShowPaymentModal] = useState(false);
  const [showCancelModal, setShowCancelModal] = useState(false);
  const [showRefundModal, setShowRefundModal] = useState(false);
  const { currentTenant } = useTenantStore();
  // Tenant defaults (T127): the API applies them when a new invoice omits
  // the tax rate / payment term; the form prefills to match.
  const defaultTaxRateId = currentTenant?.settings?.defaultTaxRateId;
  const defaultPaymentTermId = currentTenant?.settings?.defaultPaymentTermId;
  const [defaultTermName, setDefaultTermName] = useState<string | null>(null);
  const { can } = useTenantPermissions(currentTenant);

  const loadCustomers = async () => {
    setIsLoadingCustomers(true);
    try {
      // Page through all customers (API caps page_size at 100)
      setCustomers(await fetchAllCustomers());
    } catch (err) {
      console.error('Failed to load customers:', err);
    } finally {
      setIsLoadingCustomers(false);
    }
  };

  const loadReminders = useCallback(async (invoiceId: string) => {
    setRemindersLoading(true);
    try {
      const data = await paymentReminderService.getReminderHistory(invoiceId);
      setReminders(data);
    } catch (err) {
      console.error('Failed to load reminders:', err);
    } finally {
      setRemindersLoading(false);
    }
  }, []);

  const loadInvoice = useCallback(
    async (invoiceId: string, isStale: () => boolean = () => false) => {
      setIsLoading(true);
      setError(null);
      try {
        const invoice = await invoiceService.getInvoice(invoiceId);
        if (isStale()) return;
        setInvoice(invoice);
        // Convert the stored rate from decimal (0.07) to percentage (7)
        const taxRatePercent = taxRateToPercent(storedTaxRate(invoice));
        setLoadedTaxRate(taxRatePercent);
        setFormData({
          customer_id: invoice.customer_id || '',
          issue_date: invoice.issue_date
            ? invoice.issue_date.split('T')[0]
            : '',
          due_date: invoice.due_date ? invoice.due_date.split('T')[0] : '',
          tax_rate: taxRatePercent,
          notes: invoice.notes || '',
          terms_and_conditions: invoice.terms_and_conditions || '',
        });
        if (invoice.lines && invoice.lines.length > 0) {
          setLineItems(
            invoice.lines.map((line) => ({
              description: line.description,
              quantity: line.quantity,
              unit_price: line.unit_price,
              taxable: line.taxable ?? true,
            }))
          );
        }
        // Set invoice status and number
        setInvoiceStatus(invoice.status || InvoiceStatus.DRAFT);
        setInvoiceNumber(invoice.invoice_number || '');
        // Set customer email and name if available
        if (invoice.customer?.email) {
          setCustomerEmail(invoice.customer.email);
        }
        if (invoice.customer) {
          const name = invoice.customer.company_name
            ? `${invoice.customer.company_name} (${invoice.customer.first_name} ${invoice.customer.last_name})`
            : `${invoice.customer.first_name} ${invoice.customer.last_name}`;
          setCustomerName(name);
        }

        // Load reminders for non-draft invoices
        if (invoice.status && invoice.status !== InvoiceStatus.DRAFT) {
          loadReminders(invoiceId);
        }
      } catch (err) {
        if (isStale()) return;
        console.error('Failed to load invoice:', err);
        setError('Failed to load invoice details');
      } finally {
        if (!isStale()) setIsLoading(false);
      }
    },
    [loadReminders]
  );

  useEffect(() => {
    let cancelled = false;
    loadCustomers();
    if (!isNew && id) {
      loadInvoice(id, () => cancelled);
    } else {
      // Set default dates
      const today = new Date();
      const dueDate = new Date();
      dueDate.setDate(dueDate.getDate() + 30);
      setFormData((prev) => ({
        ...prev,
        issue_date: today.toISOString().split('T')[0],
        due_date: dueDate.toISOString().split('T')[0],
      }));
    }
    return () => {
      cancelled = true;
    };
  }, [id, isNew, loadInvoice]);

  // New invoice + tenant default tax rate: leave the rate blank so the API
  // applies the default (there is no tax-rate endpoint to show its value).
  useEffect(() => {
    if (isNew && defaultTaxRateId) {
      setFormData((prev) => ({ ...prev, tax_rate: '' }));
    }
  }, [isNew, defaultTaxRateId]);

  // New invoice + tenant default payment term: due date = issue date + term.
  useEffect(() => {
    if (!isNew || !defaultPaymentTermId) return;
    let cancelled = false;
    paymentTermService
      .getPaymentTerm(defaultPaymentTermId)
      .then((term) => {
        if (cancelled) return;
        setDefaultTermName(term.name);
        setFormData((prev) =>
          prev.issue_date
            ? {
                ...prev,
                due_date: addDays(prev.issue_date, term.days_until_due),
              }
            : prev
        );
      })
      .catch((err) => {
        console.error('Failed to load default payment term:', err);
      });
    return () => {
      cancelled = true;
    };
  }, [isNew, defaultPaymentTermId]);

  // Blank rate on a new invoice: the tenant's default applies on save
  const taxFromDefault =
    isNew && Boolean(defaultTaxRateId) && formData.tax_rate === '';

  const handleSendReminder = async () => {
    if (!id || isNew) return;
    setIsSendingReminder(true);
    setError(null);
    try {
      await paymentReminderService.sendManualReminder(id);
      setShowReminderConfirmation(false);
      trackEvent(AnalyticsEvents.REMINDER_SENT, {
        invoice_number: invoiceNumber,
      });
      loadReminders(id);
    } catch (err) {
      console.error('Failed to send reminder:', err);
      setError(getErrorMessage(err, 'Failed to send reminder'));
    } finally {
      setIsSendingReminder(false);
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
    value: string | number | boolean
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
    // Sum per-line totals rounded to cents, matching the server's math
    return roundToCents(
      lineItems.reduce(
        (sum, item) => sum + lineTotal(item.quantity, item.unit_price),
        0
      )
    );
  };

  const calculateTax = () => {
    // Taxable lines only, each rounded to cents (as the server does)
    const rate = percentToTaxRate(formData.tax_rate);
    return roundToCents(
      lineItems.reduce(
        (sum, item) =>
          item.taxable
            ? sum +
              roundToCents(lineTotal(item.quantity, item.unit_price) * rate)
            : sum,
        0
      )
    );
  };

  const calculateTotal = () => {
    return roundToCents(calculateSubtotal() + calculateTax());
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSaving(true);
    setError(null);

    // Convert date strings to RFC3339 format (backend expects time.Time)
    const issueDateISO = formData.issue_date
      ? new Date(formData.issue_date + 'T00:00:00Z').toISOString()
      : undefined;
    const dueDateISO = formData.due_date
      ? new Date(formData.due_date + 'T23:59:59Z').toISOString()
      : undefined;

    // Convert tax rate from percentage (e.g., 7) to decimal (e.g., 0.07)
    const taxRateDecimal = percentToTaxRate(formData.tax_rate);

    const lines: InvoiceLineRequest[] = lineItems.map((item) => ({
      description: item.description,
      quantity:
        typeof item.quantity === 'string'
          ? parseFloat(item.quantity) || 0
          : item.quantity,
      unit_price:
        typeof item.unit_price === 'string'
          ? parseFloat(item.unit_price) || 0
          : item.unit_price,
      taxable: item.taxable,
    }));

    try {
      if (isNew) {
        const createData: CreateInvoiceRequest = {
          customer_id: formData.customer_id,
          issue_date: issueDateISO,
          due_date: dueDateISO,
          tax_rate: taxFromDefault ? undefined : taxRateDecimal,
          notes: formData.notes || undefined,
          terms_and_conditions: formData.terms_and_conditions || undefined,
          lines,
        };
        await invoiceService.createInvoice(createData);
        trackEvent(AnalyticsEvents.INVOICE_CREATED, {
          total: calculateTotal(),
          line_items: lineItems.length,
        });
      } else if (id) {
        const updateData: UpdateInvoiceRequest = {
          due_date: dueDateISO,
          // Unchanged: omit it, so the invoice keeps its rate and tax_rate_id
          tax_rate:
            formData.tax_rate === loadedTaxRate ? undefined : taxRateDecimal,
          // Empty strings clear the stored values
          notes: formData.notes,
          terms_and_conditions: formData.terms_and_conditions,
          lines,
        };
        await invoiceService.updateInvoice(id, updateData);
      }
      navigate('/invoices');
    } catch (err) {
      console.error('Failed to save invoice:', err);
      setError(getErrorMessage(err, 'Failed to save invoice'));
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!id || isNew) return;
    if (!window.confirm('Are you sure you want to delete this invoice?'))
      return;

    setIsDeleting(true);
    setError(null);
    try {
      await invoiceService.deleteInvoice(id);
      navigate('/invoices');
    } catch (err) {
      console.error('Failed to delete invoice:', err);
      setError(getErrorMessage(err, 'Failed to delete invoice'));
    } finally {
      setIsDeleting(false);
    }
  };

  const handleSendInvoice = async () => {
    if (!id || isNew) return;

    setIsSending(true);
    setError(null);
    try {
      const updatedInvoice = await invoiceService.sendInvoice(id);
      setInvoiceStatus(updatedInvoice.status || InvoiceStatus.SENT);
      setShowSendConfirmation(false);
      trackEvent(AnalyticsEvents.INVOICE_SENT, { total: calculateTotal() });
      // Show success message or navigate
      alert('Invoice sent successfully!');
    } catch (err) {
      console.error('Failed to send invoice:', err);
      setError(getErrorMessage(err, 'Failed to send invoice'));
    } finally {
      setIsSending(false);
    }
  };

  const handleDownloadInvoicePDF = async () => {
    if (!id || isNew) return;

    setIsDownloadingInvoice(true);
    try {
      await invoiceService.downloadInvoicePDF(id);
      trackEvent(AnalyticsEvents.EXPORT_DOWNLOADED, {
        format: 'pdf',
        type: 'invoice',
      });
    } catch (err) {
      console.error('Failed to download invoice PDF:', err);
    } finally {
      setIsDownloadingInvoice(false);
    }
  };

  const handleDownloadReceiptPDF = async () => {
    if (!id || isNew) return;

    setIsDownloadingReceipt(true);
    try {
      await invoiceService.downloadReceiptPDF(id);
    } catch (err) {
      console.error('Failed to download receipt PDF:', err);
    } finally {
      setIsDownloadingReceipt(false);
    }
  };

  const handlePaymentRecorded = () => {
    setShowPaymentModal(false);
    if (id) loadInvoice(id);
  };

  const handleRefunded = () => {
    setShowRefundModal(false);
    if (id) loadInvoice(id);
  };

  const handleInvoiceCancelled = () => {
    setShowCancelModal(false);
    if (id) loadInvoice(id);
  };

  const amountDue = toAmount(invoice?.amount_due);
  const amountPaid = toAmount(invoice?.amount_paid);
  // Mirrors the API: payments only on open invoices with a balance; only
  // unpaid invoices can be cancelled (paid ones are refunded instead).
  const canRecordPayment =
    [
      InvoiceStatus.SENT,
      InvoiceStatus.VIEWED,
      InvoiceStatus.OVERDUE,
      InvoiceStatus.PARTIALLY_PAID,
    ].includes(invoiceStatus) && amountDue > 0;
  const canCancel =
    [
      InvoiceStatus.DRAFT,
      InvoiceStatus.SENT,
      InvoiceStatus.VIEWED,
      InvoiceStatus.OVERDUE,
    ].includes(invoiceStatus) && amountPaid === 0;
  // The API refunds up to the amount paid; payments.refund is checked there too.
  const canRefund = can('payments.refund') && amountPaid > 0;

  // Update customer email when customer selection changes
  const handleCustomerChange = (e: React.ChangeEvent<HTMLSelectElement>) => {
    const { value } = e.target;
    setFormData((prev) => ({ ...prev, customer_id: value }));
    const selectedCustomer = customers.find((c) => c.id === value);
    if (selectedCustomer?.email) {
      setCustomerEmail(selectedCustomer.email);
    } else {
      setCustomerEmail('');
    }
  };

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
              onClick={() => navigate('/invoices')}
              className="flex items-center gap-2"
            >
              <ArrowLeft className="h-4 w-4" />
              Back
            </Button>
            <h1 className="text-2xl font-bold text-neutral-900">
              {isNew ? 'Create Invoice' : `Invoice ${invoiceNumber}`}
            </h1>
            {!isNew && (
              <Badge variant={getStatusBadgeVariant(invoiceStatus)}>
                {getInvoiceStatusLabel(invoiceStatus)}
              </Badge>
            )}
          </div>
          {!isNew && invoiceStatus === InvoiceStatus.DRAFT && (
            <Button
              type="button"
              variant="primary"
              onClick={() => setShowSendConfirmation(true)}
              className="flex items-center gap-2"
            >
              <Send className="h-4 w-4" />
              Send Invoice
            </Button>
          )}
          {!isNew && invoiceStatus === InvoiceStatus.SENT && (
            <Button
              type="button"
              variant="secondary"
              onClick={() => setShowSendConfirmation(true)}
              className="flex items-center gap-2"
            >
              <Send className="h-4 w-4" />
              Resend Invoice
            </Button>
          )}
          {!isNew && (
            <div className="flex items-center gap-2">
              <Button
                type="button"
                variant="secondary"
                onClick={handleDownloadInvoicePDF}
                disabled={isDownloadingInvoice}
                className="flex items-center gap-2"
              >
                {isDownloadingInvoice ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Download className="h-4 w-4" />
                )}
                Download PDF
              </Button>
              {invoiceStatus === InvoiceStatus.PAID && (
                <Button
                  type="button"
                  variant="secondary"
                  onClick={handleDownloadReceiptPDF}
                  disabled={isDownloadingReceipt}
                  className="flex items-center gap-2"
                >
                  {isDownloadingReceipt ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <FileText className="h-4 w-4" />
                  )}
                  Download Receipt
                </Button>
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
              Invoice Details
            </h2>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="md:col-span-2">
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
                  onChange={handleCustomerChange}
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
                  htmlFor="issue_date"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Issue Date *
                </label>
                <input
                  type="date"
                  id="issue_date"
                  name="issue_date"
                  value={formData.issue_date}
                  onChange={handleChange}
                  required
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>

              <div>
                <label
                  htmlFor="due_date"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Due Date *
                </label>
                <input
                  type="date"
                  id="due_date"
                  name="due_date"
                  value={formData.due_date}
                  onChange={handleChange}
                  required
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
                {isNew && defaultTermName && (
                  <p className="mt-1 text-xs text-neutral-500">
                    Payment term: {defaultTermName}
                  </p>
                )}
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
                  placeholder={
                    isNew && defaultTaxRateId ? 'Default rate' : undefined
                  }
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
                {taxFromDefault && (
                  <p className="mt-1 text-xs text-neutral-500">
                    Left blank, your default tax rate applies.
                  </p>
                )}
                {invoice?.tax_rate?.name &&
                  formData.tax_rate === loadedTaxRate && (
                    <p className="mt-1 text-xs text-neutral-500">
                      {invoice.tax_rate.name}
                    </p>
                  )}
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
                  <label className="flex items-center gap-1 py-2 text-sm text-neutral-600">
                    <input
                      type="checkbox"
                      checked={item.taxable}
                      onChange={(e) =>
                        handleLineItemChange(index, 'taxable', e.target.checked)
                      }
                      aria-label={`Line ${index + 1} taxable`}
                    />
                    Taxable
                  </label>
                  <div className="w-24 text-right py-2 font-medium">
                    {formatDollars(lineTotal(item.quantity, item.unit_price))}
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
                      {formatDollars(calculateSubtotal())}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-neutral-600">
                      {taxFromDefault
                        ? 'Tax (default rate):'
                        : `Tax (${formData.tax_rate}%):`}
                    </span>
                    <span className="font-medium">
                      {taxFromDefault
                        ? 'Set on save'
                        : formatDollars(calculateTax())}
                    </span>
                  </div>
                  <div className="flex justify-between text-lg font-semibold pt-2 border-t">
                    <span>Total:</span>
                    <span>
                      {taxFromDefault
                        ? `${formatDollars(calculateSubtotal())} + tax`
                        : formatDollars(calculateTotal())}
                    </span>
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
                  placeholder="Additional notes..."
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>
              <div>
                <label
                  htmlFor="terms_and_conditions"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Terms & Conditions
                </label>
                <textarea
                  id="terms_and_conditions"
                  name="terms_and_conditions"
                  value={formData.terms_and_conditions}
                  onChange={handleChange}
                  rows={3}
                  placeholder="Terms and conditions..."
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>
            </div>
          </div>

          {!isNew && invoice && (
            <InvoicePaymentHistory
              invoice={invoice}
              actions={
                <>
                  {canRecordPayment && (
                    <Button
                      type="button"
                      variant="primary"
                      onClick={() => setShowPaymentModal(true)}
                      className="flex items-center gap-2"
                    >
                      <DollarSign className="h-4 w-4" />
                      Record Payment
                    </Button>
                  )}
                  {canRefund && (
                    <Button
                      type="button"
                      variant="secondary"
                      onClick={() => setShowRefundModal(true)}
                      className="flex items-center gap-2"
                    >
                      <Undo2 className="h-4 w-4" />
                      Record Refund
                    </Button>
                  )}
                  {canCancel && (
                    <Button
                      type="button"
                      variant="secondary"
                      onClick={() => setShowCancelModal(true)}
                      className="flex items-center gap-2"
                    >
                      <Ban className="h-4 w-4" />
                      Cancel Invoice
                    </Button>
                  )}
                </>
              }
            />
          )}

          {/* Payment Reminders Section - only for non-draft invoices */}
          {!isNew && invoiceStatus !== InvoiceStatus.DRAFT && (
            <div className="bg-white rounded-lg border border-neutral-200 p-6">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold text-neutral-900 flex items-center gap-2">
                  <Bell className="h-5 w-5" />
                  Payment Reminders
                </h2>
                {invoiceStatus !== InvoiceStatus.PAID &&
                  invoiceStatus !== InvoiceStatus.CANCELLED &&
                  invoiceStatus !== InvoiceStatus.REFUNDED && (
                    <Button
                      type="button"
                      variant="secondary"
                      onClick={() => setShowReminderConfirmation(true)}
                      disabled={isSendingReminder}
                      className="flex items-center gap-2"
                    >
                      {isSendingReminder ? (
                        <Loader2 className="h-4 w-4 animate-spin" />
                      ) : (
                        <Send className="h-4 w-4" />
                      )}
                      Send Reminder
                    </Button>
                  )}
              </div>

              {remindersLoading ? (
                <div className="py-4 text-center text-neutral-500">
                  Loading reminders...
                </div>
              ) : reminders.length === 0 ? (
                <div className="py-4 text-center text-neutral-500">
                  No reminders sent yet
                </div>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="border-b border-neutral-200">
                        <th className="text-left py-2 px-3 font-medium text-neutral-600">
                          #
                        </th>
                        <th className="text-left py-2 px-3 font-medium text-neutral-600">
                          Channel
                        </th>
                        <th className="text-left py-2 px-3 font-medium text-neutral-600">
                          Status
                        </th>
                        <th className="text-left py-2 px-3 font-medium text-neutral-600">
                          Tone
                        </th>
                        <th className="text-left py-2 px-3 font-medium text-neutral-600">
                          Sent At
                        </th>
                        <th className="text-left py-2 px-3 font-medium text-neutral-600">
                          Recipient
                        </th>
                      </tr>
                    </thead>
                    <tbody>
                      {reminders.map((reminder) => (
                        <tr
                          key={reminder.id}
                          className="border-b border-neutral-100"
                        >
                          <td className="py-2 px-3">
                            {reminder.is_manual
                              ? `Manual #${reminder.reminder_number}`
                              : reminder.reminder_number}
                          </td>
                          <td className="py-2 px-3">
                            <Badge
                              variant={
                                reminder.channel === 'email'
                                  ? 'info'
                                  : 'neutral'
                              }
                            >
                              {reminder.channel}
                            </Badge>
                          </td>
                          <td className="py-2 px-3">
                            <Badge
                              variant={
                                reminder.status === 'sent'
                                  ? 'success'
                                  : reminder.status === 'failed'
                                    ? 'error'
                                    : 'warning'
                              }
                            >
                              {reminder.status}
                            </Badge>
                          </td>
                          <td className="py-2 px-3 capitalize">
                            {reminder.tone}
                          </td>
                          <td className="py-2 px-3">
                            {reminder.sent_at
                              ? new Date(
                                  reminder.sent_at * 1000
                                ).toLocaleString()
                              : '-'}
                          </td>
                          <td className="py-2 px-3 text-neutral-600">
                            {reminder.recipient}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}

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
                  Delete Invoice
                </Button>
              )}
            </div>
            <div className="flex items-center gap-3">
              <Button
                type="button"
                variant="secondary"
                onClick={() => navigate('/invoices')}
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
                {isNew ? 'Create Invoice' : 'Save Changes'}
              </Button>
            </div>
          </div>
        </form>

        {!isNew && id && (
          <>
            <RecordPaymentModal
              isOpen={showPaymentModal}
              invoiceId={id}
              amountDue={amountDue}
              onClose={() => setShowPaymentModal(false)}
              onRecorded={handlePaymentRecorded}
            />
            <RefundModal
              isOpen={showRefundModal}
              invoiceId={id}
              amountPaid={amountPaid}
              onClose={() => setShowRefundModal(false)}
              onRefunded={handleRefunded}
            />
            <CancelInvoiceModal
              isOpen={showCancelModal}
              invoiceId={id}
              invoiceNumber={invoiceNumber}
              onClose={() => setShowCancelModal(false)}
              onCancelled={handleInvoiceCancelled}
            />
          </>
        )}

        {/* Send Reminder Confirmation Modal */}
        {showReminderConfirmation && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4 shadow-xl">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 bg-amber-100 rounded-full">
                  <Bell className="h-6 w-6 text-amber-600" />
                </div>
                <h2 className="text-xl font-semibold text-neutral-900">
                  Send Payment Reminder
                </h2>
              </div>

              <p className="text-neutral-600 mb-4">
                Send a payment reminder to:
              </p>

              <div className="p-3 bg-neutral-100 rounded-lg mb-6">
                <p className="font-medium text-neutral-900">
                  {customerName || 'Customer'}
                </p>
                <p className="text-sm text-neutral-600">{customerEmail}</p>
              </div>

              <div className="flex justify-end gap-3">
                <Button
                  type="button"
                  variant="secondary"
                  onClick={() => setShowReminderConfirmation(false)}
                  disabled={isSendingReminder}
                >
                  Cancel
                </Button>
                <Button
                  type="button"
                  variant="primary"
                  onClick={handleSendReminder}
                  disabled={isSendingReminder}
                  className="flex items-center gap-2"
                >
                  {isSendingReminder ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Send className="h-4 w-4" />
                  )}
                  {isSendingReminder ? 'Sending...' : 'Send Reminder'}
                </Button>
              </div>
            </div>
          </div>
        )}

        {/* Send Confirmation Modal */}
        {showSendConfirmation && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white rounded-lg p-6 max-w-md w-full mx-4 shadow-xl">
              <div className="flex items-center gap-3 mb-4">
                <div className="p-2 bg-primary-100 rounded-full">
                  <Mail className="h-6 w-6 text-primary-600" />
                </div>
                <h2 className="text-xl font-semibold text-neutral-900">
                  {invoiceStatus === InvoiceStatus.SENT
                    ? 'Resend Invoice'
                    : 'Send Invoice'}
                </h2>
              </div>

              <p className="text-neutral-600 mb-4">
                {invoiceStatus === InvoiceStatus.SENT
                  ? 'This will generate a new payment link and send the invoice again to:'
                  : 'This will send the invoice with a payment link to:'}
              </p>

              <div
                data-testid="send-confirmation"
                className="p-3 bg-neutral-100 rounded-lg mb-6"
              >
                <p className="font-medium text-neutral-900">
                  {customerEmail || 'No email address available'}
                </p>
              </div>

              {!customerEmail && (
                <p className="text-amber-600 text-sm mb-4">
                  Warning: The customer does not have an email address. The
                  invoice will be marked as sent but no email will be delivered.
                </p>
              )}

              <div className="flex justify-end gap-3">
                <Button
                  type="button"
                  variant="secondary"
                  onClick={() => setShowSendConfirmation(false)}
                  disabled={isSending}
                >
                  Cancel
                </Button>
                <Button
                  type="button"
                  variant="primary"
                  onClick={handleSendInvoice}
                  disabled={isSending}
                  data-testid="confirm-send"
                  className="flex items-center gap-2"
                >
                  {isSending ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Send className="h-4 w-4" />
                  )}
                  {isSending ? 'Sending...' : 'Confirm & Send'}
                </Button>
              </div>
            </div>
          </div>
        )}
      </div>
    </DashboardLayout>
  );
}

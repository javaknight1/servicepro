import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { DashboardLayout } from '@components/layout';
import { Button, Badge } from '@components/shared';
import {
  invoiceService,
  Invoice,
  InvoiceStatus,
} from '@services/invoiceService';
import {
  paymentReminderService,
  type PaymentReminderResponse,
} from '@services/paymentReminderService';
import { getErrorMessage } from '@/utils/error';
import { customerService, Customer } from '@services/customerService';
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
} from 'lucide-react';

interface InvoiceFormData {
  customer_id: string;
  issue_date: string;
  due_date: string;
  tax_rate: string;
  notes: string;
  terms: string;
}

interface LineItemFormData {
  description: string;
  quantity: number | string;
  unit_price: number | string;
}

const initialFormData: InvoiceFormData = {
  customer_id: '',
  issue_date: '',
  due_date: '',
  tax_rate: '0',
  notes: '',
  terms: '',
};

const emptyLineItem: LineItemFormData = {
  description: '',
  quantity: 1,
  unit_price: 0,
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

  useEffect(() => {
    loadCustomers();
    if (!isNew && id) {
      loadInvoice(id);
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
  }, [id, isNew]);

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

  const loadInvoice = async (invoiceId: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const invoice = await invoiceService.getInvoice(invoiceId);
      // Convert tax_rate from decimal (0.07) back to percentage (7) for display
      const taxRatePercent = invoice.tax_rate
        ? (invoice.tax_rate * 100).toString()
        : '0';
      setFormData({
        customer_id: invoice.customer_id || '',
        issue_date: invoice.issue_date ? invoice.issue_date.split('T')[0] : '',
        due_date: invoice.due_date ? invoice.due_date.split('T')[0] : '',
        tax_rate: taxRatePercent,
        notes: invoice.notes || '',
        terms: invoice.terms || '',
      });
      if (invoice.lines && invoice.lines.length > 0) {
        setLineItems(
          invoice.lines.map((line) => ({
            description: line.description,
            quantity: line.quantity,
            unit_price: line.unit_price,
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
      console.error('Failed to load invoice:', err);
      setError('Failed to load invoice details');
    } finally {
      setIsLoading(false);
    }
  };

  const loadReminders = async (invoiceId: string) => {
    setRemindersLoading(true);
    try {
      const data = await paymentReminderService.getReminderHistory(invoiceId);
      setReminders(data);
    } catch (err) {
      console.error('Failed to load reminders:', err);
    } finally {
      setRemindersLoading(false);
    }
  };

  const handleSendReminder = async () => {
    if (!id || isNew) return;
    setIsSendingReminder(true);
    setError(null);
    try {
      await paymentReminderService.sendManualReminder(id);
      setShowReminderConfirmation(false);
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

    // Convert date strings to RFC3339 format (backend expects time.Time)
    const issueDateISO = formData.issue_date
      ? new Date(formData.issue_date + 'T00:00:00Z').toISOString()
      : undefined;
    const dueDateISO = formData.due_date
      ? new Date(formData.due_date + 'T23:59:59Z').toISOString()
      : undefined;

    // Convert tax rate from percentage (e.g., 7) to decimal (e.g., 0.07)
    const taxRateDecimal = (parseFloat(formData.tax_rate) || 0) / 100;

    const submitData = {
      customer_id: formData.customer_id,
      issue_date: issueDateISO,
      due_date: dueDateISO,
      tax_rate: taxRateDecimal,
      notes: formData.notes || undefined,
      terms: formData.terms || undefined,
      lines: lineItems.map((item) => ({
        description: item.description,
        quantity:
          typeof item.quantity === 'string'
            ? parseFloat(item.quantity) || 0
            : item.quantity,
        unit_price:
          typeof item.unit_price === 'string'
            ? parseFloat(item.unit_price) || 0
            : item.unit_price,
      })),
    };

    try {
      if (isNew) {
        await invoiceService.createInvoice(
          submitData as unknown as Partial<Invoice>
        );
      } else if (id) {
        await invoiceService.updateInvoice(
          id,
          submitData as unknown as Partial<Invoice>
        );
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

  const getStatusBadgeVariant = (
    status: InvoiceStatus
  ): 'success' | 'warning' | 'error' | 'info' | 'neutral' => {
    switch (status) {
      case InvoiceStatus.PAID:
        return 'success';
      case InvoiceStatus.SENT:
      case InvoiceStatus.VIEWED:
        return 'info';
      case InvoiceStatus.OVERDUE:
        return 'error';
      case InvoiceStatus.PARTIALLY_PAID:
        return 'warning';
      case InvoiceStatus.CANCELLED:
        return 'neutral';
      default:
        return 'neutral';
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
                {invoiceStatus.charAt(0).toUpperCase() +
                  invoiceStatus.slice(1).replace('_', ' ')}
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
                  placeholder="Additional notes..."
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>
              <div>
                <label
                  htmlFor="terms"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Payment Terms
                </label>
                <textarea
                  id="terms"
                  name="terms"
                  value={formData.terms}
                  onChange={handleChange}
                  rows={3}
                  placeholder="Payment terms..."
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>
            </div>
          </div>

          {/* Payment Reminders Section - only for non-draft invoices */}
          {!isNew && invoiceStatus !== InvoiceStatus.DRAFT && (
            <div className="bg-white rounded-lg border border-neutral-200 p-6">
              <div className="flex items-center justify-between mb-4">
                <h2 className="text-lg font-semibold text-neutral-900 flex items-center gap-2">
                  <Bell className="h-5 w-5" />
                  Payment Reminders
                </h2>
                {invoiceStatus !== InvoiceStatus.PAID &&
                  invoiceStatus !== InvoiceStatus.CANCELLED && (
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
                            {reminder.reminder_number}
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

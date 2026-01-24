import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { DashboardLayout } from '@components/layout';
import { Button } from '@components/shared';
import {
  invoiceService,
  Invoice,
  InvoiceLineItem,
} from '@services/invoiceService';
import { customerService, Customer } from '@services/customerService';
import { ArrowLeft, Save, Trash2, Loader2, Plus, X } from 'lucide-react';

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
  const [error, setError] = useState<string | null>(null);

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
    } catch (err) {
      console.error('Failed to load invoice:', err);
      setError('Failed to load invoice details');
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
        await invoiceService.createInvoice(submitData as unknown as Partial<Invoice>);
      } else if (id) {
        await invoiceService.updateInvoice(id, submitData as unknown as Partial<Invoice>);
      }
      navigate('/invoices');
    } catch (err: any) {
      console.error('Failed to save invoice:', err);
      setError(
        err?.response?.data?.message ||
          err?.response?.data?.error ||
          'Failed to save invoice'
      );
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
    } catch (err: any) {
      console.error('Failed to delete invoice:', err);
      setError(err?.response?.data?.message || 'Failed to delete invoice');
    } finally {
      setIsDeleting(false);
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
        <div className="flex items-center gap-4 mb-6">
          <Button
            variant="ghost"
            onClick={() => navigate('/invoices')}
            className="flex items-center gap-2"
          >
            <ArrowLeft className="h-4 w-4" />
            Back
          </Button>
          <h1 className="text-2xl font-bold text-neutral-900">
            {isNew ? 'Create Invoice' : 'Edit Invoice'}
          </h1>
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
      </div>
    </DashboardLayout>
  );
}

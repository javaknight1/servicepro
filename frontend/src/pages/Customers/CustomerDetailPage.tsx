import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { DashboardLayout } from '@components/layout';
import { Button } from '@components/shared';
import { customerService } from '@services/customerService';
import type { CustomerType, CustomerStatus } from '@app-types/customer';
import { ArrowLeft, Save, Trash2, Loader2 } from 'lucide-react';

interface CustomerFormData {
  email: string;
  first_name: string;
  last_name: string;
  company_name: string;
  phone_primary: string;
  phone_secondary: string;
  billing_address_street: string;
  billing_address_city: string;
  billing_address_state: string;
  billing_address_zip: string;
  service_address_street: string;
  service_address_city: string;
  service_address_state: string;
  service_address_zip: string;
  customer_type: CustomerType;
  status: CustomerStatus;
  notes: string;
}

const initialFormData: CustomerFormData = {
  email: '',
  first_name: '',
  last_name: '',
  company_name: '',
  phone_primary: '',
  phone_secondary: '',
  billing_address_street: '',
  billing_address_city: '',
  billing_address_state: '',
  billing_address_zip: '',
  service_address_street: '',
  service_address_city: '',
  service_address_state: '',
  service_address_zip: '',
  customer_type: 'residential',
  status: 'prospect',
  notes: '',
};

export function CustomerDetailPage() {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const isNew = !id || id === 'new';

  const [formData, setFormData] = useState<CustomerFormData>(initialFormData);
  const [useServiceAddress, setUseServiceAddress] = useState(false);
  const [isLoading, setIsLoading] = useState(!isNew);
  const [isSaving, setIsSaving] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!isNew && id) {
      loadCustomer(id);
    }
  }, [id, isNew]);

  const loadCustomer = async (customerId: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const customer = await customerService.getCustomer(customerId);
      setFormData({
        email: customer.email || '',
        first_name: customer.first_name || '',
        last_name: customer.last_name || '',
        company_name: customer.company_name || '',
        phone_primary: customer.phone_primary || '',
        phone_secondary: customer.phone_secondary || '',
        billing_address_street: customer.billing_address_street || '',
        billing_address_city: customer.billing_address_city || '',
        billing_address_state: customer.billing_address_state || '',
        billing_address_zip: customer.billing_address_zip || '',
        service_address_street: customer.service_address_street || '',
        service_address_city: customer.service_address_city || '',
        service_address_state: customer.service_address_state || '',
        service_address_zip: customer.service_address_zip || '',
        customer_type: customer.customer_type || 'residential',
        status: customer.status || 'prospect',
        notes: customer.notes || '',
      });
      setUseServiceAddress(!!customer.service_address_street);
    } catch (err) {
      console.error('Failed to load customer:', err);
      setError('Failed to load customer details');
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

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSaving(true);
    setError(null);

    // Prepare data - only include service address if checkbox is checked
    const submitData: Record<string, any> = {
      ...formData,
      company_name: formData.company_name || undefined,
      phone_secondary: formData.phone_secondary || undefined,
      notes: formData.notes || undefined,
    };

    if (!useServiceAddress) {
      delete submitData.service_address_street;
      delete submitData.service_address_city;
      delete submitData.service_address_state;
      delete submitData.service_address_zip;
    } else {
      submitData.service_address_street =
        formData.service_address_street || undefined;
      submitData.service_address_city =
        formData.service_address_city || undefined;
      submitData.service_address_state =
        formData.service_address_state || undefined;
      submitData.service_address_zip =
        formData.service_address_zip || undefined;
    }

    try {
      if (isNew) {
        await customerService.createCustomer(submitData);
      } else if (id) {
        await customerService.updateCustomer(id, submitData);
      }
      navigate('/customers');
    } catch (err: any) {
      console.error('Failed to save customer:', err);
      setError(
        err?.response?.data?.message ||
          err?.response?.data?.error ||
          'Failed to save customer'
      );
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!id || isNew) return;

    if (!window.confirm('Are you sure you want to delete this customer?')) {
      return;
    }

    setIsDeleting(true);
    setError(null);
    try {
      await customerService.deleteCustomer(id);
      navigate('/customers');
    } catch (err: any) {
      console.error('Failed to delete customer:', err);
      setError(err?.response?.data?.message || 'Failed to delete customer');
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
            onClick={() => navigate('/customers')}
            className="flex items-center gap-2"
          >
            <ArrowLeft className="h-4 w-4" />
            Back
          </Button>
          <h1 className="text-2xl font-bold text-neutral-900">
            {isNew ? 'New Customer' : 'Edit Customer'}
          </h1>
        </div>

        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Basic Information */}
          <div className="bg-white rounded-lg border border-neutral-200 p-6">
            <h2 className="text-lg font-semibold text-neutral-900 mb-4">
              Basic Information
            </h2>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label
                  htmlFor="first_name"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  First Name *
                </label>
                <input
                  type="text"
                  id="first_name"
                  name="first_name"
                  value={formData.first_name}
                  onChange={handleChange}
                  required
                  maxLength={50}
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>

              <div>
                <label
                  htmlFor="last_name"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Last Name *
                </label>
                <input
                  type="text"
                  id="last_name"
                  name="last_name"
                  value={formData.last_name}
                  onChange={handleChange}
                  required
                  maxLength={50}
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>

              <div>
                <label
                  htmlFor="email"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Email *
                </label>
                <input
                  type="email"
                  id="email"
                  name="email"
                  value={formData.email}
                  onChange={handleChange}
                  required
                  maxLength={255}
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>

              <div>
                <label
                  htmlFor="phone_primary"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Primary Phone *
                </label>
                <input
                  type="tel"
                  id="phone_primary"
                  name="phone_primary"
                  value={formData.phone_primary}
                  onChange={handleChange}
                  required
                  maxLength={20}
                  placeholder="(555) 555-5555"
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>

              <div>
                <label
                  htmlFor="phone_secondary"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Secondary Phone
                </label>
                <input
                  type="tel"
                  id="phone_secondary"
                  name="phone_secondary"
                  value={formData.phone_secondary}
                  onChange={handleChange}
                  maxLength={20}
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>

              <div>
                <label
                  htmlFor="company_name"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Company Name
                </label>
                <input
                  type="text"
                  id="company_name"
                  name="company_name"
                  value={formData.company_name}
                  onChange={handleChange}
                  maxLength={100}
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>
            </div>
          </div>

          {/* Billing Address */}
          <div className="bg-white rounded-lg border border-neutral-200 p-6">
            <h2 className="text-lg font-semibold text-neutral-900 mb-4">
              Billing Address
            </h2>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div className="md:col-span-2">
                <label
                  htmlFor="billing_address_street"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Street Address *
                </label>
                <input
                  type="text"
                  id="billing_address_street"
                  name="billing_address_street"
                  value={formData.billing_address_street}
                  onChange={handleChange}
                  required
                  maxLength={100}
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>

              <div>
                <label
                  htmlFor="billing_address_city"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  City *
                </label>
                <input
                  type="text"
                  id="billing_address_city"
                  name="billing_address_city"
                  value={formData.billing_address_city}
                  onChange={handleChange}
                  required
                  maxLength={50}
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label
                    htmlFor="billing_address_state"
                    className="block text-sm font-medium text-neutral-700 mb-1"
                  >
                    State *
                  </label>
                  <input
                    type="text"
                    id="billing_address_state"
                    name="billing_address_state"
                    value={formData.billing_address_state}
                    onChange={handleChange}
                    required
                    maxLength={2}
                    placeholder="CA"
                    className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 uppercase"
                  />
                </div>

                <div>
                  <label
                    htmlFor="billing_address_zip"
                    className="block text-sm font-medium text-neutral-700 mb-1"
                  >
                    ZIP Code *
                  </label>
                  <input
                    type="text"
                    id="billing_address_zip"
                    name="billing_address_zip"
                    value={formData.billing_address_zip}
                    onChange={handleChange}
                    required
                    maxLength={10}
                    placeholder="12345"
                    className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                  />
                </div>
              </div>
            </div>
          </div>

          {/* Service Address */}
          <div className="bg-white rounded-lg border border-neutral-200 p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-neutral-900">
                Service Address
              </h2>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={useServiceAddress}
                  onChange={(e) => setUseServiceAddress(e.target.checked)}
                  className="rounded border-neutral-300 text-primary-600 focus:ring-primary-500"
                />
                <span className="text-sm text-neutral-600">
                  Different from billing address
                </span>
              </label>
            </div>

            {useServiceAddress ? (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <div className="md:col-span-2">
                  <label
                    htmlFor="service_address_street"
                    className="block text-sm font-medium text-neutral-700 mb-1"
                  >
                    Street Address
                  </label>
                  <input
                    type="text"
                    id="service_address_street"
                    name="service_address_street"
                    value={formData.service_address_street}
                    onChange={handleChange}
                    maxLength={100}
                    className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                  />
                </div>

                <div>
                  <label
                    htmlFor="service_address_city"
                    className="block text-sm font-medium text-neutral-700 mb-1"
                  >
                    City
                  </label>
                  <input
                    type="text"
                    id="service_address_city"
                    name="service_address_city"
                    value={formData.service_address_city}
                    onChange={handleChange}
                    maxLength={50}
                    className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                  />
                </div>

                <div className="grid grid-cols-2 gap-4">
                  <div>
                    <label
                      htmlFor="service_address_state"
                      className="block text-sm font-medium text-neutral-700 mb-1"
                    >
                      State
                    </label>
                    <input
                      type="text"
                      id="service_address_state"
                      name="service_address_state"
                      value={formData.service_address_state}
                      onChange={handleChange}
                      maxLength={2}
                      placeholder="CA"
                      className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500 uppercase"
                    />
                  </div>

                  <div>
                    <label
                      htmlFor="service_address_zip"
                      className="block text-sm font-medium text-neutral-700 mb-1"
                    >
                      ZIP Code
                    </label>
                    <input
                      type="text"
                      id="service_address_zip"
                      name="service_address_zip"
                      value={formData.service_address_zip}
                      onChange={handleChange}
                      maxLength={10}
                      className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                    />
                  </div>
                </div>
              </div>
            ) : (
              <p className="text-sm text-neutral-500">
                Service address will be the same as billing address.
              </p>
            )}
          </div>

          {/* Classification */}
          <div className="bg-white rounded-lg border border-neutral-200 p-6">
            <h2 className="text-lg font-semibold text-neutral-900 mb-4">
              Classification
            </h2>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
              <div>
                <label
                  htmlFor="customer_type"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Customer Type *
                </label>
                <select
                  id="customer_type"
                  name="customer_type"
                  value={formData.customer_type}
                  onChange={handleChange}
                  required
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                >
                  <option value="residential">Residential</option>
                  <option value="commercial">Commercial</option>
                </select>
              </div>

              <div>
                <label
                  htmlFor="status"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Status
                </label>
                <select
                  id="status"
                  name="status"
                  value={formData.status}
                  onChange={handleChange}
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                >
                  <option value="prospect">Prospect</option>
                  <option value="active">Active</option>
                  <option value="inactive">Inactive</option>
                </select>
              </div>

              <div className="md:col-span-2">
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
                  rows={4}
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>
            </div>
          </div>

          {/* Actions */}
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
                  Delete Customer
                </Button>
              )}
            </div>

            <div className="flex items-center gap-3">
              <Button
                type="button"
                variant="secondary"
                onClick={() => navigate('/customers')}
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
                {isNew ? 'Create Customer' : 'Save Changes'}
              </Button>
            </div>
          </div>
        </form>
      </div>
    </DashboardLayout>
  );
}

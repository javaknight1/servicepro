import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router';
import { useQuery } from '@tanstack/react-query';
import { DashboardLayout } from '@components/layout';
import {
  Button,
  Badge,
  Avatar,
  PageHeader,
  getStatusBadgeVariant,
} from '@components/shared';
import {
  BillingRulesCards,
  BillingRulesModal,
  CustomerSitesSection,
  CustomerPurchaseOrdersSection,
} from '@components/customers';
import { ConsolidatedInvoiceModal } from '@components/invoices';
import { customerService, type Customer } from '@services/customerService';
import { siteService, sitesQueryKey } from '@services/siteService';
import {
  purchaseOrderService,
  purchaseOrdersQueryKey,
} from '@services/purchaseOrderService';
import {
  agreementKeys,
  agreementService,
  describeFrequency,
  monthlyAmount,
  type Agreement,
} from '@services/agreementService';
import { formatDollars } from '@/utils/money';
import { cn } from '@utils/cn';
import { jobService, Job } from '@services/jobService';
import { getErrorMessage } from '@/utils/error';
import {
  getJobStatusLabel,
  getJobTypeLabel,
  getCustomerStatusLabel,
  getCustomerTypeLabel,
} from '@app-types';
import { formatEpochDateTime } from '@/utils/epoch';
import { useTenantTimezone } from '@/hooks/useTenantTimezone';
import type {
  CustomerType,
  CustomerStatus,
  PreferredContactMethod,
} from '@app-types/customer';
import {
  ArrowLeft,
  Save,
  Trash2,
  Loader2,
  Copy,
  ExternalLink,
  Plus,
  Pencil,
  Receipt,
} from 'lucide-react';

type CustomerTab = 'sites' | 'agreements' | 'jobs' | 'purchase_orders';

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
  // Contact preferences
  preferred_contact_method: PreferredContactMethod;
  do_not_email: boolean;
  do_not_call: boolean;
  do_not_sms: boolean;
  do_not_mail: boolean;
  // Marketing consent
  marketing_consent: boolean;
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
  // Contact preferences
  preferred_contact_method: 'any',
  do_not_email: false,
  do_not_call: false,
  do_not_sms: false,
  do_not_mail: false,
  // Marketing consent
  marketing_consent: false,
};

export function CustomerDetailPage() {
  const navigate = useNavigate();
  const timeZone = useTenantTimezone();
  const { id } = useParams<{ id: string }>();
  const isNew = !id || id === 'new';

  const [formData, setFormData] = useState<CustomerFormData>(initialFormData);
  const [useServiceAddress, setUseServiceAddress] = useState(false);
  const [isLoading, setIsLoading] = useState(!isNew);
  const [isSaving, setIsSaving] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [recentJobs, setRecentJobs] = useState<Job[]>([]);
  // The customer as last loaded, for the summary above the form.
  const [customer, setCustomer] = useState<Customer | null>(null);
  const [isLoadingJobs, setIsLoadingJobs] = useState(false);
  const [activeTab, setActiveTab] = useState<CustomerTab>('sites');
  const [isAddSiteOpen, setIsAddSiteOpen] = useState(false);
  const [isRulesOpen, setIsRulesOpen] = useState(false);
  const [isConsolidateOpen, setIsConsolidateOpen] = useState(false);

  // Shares the Sites tab's query (same key), for the tab count.
  const sitesQuery = useQuery({
    queryKey: sitesQueryKey(id, true),
    queryFn: () => siteService.list(id as string, true),
    enabled: !isNew && !!id,
  });
  const activeSiteCount = (sitesQuery.data ?? []).filter(
    (s) => s.is_active
  ).length;
  // Shares the Purchase orders tab's query (same key), for the tab count.
  const purchaseOrdersQuery = useQuery({
    queryKey: purchaseOrdersQueryKey(id ?? ''),
    queryFn: () => purchaseOrderService.list(id as string),
    enabled: !isNew && !!id,
  });
  // Shares the Sites tab's agreements query (same key).
  const agreementsQuery = useQuery({
    queryKey: agreementKeys.list({ customer_id: id }),
    queryFn: () => agreementService.list({ customer_id: id }),
    enabled: !isNew && !!id,
  });

  const loadCustomer = async (customerId: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const customer = await customerService.getCustomer(customerId);
      setCustomer(customer);
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
        // Contact preferences
        preferred_contact_method: customer.preferred_contact_method || 'any',
        do_not_email: customer.do_not_email || false,
        do_not_call: customer.do_not_call || false,
        do_not_sms: customer.do_not_sms || false,
        do_not_mail: customer.do_not_mail || false,
        // Marketing consent
        marketing_consent: customer.marketing_consent || false,
      });
      setUseServiceAddress(!!customer.service_address_street);
    } catch (err) {
      console.error('Failed to load customer:', err);
      setError('Failed to load customer details');
    } finally {
      setIsLoading(false);
    }
  };

  const loadRecentJobs = async (customerId: string) => {
    setIsLoadingJobs(true);
    try {
      const jobs = await jobService.getCustomerJobs(customerId);
      setRecentJobs(jobs.slice(0, 5));
    } catch (err) {
      console.error('Failed to load customer jobs:', err);
    } finally {
      setIsLoadingJobs(false);
    }
  };

  useEffect(() => {
    if (!isNew && id) {
      loadCustomer(id);
      loadRecentJobs(id);
    }
  }, [id, isNew]);

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
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
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
    } catch (err) {
      console.error('Failed to save customer:', err);
      setError(getErrorMessage(err, 'Failed to save customer'));
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
    } catch (err) {
      console.error('Failed to delete customer:', err);
      setError(getErrorMessage(err, 'Failed to delete customer'));
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

  const crumbName = isNew
    ? 'New customer'
    : customer?.company_name ||
      (customer ? `${customer.first_name} ${customer.last_name}`.trim() : '');
  return (
    <DashboardLayout
      breadcrumbs={[
        { label: 'Customers & Sites', to: '/customers' },
        { label: crumbName || 'Customer' },
      ]}
    >
      <div className="mx-auto max-w-[1200px] p-4 sm:p-6 lg:p-8">
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate('/customers')}
          className="mb-3 -ml-2 gap-2 text-neutral-600"
        >
          <ArrowLeft className="h-4 w-4" />
          Customers
        </Button>
        {isNew || !customer ? (
          <PageHeader
            title={isNew ? 'New customer' : 'Edit customer'}
            subtitle={
              isNew
                ? 'Who you do the work for and where to bill it.'
                : undefined
            }
          />
        ) : (
          <>
            <PageHeader
              leading={
                <Avatar
                  email={customer.email || customer.id}
                  firstName={customer.company_name || customer.first_name}
                  lastName={
                    customer.company_name ? undefined : customer.last_name
                  }
                  shape="tile"
                  size="xl"
                />
              }
              title={
                customer.company_name ||
                `${customer.first_name} ${customer.last_name}`.trim()
              }
              badge={
                <Badge
                  variant={getStatusBadgeVariant(customer.status)}
                  size="sm"
                >
                  {getCustomerStatusLabel(customer.status)}
                </Badge>
              }
              subtitle={`${getCustomerTypeLabel(customer.customer_type)}${
                sitesQuery.data
                  ? ` · ${activeSiteCount} ${activeSiteCount === 1 ? 'site' : 'sites'}`
                  : ''
              } · customer since ${new Date(
                customer.created_at
              ).toLocaleDateString('en-US', {
                month: 'long',
                year: 'numeric',
              })}`}
              actions={
                <>
                  <Button
                    variant="outline"
                    onClick={() => {
                      setActiveTab('sites');
                      setIsAddSiteOpen(true);
                    }}
                    className="gap-2"
                  >
                    <Plus className="h-4 w-4" />
                    Add site
                  </Button>
                  <Button
                    variant="primary"
                    onClick={() =>
                      navigate(`/jobs/new`, {
                        state: { prefillCustomerId: id },
                      })
                    }
                    className="gap-2"
                  >
                    <Plus className="h-4 w-4" />
                    New job
                  </Button>
                </>
              }
            />
            {/* Billing rules (T187), mockup 06. */}
            <div className="mb-2">
              <BillingRulesCards customer={customer} />
            </div>
            <div className="mb-6 flex justify-end gap-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setIsConsolidateOpen(true)}
                className="gap-1.5 text-primary-600"
              >
                <Receipt className="h-3.5 w-3.5" />
                Create consolidated invoice
              </Button>
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setIsRulesOpen(true)}
                className="gap-1.5 text-primary-600"
              >
                <Pencil className="h-3.5 w-3.5" />
                Edit billing rules
              </Button>
            </div>
            {/* One invoice for the period, a line per site (T188). */}
            <ConsolidatedInvoiceModal
              isOpen={isConsolidateOpen}
              onClose={() => setIsConsolidateOpen(false)}
              customerId={customer.id}
              customerName={
                customer.company_name ||
                `${customer.first_name} ${customer.last_name}`.trim()
              }
              onCreated={(invoice) => navigate(`/invoices/${invoice.id}`)}
            />
            <BillingRulesModal
              customer={customer}
              isOpen={isRulesOpen}
              onClose={() => setIsRulesOpen(false)}
              onSaved={setCustomer}
            />

            {/* Tabs (mockup 06). */}
            <div
              role="tablist"
              aria-label="Customer"
              className="mb-4 flex gap-6 border-b border-neutral-200"
            >
              {(
                [
                  [
                    'sites',
                    'Sites',
                    sitesQuery.data ? activeSiteCount : undefined,
                  ],
                  ['agreements', 'Agreements', agreementsQuery.data?.length],
                  [
                    'jobs',
                    'Jobs',
                    isLoadingJobs ? undefined : recentJobs.length,
                  ],
                  [
                    'purchase_orders',
                    'Purchase orders',
                    purchaseOrdersQuery.data?.length,
                  ],
                ] as const
              ).map(([tab, label, count]) => (
                <button
                  key={tab}
                  type="button"
                  role="tab"
                  aria-selected={activeTab === tab}
                  onClick={() => setActiveTab(tab)}
                  className={cn(
                    '-mb-px flex items-center gap-2 border-b-2 px-0.5 pb-3 text-[15px] font-medium transition-colors',
                    activeTab === tab
                      ? 'border-primary-600 text-primary-600'
                      : 'border-transparent text-neutral-700 hover:text-neutral-900'
                  )}
                >
                  {label}
                  {count !== undefined && (
                    <span
                      className={cn(
                        'rounded-full px-2 py-0.5 text-xs',
                        activeTab === tab
                          ? 'bg-primary-50 text-primary-700'
                          : 'bg-neutral-100 text-neutral-600'
                      )}
                    >
                      {count}
                    </span>
                  )}
                </button>
              ))}
            </div>
            <div className="mb-8" role="tabpanel">
              {activeTab === 'sites' && id ? (
                <CustomerSitesSection
                  customerId={id}
                  consolidatedBilling={!!customer.consolidated_billing}
                  isAddOpen={isAddSiteOpen}
                  onAddOpenChange={setIsAddSiteOpen}
                />
              ) : activeTab === 'agreements' && id ? (
                <CustomerAgreements
                  agreements={agreementsQuery.data}
                  isLoading={agreementsQuery.isPending}
                  onOpen={(agreementId) =>
                    navigate(`/agreements/${agreementId}`)
                  }
                  onNew={() =>
                    navigate('/agreements/new', {
                      state: { customerId: id },
                    })
                  }
                  onConsolidate={() => setIsConsolidateOpen(true)}
                />
              ) : activeTab === 'purchase_orders' && id ? (
                <CustomerPurchaseOrdersSection customerId={id} />
              ) : (
                <RecentJobs
                  jobs={recentJobs}
                  isLoading={isLoadingJobs}
                  timeZone={timeZone}
                  onView={(jobId) => navigate(`/jobs/${jobId}`)}
                  onRepeat={(jobId) => navigate(`/jobs/new?clone=${jobId}`)}
                />
              )}
            </div>
            <h2 className="mb-4 text-xl font-semibold text-neutral-900">
              Customer details
            </h2>
          </>
        )}

        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-6">
          {/* Basic Information */}
          <div className="bg-white rounded-2xl border border-neutral-200 p-6">
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
          <div className="bg-white rounded-2xl border border-neutral-200 p-6">
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
          <div className="bg-white rounded-2xl border border-neutral-200 p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-lg font-semibold text-neutral-900">
                Service Address
              </h2>
              <label className="flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={useServiceAddress}
                  onChange={(e) => setUseServiceAddress(e.target.checked)}
                  className="rounded-sm border-neutral-300 text-primary-600 focus:ring-primary-500"
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
          <div className="bg-white rounded-2xl border border-neutral-200 p-6">
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

          {/* Contact Preferences */}
          <div className="bg-white rounded-2xl border border-neutral-200 p-6">
            <h2 className="text-lg font-semibold text-neutral-900 mb-4">
              Contact Preferences
            </h2>

            <div className="space-y-6">
              <div>
                <label
                  htmlFor="preferred_contact_method"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Preferred Contact Method
                </label>
                <select
                  id="preferred_contact_method"
                  name="preferred_contact_method"
                  value={formData.preferred_contact_method}
                  onChange={handleChange}
                  className="w-full max-w-xs px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                >
                  <option value="any">Any</option>
                  <option value="email">Email</option>
                  <option value="phone">Phone</option>
                  <option value="sms">SMS</option>
                  <option value="mail">Mail</option>
                </select>
              </div>

              <div>
                <p className="text-sm font-medium text-neutral-700 mb-3">
                  Do Not Contact Via
                </p>
                <div className="flex flex-wrap gap-4">
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      name="do_not_email"
                      checked={formData.do_not_email}
                      onChange={(e) =>
                        setFormData((prev) => ({
                          ...prev,
                          do_not_email: e.target.checked,
                        }))
                      }
                      className="rounded-sm border-neutral-300 text-primary-600 focus:ring-primary-500"
                    />
                    <span className="text-sm text-neutral-600">Email</span>
                  </label>
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      name="do_not_call"
                      checked={formData.do_not_call}
                      onChange={(e) =>
                        setFormData((prev) => ({
                          ...prev,
                          do_not_call: e.target.checked,
                        }))
                      }
                      className="rounded-sm border-neutral-300 text-primary-600 focus:ring-primary-500"
                    />
                    <span className="text-sm text-neutral-600">Phone</span>
                  </label>
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      name="do_not_sms"
                      checked={formData.do_not_sms}
                      onChange={(e) =>
                        setFormData((prev) => ({
                          ...prev,
                          do_not_sms: e.target.checked,
                        }))
                      }
                      className="rounded-sm border-neutral-300 text-primary-600 focus:ring-primary-500"
                    />
                    <span className="text-sm text-neutral-600">SMS</span>
                  </label>
                  <label className="flex items-center gap-2 cursor-pointer">
                    <input
                      type="checkbox"
                      name="do_not_mail"
                      checked={formData.do_not_mail}
                      onChange={(e) =>
                        setFormData((prev) => ({
                          ...prev,
                          do_not_mail: e.target.checked,
                        }))
                      }
                      className="rounded-sm border-neutral-300 text-primary-600 focus:ring-primary-500"
                    />
                    <span className="text-sm text-neutral-600">Mail</span>
                  </label>
                </div>
              </div>

              <div className="pt-2 border-t border-neutral-200">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="checkbox"
                    name="marketing_consent"
                    checked={formData.marketing_consent}
                    onChange={(e) =>
                      setFormData((prev) => ({
                        ...prev,
                        marketing_consent: e.target.checked,
                      }))
                    }
                    className="rounded-sm border-neutral-300 text-primary-600 focus:ring-primary-500"
                  />
                  <span className="text-sm text-neutral-700">
                    Customer has consented to receive marketing communications
                  </span>
                </label>
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

interface RecentJobsProps {
  jobs: Job[];
  isLoading: boolean;
  timeZone: string | undefined;
  onView: (jobId: string) => void;
  onRepeat: (jobId: string) => void;
}

/** The Jobs tab: the customer's five most recent jobs. */
function RecentJobs({
  jobs,
  isLoading,
  timeZone,
  onView,
  onRepeat,
}: RecentJobsProps) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center rounded-2xl border border-neutral-200 bg-white py-8">
        <Loader2 className="h-6 w-6 animate-spin text-neutral-400" />
      </div>
    );
  }
  if (jobs.length === 0) {
    return (
      <p className="rounded-2xl border border-neutral-200 bg-white py-8 text-center text-sm text-neutral-500">
        No jobs yet for this customer.
      </p>
    );
  }
  return (
    <div className="space-y-3 rounded-2xl border border-neutral-200 bg-white p-4">
      {jobs.map((job) => (
        <div
          key={job.id}
          className="flex items-center justify-between rounded-xl border border-neutral-200 p-3 transition-colors hover:border-neutral-300"
        >
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2">
              <span className="font-mono text-sm text-primary-600">
                {job.job_number}
              </span>
              <span className="truncate font-medium text-neutral-900">
                {job.title}
              </span>
            </div>
            <div className="mt-1 flex items-center gap-2">
              <Badge variant={getStatusBadgeVariant(job.status)} size="sm">
                {getJobStatusLabel(job.status)}
              </Badge>
              <span className="text-xs text-neutral-500">
                {getJobTypeLabel(job.job_type)}
              </span>
              {job.scheduled_start_at && (
                <span className="text-xs text-neutral-500">
                  {formatEpochDateTime(job.scheduled_start_at, timeZone)}
                </span>
              )}
            </div>
          </div>
          <div className="ml-3 flex shrink-0 items-center gap-1">
            <Button
              variant="ghost"
              onClick={() => onView(job.id)}
              className="flex items-center gap-1 px-2 py-1 text-xs"
            >
              <ExternalLink className="h-3.5 w-3.5" />
              View
            </Button>
            <Button
              variant="outline"
              onClick={() => onRepeat(job.id)}
              className="flex items-center gap-1 px-2 py-1 text-xs"
            >
              <Copy className="h-3.5 w-3.5" />
              Repeat job
            </Button>
          </div>
        </div>
      ))}
    </div>
  );
}

interface CustomerAgreementsProps {
  agreements: Agreement[] | undefined;
  isLoading: boolean;
  onOpen: (agreementId: string) => void;
  onNew: () => void;
  /** Opens "Create consolidated invoice" (T188). */
  onConsolidate: () => void;
}

/** The Agreements tab: the customer's service agreements (T167). */
function CustomerAgreements({
  agreements,
  isLoading,
  onOpen,
  onNew,
  onConsolidate,
}: CustomerAgreementsProps) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center rounded-2xl border border-neutral-200 bg-white py-8">
        <Loader2 className="h-6 w-6 animate-spin text-neutral-400" />
      </div>
    );
  }
  const list = agreements ?? [];
  return (
    <div className="rounded-2xl border border-neutral-200 bg-white p-4">
      {list.length === 0 ? (
        <p className="py-4 text-center text-sm text-neutral-500">
          No agreements yet for this customer.
        </p>
      ) : (
        <ul className="divide-y divide-neutral-200">
          {list.map((a) => (
            <li key={a.id}>
              <button
                type="button"
                onClick={() => onOpen(a.id)}
                className="flex w-full items-center justify-between gap-4 rounded-xl px-3 py-3 text-left hover:bg-neutral-50"
              >
                <span className="min-w-0">
                  <span className="block font-medium text-neutral-900">
                    {a.title}
                  </span>
                  <span className="block text-sm text-neutral-500">
                    {describeFrequency(a)}
                  </span>
                </span>
                <span className="flex shrink-0 items-center gap-3">
                  <span className="font-semibold text-neutral-900 tabular-nums">
                    {formatDollars(monthlyAmount(a), { wholeUnits: true })}/mo
                  </span>
                  <Badge
                    size="sm"
                    variant={a.status === 'active' ? 'success' : 'neutral'}
                  >
                    {a.status === 'active' ? 'Active' : 'Paused'}
                  </Badge>
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
      <div className="mt-2 flex justify-end gap-2">
        {list.length > 0 && (
          <Button
            variant="ghost"
            size="sm"
            onClick={onConsolidate}
            className="gap-2 text-primary-600"
          >
            <Receipt className="h-4 w-4" />
            Invoice a period
          </Button>
        )}
        <Button variant="outline" size="sm" onClick={onNew} className="gap-2">
          <Plus className="h-4 w-4" />
          New agreement
        </Button>
      </div>
    </div>
  );
}

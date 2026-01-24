import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { DashboardLayout } from '@components/layout';
import {
  Card,
  CardHeader,
  CardTitle,
  CardContent,
  StatsCard,
  Badge,
  Button,
  getStatusBadgeVariant,
} from '@components/shared';
import { useAuthStore, useTenantStore } from '@store';
import { customerService, CustomerStats } from '@services/customerService';
import { jobService, JobStats, Job } from '@services/jobService';
import { quoteService } from '@services/quoteService';
import { invoiceService, InvoiceStats } from '@services/invoiceService';
import type { Quote, QuoteStats } from '@app-types/quote';
import { formatCurrency, getJobStatusLabel } from '@app-types';
import {
  Users,
  Briefcase,
  FileText,
  Receipt,
  ArrowRight,
  Building2,
  Rocket,
} from 'lucide-react';

interface DashboardStats {
  customers: CustomerStats | null;
  jobs: JobStats | null;
  quotes: QuoteStats | null;
  invoices: InvoiceStats | null;
}

export function DashboardPage() {
  const navigate = useNavigate();
  const user = useAuthStore((state) => state.user);
  const {
    currentTenant,
    tenants,
    isLoading: tenantsLoading,
    loadTenants,
  } = useTenantStore();
  const [stats, setStats] = useState<DashboardStats>({
    customers: null,
    jobs: null,
    quotes: null,
    invoices: null,
  });
  const [recentJobs, setRecentJobs] = useState<Job[]>([]);
  const [recentQuotes, setRecentQuotes] = useState<Quote[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [tenantsChecked, setTenantsChecked] = useState(false);

  // Load tenants on mount to check if user has any organizations
  useEffect(() => {
    const checkTenants = async () => {
      try {
        await loadTenants();
      } catch (err) {
        console.error('Failed to load tenants:', err);
      } finally {
        setTenantsChecked(true);
      }
    };
    checkTenants();
  }, [loadTenants]);

  // Only load dashboard data if user has an organization
  useEffect(() => {
    if (currentTenant) {
      loadDashboardData();
    } else {
      setIsLoading(false);
    }
  }, [currentTenant]);

  const loadDashboardData = async () => {
    setIsLoading(true);
    try {
      const [
        customerStats,
        jobStats,
        quoteStats,
        invoiceStats,
        jobsResponse,
        quotesResponse,
      ] = await Promise.allSettled([
        customerService.getStats(),
        jobService.getStats(),
        quoteService.getStats(),
        invoiceService.getStats(),
        jobService.getJobs({
          page: 1,
          page_size: 5,
          sort_by: 'created_at',
          sort_order: 'desc',
        }),
        quoteService.getQuotes({
          page: 1,
          page_size: 5,
          sort_by: 'created_at',
          sort_order: 'desc',
        }),
      ]);

      setStats({
        customers:
          customerStats.status === 'fulfilled' ? customerStats.value : null,
        jobs: jobStats.status === 'fulfilled' ? jobStats.value : null,
        quotes: quoteStats.status === 'fulfilled' ? quoteStats.value : null,
        invoices:
          invoiceStats.status === 'fulfilled' ? invoiceStats.value : null,
      });

      if (jobsResponse.status === 'fulfilled') {
        setRecentJobs(jobsResponse.value.jobs);
      }

      if (quotesResponse.status === 'fulfilled') {
        setRecentQuotes(quotesResponse.value.quotes);
      }
    } catch (error) {
      console.error('Failed to load dashboard data:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const quickActions = [
    {
      label: 'New Customer',
      icon: Users,
      onClick: () => navigate('/customers/new'),
      color: 'bg-blue-100 text-blue-600',
    },
    {
      label: 'New Job',
      icon: Briefcase,
      onClick: () => navigate('/jobs/new'),
      color: 'bg-green-100 text-green-600',
    },
    {
      label: 'New Quote',
      icon: FileText,
      onClick: () => navigate('/quotes/new'),
      color: 'bg-purple-100 text-purple-600',
    },
    {
      label: 'New Invoice',
      icon: Receipt,
      onClick: () => navigate('/invoices/new'),
      color: 'bg-orange-100 text-orange-600',
    },
  ];

  const getQuoteStatusLabel = (status: string): string => {
    const labels: Record<string, string> = {
      draft: 'Draft',
      sent: 'Sent',
      viewed: 'Viewed',
      accepted: 'Accepted',
      declined: 'Declined',
      rejected: 'Rejected',
      expired: 'Expired',
    };
    return labels[status] || status;
  };

  // Show loading while checking tenants
  if (!tenantsChecked || tenantsLoading) {
    return (
      <DashboardLayout>
        <div className="p-6 lg:p-8 flex items-center justify-center min-h-[400px]">
          <div className="text-center">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto"></div>
            <p className="text-neutral-600 mt-4">Loading...</p>
          </div>
        </div>
      </DashboardLayout>
    );
  }

  // Show onboarding if user has no organization
  if (!currentTenant && tenants.length === 0) {
    return (
      <DashboardLayout>
        <div className="p-6 lg:p-8">
          <div className="max-w-2xl mx-auto text-center py-12">
            <div className="inline-flex items-center justify-center h-20 w-20 rounded-full bg-primary-100 mb-6">
              <Rocket className="h-10 w-10 text-primary-600" />
            </div>
            <h1 className="text-3xl font-bold text-neutral-900 mb-4">
              Welcome to ServicePro!
            </h1>
            <p className="text-lg text-neutral-600 mb-8">
              Get started by creating your first organization. This will be your
              workspace for managing customers, jobs, quotes, and invoices.
            </p>

            <Card variant="elevated" padding="lg" className="text-left mb-6">
              <CardContent>
                <h3 className="font-semibold text-neutral-900 mb-4">
                  What you'll be able to do:
                </h3>
                <ul className="space-y-3">
                  <li className="flex items-start">
                    <Users className="h-5 w-5 text-primary-600 mr-3 mt-0.5 flex-shrink-0" />
                    <span className="text-neutral-700">
                      Manage your customer database and track interactions
                    </span>
                  </li>
                  <li className="flex items-start">
                    <Briefcase className="h-5 w-5 text-primary-600 mr-3 mt-0.5 flex-shrink-0" />
                    <span className="text-neutral-700">
                      Create and schedule jobs with status tracking
                    </span>
                  </li>
                  <li className="flex items-start">
                    <FileText className="h-5 w-5 text-primary-600 mr-3 mt-0.5 flex-shrink-0" />
                    <span className="text-neutral-700">
                      Generate professional quotes for your services
                    </span>
                  </li>
                  <li className="flex items-start">
                    <Receipt className="h-5 w-5 text-primary-600 mr-3 mt-0.5 flex-shrink-0" />
                    <span className="text-neutral-700">
                      Send invoices and track payments
                    </span>
                  </li>
                </ul>
              </CardContent>
            </Card>

            <Button
              variant="primary"
              size="lg"
              onClick={() => navigate('/settings/organization/new')}
              className="inline-flex items-center"
            >
              <Building2 className="h-5 w-5 mr-2" />
              Create Your Organization
            </Button>
          </div>
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout>
      <div className="p-6 lg:p-8">
        {/* Welcome Section */}
        <div className="mb-8">
          <h1 className="text-2xl lg:text-3xl font-bold text-neutral-900">
            Welcome back{user?.email ? `, ${user.email.split('@')[0]}` : ''}!
          </h1>
          <p className="text-neutral-600 mt-1">
            Here's an overview of your business today.
          </p>
        </div>

        {/* Stats Grid */}
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 lg:gap-6 mb-8">
          <StatsCard
            label="Total Customers"
            value={
              isLoading
                ? '...'
                : stats.customers?.total_customers?.toString() || '0'
            }
            icon={<Users className="h-6 w-6 text-primary-600" />}
            change={
              stats.customers?.active_customers
                ? `${stats.customers.active_customers} active`
                : undefined
            }
            changeType="neutral"
          />
          <StatsCard
            label="Active Jobs"
            value={
              isLoading
                ? '...'
                : (
                    (stats.jobs?.in_progress_jobs || 0) +
                    (stats.jobs?.scheduled_jobs || 0)
                  ).toString()
            }
            icon={<Briefcase className="h-6 w-6 text-primary-600" />}
            change={
              stats.jobs?.pending_jobs
                ? `${stats.jobs.pending_jobs} pending`
                : undefined
            }
            changeType="neutral"
          />
          <StatsCard
            label="Pending Quotes"
            value={
              isLoading ? '...' : stats.quotes?.sent_quotes?.toString() || '0'
            }
            icon={<FileText className="h-6 w-6 text-primary-600" />}
            change={
              stats.quotes?.draft_quotes
                ? `${stats.quotes.draft_quotes} drafts`
                : undefined
            }
            changeType="neutral"
          />
          <StatsCard
            label="Outstanding"
            value={
              isLoading
                ? '...'
                : formatCurrency(stats.invoices?.total_outstanding || 0)
            }
            icon={<Receipt className="h-6 w-6 text-primary-600" />}
            change={
              stats.invoices?.overdue_invoices
                ? `${stats.invoices.overdue_invoices} overdue`
                : undefined
            }
            changeType={
              stats.invoices?.overdue_invoices ? 'negative' : 'neutral'
            }
          />
        </div>

        {/* Quick Actions */}
        <Card variant="elevated" padding="lg" className="mb-8">
          <CardHeader>
            <CardTitle>Quick Actions</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 mt-4">
              {quickActions.map((action) => {
                const Icon = action.icon;
                return (
                  <button
                    key={action.label}
                    onClick={action.onClick}
                    className="flex flex-col items-center justify-center gap-2 p-4 rounded-lg border border-neutral-200 hover:border-primary-300 hover:bg-primary-50 transition-colors"
                  >
                    <div className={`p-2 rounded-lg ${action.color}`}>
                      <Icon className="h-5 w-5" />
                    </div>
                    <span className="text-sm font-medium text-neutral-700">
                      {action.label}
                    </span>
                  </button>
                );
              })}
            </div>
          </CardContent>
        </Card>

        {/* Recent Activity Grid */}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Recent Jobs */}
          <Card variant="elevated" padding="lg">
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>Recent Jobs</CardTitle>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => navigate('/jobs')}
                  className="flex items-center gap-1 text-primary-600"
                >
                  View all
                  <ArrowRight className="h-4 w-4" />
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-3 mt-4">
                {isLoading ? (
                  <div className="text-center py-4 text-neutral-500">
                    Loading...
                  </div>
                ) : recentJobs.length === 0 ? (
                  <div className="text-center py-4 text-neutral-500">
                    No recent jobs
                  </div>
                ) : (
                  recentJobs.map((job) => (
                    <div
                      key={job.id}
                      onClick={() => navigate(`/jobs/${job.id}`)}
                      className="flex items-center justify-between p-3 rounded-lg bg-neutral-50 hover:bg-neutral-100 cursor-pointer transition-colors"
                    >
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <span className="font-mono text-sm text-primary-600">
                            {job.job_number}
                          </span>
                          <Badge
                            variant={getStatusBadgeVariant(job.status)}
                            size="sm"
                          >
                            {getJobStatusLabel(job.status)}
                          </Badge>
                        </div>
                        <p className="text-sm text-neutral-700 font-medium truncate mt-1">
                          {job.title}
                        </p>
                        {job.customer && (
                          <p className="text-xs text-neutral-500 truncate">
                            {job.customer.first_name} {job.customer.last_name}
                          </p>
                        )}
                      </div>
                    </div>
                  ))
                )}
              </div>
            </CardContent>
          </Card>

          {/* Recent Quotes */}
          <Card variant="elevated" padding="lg">
            <CardHeader>
              <div className="flex items-center justify-between">
                <CardTitle>Recent Quotes</CardTitle>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => navigate('/quotes')}
                  className="flex items-center gap-1 text-primary-600"
                >
                  View all
                  <ArrowRight className="h-4 w-4" />
                </Button>
              </div>
            </CardHeader>
            <CardContent>
              <div className="space-y-3 mt-4">
                {isLoading ? (
                  <div className="text-center py-4 text-neutral-500">
                    Loading...
                  </div>
                ) : recentQuotes.length === 0 ? (
                  <div className="text-center py-4 text-neutral-500">
                    No recent quotes
                  </div>
                ) : (
                  recentQuotes.map((quote) => (
                    <div
                      key={quote.id}
                      onClick={() => navigate(`/quotes/${quote.id}`)}
                      className="flex items-center justify-between p-3 rounded-lg bg-neutral-50 hover:bg-neutral-100 cursor-pointer transition-colors"
                    >
                      <div className="min-w-0 flex-1">
                        <div className="flex items-center gap-2">
                          <span className="font-mono text-sm text-primary-600">
                            {quote.quote_number}
                          </span>
                          <Badge
                            variant={getStatusBadgeVariant(quote.status)}
                            size="sm"
                          >
                            {getQuoteStatusLabel(quote.status)}
                          </Badge>
                        </div>
                        {quote.customer && (
                          <p className="text-sm text-neutral-700 truncate mt-1">
                            {quote.customer.name || quote.customer.email}
                          </p>
                        )}
                        <p className="text-xs text-neutral-500 mt-0.5">
                          {formatCurrency(quote.total)}
                        </p>
                      </div>
                    </div>
                  ))
                )}
              </div>
            </CardContent>
          </Card>
        </div>
      </div>
    </DashboardLayout>
  );
}

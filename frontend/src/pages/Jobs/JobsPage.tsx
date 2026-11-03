import { useState, useEffect, useRef } from 'react';
import { useNavigate } from 'react-router';
import { DashboardLayout } from '@components/layout';
import { getErrorMessage } from '@/utils/error';
import {
  DataTable,
  Badge,
  EmptyState,
  ErrorState,
  Button,
  getStatusBadgeVariant,
  PageHeader,
} from '@components/shared';
import type { Column } from '@components/shared';
import { jobService, Job, JobFilters } from '@services/jobService';
import { JobActionsMenu, JobFlowModal } from '@components/jobs';
import { Briefcase, Plus, Search, Info, CalendarDays } from 'lucide-react';
import { getJobStatusLabel, getJobPriorityLabel } from '@app-types';
import { formatEpochDateTime } from '@/utils/epoch';
import { useTenantTimezone } from '@/hooks/useTenantTimezone';
import { useTenantStore } from '@store';
import {
  APP_PERMISSIONS,
  useTenantPermissions,
} from '@/utils/tenantPermissions';

export function JobsPage() {
  const navigate = useNavigate();
  const timeZone = useTenantTimezone();
  // Technicians see their assigned jobs (the API filters) and can't create
  // jobs (T035).
  const { currentTenant } = useTenantStore();
  const canCreateJob = useTenantPermissions(currentTenant).can(
    APP_PERMISSIONS.jobsCreate
  );
  const [jobs, setJobs] = useState<Job[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  // Ignore responses from superseded requests (fast paging/searching)
  const requestIdRef = useRef(0);
  const [filters, setFilters] = useState<JobFilters>({
    page: 1,
    page_size: 10,
    sort_by: 'created_at',
    sort_order: 'desc',
  });
  const [searchQuery, setSearchQuery] = useState('');
  const [isFlowModalOpen, setIsFlowModalOpen] = useState(false);

  const loadJobs = async () => {
    const requestId = ++requestIdRef.current;
    setIsLoading(true);
    setLoadError(null);
    try {
      const response = await jobService.getJobs({
        ...filters,
        search: searchQuery || undefined,
      });
      if (requestId !== requestIdRef.current) return;
      setJobs(response.jobs);
      setTotal(response.total);
    } catch (error) {
      if (requestId !== requestIdRef.current) return;
      console.error('Failed to load jobs:', error);
      setLoadError(getErrorMessage(error, 'Failed to load jobs'));
    } finally {
      if (requestId === requestIdRef.current) setIsLoading(false);
    }
  };

  useEffect(() => {
    loadJobs();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filters]);

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    // New filters object re-triggers the load effect (no double request)
    setFilters({ ...filters, page: 1 });
  };

  const columns: Column<Job>[] = [
    {
      key: 'job_number',
      header: 'Job #',
      sortable: true,
      render: (value) => (
        <span className="font-mono text-sm font-medium text-primary-600">
          {value as string}
        </span>
      ),
    },
    {
      key: 'customer',
      header: 'Customer',
      render: (_, row) => (
        <div>
          {row.customer ? (
            <>
              <div className="font-medium text-neutral-900">
                {row.customer.first_name} {row.customer.last_name}
              </div>
              {row.customer.company_name && (
                <div className="text-sm text-neutral-500">
                  {row.customer.company_name}
                </div>
              )}
            </>
          ) : (
            <span className="text-neutral-400">-</span>
          )}
        </div>
      ),
    },
    {
      key: 'title',
      header: 'Title',
      sortable: true,
      render: (value) => (
        <span className="text-neutral-900 font-medium">{value as string}</span>
      ),
    },
    {
      key: 'status',
      header: 'Status',
      sortable: true,
      render: (value) => (
        <Badge variant={getStatusBadgeVariant(value as string)} size="sm">
          {getJobStatusLabel(value as Job['status'])}
        </Badge>
      ),
    },
    {
      key: 'priority',
      header: 'Priority',
      sortable: true,
      render: (value) => (
        <Badge variant={getStatusBadgeVariant(value as string)} size="sm">
          {getJobPriorityLabel(value as Job['priority'])}
        </Badge>
      ),
    },
    {
      key: 'scheduled_start_at',
      header: 'Scheduled',
      sortable: true,
      render: (value) => (
        <span className="text-neutral-600">
          {value ? formatEpochDateTime(value as number, timeZone) : '-'}
        </span>
      ),
    },
    {
      key: 'assignments',
      header: 'Assigned To',
      render: (_, row) => {
        const active = row.assignments?.filter((a) => !a.unassigned_at) ?? [];
        if (active.length === 0) {
          return <span className="text-neutral-400">-</span>;
        }
        return (
          <div className="flex flex-wrap gap-1">
            {active.map((a) => (
              <span
                key={a.id}
                className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-primary-50 text-primary-700"
              >
                {a.user_name}
              </span>
            ))}
          </div>
        );
      },
    },
    {
      key: 'actions',
      header: '',
      render: (_, row) => (
        <JobActionsMenu jobId={row.id} canRepeat={canCreateJob} />
      ),
    },
  ];

  return (
    <DashboardLayout>
      <div className="mx-auto max-w-[1440px] p-4 sm:p-6 lg:p-8">
        <PageHeader
          title="Jobs"
          badge={
            <button
              onClick={() => setIsFlowModalOpen(true)}
              className="rounded-full p-1 hover:bg-neutral-100"
              aria-label="View job flow info"
            >
              <Info className="h-5 w-5 text-neutral-400" />
            </button>
          }
          subtitle="Every work order, from first call to done."
          actions={
            <>
              <Button
                variant="secondary"
                onClick={() => navigate('/jobs/calendar')}
                className="gap-2"
              >
                <CalendarDays className="h-4 w-4" />
                Calendar
              </Button>
              {canCreateJob && (
                <Button
                  variant="primary"
                  onClick={() => navigate('/jobs/new')}
                  className="gap-2"
                >
                  <Plus className="h-4 w-4" />
                  New job
                </Button>
              )}
            </>
          }
        />

        <form onSubmit={handleSearch} className="mb-6">
          <div className="relative max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-neutral-400" />
            <input
              type="text"
              placeholder="Search jobs..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full rounded-xl border border-neutral-300 bg-white py-2.5 pl-10 pr-4 text-[15px] focus:border-primary-500 focus:ring-2 focus:ring-primary-500"
            />
          </div>
        </form>

        {loadError ? (
          <ErrorState
            title="Couldn't load jobs"
            message={loadError}
            onRetry={loadJobs}
          />
        ) : !isLoading && jobs.length === 0 && !searchQuery ? (
          <EmptyState
            icon={<Briefcase className="h-12 w-12" />}
            title="No jobs yet"
            description={
              canCreateJob
                ? 'Create your first job to get started'
                : 'Jobs assigned to you show up here'
            }
            action={
              canCreateJob
                ? { label: 'New job', onClick: () => navigate('/jobs/new') }
                : undefined
            }
          />
        ) : (
          <DataTable
            data={jobs}
            columns={columns}
            keyExtractor={(row) => row.id}
            onRowClick={(row) => navigate(`/jobs/${row.id}`)}
            isLoading={isLoading}
            emptyMessage="No jobs found"
            pagination={{
              page: filters.page || 1,
              pageSize: filters.page_size || 10,
              total,
              onPageChange: (page) => setFilters({ ...filters, page }),
            }}
            sorting={{
              sortBy: filters.sort_by || 'created_at',
              sortOrder: filters.sort_order || 'desc',
              onSort: (sortBy, sortOrder) =>
                setFilters({
                  ...filters,
                  sort_by: sortBy,
                  sort_order: sortOrder,
                }),
            }}
          />
        )}

        <JobFlowModal
          isOpen={isFlowModalOpen}
          onClose={() => setIsFlowModalOpen(false)}
        />
      </div>
    </DashboardLayout>
  );
}

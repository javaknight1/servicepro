import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { DashboardLayout } from '@components/layout';
import {
  DataTable,
  Badge,
  EmptyState,
  Button,
  getStatusBadgeVariant,
} from '@components/shared';
import type { Column } from '@components/shared';
import { jobService, Job, JobFilters } from '@services/jobService';
import { JobActionsMenu, JobFlowModal } from '@components/jobs';
import { Briefcase, Plus, Search, Info, CalendarDays } from 'lucide-react';
import { getJobStatusLabel, getJobPriorityLabel } from '@app-types';
import { formatEpochDateTimeUTC } from '@/utils/epoch';

export function JobsPage() {
  const navigate = useNavigate();
  const [jobs, setJobs] = useState<Job[]>([]);
  const [total, setTotal] = useState(0);
  const [isLoading, setIsLoading] = useState(true);
  const [filters, setFilters] = useState<JobFilters>({
    page: 1,
    page_size: 10,
    sort_by: 'created_at',
    sort_order: 'desc',
  });
  const [searchQuery, setSearchQuery] = useState('');
  const [isFlowModalOpen, setIsFlowModalOpen] = useState(false);

  useEffect(() => {
    loadJobs();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [filters]);

  const loadJobs = async () => {
    setIsLoading(true);
    try {
      const response = await jobService.getJobs({
        ...filters,
        search: searchQuery || undefined,
      });
      setJobs(response.jobs);
      setTotal(response.total);
    } catch (error) {
      console.error('Failed to load jobs:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    setFilters({ ...filters, page: 1 });
    loadJobs();
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
          {value ? formatEpochDateTimeUTC(value as number) : '-'}
        </span>
      ),
    },
    {
      key: 'actions',
      header: '',
      render: (_, row) => <JobActionsMenu jobId={row.id} />,
    },
  ];

  return (
    <DashboardLayout>
      <div className="p-6 lg:p-8">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4 mb-6">
          <div className="flex items-center gap-2">
            <h1 className="text-2xl font-bold text-neutral-900">Jobs</h1>
            <button
              onClick={() => setIsFlowModalOpen(true)}
              className="p-1 rounded-full hover:bg-neutral-100"
              aria-label="View job flow info"
            >
              <Info className="h-5 w-5 text-neutral-400" />
            </button>
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              onClick={() => navigate('/jobs/calendar')}
              className="flex items-center gap-2"
            >
              <CalendarDays className="h-4 w-4" />
              Calendar
            </Button>
            <Button
              variant="primary"
              onClick={() => navigate('/jobs/new')}
              className="flex items-center gap-2"
            >
              <Plus className="h-4 w-4" />
              Create Job
            </Button>
          </div>
        </div>
        <p className="text-neutral-600 mb-6">Track and manage service jobs</p>

        <form onSubmit={handleSearch} className="mb-6">
          <div className="relative max-w-md">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-neutral-400" />
            <input
              type="text"
              placeholder="Search jobs..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full pl-10 pr-4 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>
        </form>

        {!isLoading && jobs.length === 0 && !searchQuery ? (
          <EmptyState
            icon={<Briefcase className="h-12 w-12" />}
            title="No jobs yet"
            description="Create your first job to get started"
            action={{
              label: 'Create Job',
              onClick: () => navigate('/jobs/new'),
            }}
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

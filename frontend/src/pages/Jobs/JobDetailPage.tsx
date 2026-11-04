import { useState, useEffect, useMemo, useCallback } from 'react';
import {
  useNavigate,
  useParams,
  useSearchParams,
  useLocation,
} from 'react-router';
import { DashboardLayout } from '@components/layout';
import { Button, EmptyState } from '@components/shared';
import {
  JobApprovalsSection,
  JobAssignmentsSection,
  JobPhotosSection,
  JobSitePicker,
  PendingAssignmentsSection,
  StatusTransitionButton,
} from '@components/jobs';
import { ConflictChecker } from '@components/scheduling/ConflictChecker';
import {
  jobService,
  JobStatus,
  JobPriority,
  JobType,
  JobAssignment,
  CreateJobAssignment,
  Job,
} from '@services/jobService';
import type { Customer } from '@services/customerService';
import { fetchAllCustomers } from '@services/customerLookup';
import { getErrorMessage, isAxiosErrorWithStatus } from '@/utils/error';
import { trackEvent, AnalyticsEvents } from '@services/analytics';
import { datetimeLocalToEpoch, epochToDatetimeLocal } from '@/utils/epoch';
import { useTenantTimezone } from '@/hooks/useTenantTimezone';
import { useInvoiceJob } from '@/hooks/useInvoiceJob';
import { useAuthStore, useTenantStore } from '@store';
import {
  APP_PERMISSIONS,
  useTenantPermissions,
} from '@/utils/tenantPermissions';
import {
  ArrowLeft,
  Save,
  Trash2,
  Loader2,
  Receipt,
  FileText,
} from 'lucide-react';

interface PendingAssignment extends CreateJobAssignment {
  user_name: string;
}

interface JobFormData {
  customer_id: string;
  site_id: string;
  title: string;
  description: string;
  job_type: JobType;
  status: JobStatus;
  priority: JobPriority;
  scheduled_start_at: string;
  estimated_duration: string;
  estimated_cost: string;
  internal_notes: string;
  customer_notes: string;
  special_instructions: string;
  required_materials: string;
}

const initialFormData: JobFormData = {
  customer_id: '',
  site_id: '',
  title: '',
  description: '',
  job_type: JobType.MAINTENANCE,
  status: JobStatus.NEW,
  priority: JobPriority.NORMAL,
  scheduled_start_at: '',
  estimated_duration: '',
  estimated_cost: '',
  internal_notes: '',
  customer_notes: '',
  special_instructions: '',
  required_materials: '',
};

export function JobDetailPage() {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const location = useLocation();
  const isNew = !id || id === 'new';
  const cloneFromId = searchParams.get('clone');
  const prefillCustomerId = (location.state as { prefillCustomerId?: string })
    ?.prefillCustomerId;

  // Scheduling inputs are wall-clock time in the tenant's timezone
  // (browser local if the tenant has none configured).
  const timeZone = useTenantTimezone();
  const [formData, setFormData] = useState<JobFormData>(initialFormData);
  const [currentJob, setCurrentJob] = useState<Job | null>(null);
  const [customers, setCustomers] = useState<Customer[]>([]);
  const [assignments, setAssignments] = useState<JobAssignment[]>([]);
  const [pendingAssignments, setPendingAssignments] = useState<
    PendingAssignment[]
  >([]);
  const [isLoading, setIsLoading] = useState(!isNew);
  const [isLoadingCustomers, setIsLoadingCustomers] = useState(true);
  const [isSaving, setIsSaving] = useState(false);
  const [isDeleting, setIsDeleting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  // 404: no such job in this organization, or (for a technician) not one
  // assigned to them (T035).
  const [notFound, setNotFound] = useState(false);
  const [hasConflicts, setHasConflicts] = useState(false);
  const { currentTenant } = useTenantStore();
  const { can } = useTenantPermissions(currentTenant);
  // Technicians (T035) work their assigned jobs: no prices or invoices, no
  // customer list, no dispatch tools (conflicts, approvals, assigning,
  // deleting).
  const seesMoney = can(APP_PERMISSIONS.invoicesRead);
  const isOffice = can(APP_PERMISSIONS.jobsViewAll);
  const currentUserId = useAuthStore((s) => s.user?.id);
  const canListCustomers = can(APP_PERMISSIONS.customersList);
  const canReadCustomers = can(APP_PERMISSIONS.customersRead);
  const canAssign = can(APP_PERMISSIONS.jobsAssign);
  const canDeleteJob = can(APP_PERMISSIONS.jobsDelete);
  // Completed job -> draft invoice (T029)
  const {
    invoiceJob,
    pendingJobId: invoicingJobId,
    error: invoiceError,
  } = useInvoiceJob();
  const [isCheckingConflicts, setIsCheckingConflicts] = useState(false);

  // Min/max for scheduling: no past dates, no more than 1 year ahead
  const scheduleMinDatetime = useMemo(() => {
    const now = new Date();
    now.setSeconds(0, 0);
    return epochToDatetimeLocal(Math.floor(now.getTime() / 1000), timeZone);
  }, [timeZone]);

  const scheduleMaxDatetime = useMemo(() => {
    const oneYear = new Date();
    oneYear.setFullYear(oneYear.getFullYear() + 1);
    oneYear.setSeconds(0, 0);
    return epochToDatetimeLocal(Math.floor(oneYear.getTime() / 1000), timeZone);
  }, [timeZone]);

  const startTime = useMemo(
    () => datetimeLocalToEpoch(formData.scheduled_start_at, timeZone),
    [formData.scheduled_start_at, timeZone]
  );

  const endTime = useMemo(() => {
    if (!startTime || !formData.estimated_duration) return null;
    const minutes = parseInt(formData.estimated_duration);
    if (isNaN(minutes) || minutes <= 0) return null;
    return startTime + minutes * 60;
  }, [startTime, formData.estimated_duration]);

  const assignedTechIds = useMemo(() => {
    if (isNew) {
      return pendingAssignments.map((a) => a.user_id);
    }
    return assignments.map((a) => a.user_id);
  }, [isNew, pendingAssignments, assignments]);

  const conflictJobId = id || 'new';

  const canCheckConflicts =
    isOffice &&
    startTime !== null &&
    endTime !== null &&
    assignedTechIds.length > 0;

  const handleConflictDetected = useCallback(() => {
    setHasConflicts(true);
    setIsCheckingConflicts(false);
  }, []);

  const handleConflictResolved = useCallback(() => {
    setHasConflicts(false);
    setIsCheckingConflicts(false);
  }, []);

  // Track when conflict-relevant inputs change to set checking state
  useEffect(() => {
    if (canCheckConflicts) {
      setIsCheckingConflicts(true);
    } else {
      setHasConflicts(false);
      setIsCheckingConflicts(false);
    }
  }, [startTime, endTime, assignedTechIds, canCheckConflicts]);

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

  const loadJob = useCallback(
    async (jobId: string) => {
      setIsLoading(true);
      setError(null);
      setNotFound(false);
      try {
        const job = await jobService.getJob(jobId);
        setCurrentJob(job);
        setFormData({
          customer_id: job.customer_id || '',
          site_id: job.site_id || '',
          title: job.title || '',
          description: job.description || '',
          job_type: job.job_type || JobType.MAINTENANCE,
          status: job.status || JobStatus.NEW,
          priority: job.priority || JobPriority.NORMAL,
          scheduled_start_at: job.scheduled_start_at
            ? epochToDatetimeLocal(job.scheduled_start_at, timeZone)
            : '',
          estimated_duration: job.estimated_duration?.toString() || '',
          estimated_cost: job.estimated_cost
            ? job.estimated_cost.toString()
            : '',
          internal_notes: job.internal_notes || '',
          customer_notes: job.customer_notes || '',
          special_instructions: job.special_instructions || '',
          required_materials: job.required_materials || '',
        });
        setAssignments(job.assignments || []);
      } catch (err) {
        console.error('Failed to load job:', err);
        if (isAxiosErrorWithStatus(err, 404)) {
          setNotFound(true);
        } else {
          setError('Failed to load job details');
        }
      } finally {
        setIsLoading(false);
      }
    },
    [timeZone]
  );

  // The customer list needs customers.list; without it the job's own
  // customer is shown read-only.
  useEffect(() => {
    if (canListCustomers) {
      loadCustomers();
    } else {
      setIsLoadingCustomers(false);
    }
  }, [canListCustomers]);

  useEffect(() => {
    if (!isNew && id) {
      loadJob(id);
    } else if (isNew && cloneFromId) {
      loadJobForCloning(cloneFromId);
    } else if (isNew && prefillCustomerId) {
      setFormData((prev) => ({ ...prev, customer_id: prefillCustomerId }));
    }
  }, [id, isNew, cloneFromId, prefillCustomerId, loadJob]);

  const loadJobForCloning = async (sourceJobId: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const job = await jobService.getJob(sourceJobId);
      setFormData({
        customer_id: job.customer_id || '',
        site_id: job.site_id || '',
        title: job.title || '',
        description: job.description || '',
        job_type: job.job_type || JobType.MAINTENANCE,
        status: JobStatus.NEW,
        priority: job.priority || JobPriority.NORMAL,
        scheduled_start_at: '',
        estimated_duration: job.estimated_duration?.toString() || '',
        estimated_cost: job.estimated_cost ? job.estimated_cost.toString() : '',
        internal_notes: job.internal_notes || '',
        customer_notes: job.customer_notes || '',
        special_instructions: job.special_instructions || '',
        required_materials: job.required_materials || '',
      });
    } catch (err) {
      console.error('Failed to load job for cloning:', err);
      setError('Failed to load source job for cloning');
    } finally {
      setIsLoading(false);
    }
  };

  const handleAssignmentChange = async () => {
    if (id) {
      try {
        const job = await jobService.getJob(id);
        setAssignments(job.assignments || []);
      } catch (err) {
        console.error('Failed to reload assignments:', err);
      }
    }
  };

  const handleChange = (
    e: React.ChangeEvent<
      HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement
    >
  ) => {
    const { name, value } = e.target;
    setFormData((prev) =>
      // A new customer means a new list of sites.
      name === 'customer_id'
        ? { ...prev, customer_id: value, site_id: '' }
        : { ...prev, [name]: value }
    );
  };

  const estimatedCost = formData.estimated_cost
    ? parseFloat(formData.estimated_cost)
    : undefined;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSaving(true);
    setError(null);

    // Validate scheduling date range
    if (formData.scheduled_start_at) {
      const scheduledEpoch = datetimeLocalToEpoch(
        formData.scheduled_start_at,
        timeZone
      );
      if (scheduledEpoch !== null) {
        const nowEpoch = Math.floor(Date.now() / 1000);
        if (scheduledEpoch < nowEpoch) {
          setError('Scheduled start date cannot be in the past.');
          setIsSaving(false);
          return;
        }
        const oneYearEpoch = Math.floor(
          new Date(
            new Date().setFullYear(new Date().getFullYear() + 1)
          ).getTime() / 1000
        );
        if (scheduledEpoch > oneYearEpoch) {
          setError(
            'Scheduled start date cannot be more than 1 year in the future.'
          );
          setIsSaving(false);
          return;
        }
      }
    }

    try {
      if (isNew) {
        // Create new job with assignments
        const source = cloneFromId ? 'repeat' : 'manual';
        await jobService.createJob({
          customer_id: formData.customer_id,
          site_id: formData.site_id || undefined,
          estimated_cost: seesMoney ? estimatedCost : undefined,
          title: formData.title,
          description: formData.description || undefined,
          job_type: formData.job_type,
          priority: formData.priority,
          scheduled_start_at:
            datetimeLocalToEpoch(formData.scheduled_start_at, timeZone) ??
            undefined,
          estimated_duration: formData.estimated_duration
            ? parseInt(formData.estimated_duration)
            : undefined,
          internal_notes: formData.internal_notes || undefined,
          customer_notes: formData.customer_notes || undefined,
          special_instructions: formData.special_instructions || undefined,
          required_materials: formData.required_materials || undefined,
          assignments:
            pendingAssignments.length > 0
              ? pendingAssignments.map((a) => ({
                  user_id: a.user_id,
                  role: a.role,
                }))
              : undefined,
        });
        trackEvent(AnalyticsEvents.JOB_CREATED, {
          source,
          job_type: formData.job_type,
        });
      } else if (id) {
        // Update existing job
        await jobService.updateJob(id, {
          // The API can't unset a site; an empty pick leaves it unchanged.
          site_id: formData.site_id || undefined,
          estimated_cost: seesMoney ? estimatedCost : undefined,
          title: formData.title,
          description: formData.description || undefined,
          job_type: formData.job_type,
          // Status changes only through the status actions above (the API
          // rejects a status change on update).
          priority: formData.priority,
          scheduled_start_at:
            datetimeLocalToEpoch(formData.scheduled_start_at, timeZone) ??
            undefined,
          estimated_duration: formData.estimated_duration
            ? parseInt(formData.estimated_duration)
            : undefined,
          internal_notes: formData.internal_notes || undefined,
          customer_notes: formData.customer_notes || undefined,
          special_instructions: formData.special_instructions || undefined,
          required_materials: formData.required_materials || undefined,
        });
      }
      navigate('/jobs');
    } catch (err) {
      console.error('Failed to save job:', err);
      setError(getErrorMessage(err, 'Failed to save job'));
    } finally {
      setIsSaving(false);
    }
  };

  const handleDelete = async () => {
    if (!id || isNew) return;
    if (!window.confirm('Are you sure you want to delete this job?')) return;

    setIsDeleting(true);
    setError(null);
    try {
      await jobService.deleteJob(id);
      navigate('/jobs');
    } catch (err) {
      console.error('Failed to delete job:', err);
      setError(getErrorMessage(err, 'Failed to delete job'));
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

  if (notFound) {
    return (
      <DashboardLayout>
        <div className="mx-auto max-w-5xl p-4 sm:p-6 lg:p-8">
          <EmptyState
            title="Job not found"
            description="This job doesn't exist or isn't one you can see."
            action={{ label: 'Back to jobs', onClick: () => navigate('/jobs') }}
          />
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout
      breadcrumbs={[
        { label: 'Jobs', to: '/jobs' },
        {
          label: isNew
            ? cloneFromId
              ? 'Repeat job'
              : 'New job'
            : currentJob?.job_number || 'Job',
        },
      ]}
    >
      <div className="mx-auto max-w-5xl p-4 sm:p-6 lg:p-8">
        <div className="flex items-center gap-4 mb-4">
          <Button
            variant="ghost"
            onClick={() => navigate('/jobs')}
            className="flex items-center gap-2"
          >
            <ArrowLeft className="h-4 w-4" />
            Back
          </Button>
          <h1 className="text-2xl font-bold tracking-tight text-neutral-900 md:text-3xl">
            {isNew
              ? cloneFromId
                ? 'Repeat job'
                : 'New job'
              : currentJob?.job_number || 'Edit job'}
          </h1>
        </div>

        {!isNew && currentJob && (
          <div className="mb-6">
            <StatusTransitionButton
              job={currentJob}
              onStatusChange={(updatedJob) => {
                setCurrentJob(updatedJob);
                setFormData((prev) => ({
                  ...prev,
                  status: updatedJob.status,
                }));
              }}
              disabled={isSaving}
            />
          </div>
        )}

        {!isNew && currentJob && seesMoney && (
          <JobInvoiceAction
            job={currentJob}
            canInvoice={can('invoices.create')}
            isInvoicing={invoicingJobId === currentJob.id}
            onInvoice={() => invoiceJob(currentJob.id)}
            onViewInvoice={(invoiceId) => navigate(`/invoices/${invoiceId}`)}
          />
        )}

        {(error || invoiceError) && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
            {error || invoiceError}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="bg-white rounded-2xl border border-neutral-200 p-6">
            <h2 className="text-lg font-semibold text-neutral-900 mb-4">
              Job Details
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
                  disabled={isLoadingCustomers || !canListCustomers}
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                >
                  <option value="">Select a customer</option>
                  {(canListCustomers
                    ? customers
                    : currentJob?.customer
                      ? [currentJob.customer]
                      : []
                  ).map((customer) => (
                    <option key={customer.id} value={customer.id}>
                      {customer.first_name} {customer.last_name}
                      {customer.company_name
                        ? ` (${customer.company_name})`
                        : ''}
                    </option>
                  ))}
                </select>
              </div>

              {canReadCustomers && (
                <div className="md:col-span-2">
                  <JobSitePicker
                    customerId={formData.customer_id}
                    value={formData.site_id}
                    onChange={(siteId) =>
                      setFormData((prev) => ({ ...prev, site_id: siteId }))
                    }
                  />
                </div>
              )}

              <div className="md:col-span-2">
                <label
                  htmlFor="title"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Job Title *
                </label>
                <input
                  type="text"
                  id="title"
                  name="title"
                  value={formData.title}
                  onChange={handleChange}
                  required
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>

              <div className="md:col-span-2">
                <label
                  htmlFor="description"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Description
                </label>
                <textarea
                  id="description"
                  name="description"
                  value={formData.description}
                  onChange={handleChange}
                  rows={3}
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>

              <div>
                <label
                  htmlFor="job_type"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Job Type *
                </label>
                <select
                  id="job_type"
                  name="job_type"
                  value={formData.job_type}
                  onChange={handleChange}
                  required
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                >
                  <option value={JobType.INSTALLATION}>Installation</option>
                  <option value={JobType.MAINTENANCE}>Maintenance</option>
                  <option value={JobType.REPAIR}>Repair</option>
                  <option value={JobType.INSPECTION}>Inspection</option>
                  <option value={JobType.EMERGENCY}>Emergency</option>
                </select>
              </div>

              <div>
                <label
                  htmlFor="priority"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Priority
                </label>
                <select
                  id="priority"
                  name="priority"
                  value={formData.priority}
                  onChange={handleChange}
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                >
                  <option value={JobPriority.LOW}>Low</option>
                  <option value={JobPriority.NORMAL}>Normal</option>
                  <option value={JobPriority.HIGH}>High</option>
                  <option value={JobPriority.URGENT}>Urgent</option>
                </select>
              </div>
            </div>
          </div>

          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
            <div className="bg-white rounded-2xl border border-neutral-200 p-6">
              <h2 className="text-lg font-semibold text-neutral-900 mb-4">
                Scheduling
              </h2>

              <div className="space-y-4">
                <div>
                  <label
                    htmlFor="scheduled_start_at"
                    className="block text-sm font-medium text-neutral-700 mb-1"
                  >
                    Scheduled Start
                  </label>
                  <input
                    type="datetime-local"
                    id="scheduled_start_at"
                    name="scheduled_start_at"
                    value={formData.scheduled_start_at}
                    onChange={handleChange}
                    min={scheduleMinDatetime}
                    max={scheduleMaxDatetime}
                    className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                  />
                </div>

                <div>
                  <label
                    htmlFor="estimated_duration"
                    className="block text-sm font-medium text-neutral-700 mb-1"
                  >
                    Est. Duration (minutes)
                  </label>
                  <input
                    type="number"
                    id="estimated_duration"
                    name="estimated_duration"
                    value={formData.estimated_duration}
                    onChange={handleChange}
                    min="0"
                    className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                  />
                </div>

                {seesMoney && (
                  <div>
                    <label
                      htmlFor="estimated_cost"
                      className="block text-sm font-medium text-neutral-700 mb-1"
                    >
                      Est. Cost ($)
                    </label>
                    <input
                      type="number"
                      id="estimated_cost"
                      name="estimated_cost"
                      value={formData.estimated_cost}
                      onChange={handleChange}
                      min="0"
                      step="0.01"
                      inputMode="decimal"
                      className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                    />
                    <p className="mt-1 text-xs text-neutral-500">
                      Extra work over the customer's not-to-exceed limit needs
                      the approver's OK before it's scheduled.
                    </p>
                  </div>
                )}
              </div>

              {canCheckConflicts && startTime && endTime ? (
                <div className="mt-4">
                  <ConflictChecker
                    jobId={conflictJobId}
                    startTime={startTime}
                    endTime={endTime}
                    assignedTechIds={assignedTechIds}
                    onConflictDetected={handleConflictDetected}
                    onConflictResolved={handleConflictResolved}
                  />
                </div>
              ) : startTime && endTime && assignedTechIds.length === 0 ? (
                <p className="mt-4 text-sm text-neutral-500">
                  Assign members to check for scheduling conflicts.
                </p>
              ) : null}
            </div>

            {isNew ? (
              <PendingAssignmentsSection
                assignments={pendingAssignments}
                onAssignmentsChange={setPendingAssignments}
                startTime={startTime}
                endTime={endTime}
                jobId={conflictJobId}
              />
            ) : (
              id && (
                <JobAssignmentsSection
                  jobId={id}
                  assignments={assignments}
                  onAssignmentChange={handleAssignmentChange}
                  startTime={startTime}
                  endTime={endTime}
                  readOnly={!canAssign}
                />
              )
            )}
          </div>

          <div className="bg-white rounded-2xl border border-neutral-200 p-6">
            <h2 className="text-lg font-semibold text-neutral-900 mb-4">
              Job Instructions
            </h2>

            <div className="space-y-4">
              <div>
                <label
                  htmlFor="special_instructions"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Special Instructions
                </label>
                <textarea
                  id="special_instructions"
                  name="special_instructions"
                  value={formData.special_instructions}
                  onChange={handleChange}
                  rows={3}
                  placeholder="Any special instructions for technicians..."
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>

              <div>
                <label
                  htmlFor="required_materials"
                  className="block text-sm font-medium text-neutral-700 mb-1"
                >
                  Required Materials
                </label>
                <textarea
                  id="required_materials"
                  name="required_materials"
                  value={formData.required_materials}
                  onChange={handleChange}
                  rows={3}
                  placeholder="List any materials needed for this job..."
                  className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
                />
              </div>
            </div>
          </div>

          <div className="bg-white rounded-2xl border border-neutral-200 p-6">
            <h2 className="text-lg font-semibold text-neutral-900 mb-4">
              Customer Notes
            </h2>
            <textarea
              id="customer_notes"
              name="customer_notes"
              value={formData.customer_notes}
              onChange={handleChange}
              rows={4}
              placeholder="Notes visible to customers (e.g., appointment details, preparation instructions)..."
              className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>

          <div className="bg-white rounded-2xl border border-neutral-200 p-6">
            <h2 className="text-lg font-semibold text-neutral-900 mb-4">
              Internal Notes
            </h2>
            <textarea
              id="internal_notes"
              name="internal_notes"
              value={formData.internal_notes}
              onChange={handleChange}
              rows={4}
              placeholder="Add any internal notes (not visible to customers)..."
              className="w-full px-3 py-2 border border-neutral-300 rounded-lg focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
            />
          </div>

          <div className="flex items-center justify-between pt-4">
            <div>
              {!isNew && canDeleteJob && (
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
                  Delete Job
                </Button>
              )}
            </div>
            <div className="flex items-center gap-3">
              <Button
                type="button"
                variant="secondary"
                onClick={() => navigate('/jobs')}
              >
                Cancel
              </Button>
              <Button
                type="submit"
                variant="primary"
                disabled={isSaving || hasConflicts || isCheckingConflicts}
                title={
                  hasConflicts
                    ? 'Resolve scheduling conflicts before saving'
                    : isCheckingConflicts
                      ? 'Checking for scheduling conflicts...'
                      : undefined
                }
                className="flex items-center gap-2"
              >
                {isSaving ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Save className="h-4 w-4" />
                )}
                {isNew ? 'Create Job' : 'Save Changes'}
              </Button>
            </div>
          </div>
        </form>

        {/* Outside the form: the viewer has its own form (T174). */}
        {!isNew && currentJob && (
          <div className="mt-6">
            <JobPhotosSection
              jobId={currentJob.id}
              currentUserId={currentUserId}
              canManageAll={isOffice}
              timeZone={timeZone}
            />
          </div>
        )}

        {/* Outside the form: the approval dialog has its own form. */}
        {!isNew && currentJob && isOffice && (
          <div className="mt-6">
            <JobApprovalsSection job={currentJob} />
          </div>
        )}
      </div>
    </DashboardLayout>
  );
}

/**
 * Billing for a finished job (T029): a completed job offers "Generate
 * invoice" (the server builds the draft from the job's time, materials and
 * notes, T030); an invoiced or paid job links to its invoice. Agreement
 * visits are billed on the customer's consolidated invoice instead.
 */
function JobInvoiceAction({
  job,
  canInvoice,
  isInvoicing,
  onInvoice,
  onViewInvoice,
}: {
  job: Job;
  canInvoice: boolean;
  isInvoicing: boolean;
  onInvoice: () => void;
  onViewInvoice: (invoiceId: string) => void;
}) {
  const completed = job.status === JobStatus.COMPLETED;
  const billed =
    job.status === JobStatus.INVOICED || job.status === JobStatus.PAID;

  if (billed && job.invoice_id) {
    const invoiceId = job.invoice_id;
    return (
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-neutral-200 bg-white p-4">
        <p className="text-sm text-neutral-600">
          {job.status === JobStatus.PAID
            ? 'This job has been invoiced and paid.'
            : 'This job has been invoiced.'}
        </p>
        <Button
          type="button"
          variant="secondary"
          onClick={() => onViewInvoice(invoiceId)}
          className="flex items-center gap-2"
        >
          <FileText className="h-4 w-4" />
          View invoice
        </Button>
      </div>
    );
  }

  if (!completed) return null;

  if (job.agreement_id) {
    return (
      <div className="mb-6 rounded-2xl border border-neutral-200 bg-neutral-50 p-4 text-sm text-neutral-600">
        This visit is part of a service agreement. It is billed on the
        customer&apos;s consolidated invoice.
      </div>
    );
  }

  if (!canInvoice) return null;

  return (
    <div className="mb-6 flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-primary-200 bg-primary-50 p-4">
      <div>
        <p className="font-medium text-neutral-900">Ready to invoice</p>
        <p className="text-sm text-neutral-600">
          Creates a draft from the time on site, billable materials and notes.
          You can review it before sending.
        </p>
      </div>
      <Button
        type="button"
        variant="primary"
        onClick={onInvoice}
        disabled={isInvoicing}
        className="flex items-center gap-2"
      >
        {isInvoicing ? (
          <Loader2 className="h-4 w-4 animate-spin" />
        ) : (
          <Receipt className="h-4 w-4" />
        )}
        Generate invoice
      </Button>
    </div>
  );
}

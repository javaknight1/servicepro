import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { DashboardLayout } from '@components/layout';
import { Button } from '@components/shared';
import {
  JobAssignmentsSection,
  PendingAssignmentsSection,
  StatusTransitionButton,
} from '@components/jobs';
import {
  jobService,
  JobStatus,
  JobPriority,
  JobType,
  JobAssignment,
  CreateJobAssignment,
  Job,
} from '@services/jobService';
import { customerService, Customer } from '@services/customerService';
import { ArrowLeft, Save, Trash2, Loader2 } from 'lucide-react';

interface PendingAssignment extends CreateJobAssignment {
  user_name: string;
}

interface JobFormData {
  customer_id: string;
  title: string;
  description: string;
  job_type: JobType;
  status: JobStatus;
  priority: JobPriority;
  scheduled_start_at: string;
  estimated_duration: string;
  internal_notes: string;
  customer_notes: string;
  special_instructions: string;
  required_materials: string;
}

const initialFormData: JobFormData = {
  customer_id: '',
  title: '',
  description: '',
  job_type: JobType.MAINTENANCE,
  status: JobStatus.NEW,
  priority: JobPriority.NORMAL,
  scheduled_start_at: '',
  estimated_duration: '',
  internal_notes: '',
  customer_notes: '',
  special_instructions: '',
  required_materials: '',
};

export function JobDetailPage() {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const isNew = !id || id === 'new';

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

  useEffect(() => {
    loadCustomers();
    if (!isNew && id) {
      loadJob(id);
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

  const loadJob = async (jobId: string) => {
    setIsLoading(true);
    setError(null);
    try {
      const job = await jobService.getJob(jobId);
      setCurrentJob(job);
      setFormData({
        customer_id: job.customer_id || '',
        title: job.title || '',
        description: job.description || '',
        job_type: job.job_type || JobType.MAINTENANCE,
        status: job.status || JobStatus.NEW,
        priority: job.priority || JobPriority.NORMAL,
        scheduled_start_at: job.scheduled_start_at
          ? job.scheduled_start_at.slice(0, 16)
          : '',
        estimated_duration: job.estimated_duration?.toString() || '',
        internal_notes: job.internal_notes || '',
        customer_notes: job.customer_notes || '',
        special_instructions: job.special_instructions || '',
        required_materials: job.required_materials || '',
      });
      setAssignments(job.assignments || []);
    } catch (err) {
      console.error('Failed to load job:', err);
      setError('Failed to load job details');
    } finally {
      setIsLoading(false);
    }
  };

  const handleAssignmentChange = () => {
    if (id) {
      loadJob(id);
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

    try {
      if (isNew) {
        // Create new job with assignments
        await jobService.createJob({
          customer_id: formData.customer_id,
          title: formData.title,
          description: formData.description || undefined,
          job_type: formData.job_type,
          priority: formData.priority,
          scheduled_start_at: formData.scheduled_start_at
            ? new Date(formData.scheduled_start_at).toISOString()
            : undefined,
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
      } else if (id) {
        // Update existing job
        await jobService.updateJob(id, {
          title: formData.title,
          description: formData.description || undefined,
          job_type: formData.job_type,
          status: formData.status,
          priority: formData.priority,
          scheduled_start_at: formData.scheduled_start_at
            ? new Date(formData.scheduled_start_at).toISOString()
            : undefined,
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
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (err: any) {
      console.error('Failed to save job:', err);
      setError(
        err?.response?.data?.message ||
          err?.response?.data?.error ||
          'Failed to save job'
      );
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
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (err: any) {
      console.error('Failed to delete job:', err);
      setError(err?.response?.data?.message || 'Failed to delete job');
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
        <div className="flex items-center gap-4 mb-4">
          <Button
            variant="ghost"
            onClick={() => navigate('/jobs')}
            className="flex items-center gap-2"
          >
            <ArrowLeft className="h-4 w-4" />
            Back
          </Button>
          <h1 className="text-2xl font-bold text-neutral-900">
            {isNew ? 'Create Job' : 'Edit Job'}
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

        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-6">
          <div className="bg-white rounded-lg border border-neutral-200 p-6">
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

              {!isNew && (
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
                    <option value={JobStatus.NEW}>New</option>
                    <option value={JobStatus.SCHEDULED}>Scheduled</option>
                    <option value={JobStatus.EN_ROUTE}>En Route</option>
                    <option value={JobStatus.IN_PROGRESS}>In Progress</option>
                    <option value={JobStatus.ON_HOLD}>On Hold</option>
                    <option value={JobStatus.COMPLETED}>Completed</option>
                    <option value={JobStatus.INVOICED}>Invoiced</option>
                    <option value={JobStatus.PAID}>Paid</option>
                    <option value={JobStatus.CANCELLED}>Cancelled</option>
                  </select>
                </div>
              )}
            </div>
          </div>

          <div className="bg-white rounded-lg border border-neutral-200 p-6">
            <h2 className="text-lg font-semibold text-neutral-900 mb-4">
              Scheduling
            </h2>

            <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
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
            </div>
          </div>

          <div className="bg-white rounded-lg border border-neutral-200 p-6">
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

          <div className="bg-white rounded-lg border border-neutral-200 p-6">
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

          <div className="bg-white rounded-lg border border-neutral-200 p-6">
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

          {isNew ? (
            <PendingAssignmentsSection
              assignments={pendingAssignments}
              onAssignmentsChange={setPendingAssignments}
            />
          ) : (
            id && (
              <JobAssignmentsSection
                jobId={id}
                assignments={assignments}
                onAssignmentChange={handleAssignmentChange}
              />
            )
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
                disabled={isSaving}
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
      </div>
    </DashboardLayout>
  );
}

export enum JobStatus {
  NEW = 'new',
  SCHEDULED = 'scheduled',
  EN_ROUTE = 'en_route',
  IN_PROGRESS = 'in_progress',
  ON_HOLD = 'on_hold',
  COMPLETED = 'completed',
  INVOICED = 'invoiced',
  PAID = 'paid',
  CANCELLED = 'cancelled',
}

export enum JobPriority {
  LOW = 'low',
  NORMAL = 'normal',
  HIGH = 'high',
  URGENT = 'urgent',
}

export enum JobType {
  INSTALLATION = 'installation',
  MAINTENANCE = 'maintenance',
  REPAIR = 'repair',
  INSPECTION = 'inspection',
  EMERGENCY = 'emergency',
}

export interface ServiceAddress {
  street: string;
  city: string;
  state: string;
  zip: string;
}

export interface JobAssignment {
  id: string;
  job_id: string;
  user_id: string;
  user_name: string;
  role: string;
  assigned_at: string;
  unassigned_at?: string;
  hours_worked: number;
  notes?: string;
}

export interface JobMaterial {
  id: string;
  job_id: string;
  name: string;
  description?: string;
  sku?: string;
  quantity: number;
  unit: string;
  unit_cost: number;
  total_cost: number;
  billable: boolean;
}

export interface JobNote {
  id: string;
  job_id: string;
  user_id: string;
  user_name: string;
  note: string;
  is_internal: boolean;
  created_at: string;
}

export interface Job {
  id: string;
  job_number: string;
  customer_id: string;
  customer?: {
    id: string;
    email: string;
    first_name: string;
    last_name: string;
    company_name?: string;
    phone?: string;
  };
  title: string;
  description?: string;
  job_type: JobType;
  status: JobStatus;
  priority: JobPriority;
  scheduled_start_at?: string;
  scheduled_end_at?: string;
  actual_start_at?: string;
  actual_end_at?: string;
  estimated_duration?: number;
  actual_duration?: number;
  service_address?: ServiceAddress;
  estimated_cost?: number;
  actual_cost?: number;
  tax_amount?: number;
  total_amount?: number;
  assignments?: JobAssignment[];
  materials?: JobMaterial[];
  notes?: JobNote[];
  internal_notes?: string;
  customer_notes?: string;
  completion_notes?: string;
  special_instructions?: string;
  required_materials?: string;
  next_status?: JobStatus;
  requires_follow_up?: boolean;
  follow_up_date?: string;
  warnings?: string[];
  created_at: string;
  updated_at: string;
}

export interface JobFilters {
  customer_id?: string;
  status?: JobStatus;
  priority?: JobPriority;
  assigned_to?: string;
  search?: string;
  page?: number;
  page_size?: number;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

export interface JobListResponse {
  jobs: Job[];
  total: number;
  page: number;
  page_size: number;
}

export interface JobStats {
  total_jobs: number;
  pending_jobs: number;
  scheduled_jobs: number;
  in_progress_jobs: number;
  completed_jobs: number;
  cancelled_jobs: number;
}

export interface JobStatusTransition {
  id: string;
  job_id: string;
  job_number: string;
  from_status: JobStatus;
  to_status: JobStatus;
  reason?: string;
  notes?: string;
  changed_by: string;
  changed_by_name: string;
  transitioned_at: string;
}

export interface StatusHistoryResponse {
  transitions: JobStatusTransition[];
  total: number;
}

export interface TransitionStatusRequest {
  to_status: JobStatus;
  reason?: string;
  notes?: string;
}

export const getJobStatusLabel = (status: JobStatus): string => {
  const labels: Record<JobStatus, string> = {
    [JobStatus.NEW]: 'New',
    [JobStatus.SCHEDULED]: 'Scheduled',
    [JobStatus.EN_ROUTE]: 'En Route',
    [JobStatus.IN_PROGRESS]: 'In Progress',
    [JobStatus.ON_HOLD]: 'On Hold',
    [JobStatus.COMPLETED]: 'Completed',
    [JobStatus.INVOICED]: 'Invoiced',
    [JobStatus.PAID]: 'Paid',
    [JobStatus.CANCELLED]: 'Cancelled',
  };
  return labels[status] || status;
};

export const getNextStatusLabel = (
  nextStatus: JobStatus | undefined
): string => {
  if (!nextStatus) return '';
  const labels: Partial<Record<JobStatus, string>> = {
    [JobStatus.SCHEDULED]: 'Set Scheduled',
    [JobStatus.EN_ROUTE]: 'Set En Route',
    [JobStatus.IN_PROGRESS]: 'Start Job',
    [JobStatus.COMPLETED]: 'Complete Job',
    [JobStatus.INVOICED]: 'Mark Invoiced',
    [JobStatus.PAID]: 'Mark Paid',
  };
  return labels[nextStatus] || `Set ${getJobStatusLabel(nextStatus)}`;
};

export const getJobPriorityLabel = (priority: JobPriority): string => {
  const labels: Record<JobPriority, string> = {
    [JobPriority.LOW]: 'Low',
    [JobPriority.NORMAL]: 'Normal',
    [JobPriority.HIGH]: 'High',
    [JobPriority.URGENT]: 'Urgent',
  };
  return labels[priority] || priority;
};

export const getJobTypeLabel = (jobType: JobType): string => {
  const labels: Record<JobType, string> = {
    [JobType.INSTALLATION]: 'Installation',
    [JobType.MAINTENANCE]: 'Maintenance',
    [JobType.REPAIR]: 'Repair',
    [JobType.INSPECTION]: 'Inspection',
    [JobType.EMERGENCY]: 'Emergency',
  };
  return labels[jobType] || jobType;
};

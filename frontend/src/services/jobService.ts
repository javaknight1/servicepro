import api from './api';

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

export interface CreateJobAssignment {
  user_id: string;
  role: string;
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

export interface CreateJobRequest {
  customer_id: string;
  title: string;
  description?: string;
  job_type: JobType;
  priority?: JobPriority;
  scheduled_start_at?: string;
  scheduled_end_at?: string;
  estimated_duration?: number;
  service_address?: ServiceAddress;
  estimated_cost?: number;
  internal_notes?: string;
  customer_notes?: string;
  special_instructions?: string;
  required_materials?: string;
  assignments?: CreateJobAssignment[];
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

class JobService {
  private basePath = '/v1/jobs';

  async getJobs(filters: JobFilters = {}): Promise<JobListResponse> {
    const params = new URLSearchParams();

    if (filters.customer_id) params.append('customer_id', filters.customer_id);
    if (filters.status) params.append('status', filters.status);
    if (filters.priority) params.append('priority', filters.priority);
    if (filters.assigned_to) params.append('assigned_to', filters.assigned_to);
    if (filters.search) params.append('search', filters.search);
    if (filters.page) params.append('page', filters.page.toString());
    if (filters.page_size)
      params.append('page_size', filters.page_size.toString());
    if (filters.sort_by) params.append('sort_by', filters.sort_by);
    if (filters.sort_order) params.append('sort_order', filters.sort_order);

    const response = await api.get<JobListResponse>(this.basePath, { params });
    return response.data;
  }

  async getJob(id: string): Promise<Job> {
    const response = await api.get<Job>(`${this.basePath}/${id}`);
    return response.data;
  }

  async getJobByNumber(jobNumber: string): Promise<Job> {
    const response = await api.get<Job>(`${this.basePath}/number/${jobNumber}`);
    return response.data;
  }

  async createJob(data: CreateJobRequest): Promise<Job> {
    const response = await api.post<Job>(this.basePath, data);
    return response.data;
  }

  async updateJob(id: string, data: Partial<Job>): Promise<Job> {
    const response = await api.put<Job>(`${this.basePath}/${id}`, data);
    return response.data;
  }

  async deleteJob(id: string): Promise<void> {
    await api.delete(`${this.basePath}/${id}`);
  }

  async startJob(id: string): Promise<Job> {
    const response = await api.post<Job>(`${this.basePath}/${id}/start`);
    return response.data;
  }

  async completeJob(id: string, notes?: string): Promise<Job> {
    const response = await api.post<Job>(`${this.basePath}/${id}/complete`, {
      notes,
    });
    return response.data;
  }

  async cancelJob(id: string, reason?: string): Promise<Job> {
    const response = await api.post<Job>(`${this.basePath}/${id}/cancel`, {
      reason,
    });
    return response.data;
  }

  async getStats(): Promise<JobStats> {
    const response = await api.get<JobStats>(`${this.basePath}/stats`);
    return response.data;
  }

  async getCustomerJobs(customerId: string): Promise<Job[]> {
    const response = await api.get<Job[]>(`/v1/customers/${customerId}/jobs`);
    return response.data;
  }

  async assignMember(
    jobId: string,
    userId: string,
    role: string = 'technician'
  ): Promise<void> {
    await api.post(`${this.basePath}/${jobId}/assign`, {
      technician_id: userId,
      role,
    });
  }

  async unassignMember(jobId: string, userId: string): Promise<void> {
    await api.delete(`${this.basePath}/${jobId}/assign/${userId}`);
  }

  async transitionStatus(
    jobId: string,
    toStatus: JobStatus,
    reason?: string,
    notes?: string
  ): Promise<Job> {
    const response = await api.post<Job>(
      `${this.basePath}/${jobId}/transition`,
      {
        to_status: toStatus,
        reason,
        notes,
      }
    );
    return response.data;
  }

  async getStatusHistory(
    jobId: string,
    limit: number = 20,
    offset: number = 0,
    sort: 'asc' | 'desc' = 'desc'
  ): Promise<StatusHistoryResponse> {
    const response = await api.get<StatusHistoryResponse>(
      `${this.basePath}/${jobId}/status-history`,
      {
        params: { limit, offset, sort },
      }
    );
    return response.data;
  }

  /**
   * Get jobs scheduled within a date range (for calendar view)
   * @param start - Start date of the range
   * @param end - End date of the range
   * @returns Array of jobs within the date range
   */
  async getScheduledJobs(start: Date, end: Date): Promise<Job[]> {
    const response = await api.get<Job[]>(`${this.basePath}/scheduled`, {
      params: {
        start: start.toISOString(),
        end: end.toISOString(),
      },
    });
    return response.data;
  }
}

export const jobService = new JobService();

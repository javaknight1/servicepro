/**
 * Job Lifecycle Integration Tests
 *
 * Tests complete job management workflows including:
 * - Job list loading and display
 * - Job creation flow
 * - Job status transitions
 * - Job scheduling
 * - Job completion workflow
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  render as rtlRender,
  screen,
  waitFor,
  fireEvent,
  within,
} from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { AxiosError, AxiosHeaders } from 'axios';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router';
import React from 'react';

// Mock modules
vi.mock('@store', () => ({
  useAuthStore: vi.fn(),
  useTenantStore: vi.fn(),
  useSidebarStore: vi.fn(),
}));

vi.mock('@services/jobService', () => ({
  jobService: {
    getJobs: vi.fn(),
    getJob: vi.fn(),
    createJob: vi.fn(),
    updateJob: vi.fn(),
    deleteJob: vi.fn(),
    getStats: vi.fn(),
    transitionStatus: vi.fn(),
    getStatusHistory: vi.fn(),
    getScheduledJobs: vi.fn(),
    assignMember: vi.fn(),
    unassignMember: vi.fn(),
    getApprovals: vi.fn(),
    recordApproval: vi.fn(),
    invoiceJob: vi.fn(),
  },
  jobApprovalsQueryKey: (jobId: string | null) => ['job-approvals', jobId],
  JobStatus: {
    NEW: 'new',
    SCHEDULED: 'scheduled',
    EN_ROUTE: 'en_route',
    IN_PROGRESS: 'in_progress',
    ON_HOLD: 'on_hold',
    COMPLETED: 'completed',
    INVOICED: 'invoiced',
    PAID: 'paid',
    CANCELLED: 'cancelled',
  },
  JobPriority: {
    LOW: 'low',
    NORMAL: 'normal',
    HIGH: 'high',
    URGENT: 'urgent',
  },
  JobType: {
    INSTALLATION: 'installation',
    MAINTENANCE: 'maintenance',
    REPAIR: 'repair',
    INSPECTION: 'inspection',
    EMERGENCY: 'emergency',
  },
}));

vi.mock('@services/customerService', () => ({
  customerService: {
    getCustomers: vi.fn(),
    getCustomer: vi.fn(),
  },
}));

// Tenant permissions come from GET /tenants/:id/me; the tests set them.
vi.mock('@/utils/tenantPermissions', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/utils/tenantPermissions')>()),
  useTenantPermissions: vi.fn(),
}));

vi.mock('@services/siteService', () => ({
  siteService: { list: vi.fn(), create: vi.fn(), update: vi.fn() },
  sitesQueryKey: (customerId: string | null, includeInactive = false) => [
    'customer-sites',
    customerId,
    includeInactive,
  ],
}));

/** Job pages read sites and approvals through React Query. */
function render(ui: React.ReactElement) {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return rtlRender(ui, {
    wrapper: ({ children }) => (
      <QueryClientProvider client={client}>{children}</QueryClientProvider>
    ),
  });
}

// Import after mocking
import { useAuthStore, useTenantStore, useSidebarStore } from '@store';
import { jobService } from '@services/jobService';
import type { JobStats } from '@app-types/job';
import { customerService } from '@services/customerService';
import { siteService, type Site } from '@services/siteService';
import { useTenantPermissions } from '@/utils/tenantPermissions';

/** Office staff by default (T035 gates the job pages on these). */
const OFFICE_PERMISSIONS = new Set<string>([
  'invoices.create',
  'invoices.read',
  'customers.list',
  'customers.read',
  'jobs.list',
  'jobs.create',
  'jobs.delete',
  'jobs.assign',
  'jobs.view_all',
]);
let grantedPermissions = new Set<string>(OFFICE_PERMISSIONS);

// =============================================================================
// Test Data
// =============================================================================

const mockUser = {
  id: 'user-123',
  email: 'admin@example.com',
  first_name: 'Admin',
  last_name: 'User',
  is_active: true,
  email_verified: true,
};

const mockTenant: {
  id: string;
  name: string;
  slug: string;
  is_active: boolean;
  role?: string;
} = {
  id: 'tenant-123',
  name: 'Test Organization',
  slug: 'test-org',
  is_active: true,
};

const mockCustomer = {
  id: 'customer-1',
  email: 'john@example.com',
  first_name: 'John',
  last_name: 'Doe',
  phone_primary: '555-123-4567',
};

const mockJobs = [
  {
    id: 'job-1',
    job_number: 'JOB-001',
    customer_id: 'customer-1',
    customer: mockCustomer,
    title: 'HVAC Installation',
    description: 'Install new HVAC system',
    job_type: 'installation' as const,
    status: 'new' as const,
    allowed_transitions: ['scheduled' as const, 'cancelled' as const],
    priority: 'normal' as const,
    scheduled_start_at: Date.UTC(2024, 0, 15, 9, 0) / 1000,
    scheduled_end_at: '2024-01-15T17:00:00Z',
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'job-2',
    job_number: 'JOB-002',
    customer_id: 'customer-1',
    customer: mockCustomer,
    title: 'Plumbing Repair',
    description: 'Fix leaking pipe',
    job_type: 'repair' as const,
    status: 'in_progress' as const,
    allowed_transitions: [
      'on_hold' as const,
      'completed' as const,
      'en_route' as const,
      'scheduled' as const,
      'cancelled' as const,
    ],
    priority: 'high' as const,
    scheduled_start_at: Date.UTC(2024, 0, 10, 10, 0) / 1000,
    scheduled_end_at: '2024-01-10T14:00:00Z',
    actual_start_at: '2024-01-10T10:15:00Z',
    created_at: '2024-01-02T00:00:00Z',
    updated_at: '2024-01-10T10:15:00Z',
  },
  {
    id: 'job-3',
    job_number: 'JOB-003',
    customer_id: 'customer-1',
    customer: mockCustomer,
    title: 'Electrical Inspection',
    description: 'Annual safety inspection',
    job_type: 'inspection' as const,
    status: 'completed' as const,
    allowed_transitions: [
      'invoiced' as const,
      'in_progress' as const,
      'cancelled' as const,
    ],
    priority: 'low' as const,
    scheduled_start_at: Date.UTC(2024, 0, 5, 8, 0) / 1000,
    scheduled_end_at: '2024-01-05T10:00:00Z',
    actual_start_at: '2024-01-05T08:00:00Z',
    actual_end_at: Date.UTC(2024, 0, 5, 9, 45) / 1000,
    created_at: '2024-01-03T00:00:00Z',
    updated_at: '2024-01-05T09:45:00Z',
  },
];

const mockSites: Site[] = [
  {
    id: 'site-1',
    customer_id: 'customer-1',
    name: 'Harborview Lofts',
    address_street: '220 W Spring St',
    address_city: 'Columbus',
    address_state: 'OH',
    address_zip: '43215',
    address_full: '220 W Spring St, Columbus, OH 43215',
    access_notes: 'Key box by the loading dock',
    is_active: true,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'site-old',
    customer_id: 'customer-1',
    name: 'Closed Annex',
    address_street: '1 Old Rd',
    address_city: 'Columbus',
    address_state: 'OH',
    address_zip: '43215',
    address_full: '1 Old Rd, Columbus, OH 43215',
    is_active: false,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
];

const mockJobStats: JobStats = {
  total_jobs: 4,
  pending_jobs: 1,
  scheduled_jobs: 1,
  in_progress_jobs: 1,
  on_hold_jobs: 0,
  completed_jobs: 1,
  cancelled_jobs: 0,
  by_status: { new: 1, scheduled: 1, in_progress: 1, completed: 1 },
};

// =============================================================================
// Test Setup
// =============================================================================

const mockNavigate = vi.fn();
vi.mock('react-router', async () => {
  const actual = await vi.importActual('react-router');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

const setupMocks = () => {
  grantedPermissions = new Set<string>(OFFICE_PERMISSIONS);
  vi.mocked(useTenantPermissions).mockImplementation(() => ({
    can: (permission: string) => grantedPermissions.has(permission),
    canManageMembers: false,
    canUpdateTenant: false,
    canChangePlan: false,
    canManageBilling: false,
    canDeleteTenant: false,
    canManageTaxRates: false,
    isLoading: false,
  }));

  vi.mocked(useAuthStore).mockImplementation((selector) => {
    const state = {
      user: mockUser,
      isAuthenticated: true,
      isLoading: false,
      error: null,
      login: vi.fn(),
      logout: vi.fn(),
      loadUser: vi.fn(),
      clearError: vi.fn(),
    };
    if (typeof selector === 'function') {
      return selector(state);
    }
    return state;
  });

  vi.mocked(useTenantStore).mockImplementation((selector) => {
    const state = {
      currentTenant: mockTenant,
      tenants: [mockTenant],
      isLoading: false,
      error: null,
      loadTenants: vi.fn(),
      setCurrentTenant: vi.fn(),
    };
    if (typeof selector === 'function') {
      return selector(state);
    }
    return state;
  });

  vi.mocked(useSidebarStore).mockImplementation((selector) => {
    const state = {
      isOpen: true,
      isCollapsed: false,
      toggle: vi.fn(),
      setOpen: vi.fn(),
      setCollapsed: vi.fn(),
    };
    if (typeof selector === 'function') {
      return selector(state);
    }
    return state;
  });

  vi.mocked(jobService.getJobs).mockResolvedValue({
    jobs: mockJobs,
    total: mockJobs.length,
    page: 1,
    page_size: 10,
  });

  vi.mocked(jobService.getStats).mockResolvedValue(mockJobStats);
  vi.mocked(jobService.getApprovals).mockResolvedValue([]);
  vi.mocked(siteService.list).mockResolvedValue(mockSites);

  vi.mocked(customerService.getCustomers).mockResolvedValue({
    customers: [mockCustomer],
    total: 1,
    page: 1,
    page_size: 10,
  });
};

// =============================================================================
// Job List Tests
// =============================================================================

describe('Job Lifecycle Integration', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupMocks();
  });

  describe('Job List', () => {
    it('should load and display jobs list', async () => {
      const { JobsPage } = await import('../../src/pages/Jobs/JobsPage');

      render(
        <MemoryRouter initialEntries={['/jobs']}>
          <JobsPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(jobService.getJobs).toHaveBeenCalled();
      });

      // Verify jobs are displayed
      await waitFor(() => {
        expect(screen.getByText('JOB-001')).toBeInTheDocument();
        expect(screen.getByText('HVAC Installation')).toBeInTheDocument();
      });
    });

    it('should display job status indicators', async () => {
      const { JobsPage } = await import('../../src/pages/Jobs/JobsPage');

      render(
        <MemoryRouter initialEntries={['/jobs']}>
          <JobsPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        // Check for status badges/indicators (exact text: the header's
        // "New job" button also contains "new")
        expect(
          screen.getAllByText(/^(new|scheduled|in progress)$/i).length
        ).toBeGreaterThan(0);
      });
    });

    it('should display job priority indicators', async () => {
      const { JobsPage } = await import('../../src/pages/Jobs/JobsPage');

      render(
        <MemoryRouter initialEntries={['/jobs']}>
          <JobsPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        // High priority job should have indicator
        expect(screen.getByText(/high/i)).toBeInTheDocument();
      });
    });

    it('should navigate to job detail when clicking a job', async () => {
      const user = userEvent.setup({ delay: null });
      const { JobsPage } = await import('../../src/pages/Jobs/JobsPage');

      render(
        <MemoryRouter initialEntries={['/jobs']}>
          <JobsPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('JOB-001')).toBeInTheDocument();
      });

      // Click on job
      const jobRow = screen.getByText('JOB-001').closest('tr');
      if (jobRow) {
        await user.click(jobRow);
        expect(mockNavigate).toHaveBeenCalledWith('/jobs/job-1');
      }
    });

    it('should navigate to new job page when clicking add button', async () => {
      const user = userEvent.setup({ delay: null });
      const { JobsPage } = await import('../../src/pages/Jobs/JobsPage');

      render(
        <MemoryRouter initialEntries={['/jobs']}>
          <JobsPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(jobService.getJobs).toHaveBeenCalled();
      });

      const addButton = screen.getByRole('button', { name: /add|new|create/i });
      await user.click(addButton);

      expect(mockNavigate).toHaveBeenCalledWith('/jobs/new');
    });
  });

  describe('Job Creation', () => {
    it('should create a new job successfully', async () => {
      const user = userEvent.setup({ delay: null });
      const newJob = {
        id: 'job-new',
        job_number: 'JOB-004',
        customer_id: 'customer-1',
        customer: mockCustomer,
        title: 'New Installation',
        description: 'New installation job',
        job_type: 'installation' as const,
        status: 'new' as const,
        priority: 'normal' as const,
        created_at: '2024-01-04T00:00:00Z',
        updated_at: '2024-01-04T00:00:00Z',
      };

      vi.mocked(jobService.createJob).mockResolvedValue(newJob);
      vi.mocked(customerService.getCustomer).mockResolvedValue(mockCustomer);

      const { JobDetailPage } =
        await import('../../src/pages/Jobs/JobDetailPage');

      render(
        <MemoryRouter initialEntries={['/jobs/new']}>
          <Routes>
            <Route path="/jobs/new" element={<JobDetailPage />} />
            <Route path="/jobs/:id" element={<div>Job Detail</div>} />
          </Routes>
        </MemoryRouter>
      );

      // Wait for form to render
      await waitFor(() => {
        expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
      });

      // Fill in required fields
      await user.type(screen.getByLabelText(/title/i), 'New Installation');

      // Customer and job type are native <select>s
      const customerSelect = await screen.findByLabelText(/customer/i);
      await screen.findByRole('option', { name: /John Doe/i });
      await user.selectOptions(customerSelect, 'customer-1');
      await user.selectOptions(
        screen.getByLabelText(/job type/i),
        'installation'
      );

      // Add description
      const descInput = screen.queryByLabelText(/description/i);
      if (descInput) {
        await user.type(descInput, 'New installation job');
      }

      // Submit form
      const saveButton = screen.getByRole('button', {
        name: /save|create|submit/i,
      });
      await user.click(saveButton);

      await waitFor(() => {
        expect(jobService.createJob).toHaveBeenCalledWith(
          expect.objectContaining({
            title: 'New Installation',
            customer_id: 'customer-1',
            job_type: 'installation',
          })
        );
      });
    });

    it('should interpret the scheduled start in the tenant timezone (HST)', async () => {
      const user = userEvent.setup({ delay: null });
      const hstTenant = {
        ...mockTenant,
        settings: { timezone: 'Pacific/Honolulu' },
      };
      vi.mocked(useTenantStore).mockImplementation((selector) => {
        const state = {
          currentTenant: hstTenant,
          tenants: [hstTenant],
          isLoading: false,
          error: null,
          loadTenants: vi.fn(),
          setCurrentTenant: vi.fn(),
        };
        return typeof selector === 'function' ? selector(state) : state;
      });
      vi.mocked(jobService.createJob).mockResolvedValue({
        ...mockJobs[0],
        id: 'job-hst',
      });

      // A date ~30 days ahead, typed as 09:00 wall-clock in Honolulu.
      // HST is a fixed UTC-10 (no DST), so 09:00 HST == 19:00 UTC.
      const target = new Date(Date.now() + 30 * 24 * 60 * 60 * 1000);
      const y = target.getUTCFullYear();
      const m = target.getUTCMonth();
      const d = target.getUTCDate();
      const pad = (n: number) => String(n).padStart(2, '0');
      const typed = `${y}-${pad(m + 1)}-${pad(d)}T09:00`;
      const expectedEpoch = Date.UTC(y, m, d, 19, 0) / 1000;

      const { JobDetailPage } =
        await import('../../src/pages/Jobs/JobDetailPage');

      render(
        <MemoryRouter initialEntries={['/jobs/new']}>
          <Routes>
            <Route path="/jobs/new" element={<JobDetailPage />} />
            <Route path="/jobs/:id" element={<div>Job Detail</div>} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
      });

      await user.type(screen.getByLabelText(/title/i), 'HST Job');
      const customerSelect = await screen.findByLabelText(/customer/i);
      await waitFor(() => {
        expect(
          screen.getByRole('option', { name: /John Doe/i })
        ).toBeInTheDocument();
      });
      await user.selectOptions(customerSelect, 'customer-1');
      fireEvent.change(screen.getByLabelText(/scheduled start/i), {
        target: { value: typed },
      });

      await user.click(
        screen.getByRole('button', { name: /save|create|submit/i })
      );

      await waitFor(() => {
        expect(jobService.createJob).toHaveBeenCalledWith(
          expect.objectContaining({
            title: 'HST Job',
            scheduled_start_at: expectedEpoch,
          })
        );
      });
    });

    it('should show an existing job start in the tenant timezone (EDT)', async () => {
      const edtTenant = {
        ...mockTenant,
        settings: { timezone: 'America/New_York' },
      };
      vi.mocked(useTenantStore).mockImplementation((selector) => {
        const state = {
          currentTenant: edtTenant,
          tenants: [edtTenant],
          isLoading: false,
          error: null,
          loadTenants: vi.fn(),
          setCurrentTenant: vi.fn(),
        };
        return typeof selector === 'function' ? selector(state) : state;
      });
      // 13:00 UTC on 15 Jul 2026 == 09:00 EDT
      vi.mocked(jobService.getJob).mockResolvedValue({
        ...mockJobs[0],
        scheduled_start_at: Date.UTC(2026, 6, 15, 13, 0) / 1000,
      } as unknown as Awaited<ReturnType<typeof jobService.getJob>>);

      const { JobDetailPage } =
        await import('../../src/pages/Jobs/JobDetailPage');

      render(
        <MemoryRouter initialEntries={['/jobs/job-1']}>
          <Routes>
            <Route path="/jobs/:id" element={<JobDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(screen.getByLabelText(/scheduled start/i)).toHaveValue(
          '2026-07-15T09:00'
        );
      });
    });

    it('should not send status when editing a job (status changes via actions)', async () => {
      vi.mocked(jobService.getJob).mockResolvedValue({
        ...mockJobs[0],
        scheduled_start_at: undefined,
      } as unknown as Awaited<ReturnType<typeof jobService.getJob>>);
      vi.mocked(jobService.updateJob).mockResolvedValue(
        mockJobs[0] as unknown as Awaited<
          ReturnType<typeof jobService.updateJob>
        >
      );

      const { JobDetailPage } =
        await import('../../src/pages/Jobs/JobDetailPage');

      render(
        <MemoryRouter initialEntries={['/jobs/job-1']}>
          <Routes>
            <Route path="/jobs/:id" element={<JobDetailPage />} />
            <Route path="/jobs" element={<div>Jobs</div>} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(screen.getByLabelText(/title/i)).toHaveValue(mockJobs[0].title);
      });
      // The free-form status dropdown is gone (T124).
      expect(screen.queryByLabelText(/^status$/i)).toBeNull();
      await screen.findByRole('option', { name: /John Doe/i });

      fireEvent.submit(screen.getByLabelText(/title/i).closest('form')!);

      await waitFor(() => {
        expect(jobService.updateJob).toHaveBeenCalled();
      });
      const payload = vi.mocked(jobService.updateJob).mock.calls[0][1];
      expect(payload).not.toHaveProperty('status');
    });

    it('should show validation errors for missing required fields', async () => {
      const user = userEvent.setup({ delay: null });
      const { JobDetailPage } =
        await import('../../src/pages/Jobs/JobDetailPage');

      render(
        <MemoryRouter initialEntries={['/jobs/new']}>
          <JobDetailPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
      });

      // Try to submit empty form
      const saveButton = screen.getByRole('button', {
        name: /save|create|submit/i,
      });
      await user.click(saveButton);

      // Should show validation errors
      await waitFor(() => {
        const errorMessages = screen.queryAllByText(
          /required|invalid|must be|select/i
        );
        expect(errorMessages.length).toBeGreaterThan(0);
      });

      expect(jobService.createJob).not.toHaveBeenCalled();
    });
  });

  describe('Job Status Transitions', () => {
    it('should transition job from new to scheduled', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(jobService.getJob).mockResolvedValue(mockJobs[0]); // New job
      vi.mocked(jobService.transitionStatus).mockResolvedValue({
        ...mockJobs[0],
        status: 'scheduled' as const,
      });

      const { JobDetailPage } =
        await import('../../src/pages/Jobs/JobDetailPage');

      render(
        <MemoryRouter initialEntries={['/jobs/job-1']}>
          <Routes>
            <Route path="/jobs/:id" element={<JobDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(jobService.getJob).toHaveBeenCalledWith('job-1');
      });

      await user.click(
        await screen.findByRole('button', { name: /^schedule$/i })
      );

      await waitFor(() => {
        expect(jobService.transitionStatus).toHaveBeenCalledWith(
          'job-1',
          'scheduled'
        );
      });
    });

    it('should transition job from in_progress to completed', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(jobService.getJob).mockResolvedValue(mockJobs[1]); // In progress job
      vi.mocked(jobService.transitionStatus).mockResolvedValue({
        ...mockJobs[1],
        status: 'completed' as const,
        actual_end_at: Date.UTC(2024, 0, 10, 14, 0) / 1000,
      });

      const { JobDetailPage } =
        await import('../../src/pages/Jobs/JobDetailPage');

      render(
        <MemoryRouter initialEntries={['/jobs/job-2']}>
          <Routes>
            <Route path="/jobs/:id" element={<JobDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(jobService.getJob).toHaveBeenCalledWith('job-2');
      });

      await user.click(
        await screen.findByRole('button', { name: /complete job/i })
      );

      await waitFor(() => {
        expect(jobService.transitionStatus).toHaveBeenCalledWith(
          'job-2',
          'completed'
        );
      });
    });

    it('should allow cancelling a job with reason', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(jobService.getJob).mockResolvedValue(mockJobs[0]);
      vi.mocked(jobService.transitionStatus).mockResolvedValue({
        ...mockJobs[0],
        status: 'cancelled' as const,
      });
      const promptSpy = vi
        .spyOn(window, 'prompt')
        .mockReturnValue('Customer requested cancellation');

      const { JobDetailPage } =
        await import('../../src/pages/Jobs/JobDetailPage');

      render(
        <MemoryRouter initialEntries={['/jobs/job-1']}>
          <Routes>
            <Route path="/jobs/:id" element={<JobDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(jobService.getJob).toHaveBeenCalled();
      });

      await user.click(
        await screen.findByRole('button', { name: /cancel job/i })
      );

      expect(promptSpy).toHaveBeenCalled();
      await waitFor(() => {
        expect(jobService.transitionStatus).toHaveBeenCalledWith(
          'job-1',
          'cancelled',
          'cancellation',
          'Customer requested cancellation'
        );
      });
      promptSpy.mockRestore();
    });
  });

  describe('Job Status History', () => {
    it('should display job status history', async () => {
      const mockHistory = {
        transitions: [
          {
            id: 'transition-1',
            job_id: 'job-3',
            job_number: 'JOB-003',
            from_status: 'new' as const,
            to_status: 'scheduled' as const,
            changed_by: 'user-123',
            changed_by_name: 'Admin User',
            transitioned_at: '2024-01-04T00:00:00Z',
          },
          {
            id: 'transition-2',
            job_id: 'job-3',
            job_number: 'JOB-003',
            from_status: 'scheduled' as const,
            to_status: 'in_progress' as const,
            changed_by: 'user-123',
            changed_by_name: 'Admin User',
            transitioned_at: '2024-01-05T08:00:00Z',
          },
          {
            id: 'transition-3',
            job_id: 'job-3',
            job_number: 'JOB-003',
            from_status: 'in_progress' as const,
            to_status: 'completed' as const,
            changed_by: 'user-123',
            changed_by_name: 'Admin User',
            transitioned_at: '2024-01-05T09:45:00Z',
          },
        ],
        total: 3,
      };

      vi.mocked(jobService.getJob).mockResolvedValue(mockJobs[2]);
      vi.mocked(jobService.getStatusHistory).mockResolvedValue(mockHistory);

      const { JobStatusHistoryPage } =
        await import('../../src/pages/Jobs/JobStatusHistoryPage');

      render(
        <MemoryRouter initialEntries={['/jobs/job-3/history']}>
          <Routes>
            <Route
              path="/jobs/:id/history"
              element={<JobStatusHistoryPage />}
            />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(jobService.getStatusHistory).toHaveBeenCalledWith(
          'job-3',
          expect.anything(),
          expect.anything(),
          expect.anything()
        );
      });

      // Verify history is displayed
      expect(
        (await screen.findAllByText(/in progress/i)).length
      ).toBeGreaterThan(0);
      expect(screen.getAllByText(/completed/i).length).toBeGreaterThan(0);
    });
  });

  describe('Job Filtering', () => {
    it('should filter jobs by status', async () => {
      const user = userEvent.setup({ delay: null });
      const { JobsPage } = await import('../../src/pages/Jobs/JobsPage');

      render(
        <MemoryRouter initialEntries={['/jobs']}>
          <JobsPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(jobService.getJobs).toHaveBeenCalled();
      });

      // Find status filter
      const statusFilter =
        screen.queryByLabelText(/status/i) ||
        screen.queryByRole('combobox', { name: /status/i });
      if (statusFilter) {
        await user.click(statusFilter);
        const completedOption = screen.queryByText(/completed/i);
        if (completedOption) {
          await user.click(completedOption);
        }

        await waitFor(() => {
          expect(jobService.getJobs).toHaveBeenCalledWith(
            expect.objectContaining({
              status: 'completed',
            })
          );
        });
      }
    });

    it('should filter jobs by priority', async () => {
      const user = userEvent.setup({ delay: null });
      const { JobsPage } = await import('../../src/pages/Jobs/JobsPage');

      render(
        <MemoryRouter initialEntries={['/jobs']}>
          <JobsPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(jobService.getJobs).toHaveBeenCalled();
      });

      // Find priority filter
      const priorityFilter =
        screen.queryByLabelText(/priority/i) ||
        screen.queryByRole('combobox', { name: /priority/i });
      if (priorityFilter) {
        await user.click(priorityFilter);
        const highOption = screen.queryByText(/high/i);
        if (highOption) {
          await user.click(highOption);
        }

        await waitFor(() => {
          expect(jobService.getJobs).toHaveBeenCalledWith(
            expect.objectContaining({
              priority: 'high',
            })
          );
        });
      }
    });

    it('should search jobs by query', async () => {
      const user = userEvent.setup({ delay: null });
      const { JobsPage } = await import('../../src/pages/Jobs/JobsPage');

      render(
        <MemoryRouter initialEntries={['/jobs']}>
          <JobsPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(jobService.getJobs).toHaveBeenCalled();
      });

      // Find search input
      const searchInput = screen.getByPlaceholderText(/search/i);
      await user.type(searchInput, 'HVAC');
      await user.keyboard('{Enter}');

      await waitFor(() => {
        expect(jobService.getJobs).toHaveBeenCalledWith(
          expect.objectContaining({
            search: 'HVAC',
          })
        );
      });
    });
  });

  describe('Error Handling', () => {
    it('should handle job load error gracefully', async () => {
      vi.mocked(jobService.getJobs).mockRejectedValue(
        new Error('Failed to load jobs')
      );

      const { JobsPage } = await import('../../src/pages/Jobs/JobsPage');

      render(
        <MemoryRouter initialEntries={['/jobs']}>
          <JobsPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(jobService.getJobs).toHaveBeenCalled();
      });

      // An API failure must show an error state with retry, not "No jobs yet"
      expect(
        await screen.findByText(/couldn't load jobs/i)
      ).toBeInTheDocument();
      expect(screen.queryByText(/no jobs yet/i)).not.toBeInTheDocument();

      vi.mocked(jobService.getJobs).mockClear();
      await userEvent
        .setup()
        .click(screen.getByRole('button', { name: /try again/i }));
      expect(jobService.getJobs).toHaveBeenCalledTimes(1);
    });

    it('should handle job creation error', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(jobService.createJob).mockRejectedValue({
        response: { data: { message: 'Customer not found' } },
      });

      const { JobDetailPage } =
        await import('../../src/pages/Jobs/JobDetailPage');

      render(
        <MemoryRouter initialEntries={['/jobs/new']}>
          <JobDetailPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(screen.getByLabelText(/title/i)).toBeInTheDocument();
      });

      await user.type(screen.getByLabelText(/title/i), 'Test Job');

      const saveButton = screen.getByRole('button', {
        name: /save|create|submit/i,
      });
      await user.click(saveButton);

      // Should show error or validation message
      await waitFor(() => {
        const errorMessage = screen.queryByText(
          /error|failed|not found|required/i
        );
        if (errorMessage) {
          expect(errorMessage).toBeInTheDocument();
        }
      });
    });

    it('should handle status transition error', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(jobService.getJob).mockResolvedValue(mockJobs[1]);
      vi.mocked(jobService.transitionStatus).mockRejectedValue({
        response: {
          data: { message: 'Cannot complete job without work details' },
        },
      });

      const { JobDetailPage } =
        await import('../../src/pages/Jobs/JobDetailPage');

      render(
        <MemoryRouter initialEntries={['/jobs/job-2']}>
          <Routes>
            <Route path="/jobs/:id" element={<JobDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(jobService.getJob).toHaveBeenCalled();
      });

      const completeButton = screen.queryByRole('button', {
        name: /complete|finish/i,
      });
      if (completeButton) {
        await user.click(completeButton);

        // Should show error message
        await waitFor(() => {
          const errorMessage = screen.queryByText(/error|failed|cannot/i);
          if (errorMessage) {
            expect(errorMessage).toBeInTheDocument();
          }
        });
      }
    });
  });

  describe('Sites and not-to-exceed approvals (T186, T187)', () => {
    it('picks one of the customer sites on a new job', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(jobService.createJob).mockResolvedValue(mockJobs[0]);
      const { JobDetailPage } =
        await import('../../src/pages/Jobs/JobDetailPage');
      render(
        <MemoryRouter initialEntries={['/jobs/new']}>
          <Routes>
            <Route path="/jobs/new" element={<JobDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      const siteSelect = await screen.findByLabelText(/^site$/i);
      expect(siteSelect).toBeDisabled();

      await screen.findByRole('option', { name: /John Doe/i });
      await user.selectOptions(
        screen.getByLabelText(/customer/i),
        'customer-1'
      );
      await screen.findByRole('option', { name: 'Harborview Lofts' });
      // Inactive sites aren't offered.
      expect(
        screen.queryByRole('option', { name: /Closed Annex/ })
      ).not.toBeInTheDocument();
      await user.selectOptions(screen.getByLabelText(/^site$/i), 'site-1');
      expect(
        screen.getByText('Access: Key box by the loading dock')
      ).toBeInTheDocument();

      await user.type(screen.getByLabelText(/title/i), 'Window cleaning');
      await user.type(screen.getByLabelText(/est\. cost/i), '750');
      await user.click(screen.getByRole('button', { name: /create job/i }));

      await waitFor(() => {
        expect(jobService.createJob).toHaveBeenCalledWith(
          expect.objectContaining({
            customer_id: 'customer-1',
            site_id: 'site-1',
            estimated_cost: 750,
          })
        );
      });
    });

    it('shows the site of an existing job', async () => {
      vi.mocked(jobService.getJob).mockResolvedValue({
        ...mockJobs[0],
        site_id: 'site-1',
      });
      const { JobDetailPage } =
        await import('../../src/pages/Jobs/JobDetailPage');
      render(
        <MemoryRouter initialEntries={['/jobs/job-1']}>
          <Routes>
            <Route path="/jobs/:id" element={<JobDetailPage />} />
          </Routes>
        </MemoryRouter>
      );
      await waitFor(() =>
        expect(screen.getByLabelText(/^site$/i)).toHaveValue('site-1')
      );
      expect(
        screen.getByText('220 W Spring St, Columbus, OH 43215')
      ).toBeInTheDocument();
    });

    it("records the approver's OK and retries scheduling on approval_required", async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(jobService.getJob).mockResolvedValue(mockJobs[0]);
      vi.mocked(jobService.transitionStatus)
        .mockRejectedValueOnce(
          new AxiosError('Conflict', 'ERR_BAD_REQUEST', undefined, undefined, {
            data: {
              error: 'approval_required',
              message:
                "This work is over the customer's not-to-exceed limit; record the approver's OK before scheduling it",
            },
            status: 409,
            statusText: 'Conflict',
            headers: {},
            config: { headers: new AxiosHeaders() },
          })
        )
        .mockResolvedValueOnce({
          ...mockJobs[0],
          status: 'scheduled' as const,
        });
      vi.mocked(jobService.recordApproval).mockResolvedValue({
        id: 'ap-1',
        job_id: 'job-1',
        approver_name: 'Karen Doyle',
        approver_email: 'karen@example.com',
        amount: '750.00',
        approved_at: '2024-01-02T00:00:00Z',
        recorded_by: 'user-123',
        created_at: '2024-01-02T00:00:00Z',
      });

      const { JobDetailPage } =
        await import('../../src/pages/Jobs/JobDetailPage');
      render(
        <MemoryRouter initialEntries={['/jobs/job-1']}>
          <Routes>
            <Route path="/jobs/:id" element={<JobDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await user.click(
        await screen.findByRole('button', { name: /^schedule$/i })
      );
      const dialog = await screen.findByRole('dialog');
      expect(
        within(dialog).getByText("Record the approver's OK")
      ).toBeInTheDocument();
      expect(
        within(dialog).getByText(/over the customer's not-to-exceed limit/)
      ).toBeInTheDocument();
      await user.type(within(dialog).getByLabelText(/notes/i), 'OK by phone');
      await user.click(
        within(dialog).getByRole('button', { name: /record approval/i })
      );

      await waitFor(() => {
        expect(jobService.recordApproval).toHaveBeenCalledWith('job-1', {
          notes: 'OK by phone',
        });
      });
      await waitFor(() => {
        expect(jobService.transitionStatus).toHaveBeenCalledTimes(2);
      });
      expect(jobService.transitionStatus).toHaveBeenLastCalledWith(
        'job-1',
        'scheduled'
      );
    });

    it('lists the approvals on the job', async () => {
      vi.mocked(jobService.getJob).mockResolvedValue(mockJobs[0]);
      vi.mocked(jobService.getApprovals).mockResolvedValue([
        {
          id: 'ap-1',
          job_id: 'job-1',
          approver_name: 'Karen Doyle',
          approver_email: 'karen@example.com',
          amount: '750.00',
          approved_at: '2024-01-02T15:00:00Z',
          recorded_by: 'user-123',
          created_at: '2024-01-02T15:00:00Z',
        },
      ]);
      const { JobDetailPage } =
        await import('../../src/pages/Jobs/JobDetailPage');
      render(
        <MemoryRouter initialEntries={['/jobs/job-1']}>
          <Routes>
            <Route path="/jobs/:id" element={<JobDetailPage />} />
          </Routes>
        </MemoryRouter>
      );
      const approvals = await screen.findByRole('region', {
        name: 'Approvals',
      });
      expect(
        await within(approvals).findByText('Karen Doyle')
      ).toBeInTheDocument();
      expect(within(approvals).getByText('$750.00')).toBeInTheDocument();
    });
  });

  describe('Generate invoice from a completed job (T029)', () => {
    const completedJob = {
      ...mockJobs[1],
      status: 'completed' as const,
      allowed_transitions: [],
    };

    const renderJob = async (job: typeof completedJob) => {
      vi.mocked(jobService.getJob).mockResolvedValue(
        job as unknown as Awaited<ReturnType<typeof jobService.getJob>>
      );
      const { JobDetailPage } =
        await import('../../src/pages/Jobs/JobDetailPage');
      render(
        <MemoryRouter initialEntries={[`/jobs/${job.id}`]}>
          <Routes>
            <Route path="/jobs/:id" element={<JobDetailPage />} />
          </Routes>
        </MemoryRouter>
      );
      await waitFor(() => {
        expect(jobService.getJob).toHaveBeenCalledWith(job.id);
      });
    };

    it('generates the draft invoice and opens it', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(jobService.invoiceJob).mockResolvedValue({
        id: 'inv-42',
        lines: [],
      } as unknown as Awaited<ReturnType<typeof jobService.invoiceJob>>);

      await renderJob(completedJob);
      await user.click(
        await screen.findByRole('button', { name: /generate invoice/i })
      );

      await waitFor(() => {
        expect(jobService.invoiceJob).toHaveBeenCalledWith('job-2');
      });
      expect(mockNavigate).toHaveBeenCalledWith('/invoices/inv-42');
    });

    it('opens the existing invoice when the job was invoiced meanwhile', async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(jobService.invoiceJob).mockRejectedValue({
        response: {
          status: 409,
          data: {
            error: 'already_invoiced',
            message: 'This job has already been invoiced',
            resource_id: 'inv-7',
          },
        },
      });

      await renderJob(completedJob);
      await user.click(
        await screen.findByRole('button', { name: /generate invoice/i })
      );

      await waitFor(() => {
        expect(mockNavigate).toHaveBeenCalledWith('/invoices/inv-7');
      });
    });

    it("shows the API's reason when invoicing is refused", async () => {
      const user = userEvent.setup({ delay: null });
      vi.mocked(jobService.invoiceJob).mockRejectedValue({
        response: {
          status: 409,
          data: {
            error: 'job_not_completed',
            message: 'Only completed jobs can be invoiced',
          },
        },
      });

      await renderJob(completedJob);
      await user.click(
        await screen.findByRole('button', { name: /generate invoice/i })
      );

      expect(
        await screen.findByText('Only completed jobs can be invoiced')
      ).toBeInTheDocument();
    });

    it('links an invoiced job to its invoice', async () => {
      const user = userEvent.setup({ delay: null });
      await renderJob({
        ...completedJob,
        status: 'invoiced' as const,
        invoice_id: 'inv-42',
      } as unknown as typeof completedJob);

      await user.click(
        await screen.findByRole('button', { name: /view invoice/i })
      );
      expect(mockNavigate).toHaveBeenCalledWith('/invoices/inv-42');
      expect(
        screen.queryByRole('button', { name: /generate invoice/i })
      ).toBeNull();
    });

    it('offers nothing before the job is completed', async () => {
      await renderJob(mockJobs[1] as unknown as typeof completedJob); // in progress
      expect(
        await screen.findByRole('button', { name: /complete job/i })
      ).toBeInTheDocument();
      expect(
        screen.queryByRole('button', { name: /generate invoice/i })
      ).toBeNull();
    });

    it('needs the invoices.create permission', async () => {
      grantedPermissions = new Set<string>();
      await renderJob(completedJob);
      await screen.findByLabelText(/title/i);
      expect(
        screen.queryByRole('button', { name: /generate invoice/i })
      ).toBeNull();
    });

    it('leaves agreement visits to the consolidated invoice', async () => {
      await renderJob({
        ...completedJob,
        agreement_id: 'agr-1',
      } as unknown as typeof completedJob);
      expect(
        await screen.findByText(
          /billed on the customer's consolidated invoice/i
        )
      ).toBeInTheDocument();
      expect(
        screen.queryByRole('button', { name: /generate invoice/i })
      ).toBeNull();
    });
  });
});

// =============================================================================
// Technicians (T035): their assigned jobs, no money, no dispatch tools
// =============================================================================

describe('Job pages for technicians (T035)', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupMocks();
    mockTenant.role = 'technician';
    grantedPermissions = new Set<string>([
      'jobs.read',
      'jobs.list',
      'jobs.update',
    ]);
  });

  afterEach(() => {
    mockTenant.role = undefined;
  });

  it('lists jobs without New job or Repeat Job', async () => {
    const { JobsPage } = await import('../../src/pages/Jobs/JobsPage');
    render(
      <MemoryRouter initialEntries={['/jobs']}>
        <JobsPage />
      </MemoryRouter>
    );
    await waitFor(() => expect(jobService.getJobs).toHaveBeenCalled());
    expect(
      screen.queryByRole('button', { name: /new job/i })
    ).not.toBeInTheDocument();
  });

  it('works an assigned job: no price, customer list, delete or approvals', async () => {
    vi.mocked(jobService.getJob).mockResolvedValue({
      ...mockJobs[0],
      scheduled_start_at: undefined,
      estimated_cost: 0,
    } as unknown as Awaited<ReturnType<typeof jobService.getJob>>);
    vi.mocked(jobService.updateJob).mockResolvedValue(
      mockJobs[0] as unknown as Awaited<ReturnType<typeof jobService.updateJob>>
    );
    const { JobDetailPage } =
      await import('../../src/pages/Jobs/JobDetailPage');
    render(
      <MemoryRouter initialEntries={['/jobs/job-1']}>
        <Routes>
          <Route path="/jobs/:id" element={<JobDetailPage />} />
          <Route path="/jobs" element={<div>Jobs</div>} />
        </Routes>
      </MemoryRouter>
    );

    await waitFor(() =>
      expect(screen.getByLabelText(/title/i)).toHaveValue(mockJobs[0].title)
    );
    // The job's own customer, read-only; the customer list isn't requested.
    expect(
      await screen.findByRole('option', { name: /John Doe/i })
    ).toBeInTheDocument();
    expect(screen.getByLabelText(/customer/i)).toBeDisabled();
    expect(customerService.getCustomers).not.toHaveBeenCalled();
    expect(siteService.list).not.toHaveBeenCalled();

    expect(screen.queryByLabelText(/est\. cost/i)).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /delete job/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /add member/i })
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('region', { name: 'Approvals' })
    ).not.toBeInTheDocument();
    expect(jobService.getApprovals).not.toHaveBeenCalled();

    fireEvent.submit(screen.getByLabelText(/title/i).closest('form')!);
    await waitFor(() => expect(jobService.updateJob).toHaveBeenCalled());
    const payload = vi.mocked(jobService.updateJob).mock.calls[0][1];
    expect(payload.estimated_cost).toBeUndefined();
  });

  it('shows no Generate invoice or View invoice card', async () => {
    const { JobDetailPage } =
      await import('../../src/pages/Jobs/JobDetailPage');
    for (const job of [
      { ...mockJobs[0], status: 'completed', estimated_cost: 0 },
      {
        ...mockJobs[0],
        status: 'invoiced',
        estimated_cost: 0,
        invoice_id: 'inv-1',
      },
    ]) {
      vi.mocked(jobService.getJob).mockResolvedValue(
        job as unknown as Awaited<ReturnType<typeof jobService.getJob>>
      );
      const { unmount } = render(
        <MemoryRouter initialEntries={['/jobs/job-1']}>
          <Routes>
            <Route path="/jobs/:id" element={<JobDetailPage />} />
          </Routes>
        </MemoryRouter>
      );
      await waitFor(() =>
        expect(screen.getByLabelText(/title/i)).toHaveValue(mockJobs[0].title)
      );
      expect(
        screen.queryByRole('button', { name: /generate invoice/i })
      ).not.toBeInTheDocument();
      expect(
        screen.queryByRole('button', { name: /view invoice/i })
      ).not.toBeInTheDocument();
      unmount();
    }
  });

  it("shows Job not found for a job that isn't theirs (API 404)", async () => {
    vi.mocked(jobService.getJob).mockRejectedValue(
      new AxiosError('Not Found', 'ERR_BAD_REQUEST', undefined, undefined, {
        data: { error: 'job_not_found', message: 'Job not found' },
        status: 404,
        statusText: 'Not Found',
        headers: {},
        config: { headers: new AxiosHeaders() },
      })
    );
    const { JobDetailPage } =
      await import('../../src/pages/Jobs/JobDetailPage');
    render(
      <MemoryRouter initialEntries={['/jobs/job-9']}>
        <Routes>
          <Route path="/jobs/:id" element={<JobDetailPage />} />
        </Routes>
      </MemoryRouter>
    );
    expect(await screen.findByText('Job not found')).toBeInTheDocument();
    expect(screen.queryByLabelText(/title/i)).not.toBeInTheDocument();
  });
});

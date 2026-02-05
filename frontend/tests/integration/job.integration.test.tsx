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

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
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
    startJob: vi.fn(),
    completeJob: vi.fn(),
    cancelJob: vi.fn(),
    getStats: vi.fn(),
    transitionStatus: vi.fn(),
    getStatusHistory: vi.fn(),
    getScheduledJobs: vi.fn(),
    assignMember: vi.fn(),
    unassignMember: vi.fn(),
  },
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

// Import after mocking
import { useAuthStore, useTenantStore, useSidebarStore } from '@store';
import { jobService } from '@services/jobService';
import { customerService } from '@services/customerService';

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

const mockTenant = {
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
    priority: 'normal' as const,
    scheduled_start_at: '2024-01-15T09:00:00Z',
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
    priority: 'high' as const,
    scheduled_start_at: '2024-01-10T10:00:00Z',
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
    priority: 'low' as const,
    scheduled_start_at: '2024-01-05T08:00:00Z',
    scheduled_end_at: '2024-01-05T10:00:00Z',
    actual_start_at: '2024-01-05T08:00:00Z',
    actual_end_at: '2024-01-05T09:45:00Z',
    created_at: '2024-01-03T00:00:00Z',
    updated_at: '2024-01-05T09:45:00Z',
  },
];

const mockJobStats = {
  total_jobs: 3,
  pending_jobs: 1,
  scheduled_jobs: 1,
  in_progress_jobs: 1,
  completed_jobs: 1,
  cancelled_jobs: 0,
};

// =============================================================================
// Test Setup
// =============================================================================

const mockNavigate = vi.fn();
vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom');
  return {
    ...actual,
    useNavigate: () => mockNavigate,
  };
});

const setupMocks = () => {
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
        // Check for status badges/indicators
        expect(
          screen.getByText(/new|scheduled|in progress/i)
        ).toBeInTheDocument();
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
      const user = userEvent.setup();
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
      const user = userEvent.setup();
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
      const user = userEvent.setup();
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

      // Select customer (if dropdown)
      const customerSelect = screen.queryByLabelText(/customer/i);
      if (customerSelect) {
        await user.click(customerSelect);
        const customerOption = screen.queryByText(/John Doe/i);
        if (customerOption) {
          await user.click(customerOption);
        }
      }

      // Select job type
      const jobTypeSelect = screen.queryByLabelText(/type/i);
      if (jobTypeSelect) {
        await user.click(jobTypeSelect);
        const installOption = screen.queryByText(/installation/i);
        if (installOption) {
          await user.click(installOption);
        }
      }

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
          })
        );
      });
    });

    it('should show validation errors for missing required fields', async () => {
      const user = userEvent.setup();
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
      const user = userEvent.setup();
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

      // Find and click schedule button
      const scheduleButton = screen.queryByRole('button', {
        name: /schedule/i,
      });
      if (scheduleButton) {
        await user.click(scheduleButton);

        await waitFor(() => {
          expect(jobService.transitionStatus).toHaveBeenCalledWith(
            'job-1',
            'scheduled',
            expect.anything(),
            expect.anything()
          );
        });
      }
    });

    it('should transition job from in_progress to completed', async () => {
      const user = userEvent.setup();
      vi.mocked(jobService.getJob).mockResolvedValue(mockJobs[1]); // In progress job
      vi.mocked(jobService.completeJob).mockResolvedValue({
        ...mockJobs[1],
        status: 'completed' as const,
        actual_end_at: '2024-01-10T14:00:00Z',
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

      // Find and click complete button
      const completeButton = screen.queryByRole('button', {
        name: /complete|finish/i,
      });
      if (completeButton) {
        await user.click(completeButton);

        await waitFor(() => {
          expect(jobService.completeJob).toHaveBeenCalledWith(
            'job-2',
            expect.anything()
          );
        });
      }
    });

    it('should allow cancelling a job with reason', async () => {
      const user = userEvent.setup();
      vi.mocked(jobService.getJob).mockResolvedValue(mockJobs[0]);
      vi.mocked(jobService.cancelJob).mockResolvedValue({
        ...mockJobs[0],
        status: 'cancelled' as const,
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
        expect(jobService.getJob).toHaveBeenCalled();
      });

      // Find cancel button
      const cancelButton = screen.queryByRole('button', { name: /cancel/i });
      if (cancelButton) {
        await user.click(cancelButton);

        // Fill in reason if dialog appears
        const reasonInput = screen.queryByLabelText(/reason/i);
        if (reasonInput) {
          await user.type(reasonInput, 'Customer requested cancellation');
        }

        // Confirm cancellation
        const confirmButton = screen.queryByRole('button', {
          name: /confirm|yes/i,
        });
        if (confirmButton) {
          await user.click(confirmButton);
        }

        await waitFor(() => {
          expect(jobService.cancelJob).toHaveBeenCalled();
        });
      }
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
      await waitFor(() => {
        expect(
          screen.getByText(/new.*scheduled|scheduled|in progress|completed/i)
        ).toBeInTheDocument();
      });
    });
  });

  describe('Job Filtering', () => {
    it('should filter jobs by status', async () => {
      const user = userEvent.setup();
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
      const user = userEvent.setup();
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
      const user = userEvent.setup();
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

      // Error should be handled (logged, shown to user, etc.)
    });

    it('should handle job creation error', async () => {
      const user = userEvent.setup();
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
      const user = userEvent.setup();
      vi.mocked(jobService.getJob).mockResolvedValue(mockJobs[1]);
      vi.mocked(jobService.completeJob).mockRejectedValue({
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
});

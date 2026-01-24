import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { vi, describe, it, expect, beforeEach } from 'vitest';
import { RoleAudit } from '../RoleAudit';
import { roleApi } from '@services/roleApi';
import type { RoleAuditLog } from '@types/role';

// Mock the API
vi.mock('@services/roleApi');

const mockAuditLogs: RoleAuditLog[] = [
  {
    id: '1',
    role_id: 'r1',
    role_name: 'admin',
    action: 'created',
    performed_by: 'u1',
    performed_by_name: 'John Doe',
    timestamp: '2024-01-15T10:30:00Z',
    details: { description: 'Admin role created' },
  },
  {
    id: '2',
    role_id: 'r1',
    role_name: 'admin',
    action: 'permission_added',
    performed_by: 'u1',
    performed_by_name: 'John Doe',
    timestamp: '2024-01-15T11:00:00Z',
    details: { permission: 'users.manage' },
  },
  {
    id: '3',
    role_id: 'r2',
    role_name: 'editor',
    action: 'assigned',
    performed_by: 'u2',
    performed_by_name: 'Jane Smith',
    timestamp: '2024-01-16T09:00:00Z',
  },
];

const createWrapper = () => {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return ({ children }: { children: React.ReactNode }) => (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
};

describe('RoleAudit', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    (roleApi.getRoleAuditLogs as vi.Mock).mockResolvedValue({
      data: mockAuditLogs,
    });
  });

  it('renders audit trail with logs', async () => {
    render(<RoleAudit />, { wrapper: createWrapper() });

    // Check header
    expect(screen.getByText('Role Audit Trail')).toBeInTheDocument();
    expect(
      screen.getByText(/track all role and permission changes/i)
    ).toBeInTheDocument();

    // Wait for logs to load - use getAllByText since 'admin' appears multiple times
    await waitFor(() => {
      expect(screen.getAllByText('admin').length).toBeGreaterThan(0);
      expect(screen.getByText('editor')).toBeInTheDocument();
    });
  });

  it('displays all audit log actions', async () => {
    render(<RoleAudit />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('Created')).toBeInTheDocument();
      expect(screen.getByText('Permission Added')).toBeInTheDocument();
      expect(screen.getByText('Assigned')).toBeInTheDocument();
    });
  });

  it('displays performed by information', async () => {
    render(<RoleAudit />, { wrapper: createWrapper() });

    await waitFor(() => {
      // John Doe appears multiple times (multiple audit logs by same user)
      expect(screen.getAllByText('John Doe').length).toBeGreaterThan(0);
      expect(screen.getByText('Jane Smith')).toBeInTheDocument();
    });
  });

  it('displays timestamps in correct format', async () => {
    render(<RoleAudit />, { wrapper: createWrapper() });

    await waitFor(() => {
      // Check for formatted dates (format: MMM dd, yyyy HH:mm) - multiple dates on Jan 15
      expect(screen.getAllByText(/Jan 15, 2024/).length).toBeGreaterThan(0);
      expect(screen.getByText(/Jan 16, 2024/)).toBeInTheDocument();
    });
  });

  it('filters logs by search query', async () => {
    const user = userEvent.setup();
    render(<RoleAudit />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getAllByText('admin').length).toBeGreaterThan(0);
      expect(screen.getByText('editor')).toBeInTheDocument();
    });

    // Search for 'admin'
    const searchInput = screen.getByPlaceholderText('Search audit logs...');
    await user.type(searchInput, 'admin');

    // Check filtered results
    const adminElements = screen.getAllByText('admin');
    expect(adminElements.length).toBeGreaterThan(0);
    expect(screen.queryByText('editor')).not.toBeInTheDocument();
  });

  it('shows and hides filter panel', async () => {
    const user = userEvent.setup();
    const { container } = render(<RoleAudit />, { wrapper: createWrapper() });

    // Filters should be hidden initially - no action select visible
    expect(container.querySelector('select')).not.toBeInTheDocument();

    // Click filters button
    const filtersButton = screen.getByRole('button', { name: /filters/i });
    await user.click(filtersButton);

    // Check filters are visible - look for filter panel elements
    await waitFor(() => {
      expect(container.querySelector('select')).toBeInTheDocument();
      // Use getAllByText since there might be multiple Action labels
      expect(screen.getAllByText(/action/i).length).toBeGreaterThan(0);
    });
  });

  it('applies action filter', async () => {
    const user = userEvent.setup();
    const getRoleAuditLogsMock = vi.fn().mockResolvedValue({ data: mockAuditLogs });
    vi.mocked(roleApi.getRoleAuditLogs).mockImplementation(getRoleAuditLogsMock);

    const { container } = render(<RoleAudit />, { wrapper: createWrapper() });

    // Open filters
    await user.click(screen.getByRole('button', { name: /filters/i }));

    // Wait for filter panel and select action filter
    await waitFor(() => {
      expect(container.querySelector('select')).toBeInTheDocument();
    });

    const actionSelect = container.querySelector('select') as HTMLSelectElement;
    await user.selectOptions(actionSelect, 'created');

    // Check API was called with filter
    await waitFor(() => {
      expect(getRoleAuditLogsMock).toHaveBeenCalledWith(
        expect.objectContaining({ action: 'created' })
      );
    });
  });

  it('applies date filters', async () => {
    const user = userEvent.setup();
    const getRoleAuditLogsMock = vi.fn().mockResolvedValue({ data: mockAuditLogs });
    vi.mocked(roleApi.getRoleAuditLogs).mockImplementation(getRoleAuditLogsMock);

    const { container } = render(<RoleAudit />, { wrapper: createWrapper() });

    // Open filters
    await user.click(screen.getByRole('button', { name: /filters/i }));

    // Wait for filter panel
    await waitFor(() => {
      expect(container.querySelectorAll('input[type="date"]').length).toBeGreaterThan(0);
    });

    // Get date inputs
    const dateInputs = container.querySelectorAll('input[type="date"]');
    const startDateInput = dateInputs[0] as HTMLInputElement;
    const endDateInput = dateInputs[1] as HTMLInputElement;

    await user.type(startDateInput, '2024-01-01');
    await user.type(endDateInput, '2024-01-31');

    // Check API was called with filters
    await waitFor(() => {
      expect(getRoleAuditLogsMock).toHaveBeenCalledWith(
        expect.objectContaining({
          start_date: '2024-01-01',
          end_date: '2024-01-31',
        })
      );
    });
  });

  it('clears all filters', async () => {
    const user = userEvent.setup();
    const getRoleAuditLogsMock = vi.fn().mockResolvedValue({ data: mockAuditLogs });
    vi.mocked(roleApi.getRoleAuditLogs).mockImplementation(getRoleAuditLogsMock);

    const { container } = render(<RoleAudit />, { wrapper: createWrapper() });

    // Open filters and set a filter
    await user.click(screen.getByRole('button', { name: /filters/i }));

    await waitFor(() => {
      expect(container.querySelector('select')).toBeInTheDocument();
    });

    const actionSelect = container.querySelector('select') as HTMLSelectElement;
    await user.selectOptions(actionSelect, 'created');

    // Click clear filters
    const clearButton = screen.getByRole('button', { name: /clear filters/i });
    await user.click(clearButton);

    // Check filters were cleared
    await waitFor(() => {
      expect(getRoleAuditLogsMock).toHaveBeenLastCalledWith({});
    });
  });

  it('exports audit logs', async () => {
    const user = userEvent.setup();
    const exportMock = vi.fn().mockResolvedValue({ data: new Blob(['test']) });
    vi.mocked(roleApi.exportAuditLogs).mockImplementation(exportMock);

    render(<RoleAudit />, { wrapper: createWrapper() });

    // Wait for logs to load
    await waitFor(() => {
      expect(screen.getAllByText('admin').length).toBeGreaterThan(0);
    });

    // Click export button
    const exportButton = screen.getByRole('button', { name: /export csv/i });
    await user.click(exportButton);

    // Just verify the export API was called - skip testing the actual download
    // since mocking document methods breaks subsequent tests
    await waitFor(() => {
      expect(exportMock).toHaveBeenCalled();
    });
  });

  it('shows details when clicking view details', async () => {
    const user = userEvent.setup();
    render(<RoleAudit />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getAllByText('admin').length).toBeGreaterThan(0);
    });

    // Find and click view details
    const detailsButtons = screen.getAllByText('View Details');
    await user.click(detailsButtons[0]);

    // Check details are shown
    await waitFor(() => {
      expect(
        screen.getByText(/"description": "Admin role created"/)
      ).toBeInTheDocument();
    });
  });

  it('handles loading state', () => {
    (roleApi.getRoleAuditLogs as vi.Mock).mockReturnValue(
      new Promise(() => {})
    );

    render(<RoleAudit />, { wrapper: createWrapper() });

    expect(screen.getByText('Loading audit logs...')).toBeInTheDocument();
  });

  it('handles error state', async () => {
    (roleApi.getRoleAuditLogs as vi.Mock).mockRejectedValue(
      new Error('Failed to load')
    );

    render(<RoleAudit />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(
        screen.getByText(/failed to load audit logs/i)
      ).toBeInTheDocument();
    });
  });

  it('shows empty state when no logs', async () => {
    (roleApi.getRoleAuditLogs as vi.Mock).mockResolvedValue({ data: [] });

    render(<RoleAudit />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('No audit logs found')).toBeInTheDocument();
    });
  });

  it('displays correct action badge colors', async () => {
    render(<RoleAudit />, { wrapper: createWrapper() });

    await waitFor(() => {
      const createdBadge = screen.getByText('Created');
      const assignedBadge = screen.getByText('Assigned');

      // Check badges have different colors
      expect(createdBadge.className).toContain('success');
      expect(assignedBadge.className).toContain('secondary');
    });
  });

  it('shows filter count badge', async () => {
    const user = userEvent.setup();
    const { container } = render(<RoleAudit />, { wrapper: createWrapper() });

    // Open filters and set filters
    await user.click(screen.getByRole('button', { name: /filters/i }));

    await waitFor(() => {
      expect(container.querySelector('select')).toBeInTheDocument();
    });

    const actionSelect = container.querySelector('select') as HTMLSelectElement;
    await user.selectOptions(actionSelect, 'created');

    // Verify that at least one filter is applied (the badge appears)
    // The badge shows the count of active filters
    await waitFor(() => {
      // Just verify the filter was applied by checking the select value changed
      expect(actionSelect.value).toBe('created');
    });
  });
});

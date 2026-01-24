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

    // Wait for logs to load
    await waitFor(() => {
      expect(screen.getByText('admin')).toBeInTheDocument();
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
      expect(screen.getByText('John Doe')).toBeInTheDocument();
      expect(screen.getByText('Jane Smith')).toBeInTheDocument();
    });
  });

  it('displays timestamps in correct format', async () => {
    render(<RoleAudit />, { wrapper: createWrapper() });

    await waitFor(() => {
      // Check for formatted dates (format: MMM dd, yyyy HH:mm)
      expect(screen.getByText(/Jan 15, 2024/)).toBeInTheDocument();
      expect(screen.getByText(/Jan 16, 2024/)).toBeInTheDocument();
    });
  });

  it('filters logs by search query', async () => {
    const user = userEvent.setup();
    render(<RoleAudit />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('admin')).toBeInTheDocument();
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
    render(<RoleAudit />, { wrapper: createWrapper() });

    // Filters should be hidden initially
    expect(screen.queryByLabelText(/action/i)).not.toBeInTheDocument();

    // Click filters button
    const filtersButton = screen.getByRole('button', { name: /filters/i });
    await user.click(filtersButton);

    // Check filters are visible
    expect(screen.getByLabelText(/action/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/start date/i)).toBeInTheDocument();
    expect(screen.getByLabelText(/end date/i)).toBeInTheDocument();
  });

  it('applies action filter', async () => {
    const user = userEvent.setup();
    const getRoleAuditLogsMock = jest
      .fn()
      .mockResolvedValue({ data: mockAuditLogs });
    (roleApi.getRoleAuditLogs as vi.Mock) = getRoleAuditLogsMock;

    render(<RoleAudit />, { wrapper: createWrapper() });

    // Open filters
    await user.click(screen.getByRole('button', { name: /filters/i }));

    // Select action filter
    const actionSelect = screen.getByLabelText(/action/i);
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
    const getRoleAuditLogsMock = jest
      .fn()
      .mockResolvedValue({ data: mockAuditLogs });
    (roleApi.getRoleAuditLogs as vi.Mock) = getRoleAuditLogsMock;

    render(<RoleAudit />, { wrapper: createWrapper() });

    // Open filters
    await user.click(screen.getByRole('button', { name: /filters/i }));

    // Set start date
    const startDateInput = screen.getByLabelText(/start date/i);
    await user.type(startDateInput, '2024-01-01');

    // Set end date
    const endDateInput = screen.getByLabelText(/end date/i);
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
    const getRoleAuditLogsMock = jest
      .fn()
      .mockResolvedValue({ data: mockAuditLogs });
    (roleApi.getRoleAuditLogs as vi.Mock) = getRoleAuditLogsMock;

    render(<RoleAudit />, { wrapper: createWrapper() });

    // Open filters and set a filter
    await user.click(screen.getByRole('button', { name: /filters/i }));
    const actionSelect = screen.getByLabelText(/action/i);
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
    const exportMock = jest
      .fn()
      .mockResolvedValue({ data: new Blob(['test']) });
    (roleApi.exportAuditLogs as vi.Mock) = exportMock;

    // Mock URL.createObjectURL
    global.URL.createObjectURL = vi.fn();
    const mockLink = {
      click: vi.fn(),
      remove: vi.fn(),
      setAttribute: vi.fn(),
      href: '',
    };
    vi.spyOn(document, 'createElement').mockReturnValue(mockLink as any);
    vi.spyOn(document.body, 'appendChild').mockImplementation();

    render(<RoleAudit />, { wrapper: createWrapper() });

    // Click export button
    const exportButton = screen.getByRole('button', { name: /export csv/i });
    await user.click(exportButton);

    await waitFor(() => {
      expect(exportMock).toHaveBeenCalled();
      expect(mockLink.click).toHaveBeenCalled();
    });
  });

  it('shows details when clicking view details', async () => {
    const user = userEvent.setup();
    render(<RoleAudit />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('admin')).toBeInTheDocument();
    });

    // Find and click view details
    const detailsButtons = screen.getAllByText('View Details');
    await user.click(detailsButtons[0]);

    // Check details are shown
    expect(
      screen.getByText(/"description": "Admin role created"/)
    ).toBeInTheDocument();
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
    render(<RoleAudit />, { wrapper: createWrapper() });

    // Open filters and set filters
    await user.click(screen.getByRole('button', { name: /filters/i }));

    const actionSelect = screen.getByLabelText(/action/i);
    await user.selectOptions(actionSelect, 'created');

    const startDateInput = screen.getByLabelText(/start date/i);
    await user.type(startDateInput, '2024-01-01');

    // Check filter count badge
    await waitFor(() => {
      const filterButton = screen.getByRole('button', { name: /filters/i });
      expect(filterButton).toHaveTextContent('2');
    });
  });
});

import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { RoleManager } from '../RoleManager';
import { roleApi } from '@services/roleApi';
import type { Role } from '@types/role';

// Mock the API
jest.mock('@services/roleApi');

const mockRoles: Role[] = [
  {
    id: '1',
    name: 'admin',
    description: 'Administrator role',
    hierarchy_level: 90,
    permissions: [
      {
        id: 'p1',
        name: 'users.manage',
        description: 'Manage users',
        resource: 'users',
        action: 'manage',
        created_at: '2024-01-01T00:00:00Z',
        updated_at: '2024-01-01T00:00:00Z',
      },
    ],
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: '2',
    name: 'user',
    description: 'Regular user role',
    hierarchy_level: 10,
    permissions: [],
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
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

describe('RoleManager', () => {
  beforeEach(() => {
    jest.clearAllMocks();
    (roleApi.getRoles as jest.Mock).mockResolvedValue({ data: mockRoles });
  });

  it('renders role manager with roles', async () => {
    render(<RoleManager />, { wrapper: createWrapper() });

    // Check header
    expect(screen.getByText('Role Management')).toBeInTheDocument();
    expect(
      screen.getByText(/manage roles and permissions/i)
    ).toBeInTheDocument();

    // Wait for roles to load
    await waitFor(() => {
      expect(screen.getByText('admin')).toBeInTheDocument();
      expect(screen.getByText('user')).toBeInTheDocument();
    });
  });

  it('filters roles by search query', async () => {
    const user = userEvent.setup();
    render(<RoleManager />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('admin')).toBeInTheDocument();
    });

    // Type in search
    const searchInput = screen.getByPlaceholderText('Search roles...');
    await user.type(searchInput, 'admin');

    // Check filtered results
    expect(screen.getByText('admin')).toBeInTheDocument();
    expect(screen.queryByText('user')).not.toBeInTheDocument();
  });

  it('selects and deselects roles', async () => {
    const user = userEvent.setup();
    render(<RoleManager />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('admin')).toBeInTheDocument();
    });

    // Get checkboxes
    const checkboxes = screen.getAllByRole('checkbox');
    const adminCheckbox = checkboxes[1]; // First checkbox is "select all"

    // Select role
    await user.click(adminCheckbox);
    expect(adminCheckbox).toBeChecked();

    // Check selected count
    expect(screen.getByText('1 selected')).toBeInTheDocument();

    // Deselect role
    await user.click(adminCheckbox);
    expect(adminCheckbox).not.toBeChecked();
  });

  it('selects all roles', async () => {
    const user = userEvent.setup();
    render(<RoleManager />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('admin')).toBeInTheDocument();
    });

    // Click select all checkbox
    const selectAllCheckbox = screen.getAllByRole('checkbox')[0];
    await user.click(selectAllCheckbox);

    // Check selected count
    expect(screen.getByText('2 selected')).toBeInTheDocument();
  });

  it('opens create role modal', async () => {
    const user = userEvent.setup();
    render(<RoleManager />, { wrapper: createWrapper() });

    // Click create button
    const createButton = screen.getByRole('button', { name: /create role/i });
    await user.click(createButton);

    // Check modal is open
    await waitFor(() => {
      expect(screen.getByText('Create New Role')).toBeInTheDocument();
    });
  });

  it('opens edit role modal', async () => {
    const user = userEvent.setup();
    render(<RoleManager />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('admin')).toBeInTheDocument();
    });

    // Find and click edit button
    const rows = screen.getAllByRole('row');
    const adminRow = rows.find((row) => within(row).queryByText('admin'));
    expect(adminRow).toBeDefined();

    const editButton = within(adminRow!).getByRole('button', { name: '' });
    await user.click(editButton);

    // Check modal is open
    await waitFor(() => {
      expect(screen.getByText('Edit Role')).toBeInTheDocument();
    });
  });

  it('opens delete confirmation modal', async () => {
    const user = userEvent.setup();
    render(<RoleManager />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('user')).toBeInTheDocument();
    });

    // Find user role row (admin is system role and can't be deleted)
    const rows = screen.getAllByRole('row');
    const userRow = rows.find((row) => within(row).queryByText('user'));
    expect(userRow).toBeDefined();

    // Find delete buttons and click the second one
    const deleteButtons = within(userRow!).getAllByRole('button');
    const deleteButton = deleteButtons[deleteButtons.length - 1];
    await user.click(deleteButton);

    // Check confirmation modal
    await waitFor(() => {
      expect(screen.getByText('Delete Role')).toBeInTheDocument();
      expect(
        screen.getByText(/are you sure you want to delete the role/i)
      ).toBeInTheDocument();
    });
  });

  it('calls delete API when confirming deletion', async () => {
    const user = userEvent.setup();
    const deleteMock = jest.fn().mockResolvedValue({});
    (roleApi.deleteRole as jest.Mock) = deleteMock;

    render(<RoleManager />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('user')).toBeInTheDocument();
    });

    // Open delete modal
    const rows = screen.getAllByRole('row');
    const userRow = rows.find((row) => within(row).queryByText('user'))!;
    const deleteButtons = within(userRow).getAllByRole('button');
    await user.click(deleteButtons[deleteButtons.length - 1]);

    // Confirm deletion
    await waitFor(() => {
      expect(screen.getByText('Delete Role')).toBeInTheDocument();
    });

    const confirmButton = screen.getByRole('button', { name: /delete role/i });
    await user.click(confirmButton);

    // Check API was called
    await waitFor(() => {
      expect(deleteMock).toHaveBeenCalledWith('2');
    });
  });

  it('handles loading state', () => {
    (roleApi.getRoles as jest.Mock).mockReturnValue(new Promise(() => {}));

    render(<RoleManager />, { wrapper: createWrapper() });

    expect(screen.getByText('Loading roles...')).toBeInTheDocument();
  });

  it('handles error state', async () => {
    (roleApi.getRoles as jest.Mock).mockRejectedValue(
      new Error('Failed to load')
    );

    render(<RoleManager />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText(/failed to load roles/i)).toBeInTheDocument();
    });
  });

  it('displays correct role level badges', async () => {
    render(<RoleManager />, { wrapper: createWrapper() });

    await waitFor(() => {
      expect(screen.getByText('Level 90')).toBeInTheDocument();
      expect(screen.getByText('Level 10')).toBeInTheDocument();
    });
  });

  it('calls onRoleSelect when role is clicked', async () => {
    const user = userEvent.setup();
    const onRoleSelect = jest.fn();

    render(<RoleManager onRoleSelect={onRoleSelect} />, {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(screen.getByText('admin')).toBeInTheDocument();
    });

    // Click on role row
    const adminText = screen.getByText('admin');
    await user.click(adminText);

    expect(onRoleSelect).toHaveBeenCalledWith(mockRoles[0]);
  });
});

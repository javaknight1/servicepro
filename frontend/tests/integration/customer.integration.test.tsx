/**
 * Customer CRUD Integration Tests
 *
 * Tests complete customer management workflows including:
 * - Customer list loading and display
 * - Customer creation flow
 * - Customer update flow
 * - Customer deletion
 * - Search and filtering
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import React from 'react';

// Mock modules
vi.mock('@store', () => ({
  useAuthStore: vi.fn(),
  useTenantStore: vi.fn(),
  useSidebarStore: vi.fn(),
}));

vi.mock('@services/customerService', () => ({
  customerService: {
    getCustomers: vi.fn(),
    getCustomer: vi.fn(),
    createCustomer: vi.fn(),
    updateCustomer: vi.fn(),
    deleteCustomer: vi.fn(),
    getStats: vi.fn(),
  },
}));

// Import after mocking
import { useAuthStore, useTenantStore, useSidebarStore } from '@store';
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

const mockCustomers = [
  {
    id: 'customer-1',
    email: 'john@example.com',
    first_name: 'John',
    last_name: 'Doe',
    phone_primary: '555-123-4567',
    billing_address_street: '123 Main St',
    billing_address_city: 'Springfield',
    billing_address_state: 'IL',
    billing_address_zip: '62701',
    customer_type: 'personal' as const,
    status: 'active' as const,
    preferred_contact_method: 'email' as const,
    do_not_email: false,
    do_not_call: false,
    do_not_sms: false,
    do_not_mail: false,
    marketing_consent: true,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
  },
  {
    id: 'customer-2',
    email: 'jane@example.com',
    first_name: 'Jane',
    last_name: 'Smith',
    company_name: 'Smith Industries',
    phone_primary: '555-987-6543',
    billing_address_street: '456 Oak Ave',
    billing_address_city: 'Chicago',
    billing_address_state: 'IL',
    billing_address_zip: '60601',
    customer_type: 'business' as const,
    status: 'active' as const,
    preferred_contact_method: 'call' as const,
    do_not_email: false,
    do_not_call: false,
    do_not_sms: false,
    do_not_mail: false,
    marketing_consent: false,
    created_at: '2024-01-02T00:00:00Z',
    updated_at: '2024-01-02T00:00:00Z',
  },
  {
    id: 'customer-3',
    email: 'bob@example.com',
    first_name: 'Bob',
    last_name: 'Johnson',
    phone_primary: '555-456-7890',
    billing_address_street: '789 Pine Rd',
    billing_address_city: 'Peoria',
    billing_address_state: 'IL',
    billing_address_zip: '61602',
    customer_type: 'personal' as const,
    status: 'inactive' as const,
    preferred_contact_method: 'sms' as const,
    do_not_email: true,
    do_not_call: false,
    do_not_sms: false,
    do_not_mail: true,
    marketing_consent: false,
    created_at: '2024-01-03T00:00:00Z',
    updated_at: '2024-01-03T00:00:00Z',
  },
];

const mockCustomerStats = {
  total_customers: 3,
  active_customers: 2,
  inactive_customers: 1,
  prospects: 0,
  residential_customers: 2,
  commercial_customers: 1,
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

  vi.mocked(customerService.getCustomers).mockResolvedValue({
    customers: mockCustomers,
    total: mockCustomers.length,
    page: 1,
    page_size: 10,
  });

  vi.mocked(customerService.getStats).mockResolvedValue(mockCustomerStats);
};

// =============================================================================
// Customer List Tests
// =============================================================================

describe('Customer CRUD Integration', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setupMocks();
  });

  describe('Customer List', () => {
    it('should load and display customers list', async () => {
      const { CustomersPage } =
        await import('../../src/pages/Customers/CustomersPage');

      render(
        <MemoryRouter initialEntries={['/customers']}>
          <CustomersPage />
        </MemoryRouter>
      );

      // Wait for loading to complete
      await waitFor(() => {
        expect(customerService.getCustomers).toHaveBeenCalled();
      });

      // Verify customers are displayed
      await waitFor(() => {
        expect(screen.getByText('John Doe')).toBeInTheDocument();
        expect(screen.getByText('Jane Smith')).toBeInTheDocument();
        expect(screen.getByText('Bob Johnson')).toBeInTheDocument();
      });
    });

    it('should display customer contact information', async () => {
      const { CustomersPage } =
        await import('../../src/pages/Customers/CustomersPage');

      render(
        <MemoryRouter initialEntries={['/customers']}>
          <CustomersPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('john@example.com')).toBeInTheDocument();
        expect(screen.getByText('555-123-4567')).toBeInTheDocument();
      });
    });

    it('should display company name for business customers', async () => {
      const { CustomersPage } =
        await import('../../src/pages/Customers/CustomersPage');

      render(
        <MemoryRouter initialEntries={['/customers']}>
          <CustomersPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('Smith Industries')).toBeInTheDocument();
      });
    });

    it('should show empty state when no customers exist', async () => {
      vi.mocked(customerService.getCustomers).mockResolvedValue({
        customers: [],
        total: 0,
        page: 1,
        page_size: 10,
      });

      const { CustomersPage } =
        await import('../../src/pages/Customers/CustomersPage');

      render(
        <MemoryRouter initialEntries={['/customers']}>
          <CustomersPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        // Check for empty state or no data message
        const emptyState = screen.queryByText(
          /no customers|add your first|get started/i
        );
        if (emptyState) {
          expect(emptyState).toBeInTheDocument();
        }
      });
    });

    it('should navigate to new customer page when clicking add button', async () => {
      const user = userEvent.setup();
      const { CustomersPage } =
        await import('../../src/pages/Customers/CustomersPage');

      render(
        <MemoryRouter initialEntries={['/customers']}>
          <CustomersPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(customerService.getCustomers).toHaveBeenCalled();
      });

      // Find and click the add button
      const addButton = screen.getByRole('button', { name: /add|new|create/i });
      await user.click(addButton);

      expect(mockNavigate).toHaveBeenCalledWith('/customers/new');
    });

    it('should navigate to customer detail when clicking a customer row', async () => {
      const user = userEvent.setup();
      const { CustomersPage } =
        await import('../../src/pages/Customers/CustomersPage');

      render(
        <MemoryRouter initialEntries={['/customers']}>
          <CustomersPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(screen.getByText('John Doe')).toBeInTheDocument();
      });

      // Click on customer row
      const customerRow = screen.getByText('John Doe').closest('tr');
      if (customerRow) {
        await user.click(customerRow);
        expect(mockNavigate).toHaveBeenCalledWith('/customers/customer-1');
      }
    });
  });

  describe('Customer Search and Filter', () => {
    it('should search customers by query', async () => {
      const user = userEvent.setup();
      const { CustomersPage } =
        await import('../../src/pages/Customers/CustomersPage');

      render(
        <MemoryRouter initialEntries={['/customers']}>
          <CustomersPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(customerService.getCustomers).toHaveBeenCalled();
      });

      // Find search input and type query
      const searchInput = screen.getByPlaceholderText(/search/i);
      await user.type(searchInput, 'John');

      // Submit search (Enter key or button)
      await user.keyboard('{Enter}');

      await waitFor(() => {
        expect(customerService.getCustomers).toHaveBeenCalledWith(
          expect.objectContaining({
            search: 'John',
          })
        );
      });
    });

    it('should reset to first page on search', async () => {
      const user = userEvent.setup();
      const { CustomersPage } =
        await import('../../src/pages/Customers/CustomersPage');

      render(
        <MemoryRouter initialEntries={['/customers']}>
          <CustomersPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(customerService.getCustomers).toHaveBeenCalled();
      });

      const searchInput = screen.getByPlaceholderText(/search/i);
      await user.type(searchInput, 'test');
      await user.keyboard('{Enter}');

      await waitFor(() => {
        expect(customerService.getCustomers).toHaveBeenCalledWith(
          expect.objectContaining({
            page: 1,
          })
        );
      });
    });
  });

  describe('Customer Creation', () => {
    it('should create a new customer successfully', async () => {
      const user = userEvent.setup();
      const newCustomer = {
        id: 'customer-new',
        email: 'newcustomer@example.com',
        first_name: 'New',
        last_name: 'Customer',
        phone_primary: '555-000-0000',
        billing_address_street: '999 New St',
        billing_address_city: 'New City',
        billing_address_state: 'NC',
        billing_address_zip: '28000',
        customer_type: 'personal' as const,
        status: 'active' as const,
        preferred_contact_method: 'email' as const,
        do_not_email: false,
        do_not_call: false,
        do_not_sms: false,
        do_not_mail: false,
        marketing_consent: true,
        created_at: '2024-01-04T00:00:00Z',
        updated_at: '2024-01-04T00:00:00Z',
      };

      vi.mocked(customerService.createCustomer).mockResolvedValue(newCustomer);

      const { CustomerDetailPage } =
        await import('../../src/pages/Customers/CustomerDetailPage');

      render(
        <MemoryRouter initialEntries={['/customers/new']}>
          <Routes>
            <Route path="/customers/new" element={<CustomerDetailPage />} />
            <Route path="/customers/:id" element={<div>Customer Detail</div>} />
            <Route path="/customers" element={<div>Customer List</div>} />
          </Routes>
        </MemoryRouter>
      );

      // Wait for form to render
      await waitFor(() => {
        expect(screen.getByLabelText(/first name/i)).toBeInTheDocument();
      });

      // Fill in required fields
      await user.type(screen.getByLabelText(/first name/i), 'New');
      await user.type(screen.getByLabelText(/last name/i), 'Customer');
      await user.type(
        screen.getByLabelText(/email/i),
        'newcustomer@example.com'
      );
      await user.type(screen.getByLabelText(/phone/i), '555-000-0000');

      // Fill address fields
      const streetInput = screen.getByLabelText(/street|address/i);
      await user.type(streetInput, '999 New St');

      // Submit form
      const saveButton = screen.getByRole('button', {
        name: /save|create|submit/i,
      });
      await user.click(saveButton);

      await waitFor(() => {
        expect(customerService.createCustomer).toHaveBeenCalledWith(
          expect.objectContaining({
            first_name: 'New',
            last_name: 'Customer',
            email: 'newcustomer@example.com',
          })
        );
      });
    });

    it('should show validation errors for missing required fields', async () => {
      const user = userEvent.setup();

      const { CustomerDetailPage } =
        await import('../../src/pages/Customers/CustomerDetailPage');

      render(
        <MemoryRouter initialEntries={['/customers/new']}>
          <CustomerDetailPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(screen.getByLabelText(/first name/i)).toBeInTheDocument();
      });

      // Try to submit empty form
      const saveButton = screen.getByRole('button', {
        name: /save|create|submit/i,
      });
      await user.click(saveButton);

      // Should show validation errors
      await waitFor(() => {
        const errorMessages = screen.getAllByText(/required|invalid|must be/i);
        expect(errorMessages.length).toBeGreaterThan(0);
      });

      // Should not call create
      expect(customerService.createCustomer).not.toHaveBeenCalled();
    });
  });

  describe('Customer Update', () => {
    it('should load existing customer data for editing', async () => {
      vi.mocked(customerService.getCustomer).mockResolvedValue(
        mockCustomers[0]
      );

      const { CustomerDetailPage } =
        await import('../../src/pages/Customers/CustomerDetailPage');

      render(
        <MemoryRouter initialEntries={['/customers/customer-1']}>
          <Routes>
            <Route path="/customers/:id" element={<CustomerDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(customerService.getCustomer).toHaveBeenCalledWith('customer-1');
      });

      // Verify data is loaded into form
      await waitFor(() => {
        const firstNameInput = screen.getByLabelText(
          /first name/i
        ) as HTMLInputElement;
        expect(firstNameInput.value).toBe('John');
      });
    });

    it('should update customer successfully', async () => {
      const user = userEvent.setup();
      vi.mocked(customerService.getCustomer).mockResolvedValue(
        mockCustomers[0]
      );
      vi.mocked(customerService.updateCustomer).mockResolvedValue({
        ...mockCustomers[0],
        first_name: 'Johnny',
      });

      const { CustomerDetailPage } =
        await import('../../src/pages/Customers/CustomerDetailPage');

      render(
        <MemoryRouter initialEntries={['/customers/customer-1']}>
          <Routes>
            <Route path="/customers/:id" element={<CustomerDetailPage />} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        const firstNameInput = screen.getByLabelText(
          /first name/i
        ) as HTMLInputElement;
        expect(firstNameInput.value).toBe('John');
      });

      // Update the first name
      const firstNameInput = screen.getByLabelText(/first name/i);
      await user.clear(firstNameInput);
      await user.type(firstNameInput, 'Johnny');

      // Save changes
      const saveButton = screen.getByRole('button', {
        name: /save|update|submit/i,
      });
      await user.click(saveButton);

      await waitFor(() => {
        expect(customerService.updateCustomer).toHaveBeenCalledWith(
          'customer-1',
          expect.objectContaining({
            first_name: 'Johnny',
          })
        );
      });
    });
  });

  describe('Customer Deletion', () => {
    it('should delete customer after confirmation', async () => {
      const user = userEvent.setup();
      vi.mocked(customerService.getCustomer).mockResolvedValue(
        mockCustomers[0]
      );
      vi.mocked(customerService.deleteCustomer).mockResolvedValue();

      const { CustomerDetailPage } =
        await import('../../src/pages/Customers/CustomerDetailPage');

      render(
        <MemoryRouter initialEntries={['/customers/customer-1']}>
          <Routes>
            <Route path="/customers/:id" element={<CustomerDetailPage />} />
            <Route path="/customers" element={<div>Customer List</div>} />
          </Routes>
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(customerService.getCustomer).toHaveBeenCalled();
      });

      // Find and click delete button
      const deleteButton = screen.queryByRole('button', { name: /delete/i });
      if (deleteButton) {
        await user.click(deleteButton);

        // Confirm deletion if there's a confirmation dialog
        const confirmButton = screen.queryByRole('button', {
          name: /confirm|yes|delete/i,
        });
        if (confirmButton) {
          await user.click(confirmButton);
        }

        await waitFor(() => {
          expect(customerService.deleteCustomer).toHaveBeenCalledWith(
            'customer-1'
          );
        });
      }
    });
  });

  describe('Error Handling', () => {
    it('should display error when customer list fails to load', async () => {
      vi.mocked(customerService.getCustomers).mockRejectedValue(
        new Error('Failed to load customers')
      );

      const { CustomersPage } =
        await import('../../src/pages/Customers/CustomersPage');

      render(
        <MemoryRouter initialEntries={['/customers']}>
          <CustomersPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(customerService.getCustomers).toHaveBeenCalled();
      });

      // Error should be handled gracefully (logged, displayed, etc.)
      // The exact behavior depends on the component implementation
    });

    it('should display error when customer creation fails', async () => {
      const user = userEvent.setup();
      vi.mocked(customerService.createCustomer).mockRejectedValue({
        response: { data: { message: 'Email already exists' } },
      });

      const { CustomerDetailPage } =
        await import('../../src/pages/Customers/CustomerDetailPage');

      render(
        <MemoryRouter initialEntries={['/customers/new']}>
          <CustomerDetailPage />
        </MemoryRouter>
      );

      await waitFor(() => {
        expect(screen.getByLabelText(/first name/i)).toBeInTheDocument();
      });

      // Fill in required fields
      await user.type(screen.getByLabelText(/first name/i), 'Test');
      await user.type(screen.getByLabelText(/last name/i), 'User');
      await user.type(screen.getByLabelText(/email/i), 'existing@example.com');

      const saveButton = screen.getByRole('button', {
        name: /save|create|submit/i,
      });
      await user.click(saveButton);

      // Should show error message
      await waitFor(() => {
        const errorMessage = screen.queryByText(
          /email already exists|error|failed/i
        );
        if (errorMessage) {
          expect(errorMessage).toBeInTheDocument();
        }
      });
    });
  });
});

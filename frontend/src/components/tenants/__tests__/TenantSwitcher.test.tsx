import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import React from 'react';
import { TenantSwitcher } from '../TenantSwitcher';
import { useTenantStore } from '@/store';

// Mock the tenant store
vi.mock('@/store', () => ({
  useTenantStore: vi.fn(),
}));

const mockTenants = [
  {
    id: 'tenant-1',
    name: 'Test Org 1',
    slug: 'test-org-1',
    is_active: true,
    created_at: '2024-01-01T00:00:00Z',
    updated_at: '2024-01-01T00:00:00Z',
    role: 'Admin',
  },
  {
    id: 'tenant-2',
    name: 'Test Org 2',
    slug: 'test-org-2',
    is_active: true,
    created_at: '2024-01-02T00:00:00Z',
    updated_at: '2024-01-02T00:00:00Z',
    role: 'Member',
  },
];

const renderWithRouter = (ui: React.ReactElement) => {
  return render(<BrowserRouter>{ui}</BrowserRouter>);
};

describe('TenantSwitcher', () => {
  const mockLoadTenants = vi.fn().mockResolvedValue(undefined);
  const mockSwitchTenant = vi.fn().mockResolvedValue(undefined);

  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(useTenantStore).mockReturnValue({
      currentTenant: mockTenants[0],
      tenants: mockTenants,
      isLoading: false,
      loadTenants: mockLoadTenants,
      switchTenant: mockSwitchTenant,
    } as ReturnType<typeof useTenantStore>);
  });

  describe('rendering', () => {
    it('should render current tenant name', () => {
      renderWithRouter(<TenantSwitcher />);

      expect(screen.getByText('Test Org 1')).toBeInTheDocument();
    });

    it('should render dropdown trigger button', () => {
      renderWithRouter(<TenantSwitcher />);

      const button = screen.getByRole('button');
      expect(button).toBeInTheDocument();
    });

    it('should apply custom className', () => {
      const { container } = renderWithRouter(
        <TenantSwitcher className="custom-class" />
      );

      expect(container.firstChild).toHaveClass('custom-class');
    });
  });

  describe('loading state', () => {
    it('should show loading state when loading with no tenants', () => {
      vi.mocked(useTenantStore).mockReturnValue({
        currentTenant: null,
        tenants: [],
        isLoading: true,
        loadTenants: mockLoadTenants,
        switchTenant: mockSwitchTenant,
      } as ReturnType<typeof useTenantStore>);

      renderWithRouter(<TenantSwitcher />);

      expect(screen.getByText('Loading...')).toBeInTheDocument();
    });
  });

  describe('empty state', () => {
    it('should show create organization link when no tenants', () => {
      vi.mocked(useTenantStore).mockReturnValue({
        currentTenant: null,
        tenants: [],
        isLoading: false,
        loadTenants: mockLoadTenants,
        switchTenant: mockSwitchTenant,
      } as ReturnType<typeof useTenantStore>);

      renderWithRouter(<TenantSwitcher />);

      expect(screen.getByText('Create Organization')).toBeInTheDocument();
      expect(screen.getByRole('link')).toHaveAttribute(
        'href',
        '/settings/organization/new'
      );
    });
  });

  describe('dropdown behavior', () => {
    it('should open dropdown when button is clicked', () => {
      renderWithRouter(<TenantSwitcher />);

      const button = screen.getByRole('button');
      fireEvent.click(button);

      expect(screen.getByText('Organizations')).toBeInTheDocument();
    });

    it('should show all tenants in dropdown', () => {
      renderWithRouter(<TenantSwitcher />);

      const button = screen.getByRole('button');
      fireEvent.click(button);

      // Test Org 1 appears twice (in trigger and dropdown), Test Org 2 only in dropdown
      expect(screen.getAllByText('Test Org 1').length).toBeGreaterThanOrEqual(
        1
      );
      expect(screen.getByText('Test Org 2')).toBeInTheDocument();
    });

    it('should show roles for tenants', () => {
      renderWithRouter(<TenantSwitcher />);

      const button = screen.getByRole('button');
      fireEvent.click(button);

      expect(screen.getByText('Admin')).toBeInTheDocument();
      expect(screen.getByText('Member')).toBeInTheDocument();
    });

    it('should close dropdown when clicking button again', () => {
      renderWithRouter(<TenantSwitcher />);

      const button = screen.getByRole('button');

      // Open
      fireEvent.click(button);
      expect(screen.getByText('Organizations')).toBeInTheDocument();

      // Close
      fireEvent.click(button);
      expect(screen.queryByText('Organizations')).not.toBeInTheDocument();
    });

    it('should close dropdown when clicking outside', async () => {
      renderWithRouter(
        <div>
          <div data-testid="outside">Outside</div>
          <TenantSwitcher />
        </div>
      );

      const button = screen.getByRole('button');
      fireEvent.click(button);

      expect(screen.getByText('Organizations')).toBeInTheDocument();

      fireEvent.mouseDown(screen.getByTestId('outside'));

      await waitFor(() => {
        expect(screen.queryByText('Organizations')).not.toBeInTheDocument();
      });
    });
  });

  describe('tenant selection', () => {
    it('should highlight current tenant', () => {
      renderWithRouter(<TenantSwitcher />);

      const triggerButton = screen.getByRole('button');
      fireEvent.click(triggerButton);

      // The current tenant button in the dropdown should have the highlighted class
      // Find all buttons, filter for ones with 'w-full' class which are dropdown items
      const dropdownButtons = screen
        .getAllByRole('button')
        .filter((btn) => btn.className.includes('w-full'));
      const currentTenantButton = dropdownButtons.find((btn) =>
        btn.textContent?.includes('Test Org 1')
      );
      expect(currentTenantButton).toHaveClass('bg-primary-50');
    });

    it('should call switchTenant when selecting a different tenant', async () => {
      // Mock window.location.reload
      const reloadMock = vi.fn();
      Object.defineProperty(window, 'location', {
        value: { reload: reloadMock },
        writable: true,
      });

      renderWithRouter(<TenantSwitcher />);

      const triggerButton = screen.getByRole('button');
      fireEvent.click(triggerButton);

      // Click on the second tenant
      const tenant2Button = screen
        .getAllByRole('button')
        .find((btn) => btn.textContent?.includes('Test Org 2'));

      if (tenant2Button) {
        fireEvent.click(tenant2Button);
      }

      await waitFor(() => {
        expect(mockSwitchTenant).toHaveBeenCalledWith('tenant-2');
      });
    });

    it('should not call switchTenant when selecting current tenant', async () => {
      renderWithRouter(<TenantSwitcher />);

      const triggerButton = screen.getByRole('button');
      fireEvent.click(triggerButton);

      // Click on the current tenant
      const tenant1Button = screen
        .getAllByRole('button')
        .find(
          (btn) =>
            btn.textContent?.includes('Test Org 1') && btn.type === 'button'
        );

      if (tenant1Button) {
        fireEvent.click(tenant1Button);
      }

      expect(mockSwitchTenant).not.toHaveBeenCalled();
    });
  });

  describe('create organization link', () => {
    it('should have create organization link in dropdown', () => {
      renderWithRouter(<TenantSwitcher />);

      const button = screen.getByRole('button');
      fireEvent.click(button);

      const createLinks = screen.getAllByRole('link');
      const createOrgLink = createLinks.find(
        (link) => link.textContent === 'Create Organization'
      );
      expect(createOrgLink).toBeInTheDocument();
      expect(createOrgLink).toHaveAttribute(
        'href',
        '/settings/organization/new'
      );
    });
  });

  describe('fallback text', () => {
    it('should show "Select Organization" when no current tenant', () => {
      vi.mocked(useTenantStore).mockReturnValue({
        currentTenant: null,
        tenants: mockTenants,
        isLoading: false,
        loadTenants: mockLoadTenants,
        switchTenant: mockSwitchTenant,
      } as ReturnType<typeof useTenantStore>);

      renderWithRouter(<TenantSwitcher />);

      expect(screen.getByText('Select Organization')).toBeInTheDocument();
    });
  });

  describe('load tenants on mount', () => {
    it('should call loadTenants when tenants array is empty on mount', () => {
      vi.mocked(useTenantStore).mockReturnValue({
        currentTenant: null,
        tenants: [],
        isLoading: false,
        loadTenants: mockLoadTenants,
        switchTenant: mockSwitchTenant,
      } as ReturnType<typeof useTenantStore>);

      renderWithRouter(<TenantSwitcher />);

      expect(mockLoadTenants).toHaveBeenCalled();
    });

    it('should not call loadTenants when tenants already exist', () => {
      renderWithRouter(<TenantSwitcher />);

      expect(mockLoadTenants).not.toHaveBeenCalled();
    });
  });
});

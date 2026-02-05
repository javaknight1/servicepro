/**
 * Integration Test Setup & Utilities
 *
 * Common utilities for integration testing critical user workflows.
 * These tests verify interactions between multiple components and services.
 */

import React from 'react';
import { render, RenderOptions, RenderResult } from '@testing-library/react';
import { MemoryRouter, Routes, Route } from 'react-router-dom';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { vi } from 'vitest';
import type { User } from '../../src/types';

// =============================================================================
// Mock Data Factories
// =============================================================================

export const mockUser: User = {
  id: 'user-123',
  email: 'test@example.com',
  first_name: 'Test',
  last_name: 'User',
  is_active: true,
  email_verified: true,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

export const mockTenant = {
  id: 'tenant-123',
  name: 'Test Organization',
  slug: 'test-org',
  is_active: true,
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

export const mockCustomer = {
  id: 'customer-123',
  email: 'customer@example.com',
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
};

export const mockJob = {
  id: 'job-123',
  job_number: 'JOB-001',
  customer_id: 'customer-123',
  title: 'HVAC Installation',
  description: 'Install new HVAC system',
  job_type: 'installation' as const,
  status: 'new' as const,
  priority: 'normal' as const,
  customer: {
    id: 'customer-123',
    email: 'customer@example.com',
    first_name: 'John',
    last_name: 'Doe',
  },
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

export const mockQuote = {
  id: 'quote-123',
  quote_number: 'QT-001',
  customer_id: 'customer-123',
  customer: {
    id: 'customer-123',
    email: 'customer@example.com',
    first_name: 'John',
    last_name: 'Doe',
  },
  status: 'draft' as const,
  subtotal: 1000,
  tax_rate: 0.08,
  tax_amount: 80,
  total_amount: 1080,
  valid_until: '2024-12-31T00:00:00Z',
  lines: [],
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

export const mockInvoice = {
  id: 'invoice-123',
  invoice_number: 'INV-001',
  customer_id: 'customer-123',
  customer: {
    id: 'customer-123',
    email: 'customer@example.com',
    first_name: 'John',
    last_name: 'Doe',
  },
  status: 'draft' as const,
  issue_date: '2024-01-01',
  due_date: '2024-01-31',
  subtotal: 1000,
  tax_rate: 0.08,
  tax_amount: 80,
  total_amount: 1080,
  amount_paid: 0,
  amount_due: 1080,
  lines: [],
  created_at: '2024-01-01T00:00:00Z',
  updated_at: '2024-01-01T00:00:00Z',
};

// =============================================================================
// Auth Store Mock
// =============================================================================

export interface MockAuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
}

export const createMockAuthStore = (
  initialState: Partial<MockAuthState> = {}
) => {
  const defaultState: MockAuthState = {
    user: mockUser,
    isAuthenticated: true,
    isLoading: false,
    error: null,
    ...initialState,
  };

  return {
    ...defaultState,
    login: vi.fn().mockResolvedValue(undefined),
    logout: vi.fn().mockResolvedValue(undefined),
    register: vi.fn().mockResolvedValue(undefined),
    loadUser: vi.fn().mockResolvedValue(undefined),
    clearError: vi.fn(),
  };
};

// =============================================================================
// Tenant Store Mock
// =============================================================================

export interface MockTenantState {
  currentTenant: typeof mockTenant | null;
  tenants: (typeof mockTenant)[];
  isLoading: boolean;
  error: string | null;
}

export const createMockTenantStore = (
  initialState: Partial<MockTenantState> = {}
) => {
  const defaultState: MockTenantState = {
    currentTenant: mockTenant,
    tenants: [mockTenant],
    isLoading: false,
    error: null,
    ...initialState,
  };

  return {
    ...defaultState,
    loadTenants: vi.fn().mockResolvedValue(undefined),
    setCurrentTenant: vi.fn(),
    switchTenant: vi.fn().mockResolvedValue(undefined),
    createTenant: vi.fn().mockResolvedValue(mockTenant),
    updateTenant: vi.fn().mockResolvedValue(mockTenant),
    deleteTenant: vi.fn().mockResolvedValue(undefined),
    clearError: vi.fn(),
    reset: vi.fn(),
  };
};

// =============================================================================
// API Mock Utilities
// =============================================================================

export const createMockApiResponse = <T>(data: T, status = 200) => ({
  data,
  status,
  statusText: 'OK',
  headers: {},
  config: {} as never,
});

export const createMockApiError = (
  message: string,
  status = 400,
  data?: Record<string, unknown>
) => {
  const error = new Error(message) as Error & {
    response?: {
      data: { message: string } & Record<string, unknown>;
      status: number;
    };
    isAxiosError: boolean;
  };
  error.response = {
    data: { message, ...data },
    status,
  };
  error.isAxiosError = true;
  return error;
};

// =============================================================================
// Render Utilities
// =============================================================================

interface IntegrationRenderOptions extends Omit<RenderOptions, 'wrapper'> {
  route?: string;
  routes?: React.ReactElement;
  authState?: Partial<MockAuthState>;
  tenantState?: Partial<MockTenantState>;
}

/**
 * Create a fresh QueryClient for each test
 */
export const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: {
      queries: {
        retry: false,
        gcTime: 0,
        staleTime: 0,
      },
      mutations: {
        retry: false,
      },
    },
  });

/**
 * Render a component with all necessary providers for integration testing
 */
export function renderWithProviders(
  ui: React.ReactElement,
  { route = '/', routes, ...renderOptions }: IntegrationRenderOptions = {}
): RenderResult & { queryClient: QueryClient } {
  const queryClient = createTestQueryClient();

  const Wrapper = ({ children }: { children: React.ReactNode }) => {
    return React.createElement(
      QueryClientProvider,
      { client: queryClient },
      React.createElement(
        MemoryRouter,
        { initialEntries: [route] },
        routes ? React.createElement(Routes, null, routes) : children
      )
    );
  };

  const result = render(ui, { wrapper: Wrapper, ...renderOptions });

  return {
    ...result,
    queryClient,
  };
}

/**
 * Render with routes for navigation testing
 */
export function renderWithRoutes(
  routes: React.ReactElement,
  { route = '/', ...options }: Omit<IntegrationRenderOptions, 'routes'> = {}
): RenderResult & { queryClient: QueryClient } {
  return renderWithProviders(
    React.createElement('div'), // Placeholder, routes handle rendering
    { route, routes, ...options }
  );
}

// =============================================================================
// Wait Utilities
// =============================================================================

/**
 * Wait for loading to complete (useful for async data fetching)
 */
export const waitForLoadingToComplete = async (
  queryByText: (text: string | RegExp) => HTMLElement | null,
  timeout = 5000
): Promise<void> => {
  const start = Date.now();
  while (Date.now() - start < timeout) {
    const loadingElement = queryByText(/loading/i);
    if (!loadingElement) return;
    await new Promise((resolve) => setTimeout(resolve, 50));
  }
};

// =============================================================================
// Local Storage Mock
// =============================================================================

export const createLocalStorageMock = () => {
  let store: Record<string, string> = {};

  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: vi.fn(() => {
      store = {};
    }),
    get store() {
      return store;
    },
  };
};

// =============================================================================
// Form Helpers
// =============================================================================

import userEvent from '@testing-library/user-event';

/**
 * Fill a form field by label
 */
export const fillFormField = async (
  user: ReturnType<typeof userEvent.setup>,
  getByLabelText: (text: string | RegExp) => HTMLElement,
  label: string | RegExp,
  value: string
) => {
  const field = getByLabelText(label);
  await user.clear(field);
  await user.type(field, value);
};

/**
 * Submit a form by button text
 */
export const submitForm = async (
  user: ReturnType<typeof userEvent.setup>,
  getByRole: (
    role: string,
    options?: { name?: string | RegExp }
  ) => HTMLElement,
  buttonText: string | RegExp = /submit/i
) => {
  const button = getByRole('button', { name: buttonText });
  await user.click(button);
};

// =============================================================================
// Re-export testing utilities
// =============================================================================

export { userEvent };
export { screen, waitFor, within, act } from '@testing-library/react';

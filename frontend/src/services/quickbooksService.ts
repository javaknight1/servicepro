import api from './api';
import type {
  QuickBooksConnectResponse,
  QuickBooksOptions,
  QuickBooksSettingsRequest,
  QuickBooksStatus,
  QuickBooksSyncResult,
} from '@/types/quickbooks';

const base = (tenantId: string) =>
  `/v1/tenants/${tenantId}/integrations/quickbooks`;

/** QuickBooks Online sync for an organization (T192). */
export const quickbooksApi = {
  /** Connection, mapping, counts, failed records and recent activity. */
  getStatus: (tenantId: string) => api.get<QuickBooksStatus>(base(tenantId)),

  /** A single-use Intuit consent URL to redirect to. */
  connect: (tenantId: string) =>
    api.post<QuickBooksConnectResponse>(`${base(tenantId)}/connect`),

  disconnect: (tenantId: string) =>
    api.post<QuickBooksStatus>(`${base(tenantId)}/disconnect`),

  /** Income accounts, tax codes and deposit accounts of the company. */
  getOptions: (tenantId: string) =>
    api.get<QuickBooksOptions>(`${base(tenantId)}/options`),

  updateSettings: (tenantId: string, body: QuickBooksSettingsRequest) =>
    api.put<QuickBooksStatus>(`${base(tenantId)}/settings`, body),

  syncNow: (tenantId: string) =>
    api.post<QuickBooksSyncResult>(`${base(tenantId)}/sync`),

  retry: (tenantId: string, recordId: string) =>
    api.post<QuickBooksStatus>(`${base(tenantId)}/records/${recordId}/retry`),
};

/** React Query key for an organization's QuickBooks status. */
export const quickbooksStatusQueryKey = (tenantId: string | null | undefined) =>
  ['quickbooks-status', tenantId ?? null] as const;

/** React Query key for the company's mapping options. */
export const quickbooksOptionsQueryKey = (
  tenantId: string | null | undefined
) => ['quickbooks-options', tenantId ?? null] as const;

/**
 * QuickBooks Online sync for an organization (T192):
 * /tenants/:id/integrations/quickbooks.
 */

export type QuickBooksConnectionStatus =
  'connected' | 'reauth_required' | 'disconnected';

export type QuickBooksEntity = 'customer' | 'invoice' | 'payment';

/** A QuickBooks account, item or tax code picked in the mapping. */
export interface QuickBooksRef {
  id: string;
  name: string;
}

export interface QuickBooksSyncCounts {
  customers: number;
  invoices: number;
  payments: number;
  pending: number;
  errors: number;
}

/** One record in the error list or the recent activity. */
export interface QuickBooksSyncRecord {
  id: string;
  entity: QuickBooksEntity;
  local_id: string;
  /** e.g. "Invoice INV-1042 · Harborview Property Management · $9,882.00" */
  label: string;
  status: 'pending' | 'synced' | 'error';
  attempts: number;
  last_error?: string;
  last_synced_at?: string;
  updated_at: string;
}

export interface QuickBooksStatus {
  /** False when the server has no Intuit app yet: show "Coming soon". */
  configured: boolean;
  status: QuickBooksConnectionStatus;
  environment?: string;
  company_name?: string;
  connected_at?: string;
  /** Invoices issued from this date on are synced (YYYY-MM-DD). */
  sync_from?: string;
  last_sync_at?: string;
  /** Connection-level problem (e.g. the authorization expired). */
  last_sync_error?: string;
  income_account?: QuickBooksRef;
  item?: QuickBooksRef;
  tax_code?: QuickBooksRef;
  deposit_account?: QuickBooksRef;
  counts: QuickBooksSyncCounts;
  errors: QuickBooksSyncRecord[];
  recent: QuickBooksSyncRecord[];
}

export interface QuickBooksOptions {
  income_accounts: QuickBooksRef[];
  tax_codes: QuickBooksRef[];
  deposit_accounts: QuickBooksRef[];
}

export interface QuickBooksSettingsRequest {
  income_account_id: string;
  tax_code_id: string;
  deposit_account_id: string;
}

export interface QuickBooksSyncResult {
  enqueued: number;
  synced: number;
  failed: number;
  skipped: number;
  more: boolean;
}

export interface QuickBooksConnectResponse {
  /** Intuit consent page (single use, 10 minutes). */
  url: string;
}

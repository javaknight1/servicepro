/**
 * Quote and Line Item Types
 * Matches backend API models
 */

export enum QuoteStatus {
  DRAFT = 'draft',
  SENT = 'sent',
  VIEWED = 'viewed',
  ACCEPTED = 'accepted',
  DECLINED = 'declined',
  EXPIRED = 'expired',
}

export interface LineItem {
  id: string;
  quote_id?: string;
  description: string;
  quantity: number;
  unit_price: number;
  total: number;
  sort_order: number;
}

export interface LineItemFormData {
  description: string;
  quantity: number | string;
  unit_price: number | string;
  sort_order?: number;
}

export interface Quote {
  id: string;
  customer_id: string;
  customer?: {
    id: string;
    email: string;
    name?: string;
  };
  quote_number: string;
  status: QuoteStatus;
  valid_until: string;
  is_expired: boolean;
  subtotal: number;
  tax_rate: number;
  tax_amount: number;
  total: number;
  notes?: string;
  terms?: string;
  items: LineItem[];
  created_by: string;
  updated_by?: string;
  created_at: string;
  updated_at: string;
}

export interface QuoteRequest {
  customer_id: string;
  valid_until: string;
  tax_rate: number;
  notes?: string;
  terms?: string;
  items: LineItemFormData[];
}

export interface QuoteListResponse {
  quotes: Quote[];
  total: number;
  page: number;
  page_size: number;
}

export interface QuoteStats {
  total_quotes: number;
  draft_quotes: number;
  sent_quotes: number;
  accepted_quotes: number;
  declined_quotes: number;
  expired_quotes: number;
}

export interface LineItemValidationErrors {
  description?: string;
  quantity?: string;
  unit_price?: string;
}

export interface BulkAction {
  type: 'delete' | 'duplicate' | 'move';
  itemIds: string[];
}

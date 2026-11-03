import api from './api';

/**
 * A customer's purchase order with its balance (T187,
 * models.PurchaseOrderResponse). Money is an API decimal string. Billed is
 * the total of the customer's issued invoices (not draft, not cancelled)
 * carrying the PO number; remaining = amount - billed (negative when over).
 */
export interface PurchaseOrder {
  id: string;
  customer_id: string;
  po_number: string;
  amount: string;
  billed: string;
  remaining: string;
  invoice_count: number;
  /** YYYY-MM-DD, inclusive. */
  expires_on?: string;
  expired: boolean;
  notes?: string;
  created_at: string;
  updated_at: string;
}

/** Body of POST /customers/{id}/purchase-orders. */
export interface PurchaseOrderInput {
  po_number: string;
  amount: string;
  expires_on?: string;
  notes?: string;
}

/** Body of PUT /customers/{id}/purchase-orders/{poId}. */
export interface PurchaseOrderUpdate {
  po_number?: string;
  amount?: string;
  expires_on?: string;
  clear_expires_on?: boolean;
  notes?: string;
}

interface PurchaseOrderListResponse {
  purchase_orders: PurchaseOrder[];
  total: number;
}

const basePath = (customerId: string) =>
  `/v1/customers/${customerId}/purchase-orders`;

/** Customer purchase orders (T187), scoped to the customer in the URL. */
export const purchaseOrderService = {
  async list(customerId: string): Promise<PurchaseOrder[]> {
    const response = await api.get<PurchaseOrderListResponse>(
      basePath(customerId)
    );
    return response.data.purchase_orders ?? [];
  },

  async create(
    customerId: string,
    data: PurchaseOrderInput
  ): Promise<PurchaseOrder> {
    const response = await api.post<PurchaseOrder>(basePath(customerId), data);
    return response.data;
  },

  async update(
    customerId: string,
    poId: string,
    data: PurchaseOrderUpdate
  ): Promise<PurchaseOrder> {
    const response = await api.put<PurchaseOrder>(
      `${basePath(customerId)}/${poId}`,
      data
    );
    return response.data;
  },

  async remove(customerId: string, poId: string): Promise<void> {
    await api.delete(`${basePath(customerId)}/${poId}`);
  },
};

/** React Query key for a customer's purchase orders. */
export const purchaseOrdersQueryKey = (customerId: string) =>
  ['customer-purchase-orders', customerId] as const;

import api from './api';
import type {
  CustomerType,
  CustomerStatus,
  PreferredContactMethod,
} from '../types/customer';
import { buildQueryParams } from '../utils/queryParams';

export interface Customer {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  display_name?: string;
  company_name?: string;
  phone_primary: string;
  phone_secondary?: string;
  billing_address_street: string;
  billing_address_city: string;
  billing_address_state: string;
  billing_address_zip: string;
  billing_address_full?: string;
  service_address_street?: string;
  service_address_city?: string;
  service_address_state?: string;
  service_address_zip?: string;
  service_address_full?: string;
  customer_type: CustomerType;
  status: CustomerStatus;
  notes?: string;
  // Contact preferences
  preferred_contact_method: PreferredContactMethod;
  do_not_email: boolean;
  do_not_call: boolean;
  do_not_sms: boolean;
  do_not_mail: boolean;
  // Marketing consent
  marketing_consent: boolean;
  marketing_consent_at?: string;
  // Billing rules (T187)
  po_required?: boolean;
  default_po_number?: string | null;
  billing_approver_name?: string | null;
  billing_approver_email?: string | null;
  /** Not-to-exceed limit for extra work, an API decimal ("500.00"); null = none. */
  nte_limit?: string | null;
  consolidated_billing?: boolean;
  created_at: string;
  updated_at: string;
}

/**
 * Billing rules on a customer update (models.BillingRulesRequest). Omitted
 * fields are left unchanged; "" clears a text field; `clear_nte_limit`
 * removes the not-to-exceed limit.
 */
export interface CustomerBillingRulesUpdate {
  po_required?: boolean;
  default_po_number?: string;
  billing_approver_name?: string;
  billing_approver_email?: string;
  nte_limit?: string;
  clear_nte_limit?: boolean;
  consolidated_billing?: boolean;
}

export interface CustomerFilters {
  search?: string;
  status?: CustomerStatus;
  customer_type?: CustomerType;
  page?: number;
  page_size?: number;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

export interface CustomerListResponse {
  customers: Customer[];
  total: number;
  page: number;
  page_size: number;
}

export type { CustomerStats } from '../types/customer';
import type { CustomerStats } from '../types/customer';

class CustomerService {
  private basePath = '/v1/customers';

  async getCustomers(
    filters: CustomerFilters = {}
  ): Promise<CustomerListResponse> {
    const params = buildQueryParams({
      search: filters.search,
      status: filters.status,
      customer_type: filters.customer_type,
      page: filters.page,
      page_size: filters.page_size,
      sort_by: filters.sort_by,
      sort_order: filters.sort_order,
    });

    const response = await api.get<CustomerListResponse>(this.basePath, {
      params,
    });
    return response.data;
  }

  async getCustomer(id: string): Promise<Customer> {
    const response = await api.get<Customer>(`${this.basePath}/${id}`);
    return response.data;
  }

  async createCustomer(data: Partial<Customer>): Promise<Customer> {
    const response = await api.post<Customer>(this.basePath, data);
    return response.data;
  }

  async updateCustomer(
    id: string,
    data: Partial<Customer> | CustomerBillingRulesUpdate
  ): Promise<Customer> {
    const response = await api.put<Customer>(`${this.basePath}/${id}`, data);
    return response.data;
  }

  async deleteCustomer(id: string): Promise<void> {
    await api.delete(`${this.basePath}/${id}`);
  }

  async getStats(): Promise<CustomerStats> {
    const response = await api.get<CustomerStats>(`${this.basePath}/stats`);
    return response.data;
  }
}

export const customerService = new CustomerService();

import api from './api';
import type { CustomerType, CustomerStatus } from '../types/customer';

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
  created_at: string;
  updated_at: string;
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

export interface CustomerStats {
  total_customers: number;
  active_customers: number;
  inactive_customers: number;
  prospects: number;
  residential_customers: number;
  commercial_customers: number;
}

class CustomerService {
  private basePath = '/v1/customers';

  async getCustomers(
    filters: CustomerFilters = {}
  ): Promise<CustomerListResponse> {
    const params = new URLSearchParams();

    if (filters.search) params.append('search', filters.search);
    if (filters.status) params.append('status', filters.status);
    if (filters.customer_type)
      params.append('customer_type', filters.customer_type);
    if (filters.page) params.append('page', filters.page.toString());
    if (filters.page_size)
      params.append('page_size', filters.page_size.toString());
    if (filters.sort_by) params.append('sort_by', filters.sort_by);
    if (filters.sort_order) params.append('sort_order', filters.sort_order);

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

  async updateCustomer(id: string, data: Partial<Customer>): Promise<Customer> {
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

// Customer Types

export type CustomerType = 'residential' | 'commercial';
export type CustomerStatus = 'active' | 'inactive' | 'prospect';
export type PreferredContactMethod = 'email' | 'phone' | 'sms' | 'mail' | 'any';

export interface Customer {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  company_name?: string;
  phone?: string;
  address?: string;
  city?: string;
  state?: string;
  zip_code?: string;
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

/** GET /customers/stats (T257). */
export interface CustomerStats {
  total_customers: number;
  active_customers: number;
  inactive_customers: number;
  prospects: number;
  by_status: Record<string, number>;
}

export const getCustomerStatusLabel = (status: CustomerStatus): string => {
  const labels: Record<CustomerStatus, string> = {
    active: 'Active',
    inactive: 'Inactive',
    prospect: 'Prospect',
  };
  return labels[status] || status;
};

export const getCustomerTypeLabel = (type: CustomerType): string => {
  const labels: Record<CustomerType, string> = {
    residential: 'Residential',
    commercial: 'Commercial',
  };
  return labels[type] || type;
};

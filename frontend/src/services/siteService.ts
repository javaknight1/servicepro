import api from './api';

/** One of a customer's service locations (T186, models.SiteResponse). */
export interface Site {
  id: string;
  customer_id: string;
  name: string;
  address_street: string;
  address_city: string;
  address_state: string;
  address_zip: string;
  /** The address on one line, built by the API. */
  address_full: string;
  access_notes?: string | null;
  contact_name?: string | null;
  contact_phone?: string | null;
  contact_email?: string | null;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

/** Body of POST /customers/{id}/sites (models.CreateSiteRequest). */
export interface SiteInput {
  name: string;
  address_street: string;
  address_city: string;
  address_state: string;
  address_zip: string;
  access_notes?: string;
  contact_name?: string;
  contact_phone?: string;
  contact_email?: string;
  is_active?: boolean;
}

interface SiteListResponse {
  sites: Site[];
  total: number;
}

const basePath = (customerId: string) => `/v1/customers/${customerId}/sites`;

/** Customer sites (T186). Every call is scoped to the customer in the URL. */
export const siteService = {
  async list(customerId: string, includeInactive = false): Promise<Site[]> {
    const response = await api.get<SiteListResponse>(basePath(customerId), {
      params: includeInactive ? { include_inactive: true } : undefined,
    });
    return response.data.sites ?? [];
  },

  async create(customerId: string, data: SiteInput): Promise<Site> {
    const response = await api.post<Site>(basePath(customerId), data);
    return response.data;
  },

  /** PUT: omitted fields are left unchanged. */
  async update(
    customerId: string,
    siteId: string,
    data: Partial<SiteInput>
  ): Promise<Site> {
    const response = await api.put<Site>(
      `${basePath(customerId)}/${siteId}`,
      data
    );
    return response.data;
  },
};

/** React Query key for a customer's sites. */
export const sitesQueryKey = (
  customerId: string | null | undefined,
  includeInactive = false
) => ['customer-sites', customerId ?? null, includeInactive] as const;

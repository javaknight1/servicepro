import api from './api';

export enum InvoiceStatus {
  DRAFT = 'draft',
  SENT = 'sent',
  VIEWED = 'viewed',
  PAID = 'paid',
  PARTIALLY_PAID = 'partially_paid',
  OVERDUE = 'overdue',
  CANCELLED = 'cancelled',
}

export interface InvoiceLineItem {
  id: string;
  invoice_id?: string;
  description: string;
  quantity: number;
  unit_price: number;
  total: number;
  sort_order: number;
}

export interface Invoice {
  id: string;
  invoice_number: string;
  customer_id: string;
  customer?: {
    id: string;
    email: string;
    first_name: string;
    last_name: string;
    company_name?: string;
  };
  job_id?: string;
  quote_id?: string;
  status: InvoiceStatus;
  issue_date: string;
  due_date: string;
  subtotal: number;
  tax_rate: number;
  tax_amount: number;
  total_amount: number;
  amount_paid: number;
  amount_due: number;
  notes?: string;
  terms?: string;
  lines: InvoiceLineItem[];
  created_at: string;
  updated_at: string;
}

export interface InvoiceFilters {
  customer_id?: string;
  job_id?: string;
  status?: InvoiceStatus;
  search?: string;
  overdue_only?: boolean;
  page?: number;
  page_size?: number;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

export interface InvoiceListResponse {
  invoices: Invoice[];
  total: number;
  page: number;
  page_size: number;
}

export interface InvoiceStats {
  total_invoices: number;
  draft_invoices: number;
  sent_invoices: number;
  paid_invoices: number;
  overdue_invoices: number;
  total_outstanding: number;
  total_revenue: number;
}

export interface PaymentRequest {
  amount: number;
  payment_method: string;
  reference?: string;
  notes?: string;
}

export interface PDFDownloadResponse {
  download_url: string;
  expires_at: string;
  filename: string;
}

class InvoiceService {
  private basePath = '/v1/invoices';

  async getInvoices(
    filters: InvoiceFilters = {}
  ): Promise<InvoiceListResponse> {
    const params = new URLSearchParams();

    if (filters.customer_id) params.append('customer_id', filters.customer_id);
    if (filters.job_id) params.append('job_id', filters.job_id);
    if (filters.status) params.append('status', filters.status);
    if (filters.search) params.append('search', filters.search);
    if (filters.overdue_only) params.append('overdue_only', 'true');
    if (filters.page) params.append('page', filters.page.toString());
    if (filters.page_size)
      params.append('page_size', filters.page_size.toString());
    if (filters.sort_by) params.append('sort_by', filters.sort_by);
    if (filters.sort_order) params.append('sort_order', filters.sort_order);

    const response = await api.get<InvoiceListResponse>(this.basePath, {
      params,
    });
    return response.data;
  }

  async getInvoice(id: string): Promise<Invoice> {
    const response = await api.get<Invoice>(`${this.basePath}/${id}`);
    return response.data;
  }

  async getInvoiceByNumber(invoiceNumber: string): Promise<Invoice> {
    const response = await api.get<Invoice>(
      `${this.basePath}/number/${invoiceNumber}`
    );
    return response.data;
  }

  async createInvoice(data: Partial<Invoice>): Promise<Invoice> {
    const response = await api.post<Invoice>(this.basePath, data);
    return response.data;
  }

  async updateInvoice(id: string, data: Partial<Invoice>): Promise<Invoice> {
    const response = await api.put<Invoice>(`${this.basePath}/${id}`, data);
    return response.data;
  }

  async deleteInvoice(id: string): Promise<void> {
    await api.delete(`${this.basePath}/${id}`);
  }

  async sendInvoice(id: string): Promise<Invoice> {
    const response = await api.post<Invoice>(`${this.basePath}/${id}/send`);
    return response.data;
  }

  async recordPayment(id: string, payment: PaymentRequest): Promise<Invoice> {
    const response = await api.post<Invoice>(
      `${this.basePath}/${id}/payment`,
      payment
    );
    return response.data;
  }

  async cancelInvoice(id: string): Promise<Invoice> {
    const response = await api.post<Invoice>(`${this.basePath}/${id}/cancel`);
    return response.data;
  }

  async getStats(): Promise<InvoiceStats> {
    const response = await api.get<InvoiceStats>(`${this.basePath}/stats`);
    return response.data;
  }

  async getCustomerInvoices(customerId: string): Promise<Invoice[]> {
    const response = await api.get<Invoice[]>(
      `/v1/customers/${customerId}/invoices`
    );
    return response.data;
  }

  /**
   * Get invoice PDF download URL (legacy)
   */
  async getInvoicePDF(id: string): Promise<PDFDownloadResponse> {
    const response = await api.get<PDFDownloadResponse>(
      `${this.basePath}/${id}/pdf`
    );
    return response.data;
  }

  /**
   * Get receipt PDF download URL (legacy)
   */
  async getReceiptPDF(id: string): Promise<PDFDownloadResponse> {
    const response = await api.get<PDFDownloadResponse>(
      `${this.basePath}/${id}/receipt/pdf`
    );
    return response.data;
  }

  /**
   * Download invoice PDF directly and trigger browser download
   */
  async downloadInvoicePDF(id: string, filename?: string): Promise<void> {
    const response = await api.get(`${this.basePath}/${id}/pdf`, {
      responseType: 'blob',
    });

    const contentDisposition = response.headers['content-disposition'];
    let downloadFilename = filename || `invoice-${id}.pdf`;
    if (contentDisposition) {
      const match = contentDisposition.match(/filename="?([^"]+)"?/);
      if (match) {
        downloadFilename = match[1];
      }
    }

    const blob = new Blob([response.data], { type: 'application/pdf' });
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = downloadFilename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(url);
  }

  /**
   * Download receipt PDF directly and trigger browser download
   */
  async downloadReceiptPDF(id: string, filename?: string): Promise<void> {
    const response = await api.get(`${this.basePath}/${id}/receipt/pdf`, {
      responseType: 'blob',
    });

    const contentDisposition = response.headers['content-disposition'];
    let downloadFilename = filename || `receipt-${id}.pdf`;
    if (contentDisposition) {
      const match = contentDisposition.match(/filename="?([^"]+)"?/);
      if (match) {
        downloadFilename = match[1];
      }
    }

    const blob = new Blob([response.data], { type: 'application/pdf' });
    const url = window.URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = downloadFilename;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    window.URL.revokeObjectURL(url);
  }
}

export const invoiceService = new InvoiceService();

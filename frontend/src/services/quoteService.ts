import api from './api';
import { Quote, QuoteListResponse, QuoteStats } from '../types/quote';
import { downloadPDF } from '../utils/fileDownload';

export interface PDFDownloadResponse {
  download_url: string;
  expires_at: string;
  filename: string;
}

export interface QuoteFilters {
  customer_id?: string;
  status?: string;
  page?: number;
  page_size?: number;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

/**
 * Quote Service
 * Handles all API calls related to quotes
 * Uses the shared api instance with auth interceptors
 */
class QuoteService {
  private basePath = '/v1/quotes';

  /**
   * Fetch quotes with filters and pagination
   */
  async getQuotes(filters: QuoteFilters = {}): Promise<QuoteListResponse> {
    const params = new URLSearchParams();

    if (filters.customer_id) params.append('customer_id', filters.customer_id);
    if (filters.status) params.append('status', filters.status);
    if (filters.page) params.append('page', filters.page.toString());
    if (filters.page_size)
      params.append('page_size', filters.page_size.toString());
    if (filters.sort_by) params.append('sort_by', filters.sort_by);
    if (filters.sort_order) params.append('sort_order', filters.sort_order);

    const response = await api.get<QuoteListResponse>(this.basePath, {
      params,
    });
    return response.data;
  }

  /**
   * Fetch a single quote by ID
   */
  async getQuote(id: string): Promise<Quote> {
    const response = await api.get<Quote>(`${this.basePath}/${id}`);
    return response.data;
  }

  /**
   * Fetch a quote by quote number
   */
  async getQuoteByNumber(quoteNumber: string): Promise<Quote> {
    const response = await api.get<Quote>(
      `${this.basePath}/number/${quoteNumber}`
    );
    return response.data;
  }

  /**
   * Create a new quote
   */
  async createQuote(data: Partial<Quote>): Promise<Quote> {
    const response = await api.post<Quote>(this.basePath, data);
    return response.data;
  }

  /**
   * Update an existing quote
   */
  async updateQuote(id: string, data: Partial<Quote>): Promise<Quote> {
    const response = await api.put<Quote>(`${this.basePath}/${id}`, data);
    return response.data;
  }

  /**
   * Delete a quote
   */
  async deleteQuote(id: string): Promise<void> {
    await api.delete(`${this.basePath}/${id}`);
  }

  /**
   * Send a quote to customer
   */
  async sendQuote(id: string): Promise<Quote> {
    const response = await api.post<Quote>(`${this.basePath}/${id}/send`);
    return response.data;
  }

  /**
   * Accept a quote
   */
  async acceptQuote(id: string): Promise<Quote> {
    const response = await api.post<Quote>(`${this.basePath}/${id}/accept`);
    return response.data;
  }

  /**
   * Reject a quote
   */
  async rejectQuote(id: string): Promise<Quote> {
    const response = await api.post<Quote>(`${this.basePath}/${id}/reject`);
    return response.data;
  }

  /**
   * Get quote statistics
   */
  async getStats(): Promise<QuoteStats> {
    const response = await api.get<QuoteStats>(`${this.basePath}/stats`);
    return response.data;
  }

  /**
   * Get quotes for a specific customer
   */
  async getCustomerQuotes(customerId: string): Promise<Quote[]> {
    const response = await api.get<Quote[]>(
      `/v1/customers/${customerId}/quotes`
    );
    return response.data;
  }

  /**
   * Get quote PDF download URL (legacy - returns URL)
   */
  async getQuotePDF(id: string): Promise<PDFDownloadResponse> {
    const response = await api.get<PDFDownloadResponse>(
      `${this.basePath}/${id}/pdf`
    );
    return response.data;
  }

  /**
   * Download quote PDF directly and trigger browser download
   */
  async downloadQuotePDF(id: string, filename?: string): Promise<void> {
    const response = await api.get(`${this.basePath}/${id}/pdf`, {
      responseType: 'blob',
    });

    // Get filename from Content-Disposition header or use provided/default
    const contentDisposition = response.headers['content-disposition'];
    let downloadFilename = filename || `quote-${id}.pdf`;
    if (contentDisposition) {
      const match = contentDisposition.match(/filename="?([^"]+)"?/);
      if (match) {
        downloadFilename = match[1];
      }
    }

    downloadPDF(response.data, downloadFilename);
  }
}

export const quoteService = new QuoteService();

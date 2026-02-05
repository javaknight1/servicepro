import api from './api';
import type {
  QuoteTemplate,
  TemplateCategory,
  TemplateListFilter,
  TemplateListResponse,
  TemplateRenderRequest,
  TemplateRenderResult,
  TemplateStats,
  TemplateImportExport,
} from '../types/template';
import { downloadJSON } from '../utils/fileDownload';

/**
 * Template Service
 * Handles all API calls related to quote templates
 * Uses the shared api instance with auth interceptors
 */
class TemplateService {
  private basePath = '/v1/templates';

  /**
   * Create a new template
   */
  async createTemplate(
    template: Partial<QuoteTemplate>
  ): Promise<QuoteTemplate> {
    const response = await api.post<QuoteTemplate>(this.basePath, template);
    return response.data;
  }

  /**
   * Get a single template by ID
   */
  async getTemplate(id: string): Promise<QuoteTemplate> {
    const response = await api.get<QuoteTemplate>(`${this.basePath}/${id}`);
    return response.data;
  }

  /**
   * Update a template
   */
  async updateTemplate(
    id: string,
    updates: Partial<QuoteTemplate>
  ): Promise<QuoteTemplate> {
    const response = await api.put<QuoteTemplate>(
      `${this.basePath}/${id}`,
      updates
    );
    return response.data;
  }

  /**
   * Delete a template
   */
  async deleteTemplate(id: string): Promise<void> {
    await api.delete(`${this.basePath}/${id}`);
  }

  /**
   * List templates with filters
   */
  async listTemplates(
    filter: TemplateListFilter = {}
  ): Promise<TemplateListResponse> {
    const params = new URLSearchParams();

    if (filter.category) params.append('category', filter.category);
    if (filter.tags) filter.tags.forEach((tag) => params.append('tags', tag));
    if (filter.search) params.append('search', filter.search);
    if (filter.is_active !== undefined)
      params.append('is_active', String(filter.is_active));
    if (filter.is_public !== undefined)
      params.append('is_public', String(filter.is_public));
    if (filter.created_by) params.append('created_by', filter.created_by);
    if (filter.page) params.append('page', String(filter.page));
    if (filter.page_size) params.append('page_size', String(filter.page_size));
    if (filter.sort_by) params.append('sort_by', filter.sort_by);
    if (filter.sort_order) params.append('sort_order', filter.sort_order);

    const response = await api.get<TemplateListResponse>(this.basePath, {
      params,
    });
    return response.data;
  }

  /**
   * Duplicate a template
   */
  async duplicateTemplate(id: string): Promise<QuoteTemplate> {
    const response = await api.post<QuoteTemplate>(
      `${this.basePath}/${id}/duplicate`
    );
    return response.data;
  }

  /**
   * Render a template with variables
   */
  async renderTemplate(
    request: TemplateRenderRequest
  ): Promise<TemplateRenderResult> {
    const response = await api.post<TemplateRenderResult>(
      `${this.basePath}/render`,
      request
    );
    return response.data;
  }

  /**
   * Get template statistics
   */
  async getStats(): Promise<TemplateStats> {
    const response = await api.get<TemplateStats>(`${this.basePath}/stats`);
    return response.data;
  }

  /**
   * Export templates to JSON
   */
  async exportTemplates(templateIds?: string[]): Promise<TemplateImportExport> {
    const params = new URLSearchParams();
    if (templateIds) {
      templateIds.forEach((id) => params.append('ids', id));
    }

    const response = await api.get<TemplateImportExport>(
      `${this.basePath}/export`,
      { params }
    );
    return response.data;
  }

  /**
   * Import templates from JSON
   */
  async importTemplates(
    data: TemplateImportExport,
    overwrite: boolean = false
  ): Promise<{ imported: number }> {
    const response = await api.post<{ imported: number }>(
      `${this.basePath}/import`,
      data,
      { params: { overwrite: String(overwrite) } }
    );
    return response.data;
  }

  /**
   * Download templates as JSON file
   */
  async downloadTemplatesJSON(templateIds?: string[]): Promise<void> {
    const exportData = await this.exportTemplates(templateIds);
    const filename = `quote-templates-${new Date().toISOString().split('T')[0]}.json`;
    downloadJSON(exportData, filename);
  }

  /**
   * Upload and import templates from JSON file
   */
  async uploadTemplatesJSON(
    file: File,
    overwrite: boolean = false
  ): Promise<number> {
    const text = await file.text();
    const data: TemplateImportExport = JSON.parse(text);
    const result = await this.importTemplates(data, overwrite);
    return result.imported;
  }

  // Category Management

  /**
   * Create a new category
   */
  async createCategory(
    category: Partial<TemplateCategory>
  ): Promise<TemplateCategory> {
    const response = await api.post<TemplateCategory>(
      '/v1/template-categories',
      category
    );
    return response.data;
  }

  /**
   * List all categories
   */
  async listCategories(): Promise<TemplateCategory[]> {
    const response = await api.get<TemplateCategory[]>(
      '/v1/template-categories'
    );
    return response.data;
  }

  /**
   * Update a category
   */
  async updateCategory(
    id: string,
    updates: Partial<TemplateCategory>
  ): Promise<TemplateCategory> {
    const response = await api.put<TemplateCategory>(
      `/v1/template-categories/${id}`,
      updates
    );
    return response.data;
  }

  /**
   * Delete a category
   */
  async deleteCategory(id: string): Promise<void> {
    await api.delete(`/v1/template-categories/${id}`);
  }
}

export const templateService = new TemplateService();

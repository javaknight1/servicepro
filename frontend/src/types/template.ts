/**
 * Quote Template Types
 * Matches backend API models
 */

/** Value types that can be stored in a template variable */
export type TemplateVariableValue = string | number | Date | null;

export interface TemplateVariable {
  name: string;
  type: 'text' | 'number' | 'date' | 'currency';
  label: string;
  description?: string;
  default_value?: TemplateVariableValue;
  required: boolean;
  placeholder?: string;
  validation?: VariableValidation;
}

export interface VariableValidation {
  min_length?: number;
  max_length?: number;
  min_value?: number;
  max_value?: number;
  pattern?: string;
  options?: string[];
}

export interface TemplateLineItem {
  description: string;
  quantity: number;
  unit_price: number;
  is_variable: boolean;
  category?: string;
}

export interface QuoteTemplate {
  id: string;
  name: string;
  description: string;
  category: string;
  content: string;
  variables: TemplateVariable[];
  line_items: TemplateLineItem[];
  valid_days: number;
  payment_terms: string;
  delivery_info: string;
  warranty_info: string;
  default_tax_rate: string;
  tags: string[];
  version: number;
  is_active: boolean;
  is_public: boolean;
  usage_count: number;
  created_by: string;
  updated_by?: string;
  created_at: string;
  updated_at: string;
}

export interface TemplateCategory {
  id: string;
  name: string;
  description: string;
  icon: string;
  color: string;
  sort_order: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface TemplateVariableMap {
  [key: string]: TemplateVariableValue;
}

export interface TemplateRenderRequest {
  template_id: string;
  variables: TemplateVariableMap;
  customer_id?: string;
}

export interface TemplateRenderResult {
  content: string;
  line_items: TemplateLineItem[];
  payment_terms: string;
  delivery_info: string;
  warranty_info: string;
  tax_rate: string;
  valid_until: string;
  variables: TemplateVariableMap;
}

export interface TemplateListFilter {
  category?: string;
  tags?: string[];
  search?: string;
  is_active?: boolean;
  is_public?: boolean;
  created_by?: string;
  page?: number;
  page_size?: number;
  sort_by?: string;
  sort_order?: 'asc' | 'desc';
}

export interface TemplateListResponse {
  templates: QuoteTemplate[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface TemplateStats {
  total_templates: number;
  active_templates: number;
  total_categories: number;
  total_usage: number;
  by_category: Record<string, number>;
  most_used: QuoteTemplate[];
  recently_updated: QuoteTemplate[];
}

export interface TemplateImportExport {
  version: string;
  exported_at: string;
  templates: QuoteTemplateExport[];
  categories?: TemplateCategoryExport[];
}

export interface QuoteTemplateExport {
  name: string;
  description: string;
  category: string;
  content: string;
  variables: TemplateVariable[];
  line_items: TemplateLineItem[];
  valid_days: number;
  payment_terms: string;
  delivery_info: string;
  warranty_info: string;
  default_tax_rate: string;
  tags: string[];
  is_public: boolean;
}

export interface TemplateCategoryExport {
  name: string;
  description: string;
  icon: string;
  color: string;
  sort_order: number;
}

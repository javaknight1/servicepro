import React, { useEffect, useState } from 'react';
import { useForm, Controller, useFieldArray } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { templateService } from '../../services/templateService';
import type {
  QuoteTemplate,
  TemplateVariable,
  TemplateLineItem,
} from '../../types/template';
import { VariableEditor } from './VariableEditor';

const templateSchema = z.object({
  name: z.string().min(1, 'Template name is required').max(200),
  description: z.string().max(1000).optional(),
  category: z.string().min(1, 'Category is required'),
  content: z.string().min(1, 'Template content is required'),
  variables: z.array(z.any()),
  line_items: z.array(z.any()),
  valid_days: z.number().min(1).max(365),
  payment_terms: z.string().max(500).optional(),
  delivery_info: z.string().max(500).optional(),
  warranty_info: z.string().max(500).optional(),
  default_tax_rate: z
    .string()
    .regex(/^\d+(\.\d{1,4})?$/)
    .optional(),
  tags: z.array(z.string()),
  is_public: z.boolean(),
});

type TemplateFormData = z.infer<typeof templateSchema>;

interface TemplateEditorProps {
  templateId?: string;
  onSave?: (template: QuoteTemplate) => void;
  onCancel?: () => void;
}

export const TemplateEditor: React.FC<TemplateEditorProps> = ({
  templateId,
  onSave,
  onCancel,
}) => {
  const queryClient = useQueryClient();
  const [activeTab, setActiveTab] = useState<
    'content' | 'variables' | 'items' | 'terms'
  >('content');

  const {
    register,
    control,
    handleSubmit,
    watch,
    setValue,
    formState: { errors, isDirty },
    reset,
  } = useForm<TemplateFormData>({
    resolver: zodResolver(templateSchema),
    defaultValues: {
      name: '',
      description: '',
      category: '',
      content: '',
      variables: [],
      line_items: [],
      valid_days: 30,
      payment_terms: '',
      delivery_info: '',
      warranty_info: '',
      default_tax_rate: '0.0825',
      tags: [],
      is_public: false,
    },
  });

  const {
    fields: lineItemFields,
    append: appendLineItem,
    remove: removeLineItem,
  } = useFieldArray({
    control,
    name: 'line_items',
  });

  // Load existing template
  const { data: template, isLoading } = useQuery({
    queryKey: ['template', templateId],
    queryFn: () => templateService.getTemplate(templateId!),
    enabled: !!templateId,
  });

  useEffect(() => {
    if (template) {
      reset({
        name: template.name,
        description: template.description,
        category: template.category,
        content: template.content,
        variables: template.variables,
        line_items: template.line_items,
        valid_days: template.valid_days,
        payment_terms: template.payment_terms,
        delivery_info: template.delivery_info,
        warranty_info: template.warranty_info,
        default_tax_rate: template.default_tax_rate,
        tags: template.tags || [],
        is_public: template.is_public,
      });
    }
  }, [template, reset]);

  // Save mutation
  const saveMutation = useMutation({
    mutationFn: async (data: TemplateFormData) => {
      if (templateId) {
        return templateService.updateTemplate(templateId, data);
      }
      return templateService.createTemplate(data);
    },
    onSuccess: (savedTemplate) => {
      queryClient.invalidateQueries({ queryKey: ['templates'] });
      queryClient.invalidateQueries({ queryKey: ['template', templateId] });
      onSave?.(savedTemplate);
    },
  });

  const onSubmit = (data: TemplateFormData) => {
    saveMutation.mutate(data);
  };

  // Extract variables from content
  const extractVariablesFromContent = (content: string): string[] => {
    const regex = /\{\{([^}]+)\}\}/g;
    const matches = [...content.matchAll(regex)];
    return [...new Set(matches.map((m) => m[1].trim()))];
  };

  const content = watch('content');
  const variables = watch('variables') || [];

  // Highlight undefined variables
  const usedVars = extractVariablesFromContent(content);
  const definedVars = new Set(variables.map((v: TemplateVariable) => v.name));
  const undefinedVars = usedVars.filter((v) => !definedVars.has(v));

  if (isLoading && templateId) {
    return (
      <div className="flex items-center justify-center p-8">
        <div className="text-gray-600">Loading template...</div>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-2xl font-bold text-gray-900">
            {templateId ? 'Edit Template' : 'Create Template'}
          </h2>
          {isDirty && (
            <p className="mt-1 text-sm text-amber-600">
              You have unsaved changes
            </p>
          )}
        </div>
        <div className="flex gap-2">
          <button
            type="button"
            onClick={onCancel}
            className="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={saveMutation.isPending}
            className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {saveMutation.isPending ? 'Saving...' : 'Save Template'}
          </button>
        </div>
      </div>

      {/* Basic Information */}
      <div className="rounded-lg border bg-white p-6">
        <h3 className="mb-4 text-lg font-semibold">Basic Information</h3>
        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Template Name <span className="text-red-500">*</span>
            </label>
            <input
              {...register('name')}
              type="text"
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              placeholder="e.g., Standard Service Quote"
            />
            {errors.name && (
              <p className="mt-1 text-sm text-red-600">{errors.name.message}</p>
            )}
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700">
              Category <span className="text-red-500">*</span>
            </label>
            <input
              {...register('category')}
              type="text"
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              placeholder="e.g., Service, Product, Consulting"
            />
            {errors.category && (
              <p className="mt-1 text-sm text-red-600">
                {errors.category.message}
              </p>
            )}
          </div>

          <div className="col-span-2">
            <label className="block text-sm font-medium text-gray-700">
              Description
            </label>
            <textarea
              {...register('description')}
              rows={3}
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              placeholder="Describe what this template is used for..."
            />
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700">
              Valid Days
            </label>
            <input
              {...register('valid_days', { valueAsNumber: true })}
              type="number"
              min="1"
              max="365"
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
            <p className="mt-1 text-xs text-gray-500">
              Number of days the quote will be valid
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium text-gray-700">
              Default Tax Rate
            </label>
            <input
              {...register('default_tax_rate')}
              type="text"
              className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              placeholder="0.0825"
            />
            <p className="mt-1 text-xs text-gray-500">
              Decimal format (e.g., 0.0825 for 8.25%)
            </p>
          </div>

          <div className="col-span-2">
            <label className="flex items-center">
              <input
                {...register('is_public')}
                type="checkbox"
                className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
              />
              <span className="ml-2 text-sm text-gray-700">
                Make this template public (available to all users)
              </span>
            </label>
          </div>
        </div>
      </div>

      {/* Tabs */}
      <div className="rounded-lg border bg-white">
        <div className="border-b border-gray-200">
          <nav className="flex -mb-px">
            <button
              type="button"
              onClick={() => setActiveTab('content')}
              className={`px-6 py-3 text-sm font-medium ${
                activeTab === 'content'
                  ? 'border-b-2 border-blue-500 text-blue-600'
                  : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              Content
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('variables')}
              className={`px-6 py-3 text-sm font-medium ${
                activeTab === 'variables'
                  ? 'border-b-2 border-blue-500 text-blue-600'
                  : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              Variables
              {undefinedVars.length > 0 && (
                <span className="ml-2 inline-flex h-5 w-5 items-center justify-center rounded-full bg-red-100 text-xs text-red-600">
                  {undefinedVars.length}
                </span>
              )}
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('items')}
              className={`px-6 py-3 text-sm font-medium ${
                activeTab === 'items'
                  ? 'border-b-2 border-blue-500 text-blue-600'
                  : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              Line Items
            </button>
            <button
              type="button"
              onClick={() => setActiveTab('terms')}
              className={`px-6 py-3 text-sm font-medium ${
                activeTab === 'terms'
                  ? 'border-b-2 border-blue-500 text-blue-600'
                  : 'text-gray-500 hover:text-gray-700'
              }`}
            >
              Terms
            </button>
          </nav>
        </div>

        <div className="p-6">
          {/* Content Tab */}
          {activeTab === 'content' && (
            <div>
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700">
                  Template Content <span className="text-red-500">*</span>
                </label>
                <p className="mt-1 text-sm text-gray-500">
                  Use{' '}
                  <code className="rounded bg-gray-100 px-1 text-xs">
                    {'{{variable_name}}'}
                  </code>{' '}
                  syntax to insert variables
                </p>
              </div>
              <textarea
                {...register('content')}
                rows={20}
                className="block w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                placeholder={`Dear {{customer_name}},

Thank you for your interest in {{service_name}}. This quote is valid for {{valid_days}} days.

Project Details:
- Start Date: {{project_start}}
- Duration: {{project_duration}} weeks
- Total Cost: \${{total_cost}}

Best regards,
{{company_name}}`}
              />
              {errors.content && (
                <p className="mt-1 text-sm text-red-600">
                  {errors.content.message}
                </p>
              )}

              {undefinedVars.length > 0 && (
                <div className="mt-4 rounded-md bg-amber-50 p-4">
                  <div className="flex">
                    <div className="flex-shrink-0">
                      <svg
                        className="h-5 w-5 text-amber-400"
                        viewBox="0 0 20 20"
                        fill="currentColor"
                      >
                        <path
                          fillRule="evenodd"
                          d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                          clipRule="evenodd"
                        />
                      </svg>
                    </div>
                    <div className="ml-3">
                      <h3 className="text-sm font-medium text-amber-800">
                        Undefined Variables
                      </h3>
                      <div className="mt-2 text-sm text-amber-700">
                        <p>
                          The following variables are used in the content but
                          not defined:{' '}
                          <span className="font-mono font-semibold">
                            {undefinedVars.join(', ')}
                          </span>
                        </p>
                        <button
                          type="button"
                          onClick={() => setActiveTab('variables')}
                          className="mt-2 text-sm font-medium text-amber-800 underline"
                        >
                          Go to Variables tab to define them
                        </button>
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          )}

          {/* Variables Tab */}
          {activeTab === 'variables' && (
            <Controller
              name="variables"
              control={control}
              render={({ field }) => (
                <VariableEditor
                  variables={field.value || []}
                  onChange={field.onChange}
                  undefinedVariables={undefinedVars}
                />
              )}
            />
          )}

          {/* Line Items Tab */}
          {activeTab === 'items' && (
            <div>
              <div className="mb-4 flex items-center justify-between">
                <div>
                  <h3 className="text-sm font-medium text-gray-700">
                    Line Item Templates
                  </h3>
                  <p className="mt-1 text-sm text-gray-500">
                    Define line items that can use variables in their
                    descriptions and prices
                  </p>
                </div>
                <button
                  type="button"
                  onClick={() =>
                    appendLineItem({
                      description: '',
                      quantity: 1,
                      unit_price: 0,
                      is_variable: false,
                    })
                  }
                  className="rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700"
                >
                  Add Line Item
                </button>
              </div>

              <div className="space-y-4">
                {lineItemFields.map((field, index) => (
                  <div
                    key={field.id}
                    className="rounded-md border bg-gray-50 p-4"
                  >
                    <div className="mb-3 flex items-center justify-between">
                      <span className="text-sm font-medium text-gray-700">
                        Line Item {index + 1}
                      </span>
                      <button
                        type="button"
                        onClick={() => removeLineItem(index)}
                        className="text-sm text-red-600 hover:text-red-700"
                      >
                        Remove
                      </button>
                    </div>
                    <div className="grid grid-cols-3 gap-3">
                      <div className="col-span-3">
                        <label className="block text-xs font-medium text-gray-700">
                          Description
                        </label>
                        <input
                          {...register(`line_items.${index}.description`)}
                          type="text"
                          className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                          placeholder="e.g., {{service_name}} - {{hours}} hours"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-medium text-gray-700">
                          Quantity
                        </label>
                        <input
                          {...register(`line_items.${index}.quantity`, {
                            valueAsNumber: true,
                          })}
                          type="number"
                          min="1"
                          step="1"
                          className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                        />
                      </div>
                      <div>
                        <label className="block text-xs font-medium text-gray-700">
                          Unit Price
                        </label>
                        <input
                          {...register(`line_items.${index}.unit_price`, {
                            valueAsNumber: true,
                          })}
                          type="number"
                          min="0"
                          step="0.01"
                          className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                        />
                      </div>
                      <div>
                        <label className="flex items-center">
                          <input
                            {...register(`line_items.${index}.is_variable`)}
                            type="checkbox"
                            className="h-4 w-4 rounded border-gray-300 text-blue-600"
                          />
                          <span className="ml-2 text-xs text-gray-700">
                            Contains variables
                          </span>
                        </label>
                      </div>
                    </div>
                  </div>
                ))}

                {lineItemFields.length === 0 && (
                  <div className="rounded-md bg-gray-50 p-8 text-center">
                    <p className="text-sm text-gray-500">
                      No line items defined. Click "Add Line Item" to create
                      one.
                    </p>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Terms Tab */}
          {activeTab === 'terms' && (
            <div className="space-y-4">
              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Payment Terms
                </label>
                <textarea
                  {...register('payment_terms')}
                  rows={4}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                  placeholder="e.g., Net 30, 50% deposit required, etc."
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Delivery Information
                </label>
                <textarea
                  {...register('delivery_info')}
                  rows={4}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                  placeholder="Delivery timeline and conditions..."
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Warranty Information
                </label>
                <textarea
                  {...register('warranty_info')}
                  rows={4}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                  placeholder="Warranty terms and conditions..."
                />
              </div>

              <div className="rounded-md bg-blue-50 p-4">
                <p className="text-sm text-blue-700">
                  <strong>Tip:</strong> You can use variables in these fields
                  too! For example:{' '}
                  <code className="rounded bg-blue-100 px-1">
                    {'{{payment_terms}}'}
                  </code>
                </p>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Error Display */}
      {saveMutation.isError && (
        <div className="rounded-md bg-red-50 p-4">
          <div className="flex">
            <div className="ml-3">
              <h3 className="text-sm font-medium text-red-800">
                Error saving template
              </h3>
              <div className="mt-2 text-sm text-red-700">
                {saveMutation.error instanceof Error
                  ? saveMutation.error.message
                  : 'An unknown error occurred'}
              </div>
            </div>
          </div>
        </div>
      )}
    </form>
  );
};

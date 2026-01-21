import React, { useState, useEffect } from 'react';
import { useMutation } from '@tanstack/react-query';
import { templateService } from '../../services/templateService';
import type {
  QuoteTemplate,
  TemplateVariable,
  TemplateVariableMap,
  TemplateRenderResult,
} from '../../types/template';

interface TemplatePreviewProps {
  template: QuoteTemplate;
  variables?: TemplateVariableMap;
  onVariableChange?: (variables: TemplateVariableMap) => void;
}

export const TemplatePreview: React.FC<TemplatePreviewProps> = ({
  template,
  variables: initialVariables = {},
  onVariableChange,
}) => {
  const [variables, setVariables] =
    useState<TemplateVariableMap>(initialVariables);
  const [renderResult, setRenderResult] = useState<TemplateRenderResult | null>(
    null
  );

  const renderMutation = useMutation({
    mutationFn: async (vars: TemplateVariableMap) => {
      return templateService.renderTemplate({
        template_id: template.id,
        variables: vars,
      });
    },
    onSuccess: (result) => {
      setRenderResult(result);
    },
  });

  useEffect(() => {
    // Initialize with default values
    const defaultVars: TemplateVariableMap = {};
    template.variables.forEach((varDef) => {
      if (varDef.default_value !== undefined && varDef.default_value !== null) {
        defaultVars[varDef.name] = varDef.default_value;
      }
    });
    setVariables({ ...defaultVars, ...initialVariables });
  }, [template, initialVariables]);

  const handleVariableChange = (name: string, value: any) => {
    const newVariables = { ...variables, [name]: value };
    setVariables(newVariables);
    onVariableChange?.(newVariables);
  };

  const handlePreview = () => {
    renderMutation.mutate(variables);
  };

  const getInputType = (type: TemplateVariable['type']) => {
    switch (type) {
      case 'number':
      case 'currency':
        return 'number';
      case 'date':
        return 'date';
      default:
        return 'text';
    }
  };

  const formatValue = (value: any, type: TemplateVariable['type']) => {
    if (value === null || value === undefined) return '';

    switch (type) {
      case 'currency':
        return typeof value === 'number' ? `$${value.toFixed(2)}` : value;
      case 'date':
        if (value instanceof Date) {
          return value.toLocaleDateString('en-US', {
            year: 'numeric',
            month: 'long',
            day: 'numeric',
          });
        }
        return value;
      default:
        return String(value);
    }
  };

  const validateVariable = (
    varDef: TemplateVariable,
    value: any
  ): string | null => {
    if (varDef.required && !value) {
      return `${varDef.label} is required`;
    }

    if (!varDef.validation || !value) return null;

    const validation = varDef.validation;

    if (varDef.type === 'text') {
      const strValue = String(value);
      if (validation.min_length && strValue.length < validation.min_length) {
        return `Minimum length is ${validation.min_length}`;
      }
      if (validation.max_length && strValue.length > validation.max_length) {
        return `Maximum length is ${validation.max_length}`;
      }
      if (validation.pattern) {
        const regex = new RegExp(validation.pattern);
        if (!regex.test(strValue)) {
          return 'Does not match required pattern';
        }
      }
    }

    if (varDef.type === 'number' || varDef.type === 'currency') {
      const numValue = typeof value === 'number' ? value : parseFloat(value);
      if (isNaN(numValue)) return 'Invalid number';

      if (
        validation.min_value !== undefined &&
        numValue < validation.min_value
      ) {
        return `Minimum value is ${validation.min_value}`;
      }
      if (
        validation.max_value !== undefined &&
        numValue > validation.max_value
      ) {
        return `Maximum value is ${validation.max_value}`;
      }
    }

    return null;
  };

  const hasErrors = template.variables.some(
    (varDef) => validateVariable(varDef, variables[varDef.name]) !== null
  );

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
      {/* Left Side - Variable Inputs */}
      <div className="space-y-6">
        <div className="rounded-lg border bg-white p-6">
          <h3 className="mb-4 text-lg font-semibold text-gray-900">
            Template Variables
          </h3>
          <p className="mb-4 text-sm text-gray-500">
            Fill in the variables to see how the template will look when
            rendered
          </p>

          <div className="space-y-4">
            {template.variables.map((varDef) => {
              const error = validateVariable(varDef, variables[varDef.name]);
              const value = variables[varDef.name];

              return (
                <div key={varDef.name}>
                  <label className="block text-sm font-medium text-gray-700">
                    {varDef.label}
                    {varDef.required && (
                      <span className="ml-1 text-red-500">*</span>
                    )}
                  </label>
                  {varDef.description && (
                    <p className="mt-1 text-xs text-gray-500">
                      {varDef.description}
                    </p>
                  )}
                  <input
                    type={getInputType(varDef.type)}
                    value={value || ''}
                    onChange={(e) => {
                      const inputValue = e.target.value;
                      let parsedValue: any = inputValue;

                      if (
                        varDef.type === 'number' ||
                        varDef.type === 'currency'
                      ) {
                        parsedValue = inputValue
                          ? parseFloat(inputValue)
                          : null;
                      }

                      handleVariableChange(varDef.name, parsedValue);
                    }}
                    placeholder={varDef.placeholder}
                    step={varDef.type === 'currency' ? '0.01' : undefined}
                    className={`mt-1 block w-full rounded-md border px-3 py-2 focus:outline-none focus:ring-1 ${
                      error
                        ? 'border-red-300 focus:border-red-500 focus:ring-red-500'
                        : 'border-gray-300 focus:border-blue-500 focus:ring-blue-500'
                    }`}
                  />
                  {error && (
                    <p className="mt-1 text-sm text-red-600">{error}</p>
                  )}
                </div>
              );
            })}

            {template.variables.length === 0 && (
              <div className="rounded-md bg-gray-50 p-4 text-center">
                <p className="text-sm text-gray-500">
                  This template has no variables defined
                </p>
              </div>
            )}
          </div>

          <div className="mt-6">
            <button
              type="button"
              onClick={handlePreview}
              disabled={hasErrors || renderMutation.isPending}
              className="w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
            >
              {renderMutation.isPending ? 'Rendering...' : 'Preview Template'}
            </button>
          </div>
        </div>
      </div>

      {/* Right Side - Preview */}
      <div className="space-y-6">
        {/* Content Preview */}
        <div className="rounded-lg border bg-white p-6">
          <h3 className="mb-4 text-lg font-semibold text-gray-900">Preview</h3>

          {renderMutation.isError && (
            <div className="mb-4 rounded-md bg-red-50 p-4">
              <div className="flex">
                <div className="ml-3">
                  <h3 className="text-sm font-medium text-red-800">
                    Rendering Error
                  </h3>
                  <div className="mt-2 text-sm text-red-700">
                    {renderMutation.error instanceof Error
                      ? renderMutation.error.message
                      : 'An error occurred while rendering the template'}
                  </div>
                </div>
              </div>
            </div>
          )}

          {!renderResult && (
            <div className="rounded-md bg-gray-50 p-8 text-center">
              <svg
                className="mx-auto h-12 w-12 text-gray-400"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                />
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
                />
              </svg>
              <p className="mt-2 text-sm text-gray-500">
                Fill in the variables and click "Preview Template" to see the
                result
              </p>
            </div>
          )}

          {renderResult && (
            <div className="space-y-6">
              {/* Rendered Content */}
              <div>
                <h4 className="mb-2 text-sm font-medium text-gray-700">
                  Content
                </h4>
                <div className="rounded-md bg-gray-50 p-4">
                  <pre className="whitespace-pre-wrap font-sans text-sm text-gray-900">
                    {renderResult.content}
                  </pre>
                </div>
              </div>

              {/* Line Items */}
              {renderResult.line_items &&
                renderResult.line_items.length > 0 && (
                  <div>
                    <h4 className="mb-2 text-sm font-medium text-gray-700">
                      Line Items
                    </h4>
                    <div className="overflow-hidden rounded-md border">
                      <table className="min-w-full divide-y divide-gray-200">
                        <thead className="bg-gray-50">
                          <tr>
                            <th className="px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500">
                              Description
                            </th>
                            <th className="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500">
                              Qty
                            </th>
                            <th className="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500">
                              Price
                            </th>
                            <th className="px-4 py-3 text-right text-xs font-medium uppercase tracking-wider text-gray-500">
                              Total
                            </th>
                          </tr>
                        </thead>
                        <tbody className="divide-y divide-gray-200 bg-white">
                          {renderResult.line_items.map(
                            (item: any, index: number) => (
                              <tr key={index}>
                                <td className="px-4 py-3 text-sm text-gray-900">
                                  {item.description}
                                </td>
                                <td className="px-4 py-3 text-right text-sm text-gray-900">
                                  {item.quantity}
                                </td>
                                <td className="px-4 py-3 text-right text-sm text-gray-900">
                                  ${parseFloat(item.unit_price).toFixed(2)}
                                </td>
                                <td className="px-4 py-3 text-right text-sm font-medium text-gray-900">
                                  ${parseFloat(item.total).toFixed(2)}
                                </td>
                              </tr>
                            )
                          )}
                        </tbody>
                      </table>
                    </div>
                  </div>
                )}

              {/* Terms */}
              {(renderResult.payment_terms ||
                renderResult.delivery_info ||
                renderResult.warranty_info) && (
                <div>
                  <h4 className="mb-2 text-sm font-medium text-gray-700">
                    Terms & Conditions
                  </h4>
                  <div className="space-y-3 rounded-md bg-gray-50 p-4 text-sm">
                    {renderResult.payment_terms && (
                      <div>
                        <p className="font-medium text-gray-700">
                          Payment Terms:
                        </p>
                        <p className="mt-1 text-gray-600">
                          {renderResult.payment_terms}
                        </p>
                      </div>
                    )}
                    {renderResult.delivery_info && (
                      <div>
                        <p className="font-medium text-gray-700">Delivery:</p>
                        <p className="mt-1 text-gray-600">
                          {renderResult.delivery_info}
                        </p>
                      </div>
                    )}
                    {renderResult.warranty_info && (
                      <div>
                        <p className="font-medium text-gray-700">Warranty:</p>
                        <p className="mt-1 text-gray-600">
                          {renderResult.warranty_info}
                        </p>
                      </div>
                    )}
                  </div>
                </div>
              )}

              {/* Metadata */}
              <div>
                <h4 className="mb-2 text-sm font-medium text-gray-700">
                  Quote Details
                </h4>
                <dl className="grid grid-cols-2 gap-3 text-sm">
                  <div>
                    <dt className="font-medium text-gray-500">Valid Until</dt>
                    <dd className="mt-1 text-gray-900">
                      {new Date(renderResult.valid_until).toLocaleDateString()}
                    </dd>
                  </div>
                  <div>
                    <dt className="font-medium text-gray-500">Tax Rate</dt>
                    <dd className="mt-1 text-gray-900">
                      {(parseFloat(renderResult.tax_rate) * 100).toFixed(2)}%
                    </dd>
                  </div>
                </dl>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

import React, { useState } from 'react';
import type {
  TemplateVariable,
  VariableValidation,
} from '../../types/template';

interface VariableEditorProps {
  variables: TemplateVariable[];
  onChange: (variables: TemplateVariable[]) => void;
  undefinedVariables?: string[];
}

export const VariableEditor: React.FC<VariableEditorProps> = ({
  variables,
  onChange,
  undefinedVariables = [],
}) => {
  const [editingIndex, setEditingIndex] = useState<number | null>(null);
  const [editingVariable, setEditingVariable] =
    useState<TemplateVariable | null>(null);

  const handleAddVariable = () => {
    const newVar: TemplateVariable = {
      name: '',
      type: 'text',
      label: '',
      description: '',
      required: false,
      placeholder: '',
    };
    setEditingVariable(newVar);
    setEditingIndex(variables.length);
  };

  const handleAddUndefinedVariable = (varName: string) => {
    const newVar: TemplateVariable = {
      name: varName,
      type: 'text',
      label: varName
        .replace(/_/g, ' ')
        .replace(/\b\w/g, (l) => l.toUpperCase()),
      description: '',
      required: true,
      placeholder: '',
    };
    onChange([...variables, newVar]);
  };

  const handleEditVariable = (index: number) => {
    setEditingIndex(index);
    setEditingVariable({ ...variables[index] });
  };

  const handleSaveVariable = () => {
    if (!editingVariable || editingIndex === null) return;

    const newVariables = [...variables];
    if (editingIndex < variables.length) {
      newVariables[editingIndex] = editingVariable;
    } else {
      newVariables.push(editingVariable);
    }
    onChange(newVariables);
    setEditingIndex(null);
    setEditingVariable(null);
  };

  const handleCancelEdit = () => {
    setEditingIndex(null);
    setEditingVariable(null);
  };

  const handleDeleteVariable = (index: number) => {
    onChange(variables.filter((_, i) => i !== index));
  };

  const updateEditingVariable = (updates: Partial<TemplateVariable>) => {
    if (editingVariable) {
      setEditingVariable({ ...editingVariable, ...updates });
    }
  };

  const updateValidation = (updates: Partial<VariableValidation>) => {
    if (editingVariable) {
      setEditingVariable({
        ...editingVariable,
        validation: { ...editingVariable.validation, ...updates },
      });
    }
  };

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <div>
          <h3 className="text-sm font-medium text-gray-700">Variables</h3>
          <p className="mt-1 text-sm text-gray-500">
            Define variables that can be used in your template with{' '}
            {'{{variable_name}}'} syntax
          </p>
        </div>
        <button
          type="button"
          onClick={handleAddVariable}
          className="rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white hover:bg-blue-700"
        >
          Add Variable
        </button>
      </div>

      {/* Undefined Variables Warning */}
      {undefinedVariables.length > 0 && (
        <div className="mb-4 rounded-md bg-amber-50 p-4">
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
            <div className="ml-3 flex-1">
              <h3 className="text-sm font-medium text-amber-800">
                Undefined Variables in Template
              </h3>
              <div className="mt-2 text-sm text-amber-700">
                <p>The following variables are used but not defined:</p>
                <div className="mt-2 flex flex-wrap gap-2">
                  {undefinedVariables.map((varName) => (
                    <button
                      key={varName}
                      type="button"
                      onClick={() => handleAddUndefinedVariable(varName)}
                      className="inline-flex items-center rounded-md bg-amber-100 px-2 py-1 text-xs font-medium text-amber-800 hover:bg-amber-200"
                    >
                      <span className="font-mono">{varName}</span>
                      <svg
                        className="ml-1 h-3 w-3"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          strokeWidth={2}
                          d="M12 4v16m8-8H4"
                        />
                      </svg>
                    </button>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Variables List */}
      <div className="space-y-3">
        {variables.map((variable, index) => (
          <div
            key={index}
            className="rounded-md border bg-white p-4 hover:border-gray-400"
          >
            <div className="flex items-start justify-between">
              <div className="flex-1">
                <div className="flex items-center gap-2">
                  <code className="rounded bg-gray-100 px-2 py-1 text-sm font-mono text-gray-800">
                    {`{{${variable.name}}}`}
                  </code>
                  <span className="rounded bg-blue-100 px-2 py-1 text-xs font-medium text-blue-800">
                    {variable.type}
                  </span>
                  {variable.required && (
                    <span className="rounded bg-red-100 px-2 py-1 text-xs font-medium text-red-800">
                      required
                    </span>
                  )}
                </div>
                <p className="mt-1 text-sm font-medium text-gray-900">
                  {variable.label}
                </p>
                {variable.description && (
                  <p className="mt-1 text-sm text-gray-500">
                    {variable.description}
                  </p>
                )}
                {variable.default_value && (
                  <p className="mt-1 text-xs text-gray-500">
                    Default:{' '}
                    <span className="font-mono">
                      {String(variable.default_value)}
                    </span>
                  </p>
                )}
              </div>
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => handleEditVariable(index)}
                  className="text-sm text-blue-600 hover:text-blue-700"
                >
                  Edit
                </button>
                <button
                  type="button"
                  onClick={() => handleDeleteVariable(index)}
                  className="text-sm text-red-600 hover:text-red-700"
                >
                  Delete
                </button>
              </div>
            </div>
          </div>
        ))}

        {variables.length === 0 && !editingVariable && (
          <div className="rounded-md bg-gray-50 p-8 text-center">
            <p className="text-sm text-gray-500">
              No variables defined. Click "Add Variable" to create one.
            </p>
          </div>
        )}
      </div>

      {/* Variable Editor Modal */}
      {editingVariable !== null && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="max-h-[90vh] w-full max-w-2xl overflow-y-auto rounded-lg bg-white p-6 shadow-xl">
            <h3 className="text-lg font-semibold text-gray-900">
              {editingIndex !== null && editingIndex < variables.length
                ? 'Edit Variable'
                : 'Add Variable'}
            </h3>

            <div className="mt-4 space-y-4">
              {/* Variable Name */}
              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Variable Name <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={editingVariable.name}
                  onChange={(e) =>
                    updateEditingVariable({
                      name: e.target.value.replace(/\s/g, '_'),
                    })
                  }
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="e.g., customer_name"
                />
                <p className="mt-1 text-xs text-gray-500">
                  Use snake_case (no spaces, lowercase with underscores)
                </p>
              </div>

              {/* Variable Type */}
              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Type <span className="text-red-500">*</span>
                </label>
                <select
                  value={editingVariable.type}
                  onChange={(e) =>
                    updateEditingVariable({
                      type: e.target.value as TemplateVariable['type'],
                    })
                  }
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                >
                  <option value="text">Text</option>
                  <option value="number">Number</option>
                  <option value="currency">Currency</option>
                  <option value="date">Date</option>
                </select>
              </div>

              {/* Label */}
              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Label <span className="text-red-500">*</span>
                </label>
                <input
                  type="text"
                  value={editingVariable.label}
                  onChange={(e) =>
                    updateEditingVariable({ label: e.target.value })
                  }
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="e.g., Customer Name"
                />
                <p className="mt-1 text-xs text-gray-500">
                  Human-readable label shown in forms
                </p>
              </div>

              {/* Description */}
              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Description
                </label>
                <textarea
                  value={editingVariable.description || ''}
                  onChange={(e) =>
                    updateEditingVariable({ description: e.target.value })
                  }
                  rows={2}
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="Help text for this variable"
                />
              </div>

              {/* Placeholder */}
              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Placeholder
                </label>
                <input
                  type="text"
                  value={editingVariable.placeholder || ''}
                  onChange={(e) =>
                    updateEditingVariable({ placeholder: e.target.value })
                  }
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                  placeholder="e.g., Enter customer name"
                />
              </div>

              {/* Default Value */}
              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Default Value
                </label>
                <input
                  type={
                    editingVariable.type === 'number' ||
                    editingVariable.type === 'currency'
                      ? 'number'
                      : editingVariable.type === 'date'
                        ? 'date'
                        : 'text'
                  }
                  value={
                    editingVariable.default_value instanceof Date
                      ? editingVariable.default_value
                          .toISOString()
                          .split('T')[0]
                      : (editingVariable.default_value ?? '')
                  }
                  onChange={(e) =>
                    updateEditingVariable({
                      default_value:
                        editingVariable.type === 'number' ||
                        editingVariable.type === 'currency'
                          ? parseFloat(e.target.value) || null
                          : e.target.value,
                    })
                  }
                  className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>

              {/* Required */}
              <div>
                <label className="flex items-center">
                  <input
                    type="checkbox"
                    checked={editingVariable.required}
                    onChange={(e) =>
                      updateEditingVariable({ required: e.target.checked })
                    }
                    className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
                  />
                  <span className="ml-2 text-sm text-gray-700">
                    Required field
                  </span>
                </label>
              </div>

              {/* Validation Rules */}
              <div className="rounded-md border border-gray-200 p-4">
                <h4 className="mb-3 text-sm font-medium text-gray-700">
                  Validation Rules
                </h4>

                {editingVariable.type === 'text' && (
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-xs font-medium text-gray-700">
                        Min Length
                      </label>
                      <input
                        type="number"
                        min="0"
                        value={editingVariable.validation?.min_length || ''}
                        onChange={(e) =>
                          updateValidation({
                            min_length: e.target.value
                              ? parseInt(e.target.value)
                              : undefined,
                          })
                        }
                        className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-700">
                        Max Length
                      </label>
                      <input
                        type="number"
                        min="0"
                        value={editingVariable.validation?.max_length || ''}
                        onChange={(e) =>
                          updateValidation({
                            max_length: e.target.value
                              ? parseInt(e.target.value)
                              : undefined,
                          })
                        }
                        className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                      />
                    </div>
                    <div className="col-span-2">
                      <label className="block text-xs font-medium text-gray-700">
                        Pattern (Regex)
                      </label>
                      <input
                        type="text"
                        value={editingVariable.validation?.pattern || ''}
                        onChange={(e) =>
                          updateValidation({
                            pattern: e.target.value || undefined,
                          })
                        }
                        className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 font-mono text-sm"
                        placeholder="e.g., ^[A-Z].*"
                      />
                    </div>
                  </div>
                )}

                {(editingVariable.type === 'number' ||
                  editingVariable.type === 'currency') && (
                  <div className="grid grid-cols-2 gap-3">
                    <div>
                      <label className="block text-xs font-medium text-gray-700">
                        Min Value
                      </label>
                      <input
                        type="number"
                        step="any"
                        value={editingVariable.validation?.min_value || ''}
                        onChange={(e) =>
                          updateValidation({
                            min_value: e.target.value
                              ? parseFloat(e.target.value)
                              : undefined,
                          })
                        }
                        className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                      />
                    </div>
                    <div>
                      <label className="block text-xs font-medium text-gray-700">
                        Max Value
                      </label>
                      <input
                        type="number"
                        step="any"
                        value={editingVariable.validation?.max_value || ''}
                        onChange={(e) =>
                          updateValidation({
                            max_value: e.target.value
                              ? parseFloat(e.target.value)
                              : undefined,
                          })
                        }
                        className="mt-1 block w-full rounded-md border border-gray-300 px-3 py-2 text-sm"
                      />
                    </div>
                  </div>
                )}
              </div>
            </div>

            {/* Actions */}
            <div className="mt-6 flex justify-end gap-3">
              <button
                type="button"
                onClick={handleCancelEdit}
                className="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleSaveVariable}
                disabled={!editingVariable.name || !editingVariable.label}
                className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
              >
                Save Variable
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

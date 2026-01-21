import React, { useState, useRef } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { templateService } from '../../services/templateService';
import type { TemplateImportExport } from '../../types/template';

interface TemplateImportExportProps {
  selectedTemplateIds?: string[];
  onImportComplete?: (count: number) => void;
}

export const TemplateImportExportComponent: React.FC<
  TemplateImportExportProps
> = ({ selectedTemplateIds = [], onImportComplete }) => {
  const queryClient = useQueryClient();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [overwriteExisting, setOverwriteExisting] = useState(false);
  const [importPreview, setImportPreview] =
    useState<TemplateImportExport | null>(null);
  const [showImportDialog, setShowImportDialog] = useState(false);

  // Export mutation
  const exportMutation = useMutation({
    mutationFn: async (templateIds?: string[]) => {
      await templateService.downloadTemplatesJSON(templateIds);
    },
  });

  // Import mutation
  const importMutation = useMutation({
    mutationFn: async ({
      file,
      overwrite,
    }: {
      file: File;
      overwrite: boolean;
    }) => {
      return templateService.uploadTemplatesJSON(file, overwrite);
    },
    onSuccess: (count) => {
      queryClient.invalidateQueries({ queryKey: ['templates'] });
      setShowImportDialog(false);
      setImportPreview(null);
      onImportComplete?.(count);
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    },
  });

  const handleExportAll = () => {
    exportMutation.mutate(undefined);
  };

  const handleExportSelected = () => {
    if (selectedTemplateIds.length === 0) {
      alert('Please select at least one template to export');
      return;
    }
    exportMutation.mutate(selectedTemplateIds);
  };

  const handleFileSelect = async (
    event: React.ChangeEvent<HTMLInputElement>
  ) => {
    const file = event.target.files?.[0];
    if (!file) return;

    try {
      const text = await file.text();
      const data: TemplateImportExport = JSON.parse(text);

      // Validate the import data
      if (!data.templates || !Array.isArray(data.templates)) {
        throw new Error('Invalid template file format');
      }

      setImportPreview(data);
      setShowImportDialog(true);
    } catch (error) {
      alert(
        error instanceof Error
          ? `Error reading file: ${error.message}`
          : 'Invalid file format'
      );
      if (fileInputRef.current) {
        fileInputRef.current.value = '';
      }
    }
  };

  const handleConfirmImport = () => {
    if (!fileInputRef.current?.files?.[0]) return;

    importMutation.mutate({
      file: fileInputRef.current.files[0],
      overwrite: overwriteExisting,
    });
  };

  const handleCancelImport = () => {
    setShowImportDialog(false);
    setImportPreview(null);
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  return (
    <div className="space-y-6">
      {/* Export Section */}
      <div className="rounded-lg border bg-white p-6">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-gray-900">
              Export Templates
            </h3>
            <p className="mt-1 text-sm text-gray-500">
              Download templates as JSON files for backup or sharing
            </p>
          </div>
          <svg
            className="h-8 w-8 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M9 19l3 3m0 0l3-3m-3 3V10"
            />
          </svg>
        </div>

        <div className="mt-6 flex gap-3">
          <button
            type="button"
            onClick={handleExportAll}
            disabled={exportMutation.isPending}
            className="flex-1 rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
          >
            {exportMutation.isPending ? 'Exporting...' : 'Export All Templates'}
          </button>
          <button
            type="button"
            onClick={handleExportSelected}
            disabled={
              exportMutation.isPending || selectedTemplateIds.length === 0
            }
            className="flex-1 rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:opacity-50"
          >
            {exportMutation.isPending
              ? 'Exporting...'
              : `Export Selected (${selectedTemplateIds.length})`}
          </button>
        </div>

        {exportMutation.isError && (
          <div className="mt-4 rounded-md bg-red-50 p-4">
            <p className="text-sm text-red-800">
              {exportMutation.error instanceof Error
                ? exportMutation.error.message
                : 'Export failed'}
            </p>
          </div>
        )}

        {exportMutation.isSuccess && (
          <div className="mt-4 rounded-md bg-green-50 p-4">
            <p className="text-sm text-green-800">
              Templates exported successfully!
            </p>
          </div>
        )}
      </div>

      {/* Import Section */}
      <div className="rounded-lg border bg-white p-6">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-gray-900">
              Import Templates
            </h3>
            <p className="mt-1 text-sm text-gray-500">
              Upload a JSON file to import templates into your library
            </p>
          </div>
          <svg
            className="h-8 w-8 text-gray-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"
            />
          </svg>
        </div>

        <div className="mt-6">
          <label className="block">
            <span className="sr-only">Choose template file</span>
            <input
              ref={fileInputRef}
              type="file"
              accept=".json"
              onChange={handleFileSelect}
              className="block w-full text-sm text-gray-500 file:mr-4 file:rounded-md file:border-0 file:bg-blue-50 file:px-4 file:py-2 file:text-sm file:font-medium file:text-blue-700 hover:file:bg-blue-100"
            />
          </label>
          <p className="mt-2 text-xs text-gray-500">
            Upload a JSON file exported from this or another ServicePro instance
          </p>
        </div>

        <div className="mt-4">
          <label className="flex items-center">
            <input
              type="checkbox"
              checked={overwriteExisting}
              onChange={(e) => setOverwriteExisting(e.target.checked)}
              className="h-4 w-4 rounded border-gray-300 text-blue-600 focus:ring-blue-500"
            />
            <span className="ml-2 text-sm text-gray-700">
              Overwrite existing templates with same name
            </span>
          </label>
          <p className="ml-6 mt-1 text-xs text-gray-500">
            If unchecked, templates with duplicate names will be skipped
          </p>
        </div>

        {importMutation.isError && (
          <div className="mt-4 rounded-md bg-red-50 p-4">
            <p className="text-sm text-red-800">
              {importMutation.error instanceof Error
                ? importMutation.error.message
                : 'Import failed'}
            </p>
          </div>
        )}
      </div>

      {/* Import Confirmation Dialog */}
      {showImportDialog && importPreview && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black bg-opacity-50">
          <div className="max-h-[80vh] w-full max-w-2xl overflow-y-auto rounded-lg bg-white p-6 shadow-xl">
            <h3 className="text-lg font-semibold text-gray-900">
              Confirm Import
            </h3>

            <div className="mt-4 space-y-4">
              {/* Import Summary */}
              <div className="rounded-md bg-blue-50 p-4">
                <h4 className="text-sm font-medium text-blue-900">
                  Import Summary
                </h4>
                <dl className="mt-2 grid grid-cols-2 gap-3 text-sm">
                  <div>
                    <dt className="text-blue-700">File Version</dt>
                    <dd className="mt-1 font-medium text-blue-900">
                      {importPreview.version}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-blue-700">Exported Date</dt>
                    <dd className="mt-1 font-medium text-blue-900">
                      {new Date(importPreview.exported_at).toLocaleDateString()}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-blue-700">Templates</dt>
                    <dd className="mt-1 font-medium text-blue-900">
                      {importPreview.templates.length}
                    </dd>
                  </div>
                  <div>
                    <dt className="text-blue-700">Categories</dt>
                    <dd className="mt-1 font-medium text-blue-900">
                      {importPreview.categories?.length || 0}
                    </dd>
                  </div>
                </dl>
              </div>

              {/* Template List */}
              <div>
                <h4 className="text-sm font-medium text-gray-700">
                  Templates to Import ({importPreview.templates.length})
                </h4>
                <div className="mt-2 max-h-64 overflow-y-auto rounded-md border">
                  <ul className="divide-y divide-gray-200">
                    {importPreview.templates.map((template, index) => (
                      <li key={index} className="p-3">
                        <div className="flex items-center justify-between">
                          <div className="flex-1">
                            <p className="text-sm font-medium text-gray-900">
                              {template.name}
                            </p>
                            <p className="text-xs text-gray-500">
                              {template.category}
                            </p>
                          </div>
                          <div className="flex gap-2">
                            {template.is_public && (
                              <span className="inline-flex items-center rounded-full bg-blue-100 px-2 py-0.5 text-xs font-medium text-blue-800">
                                Public
                              </span>
                            )}
                            <span className="inline-flex items-center rounded-full bg-gray-100 px-2 py-0.5 text-xs font-medium text-gray-800">
                              {template.variables.length} vars
                            </span>
                          </div>
                        </div>
                      </li>
                    ))}
                  </ul>
                </div>
              </div>

              {/* Categories */}
              {importPreview.categories &&
                importPreview.categories.length > 0 && (
                  <div>
                    <h4 className="text-sm font-medium text-gray-700">
                      Categories ({importPreview.categories.length})
                    </h4>
                    <div className="mt-2 flex flex-wrap gap-2">
                      {importPreview.categories.map((category, index) => (
                        <span
                          key={index}
                          className="inline-flex items-center rounded-full bg-gray-100 px-3 py-1 text-sm"
                          style={{ backgroundColor: `${category.color}20` }}
                        >
                          <span className="mr-1">{category.icon}</span>
                          {category.name}
                        </span>
                      ))}
                    </div>
                  </div>
                )}

              {/* Overwrite Warning */}
              {overwriteExisting && (
                <div className="rounded-md bg-amber-50 p-4">
                  <div className="flex">
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
                    <div className="ml-3">
                      <h3 className="text-sm font-medium text-amber-800">
                        Overwrite Mode Enabled
                      </h3>
                      <p className="mt-1 text-sm text-amber-700">
                        Existing templates with the same name will be
                        overwritten. This action cannot be undone.
                      </p>
                    </div>
                  </div>
                </div>
              )}
            </div>

            {/* Actions */}
            <div className="mt-6 flex justify-end gap-3">
              <button
                type="button"
                onClick={handleCancelImport}
                disabled={importMutation.isPending}
                className="rounded-md border border-gray-300 bg-white px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
              >
                Cancel
              </button>
              <button
                type="button"
                onClick={handleConfirmImport}
                disabled={importMutation.isPending}
                className="rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
              >
                {importMutation.isPending
                  ? 'Importing...'
                  : `Import ${importPreview.templates.length} Template${
                      importPreview.templates.length !== 1 ? 's' : ''
                    }`}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};

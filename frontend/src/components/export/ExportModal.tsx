import React, { useState, useEffect } from 'react';
import {
  ExportFormat,
  ExportType,
  ExportConfig,
  ExportJobResponse,
  ExportProgress,
  getFormatLabel,
  getFormatIcon,
  formatFileSize,
} from '../../types/export';
import exportService from '../../services/exportService';

interface ExportModalProps {
  isOpen: boolean;
  onClose: () => void;
  exportType: ExportType;
  query?: Record<string, unknown>;
  title?: string;
  availableFormats?: ExportFormat[];
}

type ExportPhase = 'configure' | 'exporting' | 'completed' | 'error';

const ExportModal: React.FC<ExportModalProps> = ({
  isOpen,
  onClose,
  exportType,
  query,
  title,
  availableFormats = ['csv', 'json', 'xlsx', 'pdf'],
}) => {
  const [phase, setPhase] = useState<ExportPhase>('configure');
  const [selectedFormat, setSelectedFormat] = useState<ExportFormat>('csv');
  const [job, setJob] = useState<ExportJobResponse | null>(null);
  const [progress, setProgress] = useState<ExportProgress | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Reset state when modal opens
  useEffect(() => {
    if (isOpen) {
      setPhase('configure');
      setJob(null);
      setProgress(null);
      setError(null);
    }
  }, [isOpen]);

  const handleStartExport = async () => {
    try {
      setPhase('exporting');
      setError(null);

      const config: ExportConfig = {
        format: selectedFormat,
        export_type: exportType,
        query,
      };

      // Start the export
      const exportJob = await exportService.startExport(config);
      setJob(exportJob);

      // Poll for status updates
      const completedJob = await exportService.pollExportStatus(
        exportJob.id,
        (updatedJob) => {
          setJob(updatedJob);
          setProgress({
            job_id: updatedJob.id,
            status: updatedJob.status,
            total_records: updatedJob.total_records,
            processed_records: updatedJob.processed_records,
            progress: updatedJob.progress,
            current_phase: 'processing',
            estimated_time_ms: 0,
            elapsed_time_ms: 0,
          });
        }
      );

      setJob(completedJob);
      setPhase('completed');
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Export failed');
      setPhase('error');
    }
  };

  const handleDownload = async () => {
    if (job) {
      await exportService.downloadExport(job.id, job.filename);
    }
  };

  const handleCancel = async () => {
    if (job && (job.status === 'pending' || job.status === 'processing')) {
      try {
        await exportService.cancelExport(job.id);
        setPhase('configure');
        setJob(null);
      } catch {
        setError('Failed to cancel export');
      }
    }
  };

  const handleClose = () => {
    if (phase === 'exporting') {
      // Confirm before closing during export
      if (
        window.confirm('Export is in progress. Are you sure you want to close?')
      ) {
        handleCancel();
        onClose();
      }
    } else {
      onClose();
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 overflow-y-auto">
      <div className="flex min-h-screen items-center justify-center p-4">
        {/* Backdrop */}
        <div
          className="fixed inset-0 bg-black bg-opacity-50 transition-opacity"
          onClick={handleClose}
        />

        {/* Modal */}
        <div className="relative bg-white rounded-lg shadow-xl max-w-md w-full p-6">
          {/* Header */}
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-xl font-semibold text-gray-900">
              {title ||
                `Export ${
                  exportType.charAt(0).toUpperCase() + exportType.slice(1)
                }`}
            </h2>
            <button
              onClick={handleClose}
              className="text-gray-400 hover:text-gray-500 transition-colors"
            >
              <svg
                className="w-6 h-6"
                fill="none"
                stroke="currentColor"
                viewBox="0 0 24 24"
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  strokeWidth={2}
                  d="M6 18L18 6M6 6l12 12"
                />
              </svg>
            </button>
          </div>

          {/* Configure Phase */}
          {phase === 'configure' && (
            <div>
              <p className="text-sm text-gray-600 mb-4">
                Select a format for your export:
              </p>

              <div className="grid grid-cols-2 gap-3 mb-6">
                {availableFormats.map((format) => (
                  <button
                    key={format}
                    onClick={() => setSelectedFormat(format)}
                    className={`flex items-center p-4 rounded-lg border-2 transition-all ${
                      selectedFormat === format
                        ? 'border-blue-500 bg-blue-50'
                        : 'border-gray-200 hover:border-gray-300'
                    }`}
                  >
                    <span className="text-2xl mr-3">
                      {getFormatIcon(format)}
                    </span>
                    <div className="text-left">
                      <div className="font-medium text-gray-900">
                        {getFormatLabel(format)}
                      </div>
                      <div className="text-xs text-gray-500">
                        {format === 'csv' && 'Spreadsheet compatible'}
                        {format === 'json' && 'For developers'}
                        {format === 'xlsx' && 'Microsoft Excel'}
                        {format === 'pdf' && 'Printable document'}
                      </div>
                    </div>
                  </button>
                ))}
              </div>

              <div className="flex justify-end gap-3">
                <button
                  onClick={onClose}
                  className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors"
                >
                  Cancel
                </button>
                <button
                  onClick={handleStartExport}
                  className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors flex items-center"
                >
                  <svg
                    className="w-4 h-4 mr-2"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                    />
                  </svg>
                  Start Export
                </button>
              </div>
            </div>
          )}

          {/* Exporting Phase */}
          {phase === 'exporting' && (
            <div className="text-center">
              <div className="mb-4">
                <div className="w-16 h-16 mx-auto mb-4">
                  <svg
                    className="animate-spin w-full h-full text-blue-600"
                    fill="none"
                    viewBox="0 0 24 24"
                  >
                    <circle
                      className="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      strokeWidth="4"
                    />
                    <path
                      className="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    />
                  </svg>
                </div>
                <h3 className="text-lg font-medium text-gray-900 mb-2">
                  Exporting...
                </h3>
                <p className="text-sm text-gray-600">
                  {progress?.message ||
                    'Please wait while your export is being prepared.'}
                </p>
              </div>

              {/* Progress Bar */}
              {job && (
                <div className="mb-6">
                  <div className="flex justify-between text-sm text-gray-600 mb-1">
                    <span>
                      {job.processed_records} / {job.total_records} records
                    </span>
                    <span>{Math.round(job.progress)}%</span>
                  </div>
                  <div className="w-full bg-gray-200 rounded-full h-2">
                    <div
                      className="bg-blue-600 h-2 rounded-full transition-all duration-300"
                      style={{ width: `${job.progress}%` }}
                    />
                  </div>
                </div>
              )}

              <button
                onClick={handleCancel}
                className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors"
              >
                Cancel Export
              </button>
            </div>
          )}

          {/* Completed Phase */}
          {phase === 'completed' && job && (
            <div className="text-center">
              <div className="w-16 h-16 mx-auto mb-4 bg-green-100 rounded-full flex items-center justify-center">
                <svg
                  className="w-8 h-8 text-green-600"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M5 13l4 4L19 7"
                  />
                </svg>
              </div>
              <h3 className="text-lg font-medium text-gray-900 mb-2">
                Export Complete!
              </h3>
              <p className="text-sm text-gray-600 mb-4">
                Your {getFormatLabel(job.format)} export is ready for download.
              </p>

              <div className="bg-gray-50 rounded-lg p-4 mb-6">
                <div className="flex items-center justify-between text-sm">
                  <span className="text-gray-600">File:</span>
                  <span className="font-medium text-gray-900">
                    {job.filename}
                  </span>
                </div>
                <div className="flex items-center justify-between text-sm mt-2">
                  <span className="text-gray-600">Size:</span>
                  <span className="font-medium text-gray-900">
                    {formatFileSize(job.file_size || 0)}
                  </span>
                </div>
                <div className="flex items-center justify-between text-sm mt-2">
                  <span className="text-gray-600">Records:</span>
                  <span className="font-medium text-gray-900">
                    {job.total_records.toLocaleString()}
                  </span>
                </div>
              </div>

              <div className="flex justify-center gap-3">
                <button
                  onClick={onClose}
                  className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors"
                >
                  Close
                </button>
                <button
                  onClick={handleDownload}
                  className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors flex items-center"
                >
                  <svg
                    className="w-4 h-4 mr-2"
                    fill="none"
                    stroke="currentColor"
                    viewBox="0 0 24 24"
                  >
                    <path
                      strokeLinecap="round"
                      strokeLinejoin="round"
                      strokeWidth={2}
                      d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"
                    />
                  </svg>
                  Download
                </button>
              </div>
            </div>
          )}

          {/* Error Phase */}
          {phase === 'error' && (
            <div className="text-center">
              <div className="w-16 h-16 mx-auto mb-4 bg-red-100 rounded-full flex items-center justify-center">
                <svg
                  className="w-8 h-8 text-red-600"
                  fill="none"
                  stroke="currentColor"
                  viewBox="0 0 24 24"
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    strokeWidth={2}
                    d="M6 18L18 6M6 6l12 12"
                  />
                </svg>
              </div>
              <h3 className="text-lg font-medium text-gray-900 mb-2">
                Export Failed
              </h3>
              <p className="text-sm text-red-600 mb-6">
                {error || 'An error occurred while exporting your data.'}
              </p>

              <div className="flex justify-center gap-3">
                <button
                  onClick={onClose}
                  className="px-4 py-2 text-sm font-medium text-gray-700 bg-gray-100 rounded-lg hover:bg-gray-200 transition-colors"
                >
                  Close
                </button>
                <button
                  onClick={() => setPhase('configure')}
                  className="px-4 py-2 text-sm font-medium text-white bg-blue-600 rounded-lg hover:bg-blue-700 transition-colors"
                >
                  Try Again
                </button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
};

export default ExportModal;

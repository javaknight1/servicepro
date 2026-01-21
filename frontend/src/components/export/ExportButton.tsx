import React, { useState, useRef, useEffect } from 'react';
import {
  ExportFormat,
  ExportType,
  ExportConfig,
  getFormatLabel,
  getFormatIcon,
} from '../../types/export';

interface ExportButtonProps {
  exportType: ExportType;
  query?: Record<string, unknown>;
  onExportStart?: (config: ExportConfig) => void;
  disabled?: boolean;
  className?: string;
  variant?: 'primary' | 'secondary' | 'outline';
  size?: 'sm' | 'md' | 'lg';
  showFormatDropdown?: boolean;
  defaultFormat?: ExportFormat;
  availableFormats?: ExportFormat[];
  label?: string;
}

const ExportButton: React.FC<ExportButtonProps> = ({
  exportType,
  query,
  onExportStart,
  disabled = false,
  className = '',
  variant = 'secondary',
  size = 'md',
  showFormatDropdown = true,
  defaultFormat = 'csv',
  availableFormats = ['csv', 'json', 'xlsx', 'pdf'],
  label = 'Export',
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [selectedFormat, setSelectedFormat] =
    useState<ExportFormat>(defaultFormat);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(event.target as Node)
      ) {
        setIsOpen(false);
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleExport = (format: ExportFormat) => {
    const config: ExportConfig = {
      format,
      export_type: exportType,
      query,
    };
    onExportStart?.(config);
    setIsOpen(false);
  };

  const baseClasses =
    'inline-flex items-center justify-center font-medium rounded-lg transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2';

  const sizeClasses = {
    sm: 'px-3 py-1.5 text-sm',
    md: 'px-4 py-2 text-sm',
    lg: 'px-6 py-3 text-base',
  };

  const variantClasses = {
    primary: 'bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500',
    secondary:
      'bg-gray-100 text-gray-700 hover:bg-gray-200 focus:ring-gray-500',
    outline:
      'border border-gray-300 text-gray-700 hover:bg-gray-50 focus:ring-gray-500',
  };

  const buttonClasses = `${baseClasses} ${sizeClasses[size]} ${variantClasses[variant]} ${className}`;

  if (!showFormatDropdown) {
    return (
      <button
        type="button"
        onClick={() => handleExport(selectedFormat)}
        disabled={disabled}
        className={`${buttonClasses} ${
          disabled ? 'opacity-50 cursor-not-allowed' : ''
        }`}
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
        {label}
      </button>
    );
  }

  return (
    <div className="relative inline-block" ref={dropdownRef}>
      <div className="inline-flex rounded-lg shadow-sm">
        <button
          type="button"
          onClick={() => handleExport(selectedFormat)}
          disabled={disabled}
          className={`${buttonClasses} rounded-r-none ${
            disabled ? 'opacity-50 cursor-not-allowed' : ''
          }`}
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
          {label} as {getFormatLabel(selectedFormat)}
        </button>
        <button
          type="button"
          onClick={() => setIsOpen(!isOpen)}
          disabled={disabled}
          className={`${
            variantClasses[variant]
          } px-2 py-2 rounded-l-none rounded-r-lg border-l border-gray-300 ${
            disabled ? 'opacity-50 cursor-not-allowed' : ''
          }`}
        >
          <svg
            className="w-4 h-4"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M19 9l-7 7-7-7"
            />
          </svg>
        </button>
      </div>

      {isOpen && (
        <div className="absolute right-0 mt-2 w-48 bg-white rounded-lg shadow-lg border border-gray-200 py-1 z-50">
          {availableFormats.map((format) => (
            <button
              key={format}
              onClick={() => {
                setSelectedFormat(format);
                handleExport(format);
              }}
              className="w-full px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 flex items-center"
            >
              <span className="mr-2">{getFormatIcon(format)}</span>
              {getFormatLabel(format)}
              {format === selectedFormat && (
                <svg
                  className="w-4 h-4 ml-auto text-blue-600"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path
                    fillRule="evenodd"
                    d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                    clipRule="evenodd"
                  />
                </svg>
              )}
            </button>
          ))}
        </div>
      )}
    </div>
  );
};

export default ExportButton;

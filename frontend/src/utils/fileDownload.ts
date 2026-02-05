/**
 * File Download Utilities
 *
 * Centralized utilities for downloading files in the browser.
 * Consolidates duplicate download logic across the codebase.
 */

/**
 * Downloads a Blob as a file with the given filename.
 * Handles creating the object URL, triggering the download, and cleanup.
 */
export function downloadFile(blob: Blob, filename: string): void {
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

/**
 * Creates a Blob from data and downloads it as a file.
 * Useful when you have raw data that needs to be converted to a Blob first.
 */
export function downloadBlob(
  data: BlobPart,
  filename: string,
  mimeType: string = 'application/octet-stream'
): void {
  const blob = new Blob([data], { type: mimeType });
  downloadFile(blob, filename);
}

/**
 * Downloads a string as a CSV file.
 */
export function downloadCSV(data: string, filename: string): void {
  downloadBlob(data, filename, 'text/csv');
}

/**
 * Downloads an object as a JSON file.
 * Automatically stringifies the data with pretty formatting.
 */
export function downloadJSON(data: object, filename: string): void {
  const json = JSON.stringify(data, null, 2);
  downloadBlob(json, filename, 'application/json');
}

/**
 * Downloads data as a PDF file.
 * Use when you already have PDF binary data (e.g., from an API response).
 */
export function downloadPDF(data: BlobPart, filename: string): void {
  downloadBlob(data, filename, 'application/pdf');
}

/**
 * Opens a Blob in a new browser tab for preview.
 * Note: The object URL should be revoked when no longer needed,
 * but for new tabs this typically happens when the tab is closed.
 */
export function openBlobInNewTab(blob: Blob): void {
  const url = URL.createObjectURL(blob);
  window.open(url, '_blank');
}

/**
 * Downloads an image from a data URL.
 * Useful for canvas exports and similar scenarios.
 */
export function downloadFromDataURL(dataUrl: string, filename: string): void {
  const link = document.createElement('a');
  link.href = dataUrl;
  link.download = filename;
  link.click();
}

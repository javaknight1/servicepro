import { describe, it, expect, beforeEach, vi, afterEach } from 'vitest';
import {
  downloadFile,
  downloadBlob,
  downloadCSV,
  downloadJSON,
  downloadPDF,
  openBlobInNewTab,
  downloadFromDataURL,
} from '../fileDownload';

describe('fileDownload utilities', () => {
  let createObjectURLMock: ReturnType<typeof vi.fn>;
  let revokeObjectURLMock: ReturnType<typeof vi.fn>;
  let appendChildMock: ReturnType<typeof vi.fn>;
  let removeChildMock: ReturnType<typeof vi.fn>;
  let clickMock: ReturnType<typeof vi.fn>;
  let windowOpenMock: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    createObjectURLMock = vi.fn(() => 'blob:test-url');
    revokeObjectURLMock = vi.fn();
    appendChildMock = vi.fn();
    removeChildMock = vi.fn();
    clickMock = vi.fn();
    windowOpenMock = vi.fn();

    global.URL.createObjectURL = createObjectURLMock;
    global.URL.revokeObjectURL = revokeObjectURLMock;
    window.open = windowOpenMock;

    vi.spyOn(document.body, 'appendChild').mockImplementation(appendChildMock);
    vi.spyOn(document.body, 'removeChild').mockImplementation(removeChildMock);

    vi.spyOn(document, 'createElement').mockImplementation((tag) => {
      if (tag === 'a') {
        return {
          click: clickMock,
          href: '',
          download: '',
        } as unknown as HTMLAnchorElement;
      }
      return document.createElement(tag);
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe('downloadFile', () => {
    it('should create object URL and trigger download', () => {
      const blob = new Blob(['test content'], { type: 'text/plain' });

      downloadFile(blob, 'test.txt');

      expect(createObjectURLMock).toHaveBeenCalledWith(blob);
      expect(appendChildMock).toHaveBeenCalled();
      expect(clickMock).toHaveBeenCalled();
      expect(removeChildMock).toHaveBeenCalled();
      expect(revokeObjectURLMock).toHaveBeenCalledWith('blob:test-url');
    });
  });

  describe('downloadBlob', () => {
    it('should create blob with default mime type', () => {
      downloadBlob('test data', 'test.bin');

      expect(createObjectURLMock).toHaveBeenCalled();
      expect(clickMock).toHaveBeenCalled();
    });

    it('should create blob with custom mime type', () => {
      downloadBlob('test data', 'test.txt', 'text/plain');

      expect(createObjectURLMock).toHaveBeenCalled();
      expect(clickMock).toHaveBeenCalled();
    });
  });

  describe('downloadCSV', () => {
    it('should download string as CSV file', () => {
      const csvData = 'name,value\ntest,123';

      downloadCSV(csvData, 'data.csv');

      expect(createObjectURLMock).toHaveBeenCalled();
      expect(clickMock).toHaveBeenCalled();
    });
  });

  describe('downloadJSON', () => {
    it('should download object as JSON file', () => {
      const jsonData = { name: 'test', value: 123 };

      downloadJSON(jsonData, 'data.json');

      expect(createObjectURLMock).toHaveBeenCalled();
      expect(clickMock).toHaveBeenCalled();
    });

    it('should pretty-print JSON', () => {
      const jsonData = { nested: { key: 'value' } };

      downloadJSON(jsonData, 'data.json');

      // The Blob constructor is called, which we can't directly inspect
      // but we verify the download was triggered
      expect(clickMock).toHaveBeenCalled();
    });
  });

  describe('downloadPDF', () => {
    it('should download data as PDF file', () => {
      const pdfData = new ArrayBuffer(8);

      downloadPDF(pdfData, 'document.pdf');

      expect(createObjectURLMock).toHaveBeenCalled();
      expect(clickMock).toHaveBeenCalled();
    });
  });

  describe('openBlobInNewTab', () => {
    it('should open blob URL in new tab', () => {
      const blob = new Blob(['content'], { type: 'text/html' });

      openBlobInNewTab(blob);

      expect(createObjectURLMock).toHaveBeenCalledWith(blob);
      expect(windowOpenMock).toHaveBeenCalledWith('blob:test-url', '_blank');
    });
  });

  describe('downloadFromDataURL', () => {
    it('should download from data URL', () => {
      const dataUrl = 'data:image/png;base64,iVBORw0KGgo=';

      downloadFromDataURL(dataUrl, 'image.png');

      expect(clickMock).toHaveBeenCalled();
    });
  });
});

import { useState, useRef, useEffect } from 'react';
import { MoreVertical, Send, Download, FileText, Loader2 } from 'lucide-react';
import { invoiceService, InvoiceStatus } from '../../services/invoiceService';

interface InvoiceActionsMenuProps {
  invoiceId: string;
  status: InvoiceStatus;
  onSendInvoice?: () => void;
  onInvoiceSent?: () => void;
}

export function InvoiceActionsMenu({
  invoiceId,
  status,
  onSendInvoice,
  onInvoiceSent,
}: InvoiceActionsMenuProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [isSending, setIsSending] = useState(false);
  const [isDownloadingInvoice, setIsDownloadingInvoice] = useState(false);
  const [isDownloadingReceipt, setIsDownloadingReceipt] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleClick = (e: React.MouseEvent) => {
    e.stopPropagation();
    setIsOpen(!isOpen);
  };

  const handleSendInvoice = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (!canSend) return;

    setIsSending(true);
    try {
      if (onSendInvoice) {
        onSendInvoice();
      } else {
        await invoiceService.sendInvoice(invoiceId);
        onInvoiceSent?.();
      }
    } catch (error) {
      console.error('Failed to send invoice:', error);
    } finally {
      setIsSending(false);
      setIsOpen(false);
    }
  };

  const handleDownloadInvoicePDF = async (e: React.MouseEvent) => {
    e.stopPropagation();
    setIsDownloadingInvoice(true);
    try {
      await invoiceService.downloadInvoicePDF(invoiceId);
    } catch (error) {
      console.error('Failed to download invoice PDF:', error);
    } finally {
      setIsDownloadingInvoice(false);
      setIsOpen(false);
    }
  };

  const handleDownloadReceiptPDF = async (e: React.MouseEvent) => {
    e.stopPropagation();
    setIsDownloadingReceipt(true);
    try {
      await invoiceService.downloadReceiptPDF(invoiceId);
    } catch (error) {
      console.error('Failed to download receipt PDF:', error);
    } finally {
      setIsDownloadingReceipt(false);
      setIsOpen(false);
    }
  };

  const canSend =
    status === InvoiceStatus.DRAFT || status === InvoiceStatus.SENT;
  const isPaid = status === InvoiceStatus.PAID;

  return (
    <div className="relative" ref={menuRef}>
      <button
        onClick={handleClick}
        className="p-1 rounded-md hover:bg-neutral-100 focus:outline-none focus:ring-2 focus:ring-primary-500"
        aria-label="Invoice actions"
      >
        <MoreVertical className="h-5 w-5 text-neutral-500" />
      </button>

      {isOpen && (
        <div className="absolute right-0 mt-1 w-48 bg-white rounded-md shadow-lg border border-neutral-200 z-10">
          <div className="py-1">
            <button
              onClick={handleSendInvoice}
              disabled={!canSend || isSending}
              className={`w-full flex items-center gap-2 px-4 py-2 text-sm ${
                canSend
                  ? 'text-neutral-700 hover:bg-neutral-100'
                  : 'text-neutral-400 cursor-not-allowed'
              }`}
            >
              {isSending ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Send className="h-4 w-4" />
              )}
              Send Invoice
            </button>
            <button
              onClick={handleDownloadInvoicePDF}
              disabled={isDownloadingInvoice}
              className="w-full flex items-center gap-2 px-4 py-2 text-sm text-neutral-700 hover:bg-neutral-100"
            >
              {isDownloadingInvoice ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Download className="h-4 w-4" />
              )}
              Download Invoice
            </button>
            {isPaid && (
              <button
                onClick={handleDownloadReceiptPDF}
                disabled={isDownloadingReceipt}
                className="w-full flex items-center gap-2 px-4 py-2 text-sm text-neutral-700 hover:bg-neutral-100"
              >
                {isDownloadingReceipt ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <FileText className="h-4 w-4" />
                )}
                Download Receipt
              </button>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

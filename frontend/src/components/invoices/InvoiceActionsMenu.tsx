import { useState } from 'react';
import { Send, Download, FileText } from 'lucide-react';
import { invoiceService, InvoiceStatus } from '../../services/invoiceService';
import { DropdownMenu, DropdownMenuItem } from '../shared/DropdownMenu';

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
  const [isSending, setIsSending] = useState(false);
  const [isDownloadingInvoice, setIsDownloadingInvoice] = useState(false);
  const [isDownloadingReceipt, setIsDownloadingReceipt] = useState(false);

  const handleSendInvoice = async (e: React.MouseEvent, close: () => void) => {
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
      close();
    }
  };

  const handleDownloadInvoicePDF = async (
    e: React.MouseEvent,
    close: () => void
  ) => {
    e.stopPropagation();
    setIsDownloadingInvoice(true);
    try {
      await invoiceService.downloadInvoicePDF(invoiceId);
    } catch (error) {
      console.error('Failed to download invoice PDF:', error);
    } finally {
      setIsDownloadingInvoice(false);
      close();
    }
  };

  const handleDownloadReceiptPDF = async (
    e: React.MouseEvent,
    close: () => void
  ) => {
    e.stopPropagation();
    setIsDownloadingReceipt(true);
    try {
      await invoiceService.downloadReceiptPDF(invoiceId);
    } catch (error) {
      console.error('Failed to download receipt PDF:', error);
    } finally {
      setIsDownloadingReceipt(false);
      close();
    }
  };

  const canSend =
    status === InvoiceStatus.DRAFT || status === InvoiceStatus.SENT;
  const isPaid = status === InvoiceStatus.PAID;

  return (
    <DropdownMenu buttonLabel="Invoice actions">
      {(close) => (
        <>
          <DropdownMenuItem
            onClick={(e) => handleSendInvoice(e, close)}
            disabled={!canSend}
            isLoading={isSending}
            icon={<Send className="h-4 w-4" />}
            label="Send Invoice"
          />
          <DropdownMenuItem
            onClick={(e) => handleDownloadInvoicePDF(e, close)}
            isLoading={isDownloadingInvoice}
            icon={<Download className="h-4 w-4" />}
            label="Download Invoice"
          />
          {isPaid && (
            <DropdownMenuItem
              onClick={(e) => handleDownloadReceiptPDF(e, close)}
              isLoading={isDownloadingReceipt}
              icon={<FileText className="h-4 w-4" />}
              label="Download Receipt"
            />
          )}
        </>
      )}
    </DropdownMenu>
  );
}

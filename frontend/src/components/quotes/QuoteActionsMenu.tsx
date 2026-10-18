import { useState } from 'react';
import { useNavigate } from 'react-router';
import { Send, Download, Check, X, Copy } from 'lucide-react';
import { quoteService } from '../../services/quoteService';
import { getApiErrorInfo, getErrorMessage } from '../../utils/error';
import {
  DropdownMenu,
  DropdownMenuItem,
  DropdownMenuDivider,
} from '../shared/DropdownMenu';

interface QuoteActionsMenuProps {
  quoteId: string;
  status: string;
  onSendQuote?: () => void;
  onQuoteSent?: () => void;
  onStatusChange?: () => void;
  /** Shows a failed action's message; defaults to a browser alert. */
  onError?: (message: string) => void;
}

export function QuoteActionsMenu({
  quoteId,
  status,
  onSendQuote,
  onQuoteSent,
  onStatusChange,
  onError,
}: QuoteActionsMenuProps) {
  const navigate = useNavigate();
  const [isSending, setIsSending] = useState(false);
  const [isAccepting, setIsAccepting] = useState(false);
  const [isDeclining, setIsDeclining] = useState(false);
  const [isDownloading, setIsDownloading] = useState(false);

  // The API explains a refused change (e.g. "Quote cannot be accepted in
  // current status" after someone else declined it); show it.
  const reportError = (error: unknown, fallback: string) => {
    const message =
      getApiErrorInfo(error).message || getErrorMessage(error, fallback);
    if (onError) onError(message);
    else window.alert(message);
  };

  const handleSendQuote = async (e: React.MouseEvent, close: () => void) => {
    e.stopPropagation();
    if (status !== 'draft') return;

    setIsSending(true);
    try {
      if (onSendQuote) {
        onSendQuote();
      } else {
        await quoteService.sendQuote(quoteId);
        onQuoteSent?.();
      }
    } catch (error) {
      console.error('Failed to send quote:', error);
      reportError(error, 'Failed to send quote');
    } finally {
      setIsSending(false);
      close();
    }
  };

  const handleDownloadPDF = async (e: React.MouseEvent, close: () => void) => {
    e.stopPropagation();
    setIsDownloading(true);
    try {
      await quoteService.downloadQuotePDF(quoteId);
    } catch (error) {
      console.error('Failed to download PDF:', error);
    } finally {
      setIsDownloading(false);
      close();
    }
  };

  const handleAcceptQuote = async (e: React.MouseEvent, close: () => void) => {
    e.stopPropagation();
    setIsAccepting(true);
    try {
      await quoteService.acceptQuote(quoteId);
      onStatusChange?.();
    } catch (error) {
      console.error('Failed to accept quote:', error);
      reportError(error, 'Failed to accept quote');
    } finally {
      setIsAccepting(false);
      close();
    }
  };

  const handleDeclineQuote = async (e: React.MouseEvent, close: () => void) => {
    e.stopPropagation();
    setIsDeclining(true);
    try {
      await quoteService.rejectQuote(quoteId);
      onStatusChange?.();
    } catch (error) {
      console.error('Failed to decline quote:', error);
      reportError(error, 'Failed to decline quote');
    } finally {
      setIsDeclining(false);
      close();
    }
  };

  const handleCloneQuote = (e: React.MouseEvent, close: () => void) => {
    e.stopPropagation();
    close();
    navigate(`/quotes/new?clone=${quoteId}`);
  };

  const canSend = status === 'draft';
  const canAcceptDecline = status === 'sent' || status === 'viewed';

  return (
    <DropdownMenu buttonLabel="Quote actions">
      {(close) => (
        <>
          <DropdownMenuItem
            onClick={(e) => handleSendQuote(e, close)}
            disabled={!canSend}
            isLoading={isSending}
            icon={<Send className="h-4 w-4" />}
            label="Send Quote"
          />
          <DropdownMenuItem
            onClick={(e) => handleDownloadPDF(e, close)}
            isLoading={isDownloading}
            icon={<Download className="h-4 w-4" />}
            label="Download PDF"
          />
          <DropdownMenuItem
            onClick={(e) => handleCloneQuote(e, close)}
            icon={<Copy className="h-4 w-4" />}
            label="Clone Quote"
          />
          {canAcceptDecline && (
            <>
              <DropdownMenuDivider />
              <DropdownMenuItem
                onClick={(e) => handleAcceptQuote(e, close)}
                isLoading={isAccepting}
                icon={<Check className="h-4 w-4" />}
                label="Accept Quote"
                variant="success"
              />
              <DropdownMenuItem
                onClick={(e) => handleDeclineQuote(e, close)}
                isLoading={isDeclining}
                icon={<X className="h-4 w-4" />}
                label="Decline Quote"
                variant="danger"
              />
            </>
          )}
        </>
      )}
    </DropdownMenu>
  );
}

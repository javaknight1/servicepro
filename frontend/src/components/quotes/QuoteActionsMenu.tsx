import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Send, Download, Check, X, Copy } from 'lucide-react';
import { quoteService } from '../../services/quoteService';
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
}

export function QuoteActionsMenu({
  quoteId,
  status,
  onSendQuote,
  onQuoteSent,
  onStatusChange,
}: QuoteActionsMenuProps) {
  const navigate = useNavigate();
  const [isSending, setIsSending] = useState(false);
  const [isAccepting, setIsAccepting] = useState(false);
  const [isDeclining, setIsDeclining] = useState(false);
  const [isDownloading, setIsDownloading] = useState(false);

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

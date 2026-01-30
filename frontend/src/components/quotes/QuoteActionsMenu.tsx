import { useState, useRef, useEffect } from 'react';
import { MoreVertical, Send, Download, Loader2 } from 'lucide-react';
import { quoteService } from '../../services/quoteService';

interface QuoteActionsMenuProps {
  quoteId: string;
  status: string;
  onSendQuote?: () => void;
  onQuoteSent?: () => void;
}

export function QuoteActionsMenu({
  quoteId,
  status,
  onSendQuote,
  onQuoteSent,
}: QuoteActionsMenuProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [isSending, setIsSending] = useState(false);
  const [isDownloading, setIsDownloading] = useState(false);
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

  const handleSendQuote = async (e: React.MouseEvent) => {
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
      setIsOpen(false);
    }
  };

  const handleDownloadPDF = async (e: React.MouseEvent) => {
    e.stopPropagation();
    setIsDownloading(true);
    try {
      await quoteService.downloadQuotePDF(quoteId);
    } catch (error) {
      console.error('Failed to download PDF:', error);
    } finally {
      setIsDownloading(false);
      setIsOpen(false);
    }
  };

  const canSend = status === 'draft';

  return (
    <div className="relative" ref={menuRef}>
      <button
        onClick={handleClick}
        className="p-1 rounded-md hover:bg-neutral-100 focus:outline-none focus:ring-2 focus:ring-primary-500"
        aria-label="Quote actions"
      >
        <MoreVertical className="h-5 w-5 text-neutral-500" />
      </button>

      {isOpen && (
        <div className="absolute right-0 mt-1 w-48 bg-white rounded-md shadow-lg border border-neutral-200 z-10">
          <div className="py-1">
            <button
              onClick={handleSendQuote}
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
              Send Quote
            </button>
            <button
              onClick={handleDownloadPDF}
              disabled={isDownloading}
              className="w-full flex items-center gap-2 px-4 py-2 text-sm text-neutral-700 hover:bg-neutral-100"
            >
              {isDownloading ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Download className="h-4 w-4" />
              )}
              Download PDF
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

import React, { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { QuoteStatus } from '../../types/quote';
import { quoteService } from '../../services/quoteService';
import { useQuoteStatusUpdates } from '../../hooks/useWebSocket';

interface QuoteStatusActionsProps {
  quoteId: string;
  currentStatus: QuoteStatus;
  onStatusChange?: (newStatus: QuoteStatus) => void;
}

/**
 * Action buttons for quote status transitions with WebSocket notifications
 */
export const QuoteStatusActions: React.FC<QuoteStatusActionsProps> = ({
  quoteId,
  currentStatus,
  onStatusChange,
}) => {
  const queryClient = useQueryClient();
  const [showDeclineReason, setShowDeclineReason] = useState(false);
  const [declineReason, setDeclineReason] = useState('');

  // Subscribe to status updates
  const { lastMessage, isConnected } = useQuoteStatusUpdates(
    quoteId,
    (message) => {
      if (message.type === 'status_change') {
        onStatusChange?.(message.status as QuoteStatus);
      }
    }
  );

  // Send mutation
  const sendMutation = useMutation({
    mutationFn: () => quoteService.sendQuote(quoteId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['quote', quoteId] });
      queryClient.invalidateQueries({ queryKey: ['quotes'] });
    },
  });

  // Accept mutation
  const acceptMutation = useMutation({
    mutationFn: () => quoteService.acceptQuote(quoteId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['quote', quoteId] });
      queryClient.invalidateQueries({ queryKey: ['quotes'] });
    },
  });

  // Decline mutation
  const declineMutation = useMutation({
    mutationFn: () => quoteService.rejectQuote(quoteId),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['quote', quoteId] });
      queryClient.invalidateQueries({ queryKey: ['quotes'] });
      setShowDeclineReason(false);
      setDeclineReason('');
    },
  });

  const handleDecline = () => {
    if (!showDeclineReason) {
      setShowDeclineReason(true);
      return;
    }

    declineMutation.mutate();
  };

  // Determine available actions based on current status
  const canSend = currentStatus === QuoteStatus.DRAFT;
  const canAccept =
    currentStatus === QuoteStatus.SENT || currentStatus === QuoteStatus.VIEWED;
  const canDecline =
    currentStatus === QuoteStatus.SENT || currentStatus === QuoteStatus.VIEWED;

  return (
    <div className="space-y-4">
      {/* WebSocket connection indicator */}
      <div className="flex items-center gap-2 text-sm text-gray-500">
        <div
          className={`h-2 w-2 rounded-full ${
            isConnected ? 'bg-green-500' : 'bg-red-500'
          }`}
        />
        <span>{isConnected ? 'Connected' : 'Disconnected'}</span>
      </div>

      {/* Last update notification */}
      {lastMessage && (
        <div className="rounded-md bg-blue-50 p-3">
          <p className="text-sm text-blue-700">
            Status updated to{' '}
            <span className="font-medium">{lastMessage.status}</span>
          </p>
          <p className="mt-1 text-xs text-blue-600">
            {new Date(lastMessage.timestamp).toLocaleString()}
          </p>
        </div>
      )}

      {/* Action buttons */}
      <div className="flex flex-wrap gap-2">
        {canSend && (
          <button
            onClick={() => sendMutation.mutate()}
            disabled={sendMutation.isPending}
            className="inline-flex items-center gap-2 rounded-md bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-700 disabled:opacity-50"
          >
            {sendMutation.isPending ? (
              <>
                <svg
                  className="h-4 w-4 animate-spin"
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
                Sending...
              </>
            ) : (
              <>
                <svg
                  className="h-4 w-4"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path d="M2.003 5.884L10 9.882l7.997-3.998A2 2 0 0016 4H4a2 2 0 00-1.997 1.884z" />
                  <path d="M18 8.118l-8 4-8-4V14a2 2 0 002 2h12a2 2 0 002-2V8.118z" />
                </svg>
                Send to Customer
              </>
            )}
          </button>
        )}

        {canAccept && (
          <button
            onClick={() => acceptMutation.mutate()}
            disabled={acceptMutation.isPending}
            className="inline-flex items-center gap-2 rounded-md bg-green-600 px-4 py-2 text-sm font-medium text-white hover:bg-green-700 disabled:opacity-50"
          >
            {acceptMutation.isPending ? (
              <>
                <svg
                  className="h-4 w-4 animate-spin"
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
                Accepting...
              </>
            ) : (
              <>
                <svg
                  className="h-4 w-4"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path
                    fillRule="evenodd"
                    d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                    clipRule="evenodd"
                  />
                </svg>
                Accept Quote
              </>
            )}
          </button>
        )}

        {canDecline && (
          <div className="space-y-2">
            <button
              onClick={handleDecline}
              disabled={declineMutation.isPending}
              className="inline-flex items-center gap-2 rounded-md bg-red-600 px-4 py-2 text-sm font-medium text-white hover:bg-red-700 disabled:opacity-50"
            >
              {declineMutation.isPending ? (
                <>
                  <svg
                    className="h-4 w-4 animate-spin"
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
                  Declining...
                </>
              ) : (
                <>
                  <svg
                    className="h-4 w-4"
                    fill="currentColor"
                    viewBox="0 0 20 20"
                  >
                    <path
                      fillRule="evenodd"
                      d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                      clipRule="evenodd"
                    />
                  </svg>
                  Decline Quote
                </>
              )}
            </button>

            {showDeclineReason && (
              <div className="w-full">
                <textarea
                  value={declineReason}
                  onChange={(e) => setDeclineReason(e.target.value)}
                  placeholder="Reason for declining (optional)"
                  className="w-full rounded-md border-gray-300 shadow-sm focus:border-red-500 focus:ring-red-500"
                  rows={3}
                />
              </div>
            )}
          </div>
        )}
      </div>

      {/* Error display */}
      {(sendMutation.error ||
        acceptMutation.error ||
        declineMutation.error) && (
        <div className="rounded-md bg-red-50 p-3">
          <p className="text-sm text-red-700">
            {sendMutation.error instanceof Error
              ? sendMutation.error.message
              : acceptMutation.error instanceof Error
                ? acceptMutation.error.message
                : declineMutation.error instanceof Error
                  ? declineMutation.error.message
                  : 'An error occurred'}
          </p>
        </div>
      )}
    </div>
  );
};

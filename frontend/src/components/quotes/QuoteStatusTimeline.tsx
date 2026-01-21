import React from 'react';
import { QuoteStatus } from '../../types/quote';

export interface StatusHistoryItem {
  id: string;
  from_status: QuoteStatus;
  to_status: QuoteStatus;
  event: string;
  changed_by_type: string;
  reason?: string;
  created_at: string;
}

interface QuoteStatusTimelineProps {
  history: StatusHistoryItem[];
  className?: string;
}

/**
 * Timeline component showing quote status history
 */
export const QuoteStatusTimeline: React.FC<QuoteStatusTimelineProps> = ({
  history,
  className = '',
}) => {
  const formatDate = (dateString: string): string => {
    const date = new Date(dateString);
    return date.toLocaleString('en-US', {
      month: 'short',
      day: 'numeric',
      hour: 'numeric',
      minute: '2-digit',
    });
  };

  const getStatusColor = (status: QuoteStatus): string => {
    const colors = {
      [QuoteStatus.DRAFT]: 'bg-gray-400',
      [QuoteStatus.SENT]: 'bg-blue-500',
      [QuoteStatus.VIEWED]: 'bg-purple-500',
      [QuoteStatus.ACCEPTED]: 'bg-green-500',
      [QuoteStatus.DECLINED]: 'bg-red-500',
      [QuoteStatus.EXPIRED]: 'bg-yellow-500',
    };
    return colors[status] || 'bg-gray-400';
  };

  const getEventLabel = (event: string): string => {
    const labels: Record<string, string> = {
      'quote.created': 'Quote Created',
      'quote.sent': 'Sent to Customer',
      'quote.viewed': 'Viewed by Customer',
      'quote.accepted': 'Accepted',
      'quote.declined': 'Declined',
      'quote.expired': 'Expired',
      'quote.updated': 'Updated',
    };
    return labels[event] || event;
  };

  const getChangedByLabel = (type: string): string => {
    const labels: Record<string, string> = {
      user: 'by user',
      customer: 'by customer',
      system: 'automatically',
    };
    return labels[type] || type;
  };

  if (history.length === 0) {
    return (
      <div className="text-center py-8 text-gray-500">
        No status history available
      </div>
    );
  }

  return (
    <div className={`space-y-4 ${className}`}>
      <h3 className="text-lg font-medium text-gray-900">Status History</h3>

      <div className="flow-root">
        <ul className="-mb-8">
          {history.map((item, index) => (
            <li key={item.id}>
              <div className="relative pb-8">
                {/* Connector line */}
                {index !== history.length - 1 && (
                  <span
                    className="absolute left-4 top-4 -ml-px h-full w-0.5 bg-gray-200"
                    aria-hidden="true"
                  />
                )}

                <div className="relative flex space-x-3">
                  {/* Status indicator */}
                  <div>
                    <span
                      className={`h-8 w-8 rounded-full flex items-center justify-center ring-8 ring-white ${getStatusColor(
                        item.to_status
                      )}`}
                    >
                      {item.to_status === QuoteStatus.ACCEPTED && (
                        <svg
                          className="h-5 w-5 text-white"
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
                      {item.to_status === QuoteStatus.DECLINED && (
                        <svg
                          className="h-5 w-5 text-white"
                          fill="currentColor"
                          viewBox="0 0 20 20"
                        >
                          <path
                            fillRule="evenodd"
                            d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                            clipRule="evenodd"
                          />
                        </svg>
                      )}
                      {![QuoteStatus.ACCEPTED, QuoteStatus.DECLINED].includes(
                        item.to_status
                      ) && (
                        <svg
                          className="h-5 w-5 text-white"
                          fill="currentColor"
                          viewBox="0 0 20 20"
                        >
                          <path
                            fillRule="evenodd"
                            d="M10 18a8 8 0 100-16 8 8 0 000 16zm1-11a1 1 0 10-2 0v2H7a1 1 0 100 2h2v2a1 1 0 102 0v-2h2a1 1 0 100-2h-2V7z"
                            clipRule="evenodd"
                          />
                        </svg>
                      )}
                    </span>
                  </div>

                  {/* Content */}
                  <div className="flex min-w-0 flex-1 justify-between space-x-4 pt-1.5">
                    <div>
                      <p className="text-sm font-medium text-gray-900">
                        {getEventLabel(item.event)}
                      </p>
                      <p className="mt-0.5 text-sm text-gray-500">
                        {getChangedByLabel(item.changed_by_type)}
                      </p>
                      {item.reason && (
                        <p className="mt-1 text-sm text-gray-600">
                          Reason: {item.reason}
                        </p>
                      )}
                    </div>
                    <div className="whitespace-nowrap text-right text-sm text-gray-500">
                      {formatDate(item.created_at)}
                    </div>
                  </div>
                </div>
              </div>
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
};

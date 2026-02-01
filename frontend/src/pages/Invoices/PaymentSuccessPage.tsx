import { useEffect, useState } from 'react';
import { useSearchParams } from 'react-router-dom';
import { Card } from '@components/shared';

export function PaymentSuccessPage() {
  const [searchParams] = useSearchParams();
  const [sessionId] = useState(searchParams.get('session_id'));

  useEffect(() => {
    // Session ID is captured for potential analytics/tracking
    // The actual payment verification happens server-side via Stripe webhooks
  }, [sessionId]);

  return (
    <div className="min-h-screen bg-gray-50 flex flex-col items-center justify-center p-4">
      <Card className="max-w-md w-full text-center p-8">
        <div className="mb-6">
          <div className="mx-auto w-16 h-16 bg-green-100 rounded-full flex items-center justify-center">
            <svg
              className="w-8 h-8 text-green-600"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M5 13l4 4L19 7"
              />
            </svg>
          </div>
        </div>

        <h1 className="text-2xl font-bold text-gray-900 mb-2">
          Payment Successful
        </h1>

        <p className="text-gray-600 mb-6">
          Thank you for your payment. A receipt has been sent to your email
          address.
        </p>

        <div className="bg-green-50 border border-green-200 text-green-700 px-4 py-3 rounded mb-6">
          Your invoice has been paid. You can close this window.
        </div>

        <p className="text-sm text-gray-500">
          If you have any questions about your payment, please contact us.
        </p>
      </Card>
    </div>
  );
}

import { useState, useEffect } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { MainLayout } from '@components/layout';
import { Button, Card } from '@components/shared';
import { authApi } from '@services';
import { AxiosError } from 'axios';
import type { ErrorResponse } from '@types';
import { CheckCircle, XCircle, Loader2 } from 'lucide-react';

export function VerifyEmailPage() {
  const [searchParams] = useSearchParams();
  const [status, setStatus] = useState<'loading' | 'success' | 'error'>(
    'loading'
  );
  const [message, setMessage] = useState('');

  useEffect(() => {
    const verifyEmail = async () => {
      const token = searchParams.get('token');

      if (!token) {
        setStatus('error');
        setMessage(
          'Invalid verification link. Please check your email for the correct link.'
        );
        return;
      }

      try {
        await authApi.verifyEmail({ token });
        setStatus('success');
        setMessage('Your email has been verified successfully!');
      } catch (err) {
        const axiosError = err as AxiosError<ErrorResponse>;
        setStatus('error');
        setMessage(
          axiosError.response?.data?.message ||
            'Email verification failed. The link may be expired or invalid.'
        );
      }
    };

    verifyEmail();
  }, [searchParams]);

  return (
    <MainLayout>
      <div className="min-h-[calc(100vh-16rem)] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
        <div className="w-full max-w-md">
          <Card variant="elevated" padding="lg">
            <div className="text-center">
              {status === 'loading' && (
                <>
                  <div className="mx-auto flex items-center justify-center h-12 w-12 mb-4">
                    <Loader2 className="h-8 w-8 text-primary-600 animate-spin" />
                  </div>
                  <h3 className="text-lg font-medium text-neutral-900 mb-2">
                    Verifying your email...
                  </h3>
                </>
              )}

              {status === 'success' && (
                <>
                  <div className="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-success-100 mb-4">
                    <CheckCircle className="h-6 w-6 text-success-600" />
                  </div>
                  <h3 className="text-lg font-medium text-neutral-900 mb-2">
                    Email verified!
                  </h3>
                  <p className="text-sm text-neutral-600 mb-6">{message}</p>
                  <Link to="/login">
                    <Button fullWidth>Go to Login</Button>
                  </Link>
                </>
              )}

              {status === 'error' && (
                <>
                  <div className="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-error-100 mb-4">
                    <XCircle className="h-6 w-6 text-error-600" />
                  </div>
                  <h3 className="text-lg font-medium text-neutral-900 mb-2">
                    Verification failed
                  </h3>
                  <p className="text-sm text-neutral-600 mb-6">{message}</p>
                  <div className="space-y-2">
                    <Link to="/login">
                      <Button fullWidth>Go to Login</Button>
                    </Link>
                    <p className="text-xs text-neutral-500">
                      Need a new verification link? Log in and we'll resend it.
                    </p>
                  </div>
                </>
              )}
            </div>
          </Card>
        </div>
      </div>
    </MainLayout>
  );
}

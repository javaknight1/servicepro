import { useState, useEffect } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { MainLayout } from '@components/layout';
import { Card, CardContent, Button } from '@components/shared';
import { useAuthStore } from '@store';
import { invitationApi } from '@services/tenantService';
import { CheckCircle, AlertCircle, Loader2 } from 'lucide-react';
import type { Invitation } from '@/types/tenant';

export function AcceptInvitationPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const { isAuthenticated } = useAuthStore();

  const [status, setStatus] = useState<
    'loading' | 'accepting' | 'success' | 'error' | 'login-required'
  >('loading');
  const [invitation, setInvitation] = useState<Invitation | null>(null);
  const [error, setError] = useState<string | null>(null);

  const token = searchParams.get('token');

  useEffect(() => {
    if (!token) {
      setStatus('error');
      setError('No invitation token provided');
      return;
    }

    // If not authenticated, redirect to login with return URL
    if (!isAuthenticated) {
      setStatus('login-required');
      return;
    }

    // Fetch invitation details and accept
    const acceptInvitation = async () => {
      try {
        // First, get invitation details
        const inviteResponse = await invitationApi.getByToken(token);
        setInvitation(inviteResponse.data);

        // Then accept the invitation
        setStatus('accepting');
        await invitationApi.accept(token);
        setStatus('success');
      } catch (err: unknown) {
        console.error('Failed to accept invitation:', err);
        const axiosError = err as { response?: { data?: { error?: string } } };
        setError(
          axiosError.response?.data?.error || 'Failed to accept invitation'
        );
        setStatus('error');
      }
    };

    acceptInvitation();
  }, [token, isAuthenticated]);

  const handleLogin = () => {
    // Redirect to login with return URL
    const returnUrl = `/invitations/accept?token=${token}`;
    navigate(`/login?returnUrl=${encodeURIComponent(returnUrl)}`);
  };

  const handleGoToDashboard = () => {
    navigate('/dashboard');
  };

  if (status === 'loading' || status === 'accepting') {
    return (
      <MainLayout>
        <div className="min-h-[calc(100vh-16rem)] flex items-center justify-center py-12 px-4">
          <Card variant="elevated" padding="lg" className="w-full max-w-md">
            <CardContent>
              <div className="text-center py-8">
                <Loader2 className="h-12 w-12 text-primary-600 mx-auto mb-4 animate-spin" />
                <p className="text-neutral-600">
                  {status === 'loading'
                    ? 'Loading invitation...'
                    : 'Accepting invitation...'}
                </p>
              </div>
            </CardContent>
          </Card>
        </div>
      </MainLayout>
    );
  }

  if (status === 'login-required') {
    return (
      <MainLayout>
        <div className="min-h-[calc(100vh-16rem)] flex items-center justify-center py-12 px-4">
          <Card variant="elevated" padding="lg" className="w-full max-w-md">
            <CardContent>
              <div className="text-center py-4">
                <AlertCircle className="h-12 w-12 text-warning-500 mx-auto mb-4" />
                <h2 className="text-xl font-semibold text-neutral-900 mb-2">
                  Login Required
                </h2>
                <p className="text-neutral-600 mb-6">
                  Please log in to accept this invitation.
                </p>
                <Button onClick={handleLogin} fullWidth>
                  Go to Login
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </MainLayout>
    );
  }

  if (status === 'error') {
    return (
      <MainLayout>
        <div className="min-h-[calc(100vh-16rem)] flex items-center justify-center py-12 px-4">
          <Card variant="elevated" padding="lg" className="w-full max-w-md">
            <CardContent>
              <div className="text-center py-4">
                <AlertCircle className="h-12 w-12 text-error-500 mx-auto mb-4" />
                <h2 className="text-xl font-semibold text-neutral-900 mb-2">
                  Unable to Accept Invitation
                </h2>
                <p className="text-neutral-600 mb-6">{error}</p>
                <Button onClick={handleGoToDashboard} fullWidth>
                  Go to Dashboard
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </MainLayout>
    );
  }

  // Success state
  return (
    <MainLayout>
      <div className="min-h-[calc(100vh-16rem)] flex items-center justify-center py-12 px-4">
        <Card variant="elevated" padding="lg" className="w-full max-w-md">
          <CardContent>
            <div className="text-center py-4">
              <CheckCircle className="h-12 w-12 text-success-500 mx-auto mb-4" />
              <h2 className="text-xl font-semibold text-neutral-900 mb-2">
                Invitation Accepted!
              </h2>
              <p className="text-neutral-600 mb-6">
                You've successfully joined{' '}
                <strong>{invitation?.tenant_name || 'the organization'}</strong>
                .
              </p>
              <Button onClick={handleGoToDashboard} fullWidth>
                Go to Dashboard
              </Button>
            </div>
          </CardContent>
        </Card>
      </div>
    </MainLayout>
  );
}

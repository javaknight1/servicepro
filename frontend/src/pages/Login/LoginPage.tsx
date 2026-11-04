import { useEffect, useRef, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod/mini';
import { MainLayout } from '@components/layout';
import {
  Button,
  Input,
  Card,
  CardHeader,
  CardTitle,
  CardContent,
} from '@components/shared';
import { useAuthStore } from '@store';
import {
  getApiErrorInfo,
  getErrorMessage,
  TOO_MANY_ATTEMPTS_MESSAGE,
} from '@/utils/error';
import { getSafeReturnUrl } from '@/utils/urlSanitizer';
import { isNeonAuthEnabled } from '@config/auth';
import {
  getNeonAuthErrorMessage,
  needsEmailVerification,
  OAUTH_CALLBACK_PARAM,
  verifyEmailPath,
} from '@/utils/authMessages';
import { GoogleSignInButton } from './GoogleSignInButton';

const INVALID_CREDENTIALS_MESSAGE = 'Invalid email or password.';
const SIGN_IN_FAILED_MESSAGE = 'Failed to login. Please check your credentials.';

/**
 * Map a failed login to a user-facing message. The API returns the same 401
 * for an unknown email and a wrong password, and a generic 429 once attempts
 * are throttled (there is no "account locked" state any more).
 */
function getLoginErrorMessage(err: unknown): string {
  const { status, message } = getApiErrorInfo(err);
  if (status === 429) return TOO_MANY_ATTEMPTS_MESSAGE;
  if (status === 401) return message || INVALID_CREDENTIALS_MESSAGE;
  return (
    message ||
    getErrorMessage(err, 'Failed to login. Please check your credentials.')
  );
}

const loginSchema = z.object({
  email: z.email('Invalid email address'),
  password: z
    .string()
    .check(z.minLength(8, 'Password must be at least 8 characters')),
});

type LoginFormData = z.infer<typeof loginSchema>;

export function LoginPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const returnUrl = getSafeReturnUrl(searchParams.get('returnUrl'));
  const login = useAuthStore((state) => state.login);
  const completeSignIn = useAuthStore((state) => state.completeSignIn);
  const [error, setError] = useState<string>('');
  const oauthState = isNeonAuthEnabled
    ? searchParams.get(OAUTH_CALLBACK_PARAM)
    : null;
  const [finishingOAuth, setFinishingOAuth] = useState(oauthState === '1');
  const oauthHandled = useRef(false);

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
  });

  // Neon Auth: back from Google — link/load the account, then continue.
  useEffect(() => {
    if (!oauthState || oauthHandled.current) return;
    oauthHandled.current = true;
    if (oauthState !== '1') {
      setError('Google sign-in was cancelled or failed. Please try again.');
      return;
    }
    completeSignIn()
      .then(() => navigate(returnUrl ?? '/dashboard'))
      .catch((err: unknown) => {
        setError(getNeonAuthErrorMessage(err, SIGN_IN_FAILED_MESSAGE));
        setFinishingOAuth(false);
      });
  }, [oauthState, completeSignIn, navigate, returnUrl]);

  const onSubmit = async (data: LoginFormData) => {
    try {
      setError('');
      await login(data.email, data.password);
      navigate(returnUrl ?? '/dashboard');
    } catch (err: unknown) {
      if (!isNeonAuthEnabled) {
        setError(getLoginErrorMessage(err));
        return;
      }
      if (needsEmailVerification(err)) {
        // Signed in to Neon Auth, but the API links only verified emails:
        // send a code and ask for it.
        try {
          const { sendVerificationCode } = await import('@services/neonAuth');
          await sendVerificationCode(data.email);
          navigate(verifyEmailPath(data.email, returnUrl));
        } catch (sendErr: unknown) {
          setError(
            getNeonAuthErrorMessage(
              sendErr,
              'Failed to send the verification code.'
            )
          );
        }
        return;
      }
      setError(getNeonAuthErrorMessage(err, SIGN_IN_FAILED_MESSAGE));
    }
  };

  if (finishingOAuth) {
    return (
      <MainLayout>
        <div className="min-h-[calc(100vh-16rem)] flex items-center justify-center py-12 px-4">
          <p className="text-neutral-600" role="status">
            Signing you in…
          </p>
        </div>
      </MainLayout>
    );
  }

  return (
    <MainLayout>
      <div className="min-h-[calc(100vh-16rem)] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
        <div className="w-full max-w-md">
          <Card variant="elevated" padding="lg">
            <CardHeader>
              <CardTitle className="text-center">
                Sign in to your account
              </CardTitle>
              <p className="text-center text-sm text-neutral-600">
                Or{' '}
                <Link
                  to="/register"
                  className="text-primary-600 hover:text-primary-700 font-medium"
                >
                  create a new account
                </Link>
              </p>
            </CardHeader>

            <CardContent>
              <form
                onSubmit={handleSubmit(onSubmit)}
                className="space-y-4 mt-6"
              >
                {error && (
                  <div className="bg-error-50 border border-error-200 text-error-700 px-4 py-3 rounded-lg text-sm">
                    {error}
                  </div>
                )}

                <Input
                  label="Email address"
                  type="email"
                  autoComplete="email"
                  fullWidth
                  error={errors.email?.message}
                  {...register('email')}
                />

                <Input
                  label="Password"
                  type="password"
                  autoComplete="current-password"
                  fullWidth
                  error={errors.password?.message}
                  {...register('password')}
                />

                <div className="flex items-center justify-between">
                  <div className="text-sm">
                    <Link
                      to="/forgot-password"
                      className="text-primary-600 hover:text-primary-700 font-medium"
                    >
                      Forgot your password?
                    </Link>
                  </div>
                </div>

                <Button type="submit" fullWidth isLoading={isSubmitting}>
                  Sign in
                </Button>
              </form>

              <GoogleSignInButton returnUrl={returnUrl} onError={setError} />

              {isNeonAuthEnabled && (
                <p className="mt-6 text-center text-xs text-neutral-500">
                  First time signing in since ServicePro upgraded its sign-in?{' '}
                  <Link
                    to="/register"
                    className="text-primary-600 hover:text-primary-700 font-medium"
                  >
                    Set up your sign-in
                  </Link>{' '}
                  with the email you already use.
                </p>
              )}
            </CardContent>
          </Card>
        </div>
      </div>
    </MainLayout>
  );
}

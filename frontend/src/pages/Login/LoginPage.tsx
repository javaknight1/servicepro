import { useState } from 'react';
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

const INVALID_CREDENTIALS_MESSAGE = 'Invalid email or password.';

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
  const [error, setError] = useState<string>('');

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<LoginFormData>({
    resolver: zodResolver(loginSchema),
  });

  const onSubmit = async (data: LoginFormData) => {
    try {
      setError('');
      await login(data.email, data.password);
      navigate(returnUrl ?? '/dashboard');
    } catch (err: unknown) {
      setError(getLoginErrorMessage(err));
    }
  };

  return (
    <MainLayout>
      <div className="min-h-[calc(100vh-16rem)] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
        <div className="w-full max-w-md">
          <Card variant="elevated" padding="lg">
            <CardHeader>
              <CardTitle className="text-center">
                Sign in to your account
              </CardTitle>
              <p className="text-center text-sm text-neutral-600 mt-2">
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
            </CardContent>
          </Card>
        </div>
      </div>
    </MainLayout>
  );
}

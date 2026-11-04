import { useState } from 'react';
import { Link, useNavigate } from 'react-router';
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
import { requestPasswordResetCode } from '@services/neonAuth';
import { getNeonAuthErrorMessage } from '@/utils/authMessages';
import { ArrowLeft } from 'lucide-react';

const forgotPasswordSchema = z.object({
  email: z.email('Invalid email address'),
});

type ForgotPasswordFormData = z.infer<typeof forgotPasswordSchema>;

/** Neon Auth password reset, step 1 (T158): email a reset code. */
export function NeonForgotPasswordPage() {
  const navigate = useNavigate();
  const [error, setError] = useState('');

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ForgotPasswordFormData>({
    resolver: zodResolver(forgotPasswordSchema),
  });

  const onSubmit = async (data: ForgotPasswordFormData) => {
    try {
      setError('');
      await requestPasswordResetCode(data.email);
      navigate(
        `/reset-password?${new URLSearchParams({ email: data.email }).toString()}`
      );
    } catch (err: unknown) {
      setError(
        getNeonAuthErrorMessage(
          err,
          'Failed to send the reset code. Please try again.'
        )
      );
    }
  };

  return (
    <MainLayout>
      <div className="min-h-[calc(100vh-16rem)] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
        <div className="w-full max-w-md">
          <Card variant="elevated" padding="lg">
            <CardHeader>
              <Link
                to="/login"
                className="inline-flex items-center text-sm text-neutral-600 hover:text-neutral-900 mb-5.5"
              >
                <ArrowLeft className="h-4 w-4 mr-1" />
                Back to login
              </Link>
              <CardTitle className="text-center">Reset your password</CardTitle>
              <p className="text-center text-sm text-neutral-600">
                Enter your email address and we'll send you a code to reset
                your password.
              </p>
            </CardHeader>

            <CardContent>
              <form
                onSubmit={handleSubmit(onSubmit)}
                className="space-y-4 mt-6"
              >
                {error && (
                  <div
                    className="bg-error-50 border border-error-200 text-error-700 px-4 py-3 rounded-lg text-sm"
                    role="alert"
                  >
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

                <Button type="submit" fullWidth isLoading={isSubmitting}>
                  Send reset code
                </Button>
              </form>
              <p className="mt-6 text-center text-xs text-neutral-500">
                First time signing in since ServicePro upgraded its sign-in?
                There's no password to reset yet:{' '}
                <Link
                  to="/register"
                  className="text-primary-600 hover:text-primary-700 font-medium"
                >
                  set up your sign-in
                </Link>{' '}
                with the email you already use.
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </MainLayout>
  );
}

import { useState } from 'react';
import { Link, useSearchParams } from 'react-router';
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
import { resetPasswordWithCode } from '@services/neonAuth';
import { getAuthErrorInfo } from '@services/authErrors';
import { getNeonAuthErrorMessage } from '@/utils/authMessages';
import { CheckCircle } from 'lucide-react';

const WRONG_CODE_MESSAGE =
  'That code is wrong or has expired. Request a new one and try again.';

const resetSchema = z
  .object({
    email: z.email('Invalid email address'),
    code: z
      .string()
      .check(
        z.trim(),
        z.minLength(4, 'Enter the code from the email'),
        z.maxLength(12, 'Enter the code from the email')
      ),
    password: z
      .string()
      .check(z.minLength(8, 'Password must be at least 8 characters')),
    confirmPassword: z.string(),
  })
  .check(
    z.refine((data) => data.password === data.confirmPassword, {
      error: "Passwords don't match",
      path: ['confirmPassword'],
    })
  );

type ResetFormData = z.infer<typeof resetSchema>;

/** Neon Auth password reset, step 2 (T158): code + new password. */
export function NeonResetPasswordPage() {
  const [searchParams] = useSearchParams();
  const [error, setError] = useState('');
  const [success, setSuccess] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<ResetFormData>({
    resolver: zodResolver(resetSchema),
    defaultValues: { email: searchParams.get('email') ?? '' },
  });

  const onSubmit = async (data: ResetFormData) => {
    try {
      setError('');
      await resetPasswordWithCode(data.email, data.code, data.password);
      setSuccess(true);
    } catch (err: unknown) {
      const { status } = getAuthErrorInfo(err);
      setError(
        status === 400 || status === 403
          ? WRONG_CODE_MESSAGE
          : getNeonAuthErrorMessage(err, 'Failed to reset your password.')
      );
    }
  };

  if (success) {
    return (
      <MainLayout>
        <div className="min-h-[calc(100vh-16rem)] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
          <div className="w-full max-w-md">
            <Card variant="elevated" padding="lg">
              <div className="text-center">
                <div className="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-success-100 mb-4">
                  <CheckCircle className="h-6 w-6 text-success-600" />
                </div>
                <h3 className="text-lg font-medium text-neutral-900 mb-2">
                  Password reset successful
                </h3>
                <p className="text-sm text-neutral-600 mb-6">
                  Sign in with your new password.
                </p>
                <Link to="/login">
                  <Button fullWidth>Go to Login</Button>
                </Link>
              </div>
            </Card>
          </div>
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
              <CardTitle className="text-center">Choose a new password</CardTitle>
              <p className="text-center text-sm text-neutral-600">
                Enter the code we emailed you and your new password.
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

                <Input
                  label="Reset code"
                  type="text"
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  spellCheck={false}
                  fullWidth
                  error={errors.code?.message}
                  {...register('code')}
                />

                <Input
                  label="New password"
                  type="password"
                  autoComplete="new-password"
                  fullWidth
                  helperText="Must be at least 8 characters"
                  error={errors.password?.message}
                  {...register('password')}
                />

                <Input
                  label="Confirm new password"
                  type="password"
                  autoComplete="new-password"
                  fullWidth
                  error={errors.confirmPassword?.message}
                  {...register('confirmPassword')}
                />

                <Button type="submit" fullWidth isLoading={isSubmitting}>
                  Reset password
                </Button>
              </form>
              <p className="mt-6 text-center text-sm text-neutral-600">
                <Link
                  to="/forgot-password"
                  className="text-primary-600 hover:text-primary-700 font-medium"
                >
                  Send a new code
                </Link>
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </MainLayout>
  );
}

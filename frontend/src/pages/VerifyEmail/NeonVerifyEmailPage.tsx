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
import { sendVerificationCode, verifyEmailCode } from '@services/neonAuth';
import { getAuthErrorInfo } from '@services/authErrors';
import { getNeonAuthErrorMessage } from '@/utils/authMessages';
import { getSafeReturnUrl } from '@/utils/urlSanitizer';

const WRONG_CODE_MESSAGE =
  'That code is wrong or has expired. Check the email, or send a new code.';

const verifySchema = z.object({
  email: z.email('Invalid email address'),
  code: z
    .string()
    .check(
      z.trim(),
      z.minLength(4, 'Enter the code from the email'),
      z.maxLength(12, 'Enter the code from the email')
    ),
});

type VerifyFormData = z.infer<typeof verifySchema>;

/**
 * Neon Auth email verification (T158): the user types the code Neon Auth
 * emailed. Verifying signs them in; the ServicePro account is then linked or
 * created (with the invite saved at sign-up) and the app opens.
 */
export function NeonVerifyEmailPage() {
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const returnUrl = getSafeReturnUrl(searchParams.get('returnUrl'));
  const completeSignIn = useAuthStore((state) => state.completeSignIn);
  const [error, setError] = useState('');
  const [notice, setNotice] = useState('');
  const [resending, setResending] = useState(false);

  const {
    register,
    handleSubmit,
    getValues,
    trigger,
    formState: { errors, isSubmitting },
  } = useForm<VerifyFormData>({
    resolver: zodResolver(verifySchema),
    defaultValues: { email: searchParams.get('email') ?? '', code: '' },
  });

  const onSubmit = async (data: VerifyFormData) => {
    setError('');
    setNotice('');
    try {
      await verifyEmailCode(data.email, data.code);
    } catch (err: unknown) {
      const { status } = getAuthErrorInfo(err);
      setError(
        status === 400 || status === 403
          ? WRONG_CODE_MESSAGE
          : getNeonAuthErrorMessage(err, WRONG_CODE_MESSAGE)
      );
      return;
    }
    try {
      await completeSignIn();
      navigate(returnUrl ?? '/dashboard');
    } catch (err: unknown) {
      setError(
        getNeonAuthErrorMessage(
          err,
          'Your email is verified, but signing in failed. Please sign in.'
        )
      );
    }
  };

  const resend = async () => {
    if (!(await trigger('email'))) return;
    setError('');
    setNotice('');
    setResending(true);
    try {
      await sendVerificationCode(getValues('email'));
      setNotice('We sent a new code. It may take a minute to arrive.');
    } catch (err: unknown) {
      setError(getNeonAuthErrorMessage(err, 'Failed to send a new code.'));
    } finally {
      setResending(false);
    }
  };

  return (
    <MainLayout>
      <div className="min-h-[calc(100vh-16rem)] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
        <div className="w-full max-w-md">
          <Card variant="elevated" padding="lg">
            <CardHeader>
              <CardTitle className="text-center">Check your email</CardTitle>
              <p className="text-center text-sm text-neutral-600">
                We sent a verification code to your email address. Enter it
                below to finish setting up your sign-in.
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
                {notice && (
                  <div
                    className="bg-success-50 border border-success-200 text-success-700 px-4 py-3 rounded-lg text-sm"
                    role="status"
                  >
                    {notice}
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
                  label="Verification code"
                  type="text"
                  inputMode="numeric"
                  autoComplete="one-time-code"
                  spellCheck={false}
                  fullWidth
                  error={errors.code?.message}
                  {...register('code')}
                />

                <Button type="submit" fullWidth isLoading={isSubmitting}>
                  Verify and continue
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  fullWidth
                  isLoading={resending}
                  onClick={resend}
                >
                  Send a new code
                </Button>
              </form>
              <p className="mt-6 text-center text-sm text-neutral-600">
                <Link
                  to="/login"
                  className="text-primary-600 hover:text-primary-700 font-medium"
                >
                  Back to sign in
                </Link>
              </p>
            </CardContent>
          </Card>
        </div>
      </div>
    </MainLayout>
  );
}

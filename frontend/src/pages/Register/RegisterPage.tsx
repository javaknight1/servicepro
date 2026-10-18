import { useState, useEffect } from 'react';
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
import { useAuthStore } from '@store';
import { invitationApi } from '@services/tenantService';
import {
  getApiErrorInfo,
  getErrorMessage,
  TOO_MANY_ATTEMPTS_MESSAGE,
} from '@/utils/error';
import type { Invitation } from '@/types/tenant';
import { CheckCircle, Building2, AlertCircle } from 'lucide-react';

/** Query params an invite-code link may use (`/register?invite=CODE`). */
const INVITE_CODE_PARAMS = ['invite', 'code', 'invite_code'] as const;

const INVITE_ONLY_MESSAGE =
  'ServicePro is invite-only right now. Enter your invite code to create an account.';
const INVALID_INVITE_CODE_MESSAGE =
  "That invite code isn't valid. It may have expired, been used up or been revoked. Check the code or ask the person who invited you for a new one.";
const INVALID_INVITATION_MESSAGE =
  'This invitation is no longer valid for this email address. Ask the person who invited you to send a new invitation.';

/** Messages for the API's invite-only rejection codes (403). */
const INVITE_ONLY_ERROR_MESSAGES: Record<string, string> = {
  registration_closed: INVITE_ONLY_MESSAGE,
  invalid_invite_code: INVALID_INVITE_CODE_MESSAGE,
  invalid_invitation: INVALID_INVITATION_MESSAGE,
};

/** Map a failed registration to a user-facing message. */
function getRegisterErrorMessage(err: unknown): string {
  const { status, code, message } = getApiErrorInfo(err);
  if (status === 403 && code && INVITE_ONLY_ERROR_MESSAGES[code]) {
    return INVITE_ONLY_ERROR_MESSAGES[code];
  }
  if (status === 429) return TOO_MANY_ATTEMPTS_MESSAGE;
  return (
    message ||
    getErrorMessage(err, 'Failed to create account. Please try again.')
  );
}

const registerSchema = z
  .object({
    email: z.email('Invalid email address'),
    password: z
      .string()
      .check(z.minLength(8, 'Password must be at least 8 characters')),
    confirmPassword: z.string(),
    inviteCode: z.optional(
      z.string().check(z.trim(), z.maxLength(128, 'Invite code is too long'))
    ),
  })
  .check(
    z.refine((data) => data.password === data.confirmPassword, {
      error: "Passwords don't match",
      path: ['confirmPassword'],
    })
  );

type RegisterFormData = z.infer<typeof registerSchema>;

export function RegisterPage() {
  const [searchParams] = useSearchParams();
  const register_action = useAuthStore((state) => state.register);
  const [error, setError] = useState<string>('');
  const [success, setSuccess] = useState(false);

  // Invitation state
  const [invitation, setInvitation] = useState<Invitation | null>(null);
  const [invitationLoading, setInvitationLoading] = useState(false);
  const [invitationError, setInvitationError] = useState<string | null>(null);

  const invitationToken = searchParams.get('invitation');
  const inviteCodeFromUrl =
    INVITE_CODE_PARAMS.map((key) => searchParams.get(key)).find(
      (value): value is string => !!value
    ) ?? '';

  const {
    register,
    handleSubmit,
    setValue,
    formState: { errors, isSubmitting },
  } = useForm<RegisterFormData>({
    resolver: zodResolver(registerSchema),
    defaultValues: { inviteCode: inviteCodeFromUrl },
  });

  // Fetch invitation details if token is present
  useEffect(() => {
    if (invitationToken) {
      setInvitationLoading(true);
      invitationApi
        .getByToken(invitationToken)
        .then((response) => {
          setInvitation(response.data);
          setValue('email', response.data.email);
        })
        .catch((err: unknown) => {
          console.error('Failed to fetch invitation:', err);
          // Invitation errors carry a human `message` (code in `error`).
          setInvitationError(
            getApiErrorInfo(err).message || 'Invalid or expired invitation'
          );
        })
        .finally(() => {
          setInvitationLoading(false);
        });
    }
  }, [invitationToken, setValue]);

  const onSubmit = async (data: RegisterFormData) => {
    try {
      setError('');
      await register_action(data.email, data.password, {
        invitationToken: invitationToken || undefined,
        inviteCode: data.inviteCode || undefined,
      });
      setSuccess(true);
    } catch (err: unknown) {
      setError(getRegisterErrorMessage(err));
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
                  Check your email
                </h3>
                <p className="text-sm text-neutral-600 mb-6">
                  We've sent you an email with a verification link. Please check
                  your inbox and click the link to verify your account.
                  {invitation && (
                    <>
                      <br />
                      <br />
                      Once verified, you'll be added to{' '}
                      <strong>{invitation.tenant_name}</strong>.
                    </>
                  )}
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

  // Show error if invitation is invalid
  if (invitationToken && invitationError) {
    return (
      <MainLayout>
        <div className="min-h-[calc(100vh-16rem)] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
          <div className="w-full max-w-md">
            <Card variant="elevated" padding="lg">
              <div className="text-center">
                <div className="mx-auto flex items-center justify-center h-12 w-12 rounded-full bg-error-100 mb-4">
                  <AlertCircle className="h-6 w-6 text-error-600" />
                </div>
                <h3 className="text-lg font-medium text-neutral-900 mb-2">
                  Invalid Invitation
                </h3>
                <p className="text-sm text-neutral-600 mb-6">
                  {invitationError}
                </p>
                <div className="space-y-3">
                  <Link to="/register">
                    <Button fullWidth variant="outline">
                      Register without invitation
                    </Button>
                  </Link>
                  <Link to="/login">
                    <Button fullWidth>Go to Login</Button>
                  </Link>
                </div>
              </div>
            </Card>
          </div>
        </div>
      </MainLayout>
    );
  }

  // Show loading while fetching invitation
  if (invitationToken && invitationLoading) {
    return (
      <MainLayout>
        <div className="min-h-[calc(100vh-16rem)] flex items-center justify-center py-12 px-4 sm:px-6 lg:px-8">
          <div className="w-full max-w-md">
            <Card variant="elevated" padding="lg">
              <div className="text-center py-8">
                <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-primary-600 mx-auto mb-4"></div>
                <p className="text-neutral-600">Loading invitation...</p>
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
              <CardTitle className="text-center">Create your account</CardTitle>
              <p className="text-center text-sm text-neutral-600 mt-2">
                Already have an account?{' '}
                <Link
                  to="/login"
                  className="text-primary-600 hover:text-primary-700 font-medium"
                >
                  Sign in
                </Link>
              </p>
            </CardHeader>

            <CardContent>
              {/* Invitation banner */}
              {invitation && (
                <div className="mb-6 p-4 bg-primary-50 border border-primary-200 rounded-lg">
                  <div className="flex items-start">
                    <Building2 className="h-5 w-5 text-primary-600 mt-0.5 mr-3 shrink-0" />
                    <div>
                      <p className="text-sm font-medium text-primary-900">
                        You've been invited to join
                      </p>
                      <p className="text-lg font-semibold text-primary-700 mt-1">
                        {invitation.tenant_name}
                      </p>
                      <p className="text-xs text-primary-600 mt-1">
                        as {invitation.role_name}
                      </p>
                    </div>
                  </div>
                </div>
              )}

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
                  disabled={!!invitation}
                  className={invitation ? 'bg-neutral-100' : ''}
                  {...register('email')}
                />

                {invitation && (
                  <p className="text-xs text-neutral-500 -mt-2">
                    This email is linked to your invitation and cannot be
                    changed.
                  </p>
                )}

                {!invitation && (
                  <Input
                    label="Invite code"
                    type="text"
                    autoComplete="off"
                    spellCheck={false}
                    fullWidth
                    helperText="ServicePro is invite-only right now. Enter the code from your invite."
                    error={errors.inviteCode?.message}
                    {...register('inviteCode')}
                  />
                )}

                <Input
                  label="Password"
                  type="password"
                  autoComplete="new-password"
                  fullWidth
                  helperText="Must be at least 8 characters"
                  error={errors.password?.message}
                  {...register('password')}
                />

                <Input
                  label="Confirm Password"
                  type="password"
                  autoComplete="new-password"
                  fullWidth
                  error={errors.confirmPassword?.message}
                  {...register('confirmPassword')}
                />

                <Button type="submit" fullWidth isLoading={isSubmitting}>
                  {invitation ? 'Create account & join' : 'Create account'}
                </Button>

                <p className="text-xs text-neutral-500 text-center mt-4">
                  By creating an account, you agree to our Terms of Service and
                  Privacy Policy.
                </p>
              </form>
            </CardContent>
          </Card>
        </div>
      </div>
    </MainLayout>
  );
}

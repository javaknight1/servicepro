import { useState } from 'react';
import { Button } from '@components/shared';
import { isGoogleSignInEnabled } from '@config/auth';
import {
  getNeonAuthErrorMessage,
  OAUTH_CALLBACK_PARAM,
} from '@/utils/authMessages';

interface GoogleSignInButtonProps {
  /** Resolve false to cancel (e.g. the terms box is not ticked). */
  beforeStart?: () => boolean | Promise<boolean>;
  /** Where to go after signing in (passed through the callback URL). */
  returnUrl?: string | null;
  onError: (message: string) => void;
}

/**
 * "Continue with Google" through Neon Auth (T158). Renders nothing unless
 * VITE_NEON_AUTH_GOOGLE=true on a Neon Auth build.
 */
export function GoogleSignInButton({
  beforeStart,
  returnUrl,
  onError,
}: GoogleSignInButtonProps) {
  const [starting, setStarting] = useState(false);

  if (!isGoogleSignInEnabled) return null;

  const start = async () => {
    if (beforeStart && !(await beforeStart())) return;
    setStarting(true);
    try {
      const origin = window.location.origin;
      const callback = new URL('/login', origin);
      callback.searchParams.set(OAUTH_CALLBACK_PARAM, '1');
      if (returnUrl) callback.searchParams.set('returnUrl', returnUrl);
      const errorCallback = new URL('/login', origin);
      errorCallback.searchParams.set(OAUTH_CALLBACK_PARAM, 'error');
      const { signInWithGoogle } = await import('@services/neonAuth');
      // Navigates away to Google on success.
      await signInWithGoogle(callback.toString(), errorCallback.toString());
    } catch (err: unknown) {
      setStarting(false);
      onError(getNeonAuthErrorMessage(err, 'Google sign-in failed.'));
    }
  };

  return (
    <>
      <div className="relative my-4">
        <div className="absolute inset-0 flex items-center" aria-hidden="true">
          <div className="w-full border-t border-neutral-200" />
        </div>
        <div className="relative flex justify-center text-sm">
          <span className="bg-white px-2 text-neutral-500">or</span>
        </div>
      </div>
      <Button
        type="button"
        variant="outline"
        fullWidth
        isLoading={starting}
        onClick={start}
      >
        Continue with Google
      </Button>
    </>
  );
}

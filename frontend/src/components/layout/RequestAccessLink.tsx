import { Button, type ButtonProps } from '@components/shared';
import { BETA_ACCESS_MAILTO } from '@/config/support';

interface RequestAccessLinkProps {
  variant?: ButtonProps['variant'];
  size?: ButtonProps['size'];
  label?: string;
}

/**
 * Public call to action while registration is invite-only (T255): opens an
 * email to support instead of a sign-up form visitors can't complete.
 */
export function RequestAccessLink({
  variant = 'primary',
  size,
  label = 'Request beta access',
}: RequestAccessLinkProps) {
  return (
    <a href={BETA_ACCESS_MAILTO}>
      <Button variant={variant} size={size}>
        {label}
      </Button>
    </a>
  );
}

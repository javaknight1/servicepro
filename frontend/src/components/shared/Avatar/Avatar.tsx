import { getAvatarColor, getInitials, getDisplayName } from '@/utils/avatar';
import clsx from 'clsx';

export type AvatarSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl';

export interface AvatarProps {
  email: string;
  firstName?: string | null;
  lastName?: string | null;
  profilePictureUrl?: string | null;
  size?: AvatarSize;
  className?: string;
}

const sizeClasses: Record<AvatarSize, string> = {
  xs: 'h-6 w-6 text-xs',
  sm: 'h-8 w-8 text-sm',
  md: 'h-10 w-10 text-sm',
  lg: 'h-12 w-12 text-base',
  xl: 'h-16 w-16 text-lg',
};

export function Avatar({
  email,
  firstName,
  lastName,
  profilePictureUrl,
  size = 'md',
  className,
}: AvatarProps) {
  const colors = getAvatarColor(email);
  const initials = getInitials(email, firstName, lastName);
  const displayName = getDisplayName(email, firstName, lastName);
  const sizeClass = sizeClasses[size];

  if (profilePictureUrl) {
    return (
      <img
        src={profilePictureUrl}
        alt={`${displayName}'s avatar`}
        className={clsx('rounded-full object-cover', sizeClass, className)}
      />
    );
  }

  return (
    <div
      className={clsx(
        'rounded-full flex items-center justify-center font-medium',
        sizeClass,
        colors.bg,
        colors.text,
        className
      )}
      title={displayName}
    >
      {initials}
    </div>
  );
}

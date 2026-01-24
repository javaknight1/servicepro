import { getAvatarColor, getInitials } from '@/utils/avatar';
import clsx from 'clsx';

export type AvatarSize = 'xs' | 'sm' | 'md' | 'lg' | 'xl';

export interface AvatarProps {
  email: string;
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
  profilePictureUrl,
  size = 'md',
  className,
}: AvatarProps) {
  const colors = getAvatarColor(email);
  const initials = getInitials(email);
  const sizeClass = sizeClasses[size];

  if (profilePictureUrl) {
    return (
      <img
        src={profilePictureUrl}
        alt={`${email}'s avatar`}
        className={clsx(
          'rounded-full object-cover',
          sizeClass,
          className
        )}
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
      title={email}
    >
      {initials}
    </div>
  );
}

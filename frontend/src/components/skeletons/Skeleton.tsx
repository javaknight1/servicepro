/**
 * =============================================================================
 * Base Skeleton Component
 * =============================================================================
 * Foundation skeleton component with animation and variants
 */

import { CSSProperties, ReactNode } from 'react';
import { cn } from '@utils/cn';

// =============================================================================
// Types
// =============================================================================

export interface SkeletonProps {
  className?: string;
  width?: string | number;
  height?: string | number;
  variant?: 'rectangular' | 'circular' | 'rounded' | 'text';
  animation?: 'pulse' | 'wave' | 'none';
  children?: ReactNode;
  style?: CSSProperties;
}

// =============================================================================
// Component
// =============================================================================

export function Skeleton({
  className,
  width,
  height,
  variant = 'rectangular',
  animation = 'pulse',
  children,
  style,
}: SkeletonProps) {
  const variantClasses = {
    rectangular: 'rounded-none',
    circular: 'rounded-full',
    rounded: 'rounded-lg',
    text: 'rounded h-4',
  };

  const animationClasses = {
    pulse: 'animate-pulse',
    wave: 'skeleton-wave',
    none: '',
  };

  const computedStyle: CSSProperties = {
    width: typeof width === 'number' ? `${width}px` : width,
    height: typeof height === 'number' ? `${height}px` : height,
    ...style,
  };

  return (
    <div
      className={cn(
        'bg-gray-200',
        variantClasses[variant],
        animationClasses[animation],
        className
      )}
      style={computedStyle}
      aria-hidden="true"
    >
      {children && <span className="invisible">{children}</span>}
    </div>
  );
}

// =============================================================================
// Skeleton Group
// =============================================================================

interface SkeletonGroupProps {
  count?: number;
  gap?: string | number;
  direction?: 'row' | 'column';
  children?: (index: number) => ReactNode;
  className?: string;
}

export function SkeletonGroup({
  count = 3,
  gap = 8,
  direction = 'column',
  children,
  className,
}: SkeletonGroupProps) {
  const gapValue = typeof gap === 'number' ? `${gap}px` : gap;

  return (
    <div
      className={cn(
        'flex',
        direction === 'column' ? 'flex-col' : 'flex-row',
        className
      )}
      style={{ gap: gapValue }}
    >
      {Array.from({ length: count }).map((_, index) =>
        children ? children(index) : <Skeleton key={index} height={16} />
      )}
    </div>
  );
}

// =============================================================================
// Text Skeleton
// =============================================================================

interface TextSkeletonProps {
  lines?: number;
  lastLineWidth?: string;
  lineHeight?: number;
  gap?: number;
  className?: string;
}

export function TextSkeleton({
  lines = 3,
  lastLineWidth = '60%',
  lineHeight = 16,
  gap = 8,
  className,
}: TextSkeletonProps) {
  return (
    <div className={cn('space-y-2', className)} style={{ gap: `${gap}px` }}>
      {Array.from({ length: lines }).map((_, index) => (
        <Skeleton
          key={index}
          variant="text"
          height={lineHeight}
          width={index === lines - 1 ? lastLineWidth : '100%'}
        />
      ))}
    </div>
  );
}

// =============================================================================
// Avatar Skeleton
// =============================================================================

interface AvatarSkeletonProps {
  size?: 'sm' | 'md' | 'lg' | 'xl' | number;
  className?: string;
}

const avatarSizes = {
  sm: 32,
  md: 40,
  lg: 48,
  xl: 64,
};

export function AvatarSkeleton({
  size = 'md',
  className,
}: AvatarSkeletonProps) {
  const sizeValue = typeof size === 'number' ? size : avatarSizes[size];

  return (
    <Skeleton
      variant="circular"
      width={sizeValue}
      height={sizeValue}
      className={className}
    />
  );
}

// =============================================================================
// Button Skeleton
// =============================================================================

interface ButtonSkeletonProps {
  size?: 'sm' | 'md' | 'lg';
  width?: string | number;
  className?: string;
}

const buttonSizes = {
  sm: { height: 32, width: 64 },
  md: { height: 40, width: 80 },
  lg: { height: 48, width: 96 },
};

export function ButtonSkeleton({
  size = 'md',
  width,
  className,
}: ButtonSkeletonProps) {
  const sizeConfig = buttonSizes[size];

  return (
    <Skeleton
      variant="rounded"
      height={sizeConfig.height}
      width={width || sizeConfig.width}
      className={className}
    />
  );
}

// =============================================================================
// Input Skeleton
// =============================================================================

interface InputSkeletonProps {
  hasLabel?: boolean;
  className?: string;
}

export function InputSkeleton({
  hasLabel = true,
  className,
}: InputSkeletonProps) {
  return (
    <div className={cn('space-y-2', className)}>
      {hasLabel && <Skeleton variant="text" width={80} height={14} />}
      <Skeleton variant="rounded" height={40} width="100%" />
    </div>
  );
}

// =============================================================================
// Image Skeleton
// =============================================================================

interface ImageSkeletonProps {
  aspectRatio?: string;
  width?: string | number;
  height?: string | number;
  className?: string;
}

export function ImageSkeleton({
  aspectRatio,
  width = '100%',
  height,
  className,
}: ImageSkeletonProps) {
  const style: CSSProperties = {
    width: typeof width === 'number' ? `${width}px` : width,
    aspectRatio,
  };

  if (height && !aspectRatio) {
    style.height = typeof height === 'number' ? `${height}px` : height;
  }

  return (
    <Skeleton
      variant="rounded"
      className={cn('flex items-center justify-center', className)}
      style={style}
    >
      <svg
        className="w-10 h-10 text-gray-300"
        fill="currentColor"
        viewBox="0 0 24 24"
      >
        <path d="M4 16l4.586-4.586a2 2 0 012.828 0L16 16m-2-2l1.586-1.586a2 2 0 012.828 0L20 14m-6-6h.01M6 20h12a2 2 0 002-2V6a2 2 0 00-2-2H6a2 2 0 00-2 2v12a2 2 0 002 2z" />
      </svg>
    </Skeleton>
  );
}

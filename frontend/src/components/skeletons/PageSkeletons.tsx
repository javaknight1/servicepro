/**
 * =============================================================================
 * Page Skeleton Components
 * =============================================================================
 * Full-page skeleton components for different page layouts
 */

import { cn } from '@utils/cn';
import {
  Skeleton,
  SkeletonGroup,
  TextSkeleton,
  AvatarSkeleton,
  ButtonSkeleton,
  InputSkeleton,
  ImageSkeleton,
} from './Skeleton';

// =============================================================================
// Card Skeleton
// =============================================================================

interface CardSkeletonProps {
  hasImage?: boolean;
  hasActions?: boolean;
  className?: string;
}

export function CardSkeleton({
  hasImage = false,
  hasActions = false,
  className,
}: CardSkeletonProps) {
  return (
    <div className={cn('bg-white rounded-lg shadow p-6', className)}>
      {hasImage && <ImageSkeleton aspectRatio="16/9" className="mb-4" />}
      <Skeleton height={24} width="60%" className="mb-4" />
      <TextSkeleton lines={3} />
      {hasActions && (
        <div className="flex gap-2 mt-4">
          <ButtonSkeleton size="sm" />
          <ButtonSkeleton size="sm" />
        </div>
      )}
    </div>
  );
}

// =============================================================================
// Table Skeleton
// =============================================================================

interface TableSkeletonProps {
  rows?: number;
  columns?: number;
  hasHeader?: boolean;
  hasActions?: boolean;
  className?: string;
}

export function TableSkeleton({
  rows = 5,
  columns = 4,
  hasHeader = true,
  hasActions = true,
  className,
}: TableSkeletonProps) {
  const totalColumns = hasActions ? columns + 1 : columns;

  return (
    <div
      className={cn('bg-white rounded-lg shadow overflow-hidden', className)}
    >
      {/* Header */}
      {hasHeader && (
        <div className="px-6 py-4 border-b bg-gray-50">
          <div className="flex items-center justify-between">
            <Skeleton height={24} width={150} />
            <div className="flex gap-2">
              <ButtonSkeleton size="sm" width={100} />
              <ButtonSkeleton size="sm" width={80} />
            </div>
          </div>
        </div>
      )}

      {/* Table Header Row */}
      <div className="px-6 py-3 border-b bg-gray-50 flex gap-4">
        {Array.from({ length: totalColumns }).map((_, i) => (
          <Skeleton
            key={i}
            height={14}
            width={i === 0 ? '20%' : `${60 / (totalColumns - 1)}%`}
          />
        ))}
      </div>

      {/* Table Rows */}
      <div className="divide-y">
        {Array.from({ length: rows }).map((_, rowIndex) => (
          <div key={rowIndex} className="px-6 py-4 flex gap-4 items-center">
            {Array.from({ length: columns }).map((_, colIndex) => (
              <Skeleton
                key={colIndex}
                height={16}
                width={colIndex === 0 ? '20%' : `${60 / (columns - 1)}%`}
              />
            ))}
            {hasActions && (
              <div className="flex gap-2 ml-auto">
                <Skeleton variant="circular" width={32} height={32} />
                <Skeleton variant="circular" width={32} height={32} />
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Pagination */}
      <div className="px-6 py-4 border-t flex items-center justify-between">
        <Skeleton height={14} width={120} />
        <div className="flex gap-2">
          <ButtonSkeleton size="sm" width={32} />
          <ButtonSkeleton size="sm" width={32} />
          <ButtonSkeleton size="sm" width={32} />
        </div>
      </div>
    </div>
  );
}

// =============================================================================
// Dashboard Skeleton
// =============================================================================

interface DashboardSkeletonProps {
  statsCount?: number;
  className?: string;
}

export function DashboardSkeleton({
  statsCount = 4,
  className,
}: DashboardSkeletonProps) {
  return (
    <div className={cn('space-y-6', className)}>
      {/* Page Header */}
      <div className="flex items-center justify-between">
        <div>
          <Skeleton height={32} width={200} className="mb-2" />
          <Skeleton height={16} width={300} />
        </div>
        <div className="flex gap-2">
          <ButtonSkeleton width={120} />
          <ButtonSkeleton width={100} />
        </div>
      </div>

      {/* Stats Row */}
      <div className={`grid grid-cols-1 md:grid-cols-${statsCount} gap-4`}>
        {Array.from({ length: statsCount }).map((_, i) => (
          <div key={i} className="bg-white rounded-lg shadow p-6">
            <div className="flex items-center justify-between mb-2">
              <Skeleton height={14} width={80} />
              <Skeleton variant="circular" width={40} height={40} />
            </div>
            <Skeleton height={32} width={100} className="mb-1" />
            <Skeleton height={12} width={60} />
          </div>
        ))}
      </div>

      {/* Main Content */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2">
          <CardSkeleton />
        </div>
        <div>
          <CardSkeleton />
        </div>
      </div>

      {/* Table */}
      <TableSkeleton rows={5} />
    </div>
  );
}

// =============================================================================
// Form Skeleton
// =============================================================================

interface FormSkeletonProps {
  fields?: number;
  hasSubmit?: boolean;
  className?: string;
}

export function FormSkeleton({
  fields = 4,
  hasSubmit = true,
  className,
}: FormSkeletonProps) {
  return (
    <div className={cn('bg-white rounded-lg shadow p-6', className)}>
      <Skeleton height={24} width={150} className="mb-6" />

      <div className="space-y-4">
        {Array.from({ length: fields }).map((_, i) => (
          <InputSkeleton key={i} />
        ))}
      </div>

      {hasSubmit && (
        <div className="flex gap-3 mt-6 pt-6 border-t">
          <ButtonSkeleton width={100} />
          <ButtonSkeleton width={80} />
        </div>
      )}
    </div>
  );
}

// =============================================================================
// List Skeleton
// =============================================================================

interface ListSkeletonProps {
  items?: number;
  hasAvatar?: boolean;
  hasActions?: boolean;
  className?: string;
}

export function ListSkeleton({
  items = 5,
  hasAvatar = true,
  hasActions = false,
  className,
}: ListSkeletonProps) {
  return (
    <div className={cn('bg-white rounded-lg shadow divide-y', className)}>
      {Array.from({ length: items }).map((_, i) => (
        <div key={i} className="p-4 flex items-center gap-4">
          {hasAvatar && <AvatarSkeleton size="md" />}
          <div className="flex-1 min-w-0">
            <Skeleton height={16} width="40%" className="mb-2" />
            <Skeleton height={14} width="60%" />
          </div>
          {hasActions && (
            <div className="flex gap-2">
              <Skeleton variant="circular" width={32} height={32} />
            </div>
          )}
        </div>
      ))}
    </div>
  );
}

// =============================================================================
// Profile Skeleton
// =============================================================================

interface ProfileSkeletonProps {
  className?: string;
}

export function ProfileSkeleton({ className }: ProfileSkeletonProps) {
  return (
    <div className={cn('bg-white rounded-lg shadow', className)}>
      {/* Cover Image */}
      <ImageSkeleton height={200} className="rounded-t-lg" />

      {/* Profile Info */}
      <div className="p-6 relative">
        <div className="absolute -top-16 left-6">
          <AvatarSkeleton size={128} />
        </div>

        <div className="ml-40">
          <Skeleton height={28} width={200} className="mb-2" />
          <Skeleton height={16} width={150} className="mb-4" />

          <div className="flex gap-4 mb-4">
            <Skeleton height={14} width={80} />
            <Skeleton height={14} width={100} />
            <Skeleton height={14} width={120} />
          </div>

          <TextSkeleton lines={2} />
        </div>
      </div>

      {/* Stats */}
      <div className="px-6 py-4 border-t flex gap-8">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i}>
            <Skeleton height={24} width={40} className="mb-1" />
            <Skeleton height={14} width={60} />
          </div>
        ))}
      </div>
    </div>
  );
}

// =============================================================================
// Settings Skeleton
// =============================================================================

interface SettingsSkeletonProps {
  sections?: number;
  className?: string;
}

export function SettingsSkeleton({
  sections = 3,
  className,
}: SettingsSkeletonProps) {
  return (
    <div className={cn('space-y-6', className)}>
      {/* Page Header */}
      <div>
        <Skeleton height={32} width={150} className="mb-2" />
        <Skeleton height={16} width={300} />
      </div>

      {/* Settings Sections */}
      {Array.from({ length: sections }).map((_, sectionIndex) => (
        <div key={sectionIndex} className="bg-white rounded-lg shadow">
          <div className="p-6 border-b">
            <Skeleton height={20} width={120} className="mb-2" />
            <Skeleton height={14} width={250} />
          </div>

          <div className="p-6 space-y-6">
            {Array.from({ length: 3 }).map((_, i) => (
              <div key={i} className="flex items-center justify-between">
                <div className="flex-1">
                  <Skeleton height={16} width={150} className="mb-1" />
                  <Skeleton height={14} width={250} />
                </div>
                <Skeleton variant="rounded" width={48} height={24} />
              </div>
            ))}
          </div>
        </div>
      ))}
    </div>
  );
}

// =============================================================================
// Chart Skeleton
// =============================================================================

interface ChartSkeletonProps {
  type?: 'bar' | 'line' | 'pie';
  height?: number;
  className?: string;
}

export function ChartSkeleton({
  type = 'bar',
  height = 300,
  className,
}: ChartSkeletonProps) {
  return (
    <div className={cn('bg-white rounded-lg shadow p-6', className)}>
      <div className="flex items-center justify-between mb-4">
        <Skeleton height={20} width={150} />
        <div className="flex gap-2">
          <ButtonSkeleton size="sm" width={60} />
          <ButtonSkeleton size="sm" width={60} />
        </div>
      </div>

      <div
        className="relative bg-gray-100 rounded-lg overflow-hidden"
        style={{ height }}
      >
        {type === 'bar' && (
          <div className="absolute bottom-0 left-0 right-0 flex items-end justify-around p-4 h-full">
            {Array.from({ length: 7 }).map((_, i) => (
              <Skeleton
                key={i}
                width={40}
                height={`${30 + Math.random() * 60}%`}
                className="flex-shrink-0"
              />
            ))}
          </div>
        )}

        {type === 'line' && (
          <svg
            className="absolute inset-0 w-full h-full"
            preserveAspectRatio="none"
          >
            <defs>
              <linearGradient
                id="skeleton-gradient"
                x1="0"
                y1="0"
                x2="1"
                y2="0"
              >
                <stop offset="0%" stopColor="#e5e7eb" />
                <stop offset="50%" stopColor="#d1d5db" />
                <stop offset="100%" stopColor="#e5e7eb" />
              </linearGradient>
            </defs>
            <path
              d="M 0 200 Q 50 150, 100 180 T 200 120 T 300 160 T 400 100 T 500 140 T 600 80"
              fill="none"
              stroke="url(#skeleton-gradient)"
              strokeWidth="3"
              className="animate-pulse"
            />
          </svg>
        )}

        {type === 'pie' && (
          <div className="absolute inset-0 flex items-center justify-center">
            <Skeleton variant="circular" width={200} height={200} />
          </div>
        )}
      </div>

      {/* Legend */}
      <div className="flex gap-4 mt-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="flex items-center gap-2">
            <Skeleton variant="circular" width={12} height={12} />
            <Skeleton height={12} width={60} />
          </div>
        ))}
      </div>
    </div>
  );
}

// =============================================================================
// Navigation Skeleton
// =============================================================================

interface NavSkeletonProps {
  items?: number;
  className?: string;
}

export function NavSkeleton({ items = 5, className }: NavSkeletonProps) {
  return (
    <nav className={cn('space-y-1', className)}>
      {Array.from({ length: items }).map((_, i) => (
        <div key={i} className="flex items-center gap-3 px-3 py-2">
          <Skeleton variant="rounded" width={20} height={20} />
          <Skeleton height={16} width={`${60 + Math.random() * 30}%`} />
        </div>
      ))}
    </nav>
  );
}

// =============================================================================
// Header Skeleton
// =============================================================================

interface HeaderSkeletonProps {
  className?: string;
}

export function HeaderSkeleton({ className }: HeaderSkeletonProps) {
  return (
    <header
      className={cn(
        'bg-white border-b px-6 py-4 flex items-center justify-between',
        className
      )}
    >
      {/* Logo */}
      <Skeleton variant="rounded" width={120} height={32} />

      {/* Navigation */}
      <div className="hidden md:flex items-center gap-6">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} height={16} width={60 + i * 10} />
        ))}
      </div>

      {/* User Menu */}
      <div className="flex items-center gap-4">
        <Skeleton variant="circular" width={32} height={32} />
        <Skeleton variant="circular" width={40} height={40} />
      </div>
    </header>
  );
}

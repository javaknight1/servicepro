import React, { useState, useEffect, useRef, useCallback, memo } from 'react';
import { cn } from '../../utils/cn';
import {
  KPIWidgetProps,
  KPIData,
  KPITrend,
  KPITargetProgress,
  formatKPIValue,
  calculateTrend,
  calculateTargetProgress,
  getStatusColor,
  getStatusBgColor,
  getTrendColor,
} from '../../types/kpi';

// Trend indicator icons
const TrendUpIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg
    className={className}
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
    strokeWidth={2}
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      d="M7 11l5-5m0 0l5 5m-5-5v12"
    />
  </svg>
);

const TrendDownIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg
    className={className}
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
    strokeWidth={2}
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      d="M17 13l-5 5m0 0l-5-5m5 5V6"
    />
  </svg>
);

const TrendNeutralIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg
    className={className}
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
    strokeWidth={2}
  >
    <path strokeLinecap="round" strokeLinejoin="round" d="M5 12h14" />
  </svg>
);

// Info icon for tooltip
const InfoIcon: React.FC<{ className?: string }> = ({ className }) => (
  <svg
    className={className}
    fill="none"
    viewBox="0 0 24 24"
    stroke="currentColor"
    strokeWidth={2}
  >
    <path
      strokeLinecap="round"
      strokeLinejoin="round"
      d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
    />
  </svg>
);

// Loading skeleton
const KPIWidgetSkeleton: React.FC<{ size?: 'sm' | 'md' | 'lg' | 'xl' }> = ({
  size = 'md',
}) => {
  const sizeClasses = {
    sm: 'h-16',
    md: 'h-24',
    lg: 'h-32',
    xl: 'h-40',
  };

  return (
    <div
      className={cn(
        'bg-white rounded-lg border border-gray-200 p-4 animate-pulse',
        sizeClasses[size]
      )}
      data-testid="kpi-skeleton"
    >
      <div className="h-3 bg-gray-200 rounded w-1/3 mb-3" />
      <div className="h-6 bg-gray-200 rounded w-2/3 mb-2" />
      <div className="h-3 bg-gray-200 rounded w-1/4" />
    </div>
  );
};

// Error state
const KPIWidgetError: React.FC<{ error: string; onRetry?: () => void }> = ({
  error,
  onRetry,
}) => (
  <div
    className="bg-red-50 rounded-lg border border-red-200 p-4 text-center"
    data-testid="kpi-error"
  >
    <div className="text-red-600 text-sm mb-2">{error}</div>
    {onRetry && (
      <button
        onClick={onRetry}
        className="text-red-700 text-xs underline hover:no-underline"
      >
        Try again
      </button>
    )}
  </div>
);

// Sparkline component
const Sparkline: React.FC<{ data: number[]; className?: string }> = memo(
  ({ data, className }) => {
    if (!data || data.length < 2) return null;

    const width = 80;
    const height = 24;
    const padding = 2;

    const min = Math.min(...data);
    const max = Math.max(...data);
    const range = max - min || 1;

    const points = data
      .map((value, index) => {
        const x = padding + (index / (data.length - 1)) * (width - padding * 2);
        const y =
          height - padding - ((value - min) / range) * (height - padding * 2);
        return `${x},${y}`;
      })
      .join(' ');

    const lastValue = data[data.length - 1];
    const prevValue = data[data.length - 2];
    const isUp = lastValue >= prevValue;

    return (
      <svg
        viewBox={`0 0 ${width} ${height}`}
        className={cn('w-20 h-6', className)}
        data-testid="kpi-sparkline"
      >
        <polyline
          points={points}
          fill="none"
          stroke={isUp ? '#22c55e' : '#ef4444'}
          strokeWidth="1.5"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
        <circle
          cx={width - padding}
          cy={
            height -
            padding -
            ((lastValue - min) / range) * (height - padding * 2)
          }
          r="2"
          fill={isUp ? '#22c55e' : '#ef4444'}
        />
      </svg>
    );
  }
);

Sparkline.displayName = 'Sparkline';

// Tooltip component
const Tooltip: React.FC<{
  content: string;
  children: React.ReactNode;
  visible: boolean;
}> = ({ content, children, visible }) => (
  <div className="relative inline-block">
    {children}
    {visible && (
      <div
        className="absolute bottom-full left-1/2 -translate-x-1/2 mb-2 px-3 py-2 bg-gray-900 text-white text-xs rounded-lg shadow-lg whitespace-nowrap z-50"
        role="tooltip"
      >
        {content}
        <div className="absolute top-full left-1/2 -translate-x-1/2 border-4 border-transparent border-t-gray-900" />
      </div>
    )}
  </div>
);

// Progress bar for target
const TargetProgress: React.FC<{ progress: KPITargetProgress }> = ({
  progress,
}) => (
  <div className="mt-2" data-testid="kpi-target-progress">
    <div className="flex justify-between text-xs text-gray-500 mb-1">
      <span>Progress</span>
      <span>{progress.percentage.toFixed(0)}%</span>
    </div>
    <div className="h-1.5 bg-gray-200 rounded-full overflow-hidden">
      <div
        className={cn(
          'h-full rounded-full transition-all duration-500',
          progress.isOnTrack ? 'bg-green-500' : 'bg-amber-500'
        )}
        style={{ width: `${Math.min(progress.percentage, 100)}%` }}
      />
    </div>
  </div>
);

// Animated number component
const AnimatedValue: React.FC<{
  value: number;
  format: (v: number) => string;
  animate?: boolean;
}> = ({ value, format, animate = true }) => {
  const [displayValue, setDisplayValue] = useState(value);
  const prevValueRef = useRef(value);

  useEffect(() => {
    if (!animate || prevValueRef.current === value) {
      setDisplayValue(value);
      return;
    }

    const startValue = prevValueRef.current;
    const endValue = value;
    const duration = 500;
    const startTime = performance.now();

    const animateValue = (currentTime: number) => {
      const elapsed = currentTime - startTime;
      const progress = Math.min(elapsed / duration, 1);

      // Ease-out cubic
      const easeOut = 1 - Math.pow(1 - progress, 3);
      const currentValue = startValue + (endValue - startValue) * easeOut;

      setDisplayValue(currentValue);

      if (progress < 1) {
        requestAnimationFrame(animateValue);
      } else {
        setDisplayValue(endValue);
      }
    };

    requestAnimationFrame(animateValue);
    prevValueRef.current = value;
  }, [value, animate]);

  return <>{format(displayValue)}</>;
};

// Main KPI Widget Component
const KPIWidget: React.FC<KPIWidgetProps> = ({
  data,
  showTrend = true,
  showTarget = false,
  showSparkline = false,
  sparklineData,
  loading = false,
  error = null,
  size = 'md',
  realTime = false,
  updateInterval,
  onClick,
  className,
  animate = true,
  showTooltip = true,
  compact = false,
}) => {
  const [tooltipVisible, setTooltipVisible] = useState(false);
  const [isUpdating, setIsUpdating] = useState(false);
  const prevValueRef = useRef(data.value);

  // Detect value changes for animation
  useEffect(() => {
    if (prevValueRef.current !== data.value) {
      setIsUpdating(true);
      const timeout = setTimeout(() => setIsUpdating(false), 1000);
      prevValueRef.current = data.value;
      return () => clearTimeout(timeout);
    }
  }, [data.value]);

  // Size-based styles
  const sizeStyles = {
    sm: {
      container: 'p-3',
      label: 'text-xs',
      value: 'text-lg font-semibold',
      trend: 'text-xs',
      icon: 'w-3 h-3',
    },
    md: {
      container: 'p-4',
      label: 'text-sm',
      value: 'text-2xl font-bold',
      trend: 'text-sm',
      icon: 'w-4 h-4',
    },
    lg: {
      container: 'p-5',
      label: 'text-base',
      value: 'text-3xl font-bold',
      trend: 'text-base',
      icon: 'w-5 h-5',
    },
    xl: {
      container: 'p-6',
      label: 'text-lg',
      value: 'text-4xl font-bold',
      trend: 'text-lg',
      icon: 'w-6 h-6',
    },
  };

  const styles = sizeStyles[size];

  // Calculate trend
  const trend: KPITrend | null = showTrend
    ? calculateTrend(data.value, data.previousValue)
    : null;

  // Calculate target progress
  const targetProgress: KPITargetProgress | null = showTarget
    ? calculateTargetProgress(data.value, data.target)
    : null;

  // Format value
  const formatValue = useCallback(
    (v: number) => {
      const formatted = formatKPIValue(v, data.format, {
        currency: data.currency,
        decimals: data.decimals,
      });
      return `${data.prefix || ''}${formatted}${data.suffix || ''}`;
    },
    [data.format, data.currency, data.decimals, data.prefix, data.suffix]
  );

  // Loading state
  if (loading) {
    return <KPIWidgetSkeleton size={size} />;
  }

  // Error state
  if (error) {
    return <KPIWidgetError error={error} />;
  }

  // Get trend icon
  const getTrendIcon = () => {
    if (!trend) return null;
    const iconClass = cn(styles.icon, getTrendColor(trend.direction));
    switch (trend.direction) {
      case 'up':
        return <TrendUpIcon className={iconClass} />;
      case 'down':
        return <TrendDownIcon className={iconClass} />;
      default:
        return <TrendNeutralIcon className={iconClass} />;
    }
  };

  return (
    <div
      className={cn(
        'bg-white rounded-lg border border-gray-200 transition-all duration-200',
        styles.container,
        onClick && 'cursor-pointer hover:shadow-md hover:border-gray-300',
        isUpdating && 'ring-2 ring-blue-400 ring-opacity-50',
        data.status && getStatusBgColor(data.status),
        className
      )}
      onClick={onClick}
      onMouseEnter={() => setTooltipVisible(true)}
      onMouseLeave={() => setTooltipVisible(false)}
      data-testid="kpi-widget"
      role={onClick ? 'button' : undefined}
      tabIndex={onClick ? 0 : undefined}
      onKeyDown={(e) => {
        if (onClick && (e.key === 'Enter' || e.key === ' ')) {
          e.preventDefault();
          onClick();
        }
      }}
    >
      {/* Header with label and info icon */}
      <div className="flex items-center justify-between mb-1">
        {!compact && (
          <div className="flex items-center gap-1">
            {data.icon && (
              <span className="text-gray-400 mr-1">
                {typeof data.icon === 'string' ? (
                  <span className="text-lg">{data.icon}</span>
                ) : (
                  data.icon
                )}
              </span>
            )}
            <span
              className={cn(styles.label, 'text-gray-600 font-medium truncate')}
            >
              {data.label}
            </span>
          </div>
        )}

        {showTooltip && data.description && (
          <Tooltip content={data.description} visible={tooltipVisible}>
            <InfoIcon
              className={cn(styles.icon, 'text-gray-400 hover:text-gray-600')}
            />
          </Tooltip>
        )}
      </div>

      {/* Main value */}
      <div className="flex items-baseline gap-2">
        <span
          className={cn(
            styles.value,
            data.status ? getStatusColor(data.status) : 'text-gray-900'
          )}
          data-testid="kpi-value"
        >
          <AnimatedValue
            value={data.value}
            format={formatValue}
            animate={animate}
          />
        </span>

        {/* Trend indicator */}
        {trend && trend.direction !== 'neutral' && (
          <div
            className={cn(
              'flex items-center gap-0.5',
              getTrendColor(trend.direction)
            )}
            data-testid="kpi-trend"
          >
            {getTrendIcon()}
            <span className={styles.trend}>{trend.label}</span>
          </div>
        )}
      </div>

      {/* Sparkline */}
      {showSparkline && sparklineData && sparklineData.length > 1 && (
        <div className="mt-2">
          <Sparkline data={sparklineData} />
        </div>
      )}

      {/* Target progress */}
      {targetProgress && <TargetProgress progress={targetProgress} />}

      {/* Last updated */}
      {data.lastUpdated && (
        <div
          className="mt-2 text-xs text-gray-400"
          data-testid="kpi-last-updated"
        >
          Updated {new Date(data.lastUpdated).toLocaleTimeString()}
        </div>
      )}

      {/* Real-time indicator */}
      {realTime && (
        <div className="absolute top-2 right-2">
          <span className="relative flex h-2 w-2">
            <span className="animate-ping absolute inline-flex h-full w-full rounded-full bg-green-400 opacity-75" />
            <span className="relative inline-flex rounded-full h-2 w-2 bg-green-500" />
          </span>
        </div>
      )}
    </div>
  );
};

export default memo(KPIWidget);
export { KPIWidgetSkeleton, KPIWidgetError, Sparkline, TargetProgress };

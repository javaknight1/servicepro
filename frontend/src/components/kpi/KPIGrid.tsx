import React, { memo } from 'react';
import { cn } from '../../utils/cn';
import { KPIGridProps } from '../../types/kpi';
import KPIWidget, { KPIWidgetSkeleton } from './KPIWidget';

const KPIGrid: React.FC<KPIGridProps> = ({
  kpis,
  columns = 4,
  showTrends = true,
  gap = 'md',
  loading = false,
  error = null,
  className,
}) => {
  const columnClasses = {
    1: 'grid-cols-1',
    2: 'grid-cols-1 sm:grid-cols-2',
    3: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3',
    4: 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-4',
    6: 'grid-cols-2 sm:grid-cols-3 lg:grid-cols-6',
  };

  const gapClasses = {
    sm: 'gap-2',
    md: 'gap-4',
    lg: 'gap-6',
  };

  if (error) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-6 text-center">
        <div className="text-red-600 font-medium mb-2">Failed to load KPIs</div>
        <div className="text-red-500 text-sm">{error}</div>
      </div>
    );
  }

  if (loading) {
    return (
      <div
        className={cn(
          'grid',
          columnClasses[columns],
          gapClasses[gap],
          className
        )}
        data-testid="kpi-grid-loading"
      >
        {Array.from({ length: columns }).map((_, index) => (
          <KPIWidgetSkeleton key={index} />
        ))}
      </div>
    );
  }

  if (!kpis || kpis.length === 0) {
    return (
      <div className="bg-gray-50 border border-gray-200 rounded-lg p-6 text-center">
        <div className="text-gray-600 font-medium">No KPIs available</div>
        <div className="text-gray-400 text-sm mt-1">
          KPIs will appear here once data is available
        </div>
      </div>
    );
  }

  return (
    <div
      className={cn('grid', columnClasses[columns], gapClasses[gap], className)}
      data-testid="kpi-grid"
    >
      {kpis.map((kpi) => (
        <KPIWidget key={kpi.id} data={kpi} showTrend={showTrends} />
      ))}
    </div>
  );
};

export default memo(KPIGrid);

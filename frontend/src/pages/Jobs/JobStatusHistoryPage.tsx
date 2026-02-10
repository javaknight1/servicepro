import { useState, useEffect } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { DashboardLayout } from '@components/layout';
import { Button, Badge, getStatusBadgeVariant } from '@components/shared';
import {
  jobService,
  Job,
  JobStatusTransition,
  StatusHistoryResponse,
} from '@services/jobService';
import { getJobStatusLabel } from '@app-types';
import { formatEpochDateTimeUTC } from '@/utils/epoch';
import {
  ArrowLeft,
  ArrowRight,
  Loader2,
  SortAsc,
  SortDesc,
} from 'lucide-react';

export function JobStatusHistoryPage() {
  const navigate = useNavigate();
  const { id } = useParams<{ id: string }>();
  const [job, setJob] = useState<Job | null>(null);
  const [history, setHistory] = useState<StatusHistoryResponse | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [sortOrder, setSortOrder] = useState<'asc' | 'desc'>('desc');

  useEffect(() => {
    if (id) {
      loadData();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [id, sortOrder]);

  const loadData = async () => {
    if (!id) return;

    setIsLoading(true);
    setError(null);
    try {
      const [jobData, historyData] = await Promise.all([
        jobService.getJob(id),
        jobService.getStatusHistory(id, 50, 0, sortOrder),
      ]);
      setJob(jobData);
      setHistory(historyData);
    } catch (err) {
      console.error('Failed to load data:', err);
      setError('Failed to load status history');
    } finally {
      setIsLoading(false);
    }
  };

  const toggleSort = () => {
    setSortOrder((prev) => (prev === 'desc' ? 'asc' : 'desc'));
  };

  const formatDate = (epoch: number) => {
    return formatEpochDateTimeUTC(epoch);
  };

  if (isLoading) {
    return (
      <DashboardLayout>
        <div className="flex items-center justify-center min-h-96">
          <Loader2 className="h-8 w-8 animate-spin text-primary-500" />
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout>
      <div className="p-6 lg:p-8 max-w-4xl mx-auto">
        <div className="flex items-center gap-4 mb-6">
          <Button
            variant="ghost"
            onClick={() => navigate('/jobs')}
            className="flex items-center gap-2"
          >
            <ArrowLeft className="h-4 w-4" />
            Back to Jobs
          </Button>
          <div>
            <h1 className="text-2xl font-bold text-neutral-900">
              Status History
            </h1>
            {job && (
              <p className="text-neutral-600 mt-1">
                {job.job_number} - {job.title}
              </p>
            )}
          </div>
        </div>

        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
            {error}
          </div>
        )}

        <div className="bg-white rounded-lg border border-neutral-200">
          <div className="px-6 py-4 border-b border-neutral-200 flex items-center justify-between">
            <h2 className="text-lg font-semibold text-neutral-900">
              Transition History
            </h2>
            <button
              onClick={toggleSort}
              className="flex items-center gap-2 text-sm text-neutral-600 hover:text-neutral-900"
            >
              {sortOrder === 'desc' ? (
                <>
                  <SortDesc className="h-4 w-4" />
                  Newest first
                </>
              ) : (
                <>
                  <SortAsc className="h-4 w-4" />
                  Oldest first
                </>
              )}
            </button>
          </div>

          {history && history.transitions.length > 0 ? (
            <div className="divide-y divide-neutral-200">
              {history.transitions.map((transition: JobStatusTransition) => (
                <div key={transition.id} className="p-6">
                  <div className="flex items-start justify-between">
                    <div className="flex items-center gap-3">
                      <Badge
                        variant={getStatusBadgeVariant(transition.from_status)}
                        size="sm"
                      >
                        {transition.from_status
                          ? getJobStatusLabel(transition.from_status)
                          : 'Created'}
                      </Badge>
                      <ArrowRight className="h-4 w-4 text-neutral-400" />
                      <Badge
                        variant={getStatusBadgeVariant(transition.to_status)}
                        size="sm"
                      >
                        {getJobStatusLabel(transition.to_status)}
                      </Badge>
                    </div>
                    <span className="text-sm text-neutral-500">
                      {formatDate(transition.transitioned_at)}
                    </span>
                  </div>

                  <div className="mt-3 text-sm text-neutral-600">
                    <span className="font-medium">Changed by:</span>{' '}
                    {transition.changed_by_name || 'Unknown'}
                  </div>

                  {transition.reason && (
                    <div className="mt-2 text-sm text-neutral-600">
                      <span className="font-medium">Reason:</span>{' '}
                      {transition.reason.replace(/_/g, ' ')}
                    </div>
                  )}

                  {transition.notes && (
                    <div className="mt-2 p-3 bg-neutral-50 rounded-lg text-sm text-neutral-700">
                      {transition.notes}
                    </div>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <div className="p-6 text-center text-neutral-500">
              No status history found
            </div>
          )}
        </div>
      </div>
    </DashboardLayout>
  );
}

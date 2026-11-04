import { FileSpreadsheet } from 'lucide-react';
import { Link } from 'react-router';
import { Card } from '@components/shared';
import { QuickBooksCard } from './QuickBooksCard';

const IMPORT_SOURCES = [
  { icon: 'J', label: 'Jobber', source: 'jobber' },
  { icon: 'H', label: 'Housecall Pro', source: 'housecall_pro' },
  { icon: null, label: 'CSV file', source: 'generic' },
] as const;

/**
 * Settings → Integrations (mockup 14): QuickBooks Online sync (T192), Stripe
 * payouts (T169, on its own page) and imports (T193).
 */
export function IntegrationsTab({
  tenantId,
  canManage,
  callback,
  onCallbackSeen,
}: {
  tenantId: string;
  canManage: boolean;
  callback?: { result: string | null; reason: string | null };
  onCallbackSeen?: () => void;
}) {
  return (
    <div className="space-y-6">
      <QuickBooksCard
        tenantId={tenantId}
        canManage={canManage}
        callback={callback}
        onCallbackSeen={onCallbackSeen}
      />
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card variant="elevated" padding="lg">
          <div className="flex items-start gap-3">
            <span
              aria-hidden
              className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg border border-neutral-200 text-xl font-bold text-[#635bff]"
            >
              S
            </span>
            <div className="min-w-0 flex-1">
              <h2 className="font-semibold text-neutral-900">Stripe payouts</h2>
              <p className="mt-0.5 text-sm text-neutral-600">
                Online invoice payments, deposited to your bank.
              </p>
            </div>
            <Link
              to="/settings/payments"
              className="text-sm font-medium text-primary-600 hover:underline"
            >
              Manage
            </Link>
          </div>
        </Card>
        <Card variant="elevated" padding="lg">
          <h2 className="font-semibold text-neutral-900">Import data</h2>
          <p className="mt-0.5 text-sm text-neutral-600">
            Customers, sites, jobs and open invoices.
          </p>
          <div className="mt-4 flex flex-wrap gap-2">
            {IMPORT_SOURCES.map((s) => (
              <Link
                key={s.source}
                to={`/settings/import?source=${s.source}`}
                className="flex items-center gap-2 rounded-lg border border-neutral-200 px-3 py-2 text-sm text-neutral-800 hover:border-primary-600"
              >
                <span className="flex h-6 w-6 items-center justify-center rounded-md bg-neutral-100 text-xs font-medium text-neutral-600">
                  {s.icon ?? <FileSpreadsheet className="h-3.5 w-3.5" />}
                </span>
                {s.label}
              </Link>
            ))}
          </div>
        </Card>
      </div>
    </div>
  );
}

import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router';
import { useQueries, useQuery } from '@tanstack/react-query';
import { FilePenLine, Plus, Repeat, Search } from 'lucide-react';
import { DashboardLayout } from '@components/layout';
import {
  Badge,
  Button,
  ErrorState,
  PageHeader,
  StatsCard,
} from '@components/shared';
import {
  agreementKeys,
  agreementService,
  describeFrequency,
  monthlyAmount,
  type Agreement,
  type AgreementStatus,
} from '@services/agreementService';
import { fetchAllCustomers } from '@services/customerLookup';
import type { Customer } from '@services/customerService';
import { siteService, sitesQueryKey, type Site } from '@services/siteService';
import { formatDollars } from '@/utils/money';
import { cn } from '@utils/cn';
import { customerName, formatVisit } from './agreementDisplay';

type StatusFilter = 'all' | AgreementStatus;
const PAGE_SIZE = 20;

/** Service agreements list (T167, mockup 07). */
export function AgreementsPage() {
  const navigate = useNavigate();
  const [filter, setFilter] = useState<StatusFilter>('all');
  const [search, setSearch] = useState('');
  const [page, setPage] = useState(0);

  const agreementsQuery = useQuery({
    queryKey: agreementKeys.list(),
    queryFn: () => agreementService.list(),
  });
  const customersQuery = useQuery({
    queryKey: ['customers', 'all'],
    queryFn: () => fetchAllCustomers(),
  });
  const agreements = useMemo(
    () => agreementsQuery.data ?? [],
    [agreementsQuery.data]
  );

  // Site names: one request per customer with agreements.
  const customerIds = useMemo(
    () => [...new Set(agreements.map((a) => a.customer_id))],
    [agreements]
  );
  const sitesQueries = useQueries({
    queries: customerIds.map((customerId) => ({
      queryKey: sitesQueryKey(customerId, true),
      queryFn: () => siteService.list(customerId, true),
    })),
  });
  const sitesById = new Map<string, Site>();
  for (const q of sitesQueries) {
    for (const site of q.data ?? []) sitesById.set(site.id, site);
  }
  const customersById = new Map(
    (customersQuery.data ?? []).map((c) => [c.id, c])
  );

  const active = agreements.filter((a) => a.status === 'active');
  const paused = agreements.filter((a) => a.status === 'paused');
  const recurring = active.reduce((sum, a) => sum + monthlyAmount(a), 0);
  const activeCustomers = new Set(active.map((a) => a.customer_id)).size;

  const term = search.trim().toLowerCase();
  const filtered = agreements.filter((a) => {
    if (filter !== 'all' && a.status !== filter) return false;
    if (!term) return true;
    const site = sitesById.get(a.site_id)?.name ?? '';
    const customer = customerName(customersById.get(a.customer_id));
    return [site, customer, a.title].some((v) =>
      v.toLowerCase().includes(term)
    );
  });
  const pageCount = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const currentPage = Math.min(page, pageCount - 1);
  const rows = filtered.slice(
    currentPage * PAGE_SIZE,
    (currentPage + 1) * PAGE_SIZE
  );
  const shownMonthly = rows.reduce((sum, a) => sum + monthlyAmount(a), 0);

  // Next visit for the active rows on this page.
  const previewQueries = useQueries({
    queries: rows.map((a) => ({
      queryKey: agreementKeys.preview(a.id, 14),
      queryFn: () => agreementService.preview(a.id, 14),
      enabled: a.status === 'active',
    })),
  });

  const tabs: Array<[StatusFilter, string, number]> = [
    ['all', 'All', agreements.length],
    ['active', 'Active', active.length],
    ['paused', 'Paused', paused.length],
  ];

  return (
    <DashboardLayout breadcrumbs={[{ label: 'Agreements' }]}>
      <div className="mx-auto max-w-[1200px] p-4 sm:p-6 lg:p-8">
        <PageHeader
          title="Service agreements"
          subtitle="Recurring work per site. Visits are scheduled from these automatically."
          actions={
            <Button
              onClick={() => navigate('/agreements/new')}
              className="gap-2"
            >
              <Plus className="h-4 w-4" />
              New agreement
            </Button>
          }
        />

        <div className="mb-6 grid grid-cols-1 gap-4 sm:grid-cols-2">
          <StatsCard
            label="Active agreements"
            value={agreementsQuery.isPending ? '—' : active.length}
            icon={<FilePenLine className="h-6 w-6" />}
            change={`across ${activeCustomers} ${activeCustomers === 1 ? 'customer' : 'customers'}`}
          />
          <StatsCard
            label="Recurring revenue"
            value={
              agreementsQuery.isPending
                ? '—'
                : `${formatDollars(recurring, { wholeUnits: true })}/mo`
            }
            icon={<Repeat className="h-6 w-6" />}
            change="before sales tax"
          />
        </div>

        <div className="mb-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div role="tablist" aria-label="Status" className="flex gap-1">
            {tabs.map(([value, label, count]) => (
              <button
                key={value}
                type="button"
                role="tab"
                aria-selected={filter === value}
                onClick={() => {
                  setFilter(value);
                  setPage(0);
                }}
                className={cn(
                  'rounded-lg px-3.5 py-2 text-[15px] font-medium transition-colors',
                  filter === value
                    ? 'bg-primary-50 text-primary-600'
                    : 'text-neutral-700 hover:bg-neutral-100'
                )}
              >
                {label} {count}
              </button>
            ))}
          </div>
          <label className="relative block sm:w-80">
            <span className="sr-only">Search sites or customers</span>
            <Search className="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-neutral-400" />
            <input
              type="search"
              value={search}
              onChange={(e) => {
                setSearch(e.target.value);
                setPage(0);
              }}
              placeholder="Search sites or customers"
              className="w-full rounded-lg border border-neutral-300 py-2 pr-3 pl-9 focus:border-primary-500 focus:ring-2 focus:ring-primary-500"
            />
          </label>
        </div>

        {agreementsQuery.isError ? (
          <ErrorState
            title="Couldn't load agreements"
            onRetry={() => void agreementsQuery.refetch()}
          />
        ) : (
          <section className="overflow-hidden rounded-2xl border border-neutral-200 bg-white">
            <div className="overflow-x-auto">
              <table className="w-full min-w-[860px] text-left text-[15px]">
                <thead className="border-b border-neutral-200 bg-neutral-50 text-sm font-semibold text-neutral-700">
                  <tr>
                    <th className="px-5 py-3">Site</th>
                    <th className="px-4 py-3">Customer</th>
                    <th className="px-4 py-3">Frequency</th>
                    <th className="px-4 py-3">Next visit</th>
                    <th className="px-4 py-3">Invoice style</th>
                    <th className="px-4 py-3 text-right">Monthly</th>
                    <th className="px-5 py-3">Status</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-neutral-200">
                  {agreementsQuery.isPending ? (
                    <tr>
                      <td
                        colSpan={7}
                        className="px-5 py-10 text-center text-neutral-500"
                      >
                        Loading agreements…
                      </td>
                    </tr>
                  ) : rows.length === 0 ? (
                    <tr>
                      <td colSpan={7} className="px-5 py-10 text-center">
                        <p className="font-medium text-neutral-900">
                          {agreements.length === 0
                            ? 'No agreements yet'
                            : 'No agreements match'}
                        </p>
                        {agreements.length === 0 && (
                          <p className="mt-1 text-sm text-neutral-500">
                            Set up recurring work for a site and its visits are
                            scheduled for you.
                          </p>
                        )}
                      </td>
                    </tr>
                  ) : (
                    rows.map((a, i) => (
                      <AgreementRow
                        key={a.id}
                        agreement={a}
                        site={sitesById.get(a.site_id)}
                        customer={customersById.get(a.customer_id)}
                        nextVisit={previewQueries[i]?.data?.visits[0]}
                        timeZone={previewQueries[i]?.data?.timezone}
                        onOpen={() => navigate(`/agreements/${a.id}`)}
                      />
                    ))
                  )}
                </tbody>
              </table>
            </div>
            {filtered.length > 0 && (
              <footer className="flex flex-col gap-2 border-t border-neutral-200 bg-neutral-50/60 px-5 py-4 text-[15px] text-neutral-600 sm:flex-row sm:items-center sm:justify-between">
                <p>
                  Showing {rows.length} of {filtered.length} ·{' '}
                  <span className="font-semibold text-neutral-900">
                    {formatDollars(shownMonthly, { wholeUnits: true })}
                  </span>{' '}
                  of {formatDollars(recurring, { wholeUnits: true })}/mo
                </p>
                {pageCount > 1 && (
                  <div className="flex items-center gap-2">
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={currentPage === 0}
                      onClick={() => setPage(currentPage - 1)}
                    >
                      Previous
                    </Button>
                    <span>
                      Page {currentPage + 1} of {pageCount}
                    </span>
                    <Button
                      variant="ghost"
                      size="sm"
                      disabled={currentPage >= pageCount - 1}
                      onClick={() => setPage(currentPage + 1)}
                    >
                      Next
                    </Button>
                  </div>
                )}
              </footer>
            )}
          </section>
        )}
      </div>
    </DashboardLayout>
  );
}

interface AgreementRowProps {
  agreement: Agreement;
  site?: Site;
  customer?: Customer;
  nextVisit?: { starts_at: string };
  timeZone?: string;
  onOpen: () => void;
}

function AgreementRow({
  agreement: a,
  site,
  customer,
  nextVisit,
  timeZone,
  onOpen,
}: AgreementRowProps) {
  return (
    <tr
      onClick={onOpen}
      className="cursor-pointer transition-colors hover:bg-neutral-50"
    >
      <td className="px-5 py-3.5">
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            onOpen();
          }}
          className="text-left font-medium text-neutral-900 hover:text-primary-600"
        >
          {site?.name ?? a.title}
        </button>
        {site && <p className="text-sm text-neutral-500">{a.title}</p>}
      </td>
      <td className="px-4 py-3.5 text-neutral-700">{customerName(customer)}</td>
      <td className="px-4 py-3.5 text-neutral-700">{describeFrequency(a)}</td>
      <td className="px-4 py-3.5 text-neutral-700">
        {a.status === 'paused'
          ? '—'
          : nextVisit
            ? formatVisit(nextVisit.starts_at, timeZone)
            : 'None in 2 weeks'}
      </td>
      <td className="px-4 py-3.5 text-neutral-700">
        {customer?.consolidated_billing ? 'Consolidated' : 'Per job'}
      </td>
      <td className="px-4 py-3.5 text-right font-semibold text-neutral-900 tabular-nums">
        {formatDollars(monthlyAmount(a), { wholeUnits: true })}
        {a.billing_period === 'visit' && (
          <span className="block text-xs font-normal text-neutral-500">
            {formatDollars(a.price_per_period)}/visit
          </span>
        )}
      </td>
      <td className="px-5 py-3.5">
        <Badge
          size="sm"
          variant={a.status === 'active' ? 'success' : 'neutral'}
        >
          {a.status === 'active' ? 'Active' : 'Paused'}
        </Badge>
      </td>
    </tr>
  );
}

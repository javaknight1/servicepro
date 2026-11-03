import { useState } from 'react';
import { Link } from 'react-router';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Building2, Loader2, Pencil } from 'lucide-react';
import { Badge, Button } from '@components/shared';
import { siteService, sitesQueryKey, type Site } from '@services/siteService';
import {
  agreementKeys,
  agreementService,
  describeFrequency,
  monthlyAmount,
  type Agreement,
} from '@services/agreementService';
import { getErrorMessage } from '@/utils/error';
import { formatDollars } from '@/utils/money';
import { SiteFormModal } from './SiteFormModal';

interface CustomerSitesSectionProps {
  customerId: string;
  consolidatedBilling: boolean;
  /** Controlled "Add site" dialog (the page header has the button too). */
  isAddOpen: boolean;
  onAddOpenChange: (open: boolean) => void;
}

/** The Sites tab of mockup 06 (T186): a customer's service locations. */
export function CustomerSitesSection({
  customerId,
  consolidatedBilling,
  isAddOpen,
  onAddOpenChange,
}: CustomerSitesSectionProps) {
  const queryClient = useQueryClient();
  const [editing, setEditing] = useState<Site | null>(null);

  const sitesQuery = useQuery({
    queryKey: sitesQueryKey(customerId, true),
    queryFn: () => siteService.list(customerId, true),
  });

  const toggleActive = useMutation({
    mutationFn: (site: Site) =>
      siteService.update(customerId, site.id, { is_active: !site.is_active }),
    onSuccess: () =>
      queryClient.invalidateQueries({
        queryKey: ['customer-sites', customerId],
      }),
  });

  // Each site's agreements (T167): the Agreement and Monthly columns.
  const agreementsQuery = useQuery({
    queryKey: agreementKeys.list({ customer_id: customerId }),
    queryFn: () => agreementService.list({ customer_id: customerId }),
  });
  const agreementsBySite = new Map<string, Agreement[]>();
  for (const a of agreementsQuery.data ?? []) {
    agreementsBySite.set(a.site_id, [
      ...(agreementsBySite.get(a.site_id) ?? []),
      a,
    ]);
  }
  const siteMonthly = (siteId: string) =>
    (agreementsBySite.get(siteId) ?? [])
      .filter((a) => a.status === 'active')
      .reduce((sum, a) => sum + monthlyAmount(a), 0);

  const sites = sitesQuery.data ?? [];
  const totalMonthly = sites.reduce((sum, s) => sum + siteMonthly(s.id), 0);
  const activeCount = sites.filter((s) => s.is_active).length;

  const handleToggle = (site: Site) => {
    if (
      site.is_active &&
      !window.confirm(
        `Deactivate ${site.name}? New jobs and agreements can't use it until you reactivate it.`
      )
    ) {
      return;
    }
    toggleActive.mutate(site);
  };

  return (
    <section
      aria-label="Sites"
      className="overflow-hidden rounded-2xl border border-neutral-200 bg-white"
    >
      {toggleActive.error && (
        <p
          role="alert"
          className="border-b border-error-200 bg-error-50 px-6 py-3 text-sm text-error-700"
        >
          {getErrorMessage(toggleActive.error, 'Could not update the site')}
        </p>
      )}
      {sitesQuery.isPending ? (
        <div className="flex justify-center py-10">
          <Loader2 className="h-6 w-6 animate-spin text-neutral-400" />
        </div>
      ) : sitesQuery.isError ? (
        <p
          role="alert"
          className="px-6 py-8 text-center text-sm text-error-700"
        >
          {getErrorMessage(sitesQuery.error, 'Could not load the sites')}
        </p>
      ) : sites.length === 0 ? (
        <div className="px-6 py-10 text-center">
          <p className="font-medium text-neutral-900">No sites yet</p>
          <p className="mt-1 text-sm text-neutral-500">
            Add each building or property you service for this customer.
          </p>
          <Button
            variant="outline"
            size="sm"
            className="mt-4"
            onClick={() => onAddOpenChange(true)}
          >
            Add site
          </Button>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[960px] text-left text-[15px]">
            <thead className="border-b border-neutral-200 bg-neutral-50 text-sm font-semibold text-neutral-700">
              <tr>
                <th className="px-6 py-3">Site</th>
                <th className="px-4 py-3">Address</th>
                <th className="px-4 py-3">Agreement</th>
                <th className="px-4 py-3">Site contact</th>
                <th className="px-4 py-3 text-right">Monthly</th>
                <th className="px-4 py-3">Status</th>
                <th className="px-6 py-3">
                  <span className="sr-only">Actions</span>
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-neutral-200">
              {sites.map((site) => (
                <tr
                  key={site.id}
                  className={site.is_active ? undefined : 'text-neutral-500'}
                >
                  <td className="px-6 py-4">
                    <div className="flex items-center gap-3">
                      <span className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-primary-50 text-primary-600">
                        <Building2 className="h-4 w-4" />
                      </span>
                      <span className="font-medium text-neutral-900">
                        {site.name}
                      </span>
                    </div>
                  </td>
                  <td className="px-4 py-4 text-neutral-700">
                    {site.address_full}
                    {site.access_notes && (
                      <p
                        className="mt-0.5 line-clamp-2 text-sm text-neutral-500"
                        title={site.access_notes}
                      >
                        Access: {site.access_notes}
                      </p>
                    )}
                  </td>
                  <td className="px-4 py-4">
                    <SiteAgreements
                      agreements={agreementsBySite.get(site.id) ?? []}
                      customerId={customerId}
                      siteId={site.id}
                      canAdd={site.is_active}
                    />
                  </td>
                  <td className="px-4 py-4">
                    {site.contact_name ? (
                      <>
                        <p className="whitespace-nowrap text-neutral-800">
                          {site.contact_name}
                        </p>
                        <p className="text-sm text-neutral-500">
                          {site.contact_phone || site.contact_email || ''}
                        </p>
                      </>
                    ) : (
                      <span className="text-neutral-400">—</span>
                    )}
                  </td>
                  <td className="px-4 py-4 text-right font-semibold text-neutral-900 tabular-nums">
                    {siteMonthly(site.id) > 0
                      ? formatDollars(siteMonthly(site.id), {
                          wholeUnits: true,
                        })
                      : '—'}
                  </td>
                  <td className="px-4 py-4">
                    <Badge
                      size="sm"
                      variant={site.is_active ? 'success' : 'neutral'}
                    >
                      {site.is_active ? 'Active' : 'Inactive'}
                    </Badge>
                  </td>
                  <td className="px-6 py-4">
                    <div className="flex justify-end gap-1 whitespace-nowrap">
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => setEditing(site)}
                        aria-label={`Edit ${site.name}`}
                        className="gap-1"
                      >
                        <Pencil className="h-3.5 w-3.5" />
                        Edit
                      </Button>
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleToggle(site)}
                        disabled={toggleActive.isPending}
                      >
                        {site.is_active ? 'Deactivate' : 'Reactivate'}
                      </Button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      {sites.length > 0 && (
        <footer className="flex flex-wrap items-center justify-between gap-2 border-t border-neutral-200 bg-neutral-50/60 px-6 py-4 text-[15px] font-medium text-neutral-800">
          <span>
            {activeCount} active {activeCount === 1 ? 'site' : 'sites'} ·{' '}
            {consolidatedBilling
              ? 'billed together on one invoice'
              : 'invoiced job by job'}
          </span>
          {totalMonthly > 0 && (
            <span>
              <span className="font-semibold text-neutral-900">
                {formatDollars(totalMonthly, { wholeUnits: true })}
              </span>{' '}
              <span className="font-normal text-neutral-500">
                per month, before tax
              </span>
            </span>
          )}
        </footer>
      )}

      <SiteFormModal
        customerId={customerId}
        isOpen={isAddOpen || editing !== null}
        site={editing}
        onClose={() => {
          setEditing(null);
          onAddOpenChange(false);
        }}
      />
    </section>
  );
}

interface SiteAgreementsProps {
  agreements: Agreement[];
  customerId: string;
  siteId: string;
  canAdd: boolean;
}

/** A site's agreements in the Sites table, or a link to add one. */
function SiteAgreements({
  agreements,
  customerId,
  siteId,
  canAdd,
}: SiteAgreementsProps) {
  if (agreements.length === 0) {
    return canAdd ? (
      <Link
        to="/agreements/new"
        state={{ customerId, siteId }}
        className="text-sm font-medium text-primary-600 hover:text-primary-700"
      >
        Add agreement
      </Link>
    ) : (
      <span className="text-neutral-400">—</span>
    );
  }
  return (
    <ul className="space-y-1">
      {agreements.map((a) => (
        <li key={a.id}>
          <Link
            to={`/agreements/${a.id}`}
            className="font-medium text-primary-600 hover:text-primary-700"
          >
            {a.title}
          </Link>
          <p className="text-sm text-neutral-600">
            {describeFrequency(a)}
            {a.status === 'paused' ? ' · paused' : ''}
          </p>
        </li>
      ))}
    </ul>
  );
}

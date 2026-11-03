import { useState } from 'react';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Building2, Loader2, Pencil } from 'lucide-react';
import { Badge, Button } from '@components/shared';
import { siteService, sitesQueryKey, type Site } from '@services/siteService';
import { getErrorMessage } from '@/utils/error';
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

  const sites = sitesQuery.data ?? [];
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
          <table className="w-full min-w-[760px] text-left text-[15px]">
            <thead className="border-b border-neutral-200 bg-neutral-50 text-sm font-semibold text-neutral-700">
              <tr>
                <th className="px-6 py-3">Site</th>
                <th className="px-4 py-3">Address</th>
                <th className="px-4 py-3">Site contact</th>
                <th className="px-4 py-3">Access notes</th>
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
                  </td>
                  <td className="px-4 py-4">
                    {site.contact_name ? (
                      <>
                        <p className="text-neutral-800">{site.contact_name}</p>
                        <p className="text-sm text-neutral-500">
                          {site.contact_phone || site.contact_email || ''}
                        </p>
                      </>
                    ) : (
                      <span className="text-neutral-400">—</span>
                    )}
                  </td>
                  <td className="max-w-[16rem] px-4 py-4 text-sm text-neutral-600">
                    <p className="line-clamp-2">{site.access_notes || '—'}</p>
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
                    <div className="flex justify-end gap-1">
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
        <footer className="border-t border-neutral-200 bg-neutral-50/60 px-6 py-4 text-[15px] font-medium text-neutral-800">
          {activeCount} active {activeCount === 1 ? 'site' : 'sites'} ·{' '}
          {consolidatedBilling
            ? 'billed together on one invoice'
            : 'invoiced job by job'}
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

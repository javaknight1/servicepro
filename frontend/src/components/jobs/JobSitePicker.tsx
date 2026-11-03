import { useQuery } from '@tanstack/react-query';
import { siteService, sitesQueryKey } from '@services/siteService';

interface JobSitePickerProps {
  /** The job's customer; no customer, no sites to pick. */
  customerId: string;
  value: string;
  onChange: (siteId: string) => void;
  disabled?: boolean;
}

/**
 * Picks one of the customer's sites for a job (T186). Inactive sites are
 * hidden unless the job already uses one.
 */
export function JobSitePicker({
  customerId,
  value,
  onChange,
  disabled,
}: JobSitePickerProps) {
  const sitesQuery = useQuery({
    queryKey: sitesQueryKey(customerId, true),
    queryFn: () => siteService.list(customerId, true),
    enabled: !!customerId,
  });
  const sites = (sitesQuery.data ?? []).filter(
    (s) => s.is_active || s.id === value
  );
  const selected = sites.find((s) => s.id === value);

  return (
    <div>
      <label
        htmlFor="site_id"
        className="mb-1 block text-sm font-medium text-neutral-700"
      >
        Site
      </label>
      <select
        id="site_id"
        name="site_id"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled || !customerId || sitesQuery.isPending}
        className="w-full rounded-lg border border-neutral-300 px-3 py-2 focus:border-primary-500 focus:ring-2 focus:ring-primary-500"
      >
        <option value="">
          {!customerId
            ? 'Select a customer first'
            : sites.length === 0 && !sitesQuery.isPending
              ? 'This customer has no sites'
              : 'No site'}
        </option>
        {sites.map((site) => (
          <option key={site.id} value={site.id}>
            {site.name}
            {site.is_active ? '' : ' (inactive)'}
          </option>
        ))}
      </select>
      {selected && (
        <div className="mt-2 text-sm text-neutral-500">
          <p>{selected.address_full}</p>
          {selected.access_notes && (
            <p className="mt-0.5">Access: {selected.access_notes}</p>
          )}
          {selected.contact_name && (
            <p className="mt-0.5">
              Site contact: {selected.contact_name}
              {selected.contact_phone ? ` · ${selected.contact_phone}` : ''}
            </p>
          )}
        </div>
      )}
      {sitesQuery.isError && (
        <p className="mt-1 text-sm text-error-600">Could not load the sites.</p>
      )}
    </div>
  );
}

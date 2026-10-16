import { useTenantStore } from '@store';
import { normalizeTimeZone } from '@/utils/epoch';

/**
 * The current tenant's configured IANA timezone (Settings → Business Hours),
 * or undefined to fall back to the browser's local timezone.
 */
export function useTenantTimezone(): string | undefined {
  const timezone = useTenantStore(
    (state) => state.currentTenant?.settings?.timezone
  );
  return normalizeTimeZone(timezone);
}

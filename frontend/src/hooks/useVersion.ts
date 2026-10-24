import { useQuery } from '@tanstack/react-query';
import { getVersion } from '@services/versionService';

/**
 * The deployed API's build identifiers, for the footer build stamp.
 *
 * `staleTime: Infinity` because the answer only changes with a deploy, and a
 * tab left open across one is not worth polling for. `retry: false` keeps a
 * down API from firing repeated failed requests on every page load; the stamp
 * then shows the web version alone.
 */
export function useVersion() {
  return useQuery({
    queryKey: ['version'],
    queryFn: getVersion,
    staleTime: Infinity,
    retry: false,
    refetchOnWindowFocus: false,
  });
}

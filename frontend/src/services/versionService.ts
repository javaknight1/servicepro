import api from './api';
import { VERSION_URL } from '@config/endpoints';

/** GET /version body (`version.Info`). */
export interface VersionInfo {
  git_sha?: string;
  app_version?: string;
}

/**
 * GET /version: the deployed API's build identity. The path sits outside the
 * "/api" base, so the instance baseURL is cleared for this one call. A
 * non-object body (e.g. the SPA's index.html when nothing proxies /version)
 * counts as "unknown" rather than a version.
 */
export async function getVersion(): Promise<VersionInfo> {
  const { data } = await api.get<unknown>(VERSION_URL, { baseURL: '' });
  if (!data || typeof data !== 'object') {
    return {};
  }
  const body = data as Record<string, unknown>;
  return {
    git_sha: typeof body.git_sha === 'string' ? body.git_sha : undefined,
    app_version:
      typeof body.app_version === 'string' ? body.app_version : undefined,
  };
}

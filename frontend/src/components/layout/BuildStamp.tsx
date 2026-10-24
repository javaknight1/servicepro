import { useVersion } from '@hooks/useVersion';
import { cn } from '@utils/cn';

// The API reports these when its env vars are unset (backend
// internal/version). They mean "unknown", not a version.
const API_UNKNOWN_VALUES = new Set(['', 'dev', 'unknown']);

function known(value: string | undefined): string | undefined {
  return value && !API_UNKNOWN_VALUES.has(value) ? value : undefined;
}

/** Short SHA for the tooltip; the full 40 chars are noise at a glance. */
function shortSha(sha: string | undefined): string | undefined {
  return sha ? sha.slice(0, 7) : undefined;
}

/** "V0.1.0" when known, an em dash when not. Never renders a bare "V". */
function stamp(version: string | undefined): string {
  return version ? `V${version}` : '—';
}

function withSha(label: string, sha: string | undefined): string {
  const short = shortSha(sha);
  return short ? `${label} (${short})` : label;
}

interface BuildStampProps {
  className?: string;
}

/**
 * "WEB V<x> · API V<y>" so a bug report can name the exact build of both
 * halves without opening devtools (League Looper's footer stamp).
 *
 * The web half is baked in at build time by release-deploy.yml
 * (VITE_APP_VERSION / VITE_GIT_SHA); the API half comes from GET /version.
 * They ship from the same tag as separate jobs, so a mismatch is normal for
 * the minute between them and a real signal after: it is flagged. The whole
 * group is omitted when neither is known (local builds with no API).
 */
export function BuildStamp({ className }: BuildStampProps) {
  const { data: apiBuild } = useVersion();

  const webVersion = known(import.meta.env.VITE_APP_VERSION);
  const webSha = known(import.meta.env.VITE_GIT_SHA);
  const apiVersion = known(apiBuild?.app_version);
  const apiSha = known(apiBuild?.git_sha);

  if (!webVersion && !apiVersion) {
    return null;
  }

  const drift = Boolean(webVersion && apiVersion && webVersion !== apiVersion);
  const webTitle = withSha(`Web build ${stamp(webVersion)}`, webSha);
  const apiTitle = drift
    ? `${withSha(`API is serving ${stamp(apiVersion)}`, apiSha)}, which differs from this web build ${stamp(webVersion)}`
    : withSha(`API build ${stamp(apiVersion)}`, apiSha);

  return (
    <p
      data-testid="build-stamp"
      className={cn(
        'flex items-center justify-center gap-x-2 font-mono text-xs uppercase tracking-wider text-neutral-400',
        className
      )}
    >
      <span title={webTitle}>WEB {stamp(webVersion)}</span>
      <span aria-hidden="true">·</span>
      <span
        title={apiTitle}
        data-drift={drift ? 'true' : undefined}
        className={cn(drift && 'font-semibold text-warning-600')}
      >
        API {stamp(apiVersion)}
      </span>
    </p>
  );
}

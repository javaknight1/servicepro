/**
 * Public support contact. Configure with VITE_SUPPORT_EMAIL at build time.
 */
export const SUPPORT_EMAIL: string =
  import.meta.env.VITE_SUPPORT_EMAIL || 'support@sharkfins.xyz';

/** Build a mailto: URL with an optional subject and body. */
export function buildMailto(
  to: string,
  { subject, body }: { subject?: string; body?: string } = {}
): string {
  const params = new URLSearchParams();
  if (subject) params.set('subject', subject);
  if (body) params.set('body', body);
  // mailto expects %20 for spaces, not '+'
  const query = params.toString().replace(/\+/g, '%20');
  return `mailto:${to}${query ? `?${query}` : ''}`;
}

/** Fixed date of the current (unreviewed) legal drafts. */
export const LEGAL_DRAFT_DATE = 'October 16, 2026';

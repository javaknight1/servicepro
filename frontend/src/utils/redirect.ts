/**
 * Full-page navigation to another site (e.g. Stripe-hosted onboarding).
 * Kept in its own module so tests can replace it.
 */
export function redirectTo(url: string): void {
  window.location.assign(url);
}

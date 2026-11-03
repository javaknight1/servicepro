import { Check } from 'lucide-react';
import { Card, CardContent, Button, Badge } from '@components/shared';
import type { MembershipTier, BillingCycle } from '@/types/membership';
import {
  FOUNDING_DISCOUNT_CENTS,
  isUnlimited,
  usersLabel,
} from '@/types/membership';
import { formatCents } from '@/utils/money';

interface TierCardProps {
  tier: MembershipTier;
  isCurrentTier?: boolean;
  onSelect?: (tier: MembershipTier) => void;
  disabled?: boolean;
  showSelectButton?: boolean;
  highlighted?: boolean;
  billingCycle?: BillingCycle;
  /** Button text (default "Select Plan"). */
  selectLabel?: string;
  /** Show the founding price ($40 off Team monthly, for life). */
  foundingOffer?: boolean;
  /** Small label on top of the card, e.g. "Most teams". */
  ribbon?: string;
}

/** Bullets for a plan: its highlights, or the legacy count limits. */
function bulletsFor(tier: MembershipTier): string[] {
  const f = tier.features;
  if (f.highlights && f.highlights.length > 0) return f.highlights;
  const limit = (v: number | undefined, label: string, unit = '') =>
    v === undefined
      ? null
      : `${isUnlimited(v) ? 'Unlimited' : `${v}${unit}`} ${label}`;
  return [
    limit(f.customers, 'customers'),
    limit(f.jobs_per_month, 'jobs / month'),
    limit(f.quotes_per_month, 'quotes / month'),
    limit(f.storage_gb, 'storage', ' GB'),
    f.advanced_reporting ? 'Advanced reporting' : null,
  ].filter((b): b is string => b !== null);
}

export function TierCard({
  tier,
  isCurrentTier = false,
  onSelect,
  disabled = false,
  showSelectButton = true,
  highlighted = false,
  billingCycle = 'monthly',
  selectLabel = 'Select Plan',
  foundingOffer = false,
  ribbon,
}: TierCardProps) {
  const isFree = tier.monthly_price_cents === 0;
  const isAnnual = billingCycle === 'annual';
  const showFounding = foundingOffer && !isAnnual && !isFree;

  const listPrice = isAnnual
    ? tier.annual_price_cents
    : tier.monthly_price_cents;
  const price = showFounding
    ? tier.monthly_price_cents - FOUNDING_DISCOUNT_CENTS
    : listPrice;

  return (
    <Card
      variant="elevated"
      padding="lg"
      className={`relative ${highlighted ? 'ring-2 ring-primary-500' : ''} ${
        isCurrentTier ? 'bg-neutral-50' : ''
      }`}
      data-testid={`tier-card-${tier.name}`}
    >
      {isCurrentTier ? (
        <div className="absolute -top-3 left-1/2 z-10 -translate-x-1/2">
          <Badge variant="success">Current Plan</Badge>
        </div>
      ) : (
        ribbon && (
          <div className="absolute -top-3 left-1/2 z-10 -translate-x-1/2">
            <Badge variant="info">{ribbon}</Badge>
          </div>
        )
      )}

      <CardContent className="pt-4">
        <h3 className="text-lg font-semibold text-neutral-900">
          {tier.display_name}
        </h3>
        {tier.description && (
          <p className="mt-1 text-sm text-neutral-600">{tier.description}</p>
        )}

        <div className="mt-4">
          <span className="text-4xl font-bold text-neutral-900">
            {isFree ? 'Free' : formatCents(price)}
          </span>
          {!isFree && (
            <span className="text-sm text-neutral-500">
              {isAnnual ? '/yr' : '/mo'}
            </span>
          )}
          {showFounding && (
            <p className="mt-1 text-xs text-neutral-500">
              <span className="line-through">{formatCents(listPrice)}</span>{' '}
              <span className="font-medium text-green-700">
                Founding price, for life
              </span>
            </p>
          )}
          {!isFree && isAnnual && (
            <p className="mt-1 text-xs text-green-700">
              Two months free vs monthly
            </p>
          )}
          <p className="mt-1 text-sm text-neutral-600">
            {usersLabel(tier.features.team_members)}
          </p>
        </div>

        <ul className="mb-6 mt-4 space-y-2">
          {bulletsFor(tier).map((b) => (
            <li key={b} className="flex items-start text-sm text-neutral-700">
              <Check className="mr-2 mt-0.5 h-4 w-4 shrink-0 text-primary-600" />
              {b}
            </li>
          ))}
        </ul>

        {showSelectButton && (
          <Button
            variant={highlighted && !isCurrentTier ? 'primary' : 'outline'}
            fullWidth
            disabled={disabled || isCurrentTier}
            onClick={() => onSelect?.(tier)}
          >
            {isCurrentTier ? 'Current Plan' : selectLabel}
          </Button>
        )}
      </CardContent>
    </Card>
  );
}

export default TierCard;

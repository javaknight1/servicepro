import { Check, X } from 'lucide-react';
import { Card, CardContent, Button, Badge } from '@components/shared';
import type {
  MembershipTier,
  TierFeatures,
  BillingCycle,
} from '@/types/membership';
import { formatPrice, isUnlimited } from '@/types/membership';

interface TierCardProps {
  tier: MembershipTier;
  isCurrentTier?: boolean;
  onSelect?: (tier: MembershipTier) => void;
  disabled?: boolean;
  showSelectButton?: boolean;
  highlighted?: boolean;
  billingCycle?: BillingCycle;
}

const featureLabels: Record<keyof TierFeatures, string> = {
  team_members: 'Team Members',
  customers: 'Customers',
  jobs_per_month: 'Jobs / Month',
  quotes_per_month: 'Quotes / Month',
  storage_gb: 'Storage',
  advanced_reporting: 'Advanced Reporting',
  api_access: 'API Access',
  webhooks: 'Webhooks',
  custom_branding: 'Custom Branding',
};

export function TierCard({
  tier,
  isCurrentTier = false,
  onSelect,
  disabled = false,
  showSelectButton = true,
  highlighted = false,
  billingCycle = 'monthly',
}: TierCardProps) {
  const isFree = tier.monthly_price_cents === 0;
  const isAnnual = billingCycle === 'annual';

  // Calculate the price to display
  const displayPrice = isFree
    ? 0
    : isAnnual
      ? Math.round(tier.annual_price_cents / 12) // Monthly equivalent for annual
      : tier.monthly_price_cents;

  return (
    <Card
      variant="elevated"
      padding="lg"
      className={`relative ${highlighted ? 'ring-2 ring-primary-500 shadow-lg' : ''} ${
        isCurrentTier ? 'bg-neutral-50' : ''
      }`}
    >
      {highlighted && !isCurrentTier && (
        <div className="absolute -top-3 left-1/2 transform -translate-x-1/2 z-10">
          <Badge variant="info">Popular</Badge>
        </div>
      )}
      {isCurrentTier && (
        <div className="absolute -top-3 left-1/2 transform -translate-x-1/2 z-10">
          <Badge variant="success">Current Plan</Badge>
        </div>
      )}

      <CardContent className="pt-6">
        <div className="text-center mb-6">
          <h3 className="text-xl font-bold text-neutral-900">
            {tier.display_name}
          </h3>
          {tier.description && (
            <p className="text-sm text-neutral-600 mt-1">{tier.description}</p>
          )}
        </div>

        <div className="text-center mb-6">
          <span className="text-4xl font-bold text-neutral-900">
            {isFree ? 'Free' : formatPrice(displayPrice)}
          </span>
          {!isFree && <span className="text-neutral-500 text-sm">/month</span>}
          {!isFree && isAnnual && (
            <p className="text-xs text-green-600 mt-1">
              {formatPrice(tier.annual_price_cents)} billed annually
            </p>
          )}
        </div>

        <ul className="space-y-3 mb-6">
          {/* Numeric features */}
          <FeatureItem
            label={featureLabels.team_members}
            value={tier.features.team_members}
            type="number"
          />
          <FeatureItem
            label={featureLabels.customers}
            value={tier.features.customers}
            type="number"
          />
          <FeatureItem
            label={featureLabels.jobs_per_month}
            value={tier.features.jobs_per_month}
            type="number"
          />
          <FeatureItem
            label={featureLabels.quotes_per_month}
            value={tier.features.quotes_per_month}
            type="number"
          />
          <FeatureItem
            label={featureLabels.storage_gb}
            value={tier.features.storage_gb}
            type="number"
            unit="GB"
          />

          {/* Boolean features */}
          <FeatureItem
            label={featureLabels.advanced_reporting}
            value={tier.features.advanced_reporting}
            type="boolean"
          />
          <FeatureItem
            label={featureLabels.api_access}
            value={tier.features.api_access}
            type="boolean"
          />
          <FeatureItem
            label={featureLabels.webhooks}
            value={tier.features.webhooks}
            type="boolean"
          />
          <FeatureItem
            label={featureLabels.custom_branding}
            value={tier.features.custom_branding}
            type="boolean"
          />
        </ul>

        {showSelectButton && (
          <Button
            variant={
              isCurrentTier ? 'outline' : highlighted ? 'primary' : 'outline'
            }
            fullWidth
            disabled={disabled || isCurrentTier}
            onClick={() => onSelect?.(tier)}
          >
            {isCurrentTier ? 'Current Plan' : 'Select Plan'}
          </Button>
        )}
      </CardContent>
    </Card>
  );
}

interface FeatureItemProps {
  label: string;
  value: number | boolean;
  type: 'number' | 'boolean';
  unit?: string;
}

function FeatureItem({ label, value, type, unit }: FeatureItemProps) {
  if (type === 'boolean') {
    const enabled = value as boolean;
    return (
      <li className="flex items-center">
        {enabled ? (
          <Check className="h-5 w-5 text-green-500 mr-3 flex-shrink-0" />
        ) : (
          <X className="h-5 w-5 text-neutral-300 mr-3 flex-shrink-0" />
        )}
        <span className={enabled ? 'text-neutral-700' : 'text-neutral-400'}>
          {label}
        </span>
      </li>
    );
  }

  const numValue = value as number;
  const displayValue = isUnlimited(numValue)
    ? 'Unlimited'
    : `${numValue}${unit ? ` ${unit}` : ''}`;

  return (
    <li className="flex items-center">
      <Check className="h-5 w-5 text-green-500 mr-3 flex-shrink-0" />
      <span className="text-neutral-700">
        <strong>{displayValue}</strong> {label}
      </span>
    </li>
  );
}

export default TierCard;

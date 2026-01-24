import { Check, X } from 'lucide-react';
import { Button, Badge } from '@components/shared';
import type { MembershipTier, BillingCycle } from '@/types/membership';
import {
  formatPrice,
  isUnlimited,
  featureDisplayConfig,
} from '@/types/membership';

interface TierComparisonTableProps {
  tiers: MembershipTier[];
  currentTierId?: string;
  onSelectTier?: (tier: MembershipTier) => void;
  showSelectButtons?: boolean;
  billingCycle?: BillingCycle;
}

export function TierComparisonTable({
  tiers,
  currentTierId,
  onSelectTier,
  showSelectButtons = true,
  billingCycle = 'monthly',
}: TierComparisonTableProps) {
  // Sort tiers by sort_order
  const sortedTiers = [...tiers].sort((a, b) => a.sort_order - b.sort_order);
  const isAnnual = billingCycle === 'annual';

  return (
    <div className="overflow-x-auto">
      <table className="w-full border-collapse">
        <thead>
          <tr>
            <th className="text-left p-4 bg-neutral-50 border-b border-neutral-200 min-w-[200px]">
              <span className="text-sm font-medium text-neutral-600">
                Features
              </span>
            </th>
            {sortedTiers.map((tier) => {
              const isFree = tier.monthly_price_cents === 0;
              const displayPrice = isFree
                ? 0
                : isAnnual
                  ? Math.round(tier.annual_price_cents / 12)
                  : tier.monthly_price_cents;

              return (
                <th
                  key={tier.id}
                  className={`text-center p-4 bg-neutral-50 border-b border-neutral-200 min-w-[180px] ${
                    tier.name === 'basic' ? 'bg-primary-50' : ''
                  }`}
                >
                  <div className="space-y-2">
                    {tier.id === currentTierId && (
                      <Badge variant="success" className="mb-2">
                        Current
                      </Badge>
                    )}
                    {tier.name === 'basic' && tier.id !== currentTierId && (
                      <Badge variant="info" className="mb-2">
                        Popular
                      </Badge>
                    )}
                    <div className="text-lg font-bold text-neutral-900">
                      {tier.display_name}
                    </div>
                    <div className="text-2xl font-bold text-neutral-900">
                      {isFree ? 'Free' : formatPrice(displayPrice)}
                      {!isFree && (
                        <span className="text-sm font-normal text-neutral-500">
                          /mo
                        </span>
                      )}
                    </div>
                    {!isFree && isAnnual && (
                      <div className="text-xs text-green-600">
                        {formatPrice(tier.annual_price_cents)}/yr
                      </div>
                    )}
                  </div>
                </th>
              );
            })}
          </tr>
        </thead>
        <tbody>
          {featureDisplayConfig.map((feature) => (
            <tr key={feature.key} className="border-b border-neutral-100">
              <td className="p-4 text-sm text-neutral-700">{feature.label}</td>
              {sortedTiers.map((tier) => (
                <td
                  key={tier.id}
                  className={`p-4 text-center ${
                    tier.name === 'basic' ? 'bg-primary-50/50' : ''
                  }`}
                >
                  <FeatureValue
                    value={tier.features[feature.key]}
                    type={feature.type}
                    unit={feature.unit}
                  />
                </td>
              ))}
            </tr>
          ))}

          {showSelectButtons && (
            <tr>
              <td className="p-4"></td>
              {sortedTiers.map((tier) => (
                <td
                  key={tier.id}
                  className={`p-4 text-center ${
                    tier.name === 'basic' ? 'bg-primary-50/50' : ''
                  }`}
                >
                  <Button
                    variant={
                      tier.id === currentTierId
                        ? 'outline'
                        : tier.name === 'basic'
                          ? 'primary'
                          : 'outline'
                    }
                    size="sm"
                    disabled={tier.id === currentTierId}
                    onClick={() => onSelectTier?.(tier)}
                  >
                    {tier.id === currentTierId ? 'Current' : 'Select'}
                  </Button>
                </td>
              ))}
            </tr>
          )}
        </tbody>
      </table>
    </div>
  );
}

interface FeatureValueProps {
  value: number | boolean;
  type: 'number' | 'boolean';
  unit?: string;
}

function FeatureValue({ value, type, unit }: FeatureValueProps) {
  if (type === 'boolean') {
    const enabled = value as boolean;
    return enabled ? (
      <Check className="h-5 w-5 text-green-500 mx-auto" />
    ) : (
      <X className="h-5 w-5 text-neutral-300 mx-auto" />
    );
  }

  const numValue = value as number;
  if (isUnlimited(numValue)) {
    return (
      <span className="text-sm font-medium text-neutral-900">Unlimited</span>
    );
  }

  return (
    <span className="text-sm font-medium text-neutral-900">
      {numValue}
      {unit ? ` ${unit}` : ''}
    </span>
  );
}

export default TierComparisonTable;

import { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { CreditCard, ExternalLink, Crown } from 'lucide-react';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  Button,
  Badge,
} from '@components/shared';
import { useMembershipStore, useTenantStore } from '@store';
import { formatPrice } from '@/types/membership';

export function MembershipTab() {
  const navigate = useNavigate();
  const { currentTenant } = useTenantStore();
  const { currentMembership, isLoading, error, loadMembership } =
    useMembershipStore();

  useEffect(() => {
    if (currentTenant?.id) {
      loadMembership(currentTenant.id);
    }
  }, [currentTenant?.id, loadMembership]);

  if (isLoading) {
    return (
      <Card variant="elevated" padding="lg">
        <CardContent>
          <div className="animate-pulse space-y-4">
            <div className="h-8 bg-neutral-200 rounded w-1/3"></div>
            <div className="h-4 bg-neutral-200 rounded w-1/2"></div>
            <div className="h-32 bg-neutral-200 rounded"></div>
          </div>
        </CardContent>
      </Card>
    );
  }

  const tier = currentMembership?.tier;
  const subscription = currentMembership?.subscription;

  const getBadgeVariant = (tierName: string) => {
    switch (tierName) {
      case 'pro':
        return 'info';
      case 'basic':
        return 'success';
      default:
        return 'neutral';
    }
  };

  return (
    <div className="space-y-6">
      {/* Current Plan */}
      <Card variant="elevated" padding="lg">
        <CardHeader>
          <CardTitle>Current Plan</CardTitle>
          <CardDescription>
            Your organization's membership tier and features
          </CardDescription>
        </CardHeader>
        <CardContent>
          {error ? (
            <div className="text-red-600 py-4">
              Failed to load membership information. Please try again later.
            </div>
          ) : !tier ? (
            <div className="text-center py-8">
              <Crown className="h-12 w-12 text-neutral-400 mx-auto mb-4" />
              <p className="text-neutral-600 mb-4">
                No active membership found
              </p>
              <Button
                variant="primary"
                onClick={() =>
                  navigate('/settings/organization/membership/change')
                }
              >
                Choose a Plan
              </Button>
            </div>
          ) : (
            <div className="space-y-6">
              <div className="flex items-center space-x-4">
                <div className="h-12 w-12 bg-primary-100 rounded-lg flex items-center justify-center">
                  <Crown className="h-6 w-6 text-primary-600" />
                </div>
                <div>
                  <div className="flex items-center space-x-2">
                    <h3 className="text-xl font-semibold text-neutral-900">
                      {tier.display_name}
                    </h3>
                    <Badge variant={getBadgeVariant(tier.name)}>
                      {tier.name.charAt(0).toUpperCase() + tier.name.slice(1)}
                    </Badge>
                    {subscription?.is_legacy && (
                      <Badge variant="warning">Legacy</Badge>
                    )}
                  </div>
                  <p className="text-sm text-neutral-600">
                    {subscription?.is_legacy
                      ? 'This plan is no longer available to new customers'
                      : tier.description || 'Your current membership plan'}
                  </p>
                </div>
              </div>

              <div className="bg-neutral-50 rounded-lg p-4">
                <div className="grid grid-cols-2 gap-4 text-sm">
                  <div>
                    <p className="text-neutral-500">Current Price</p>
                    <p className="font-medium text-neutral-900">
                      {subscription?.price_cents === 0
                        ? 'Free'
                        : subscription?.billing_cycle === 'annual'
                          ? `${formatPrice(subscription?.price_cents || 0)}/year`
                          : `${formatPrice(subscription?.price_cents || 0)}/month`}
                    </p>
                  </div>
                  {subscription && (
                    <>
                      <div>
                        <p className="text-neutral-500">Status</p>
                        <p className="font-medium text-neutral-900 capitalize">
                          {subscription.status}
                        </p>
                      </div>
                      <div>
                        <p className="text-neutral-500">Billing Cycle</p>
                        <p className="font-medium text-neutral-900 capitalize">
                          {subscription.billing_cycle}
                        </p>
                      </div>
                      <div>
                        <p className="text-neutral-500">Started</p>
                        <p className="font-medium text-neutral-900">
                          {new Date(
                            subscription.started_at
                          ).toLocaleDateString()}
                        </p>
                      </div>
                      {subscription.current_period_end && (
                        <div>
                          <p className="text-neutral-500">Renews</p>
                          <p className="font-medium text-neutral-900">
                            {new Date(
                              subscription.current_period_end
                            ).toLocaleDateString()}
                          </p>
                        </div>
                      )}
                    </>
                  )}
                </div>
                {/* Savings message for monthly billing */}
                {subscription?.billing_cycle === 'monthly' &&
                  tier.monthly_price_cents > 0 && (
                    <div className="mt-4 pt-4 border-t border-neutral-200">
                      <p className="text-sm text-green-600">
                        💡 Switch to annual billing and save{' '}
                        <span className="font-semibold">
                          {formatPrice(
                            tier.monthly_price_cents * 12 -
                              tier.annual_price_cents
                          )}
                        </span>{' '}
                        per year (17% off)
                      </p>
                    </div>
                  )}
              </div>

              <div className="flex space-x-3">
                <Button
                  variant="primary"
                  onClick={() =>
                    navigate('/settings/organization/membership/change')
                  }
                >
                  Change Plan
                </Button>
                <Button variant="outline" onClick={() => navigate('/pricing')}>
                  <ExternalLink className="h-4 w-4 mr-2" />
                  View All Plans
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Payment Method */}
      <Card variant="elevated" padding="lg">
        <CardHeader>
          <CardTitle>Payment Method</CardTitle>
          <CardDescription>
            Manage your billing and payment information
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-between py-4">
            <div className="flex items-center space-x-4">
              <div className="h-10 w-16 bg-neutral-100 rounded flex items-center justify-center">
                <CreditCard className="h-6 w-6 text-neutral-400" />
              </div>
              <div>
                <p className="font-medium text-neutral-900">
                  Visa ending in 4242
                </p>
                <p className="text-sm text-neutral-500">Expires 12/2025</p>
              </div>
            </div>
            <Button variant="outline" disabled>
              Update
            </Button>
          </div>
          <p className="text-xs text-neutral-400 mt-2">
            Payment method management is coming soon.
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

export default MembershipTab;

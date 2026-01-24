import { useEffect, useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { DashboardLayout } from '@components/layout';
import { Card, CardContent, Button } from '@components/shared';
import { TierCard, BillingToggle } from '@components/membership';
import { useMembershipStore, useTenantStore } from '@store';
import type { MembershipTier, BillingCycle } from '@/types/membership';
import { ArrowLeft, AlertTriangle } from 'lucide-react';

export function ChangeMembershipPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { currentTenant } = useTenantStore();
  const {
    tiers,
    currentMembership,
    isLoading,
    error,
    loadTiers,
    loadMembership,
    updateMembership,
    clearError,
  } = useMembershipStore();

  const [isUpdating, setIsUpdating] = useState(false);
  const [updateError, setUpdateError] = useState<string | null>(null);
  const [billingCycle, setBillingCycle] = useState<BillingCycle>('monthly');

  // Get pre-selected tier from location state (if coming from pricing page)
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const preSelectedTierId = (location.state as any)?.selectedTierId;

  useEffect(() => {
    loadTiers();
  }, [loadTiers]);

  useEffect(() => {
    if (currentTenant?.id) {
      loadMembership(currentTenant.id);
    }
  }, [currentTenant?.id, loadMembership]);

  const handleSelectTier = async (tier: MembershipTier) => {
    if (!currentTenant?.id) return;

    setIsUpdating(true);
    setUpdateError(null);
    clearError();

    try {
      await updateMembership(currentTenant.id, tier.id, billingCycle);
      // Navigate back to org settings on success
      navigate('/settings/organization', {
        state: { activeTab: 'membership', success: true },
      });
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (err: any) {
      setUpdateError(
        err.response?.data?.message || 'Failed to update membership'
      );
    } finally {
      setIsUpdating(false);
    }
  };

  const currentTierId = currentMembership?.tier?.id;
  const isLegacyPlan = currentMembership?.subscription?.is_legacy ?? false;

  if (!currentTenant) {
    return (
      <DashboardLayout>
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <Card variant="elevated" padding="lg">
            <CardContent>
              <div className="text-center py-8">
                <p className="text-neutral-600 mb-4">
                  Please select an organization first.
                </p>
                <Button
                  variant="primary"
                  onClick={() => navigate('/settings/organization')}
                >
                  Go to Organization Settings
                </Button>
              </div>
            </CardContent>
          </Card>
        </div>
      </DashboardLayout>
    );
  }

  return (
    <DashboardLayout>
      <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="mb-8">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate('/settings/organization')}
            className="mb-4"
          >
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Settings
          </Button>
          <h1 className="text-3xl font-bold text-neutral-900">
            Change Membership Plan
          </h1>
          <p className="text-neutral-600 mt-2">
            Select a new plan for{' '}
            <span className="font-medium">{currentTenant.name}</span>
          </p>
        </div>

        {/* Error Message */}
        {(error || updateError) && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700">
            {error || updateError}
          </div>
        )}

        {/* Loading State */}
        {isLoading ? (
          <div className="text-center py-12">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto"></div>
            <p className="text-neutral-600 mt-4">Loading plans...</p>
          </div>
        ) : (
          <>
            {/* Legacy Plan Warning */}
            {isLegacyPlan && (
              <div className="mb-6 p-4 bg-amber-50 border border-amber-200 rounded-lg flex items-start space-x-3">
                <AlertTriangle className="h-5 w-5 text-amber-600 flex-shrink-0 mt-0.5" />
                <div>
                  <p className="text-amber-800 font-medium">
                    You're on a legacy plan
                  </p>
                  <p className="text-amber-700 text-sm mt-1">
                    Your current plan ({currentMembership?.tier?.display_name})
                    is no longer available to new customers. If you change your
                    plan, you will not be able to return to this pricing.
                  </p>
                </div>
              </div>
            )}

            {/* Billing Toggle */}
            <div className="flex justify-center mb-8">
              <BillingToggle value={billingCycle} onChange={setBillingCycle} />
            </div>

            {/* Tier Cards */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6">
              {tiers.map((tier) => (
                <TierCard
                  key={tier.id}
                  tier={tier}
                  isCurrentTier={!isLegacyPlan && tier.id === currentTierId}
                  onSelect={handleSelectTier}
                  disabled={isUpdating}
                  highlighted={
                    tier.name === 'basic' ||
                    (preSelectedTierId && tier.id === preSelectedTierId)
                  }
                  billingCycle={billingCycle}
                />
              ))}
            </div>

            {/* Help Text */}
            <div className="mt-8 text-center">
              <p className="text-sm text-neutral-500">
                Need help choosing?{' '}
                <a
                  href="/pricing"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-primary-600 hover:text-primary-700"
                >
                  View full comparison
                </a>
              </p>
            </div>
          </>
        )}
      </div>
    </DashboardLayout>
  );
}

export default ChangeMembershipPage;

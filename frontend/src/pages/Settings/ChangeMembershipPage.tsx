import { useEffect, useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { useStripe, useElements, CardElement } from '@stripe/react-stripe-js';
import { DashboardLayout } from '@components/layout';
import { Card, CardContent, Button, Modal } from '@components/shared';
import { TierCard, BillingToggle } from '@components/membership';
import { useMembershipStore, useTenantStore, useBillingStore } from '@store';
import { useStripeContext } from '@/providers/StripeProvider';
import { billingApi } from '@/services';
import type { MembershipTier, BillingCycle } from '@/types/membership';

/** Expected shape of location.state when navigating to this page */
interface ChangeMembershipLocationState {
  selectedTierId?: string;
}
import {
  ArrowLeft,
  AlertTriangle,
  CreditCard,
  Loader2,
  AlertCircle,
} from 'lucide-react';

// Separate component for adding a card (needs Elements context)
interface AddCardModalContentProps {
  tenantId: string;
  onSuccess: () => void;
  onCancel: () => void;
}

function AddCardModalContent({
  tenantId,
  onSuccess,
  onCancel,
}: AddCardModalContentProps) {
  const stripe = useStripe();
  const elements = useElements();
  const { addPaymentMethod } = useBillingStore();

  const [isProcessing, setIsProcessing] = useState(false);
  const [addError, setAddError] = useState<string | null>(null);

  const handleAddCard = async () => {
    if (!stripe || !elements) {
      return;
    }

    setIsProcessing(true);
    setAddError(null);

    try {
      // Create setup intent
      const setupResponse = await billingApi.createSetupIntent(tenantId);
      const clientSecret = setupResponse.data.client_secret;

      // Confirm the setup intent with the card element
      const cardElement = elements.getElement(CardElement);
      if (!cardElement) {
        throw new Error('Card element not found');
      }

      const { setupIntent, error } = await stripe.confirmCardSetup(
        clientSecret,
        {
          payment_method: {
            card: cardElement,
          },
        }
      );

      if (error) {
        throw new Error(error.message || 'Failed to add payment method');
      }

      if (!setupIntent?.payment_method) {
        throw new Error('No payment method returned');
      }

      // Add the payment method to our backend (set as default since it's for subscription)
      await addPaymentMethod(
        tenantId,
        setupIntent.payment_method as string,
        true // Always set as default for subscription payment
      );

      cardElement.clear();
      onSuccess();
    } catch (error) {
      const message =
        error instanceof Error ? error.message : 'Failed to add card';
      setAddError(message);
    } finally {
      setIsProcessing(false);
    }
  };

  return (
    <div className="space-y-4">
      <div className="flex items-center space-x-3 text-amber-700 bg-amber-50 p-3 rounded-lg">
        <CreditCard className="h-5 w-5 flex-shrink-0" />
        <p className="text-sm">
          A payment method is required for paid plans. Your card will be charged
          at the start of each billing cycle.
        </p>
      </div>

      {addError && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg flex items-center text-red-700 text-sm">
          <AlertCircle className="h-4 w-4 mr-2 flex-shrink-0" />
          {addError}
        </div>
      )}

      <div className="p-3 border border-neutral-300 rounded-lg">
        <CardElement
          options={{
            style: {
              base: {
                fontSize: '16px',
                color: '#1f2937',
                '::placeholder': {
                  color: '#9ca3af',
                },
              },
              invalid: {
                color: '#dc2626',
              },
            },
          }}
        />
      </div>

      <div className="flex justify-end space-x-3 pt-2">
        <Button variant="outline" onClick={onCancel} disabled={isProcessing}>
          Cancel
        </Button>
        <Button
          variant="primary"
          onClick={handleAddCard}
          disabled={isProcessing || !stripe}
        >
          {isProcessing ? (
            <>
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              Processing...
            </>
          ) : (
            'Add Card & Continue'
          )}
        </Button>
      </div>
    </div>
  );
}

export function ChangeMembershipPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { currentTenant } = useTenantStore();
  const { isAvailable: isStripeAvailable } = useStripeContext();
  const {
    tiers,
    currentMembership,
    isLoading,
    error,
    loadTiers,
    loadMembership,
  } = useMembershipStore();
  const { paymentMethods, loadPaymentMethods, loadConfig, isStripeEnabled } =
    useBillingStore();

  const [updateError, setUpdateError] = useState<string | null>(null);
  const [billingCycle, setBillingCycle] = useState<BillingCycle>('monthly');
  const [showPaymentModal, setShowPaymentModal] = useState(false);
  const [pendingTier, setPendingTier] = useState<MembershipTier | null>(null);

  // Get pre-selected tier from location state (if coming from pricing page)
  const locationState = location.state as ChangeMembershipLocationState | null;
  const preSelectedTierId = locationState?.selectedTierId;

  useEffect(() => {
    loadTiers();
    loadConfig();
  }, [loadTiers, loadConfig]);

  useEffect(() => {
    if (currentTenant?.id) {
      loadMembership(currentTenant.id);
      if (isStripeEnabled) {
        loadPaymentMethods(currentTenant.id);
      }
    }
  }, [currentTenant?.id, isStripeEnabled, loadMembership, loadPaymentMethods]);

  const hasPaymentMethod = paymentMethods.length > 0;

  const handleSelectTier = async (tier: MembershipTier) => {
    if (!currentTenant?.id) return;

    // Check if this is a paid tier
    const isPaidTier =
      tier.monthly_price_cents > 0 || tier.annual_price_cents > 0;

    // If paid tier, check payment requirements
    if (isPaidTier) {
      // If Stripe is configured and available
      if (isStripeEnabled && isStripeAvailable) {
        // Check if user has a payment method
        if (!hasPaymentMethod) {
          setPendingTier(tier);
          setShowPaymentModal(true);
          return;
        }
      } else {
        // Stripe is not configured - show error for paid tiers
        setUpdateError(
          'Payment processing is not configured. Please add a payment method in your organization settings first, or contact support.'
        );
        return;
      }
    }

    // Proceed to confirmation page
    navigateToConfirmation(tier);
  };

  const navigateToConfirmation = (tier: MembershipTier) => {
    const priceCents =
      billingCycle === 'annual'
        ? tier.annual_price_cents
        : tier.monthly_price_cents;

    // Navigate to confirmation page with tier details
    navigate('/settings/organization/membership/confirm', {
      state: {
        tierId: tier.id,
        tierName: tier.name,
        tierDisplayName: tier.display_name,
        billingCycle,
        priceCents,
      },
    });
  };

  const handlePaymentSuccess = async () => {
    setShowPaymentModal(false);
    // Reload payment methods
    if (currentTenant?.id) {
      await loadPaymentMethods(currentTenant.id);
    }
    // Navigate to confirmation page
    if (pendingTier) {
      navigateToConfirmation(pendingTier);
      setPendingTier(null);
    }
  };

  const handlePaymentCancel = () => {
    setShowPaymentModal(false);
    setPendingTier(null);
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
                  disabled={false}
                  highlighted={
                    tier.name === 'basic' ||
                    (!!preSelectedTierId && tier.id === preSelectedTierId)
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

      {/* Payment Method Modal - Only show when Stripe is available */}
      {isStripeAvailable && (
        <Modal
          isOpen={showPaymentModal}
          onClose={handlePaymentCancel}
          title="Add Payment Method"
        >
          <AddCardModalContent
            tenantId={currentTenant.id}
            onSuccess={handlePaymentSuccess}
            onCancel={handlePaymentCancel}
          />
        </Modal>
      )}
    </DashboardLayout>
  );
}

export default ChangeMembershipPage;

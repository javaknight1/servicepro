import { useEffect, useState } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { useStripe, useElements, CardElement } from '@stripe/react-stripe-js';
import { DashboardLayout } from '@components/layout';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  Button,
} from '@components/shared';
import { useMembershipStore, useTenantStore, useBillingStore } from '@store';
import { useStripeContext } from '@/providers/StripeProvider';
import { billingApi, membershipApi } from '@/services';
import { formatPrice } from '@/types/membership';
import type {
  BillingCycle,
  SubscriptionChangePreview,
} from '@/types/membership';
import {
  ArrowLeft,
  CreditCard,
  Check,
  Plus,
  Loader2,
  AlertCircle,
  Shield,
  AlertTriangle,
  Calendar,
  TrendingUp,
  TrendingDown,
} from 'lucide-react';

// Component for adding a new card
function AddCardForm({
  tenantId,
  onSuccess,
  onCancel,
}: {
  tenantId: string;
  onSuccess: () => void;
  onCancel: () => void;
}) {
  const stripe = useStripe();
  const elements = useElements();
  const { addPaymentMethod } = useBillingStore();

  const [isProcessing, setIsProcessing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    if (!stripe || !elements) return;

    setIsProcessing(true);
    setError(null);

    try {
      const setupResponse = await billingApi.createSetupIntent(tenantId);
      const clientSecret = setupResponse.data.client_secret;

      const cardElement = elements.getElement(CardElement);
      if (!cardElement) throw new Error('Card element not found');

      const { setupIntent, error: stripeError } = await stripe.confirmCardSetup(
        clientSecret,
        { payment_method: { card: cardElement } }
      );

      if (stripeError) throw new Error(stripeError.message);
      if (!setupIntent?.payment_method)
        throw new Error('No payment method returned');

      await addPaymentMethod(
        tenantId,
        setupIntent.payment_method as string,
        true
      );
      cardElement.clear();
      onSuccess();
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add card');
    } finally {
      setIsProcessing(false);
    }
  };

  return (
    <div className="border border-neutral-200 rounded-lg p-4 space-y-4">
      <h4 className="font-medium text-neutral-900">Add New Payment Method</h4>

      {error && (
        <div className="p-3 bg-red-50 border border-red-200 rounded-lg flex items-center text-red-700 text-sm">
          <AlertCircle className="h-4 w-4 mr-2 flex-shrink-0" />
          {error}
        </div>
      )}

      <div className="p-3 border border-neutral-300 rounded-lg">
        <CardElement
          options={{
            style: {
              base: {
                fontSize: '16px',
                color: '#1f2937',
                '::placeholder': { color: '#9ca3af' },
              },
              invalid: { color: '#dc2626' },
            },
          }}
        />
      </div>

      <div className="flex justify-end space-x-3">
        <Button variant="outline" onClick={onCancel} disabled={isProcessing}>
          Cancel
        </Button>
        <Button
          variant="primary"
          onClick={handleSubmit}
          disabled={isProcessing || !stripe}
        >
          {isProcessing ? (
            <>
              <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              Adding...
            </>
          ) : (
            'Add Card'
          )}
        </Button>
      </div>
    </div>
  );
}

interface LocationState {
  tierId: string;
  tierName: string;
  tierDisplayName: string;
  billingCycle: BillingCycle;
  priceCents: number;
}

export function ConfirmMembershipPage() {
  const navigate = useNavigate();
  const location = useLocation();
  const { currentTenant } = useTenantStore();
  const { isAvailable: isStripeAvailable } = useStripeContext();
  const { updateMembership, currentMembership } = useMembershipStore();
  const {
    paymentMethods,
    loadPaymentMethods,
    isLoadingMethods,
    isStripeEnabled,
    loadConfig,
    setDefaultPaymentMethod,
  } = useBillingStore();

  const [selectedPaymentMethodId, setSelectedPaymentMethodId] = useState<
    string | null
  >(null);
  const [isAddingCard, setIsAddingCard] = useState(false);
  const [isConfirming, setIsConfirming] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [preview, setPreview] = useState<SubscriptionChangePreview | null>(
    null
  );
  const [isLoadingPreview, setIsLoadingPreview] = useState(false);

  // Get state from navigation
  const state = location.state as LocationState | null;

  useEffect(() => {
    loadConfig();
  }, [loadConfig]);

  useEffect(() => {
    if (currentTenant?.id && isStripeEnabled) {
      loadPaymentMethods(currentTenant.id);
    }
  }, [currentTenant?.id, isStripeEnabled, loadPaymentMethods]);

  // Set default payment method when loaded
  useEffect(() => {
    if (paymentMethods.length > 0 && !selectedPaymentMethodId) {
      const defaultMethod = paymentMethods.find((pm) => pm.is_default);
      setSelectedPaymentMethodId(defaultMethod?.id || paymentMethods[0].id);
    }
  }, [paymentMethods, selectedPaymentMethodId]);

  // Fetch subscription change preview
  useEffect(() => {
    async function fetchPreview() {
      if (!currentTenant?.id || !state?.tierId) return;

      setIsLoadingPreview(true);
      try {
        const response = await membershipApi.previewSubscriptionChange(
          currentTenant.id,
          {
            tier_id: state.tierId,
            billing_cycle: state.billingCycle,
          }
        );
        setPreview(response.data);
      } catch (err) {
        console.error('Failed to fetch subscription preview:', err);
        // Don't set error - we can still proceed without preview
      } finally {
        setIsLoadingPreview(false);
      }
    }

    fetchPreview();
  }, [currentTenant?.id, state?.tierId, state?.billingCycle]);

  // Redirect if no state
  if (!state || !currentTenant) {
    return (
      <DashboardLayout>
        <div className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
          <Card variant="elevated" padding="lg">
            <CardContent>
              <div className="text-center py-8">
                <p className="text-neutral-600 mb-4">
                  No plan selected. Please choose a plan first.
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
            </CardContent>
          </Card>
        </div>
      </DashboardLayout>
    );
  }

  const { tierDisplayName, billingCycle, priceCents, tierId } = state;
  const isFree = priceCents === 0;

  const handleConfirm = async () => {
    if (!currentTenant?.id) return;

    // For paid plans, require a payment method
    if (!isFree && !selectedPaymentMethodId) {
      setError('Please select a payment method');
      return;
    }

    setIsConfirming(true);
    setError(null);

    try {
      // If a non-default payment method was selected, set it as the new default
      if (!isFree && selectedPaymentMethodId) {
        const currentDefault = paymentMethods.find((pm) => pm.is_default);
        if (currentDefault?.id !== selectedPaymentMethodId) {
          await setDefaultPaymentMethod(
            currentTenant.id,
            selectedPaymentMethodId
          );
        }
      }

      await updateMembership(currentTenant.id, tierId, billingCycle);
      // Success - redirect to dashboard
      navigate('/dashboard', {
        state: { membershipUpdated: true, newPlan: tierDisplayName },
      });
    } catch (err: unknown) {
      const errorMessage =
        err instanceof Error
          ? err.message
          : (err as { response?: { data?: { message?: string } } })?.response
              ?.data?.message || 'Failed to update membership';
      setError(errorMessage);
    } finally {
      setIsConfirming(false);
    }
  };

  const handleCardAdded = () => {
    setIsAddingCard(false);
    if (currentTenant?.id) {
      loadPaymentMethods(currentTenant.id);
    }
  };

  const currentPlanName = currentMembership?.tier?.display_name || 'Free';
  const isUpgrade =
    priceCents > (currentMembership?.subscription?.price_cents || 0);

  return (
    <DashboardLayout>
      <div className="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
        {/* Header */}
        <div className="mb-8">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate('/settings/organization/membership/change')}
            className="mb-4"
          >
            <ArrowLeft className="h-4 w-4 mr-2" />
            Back to Plans
          </Button>
          <h1 className="text-3xl font-bold text-neutral-900">
            Confirm Your Plan
          </h1>
          <p className="text-neutral-600 mt-2">
            Review your selection and confirm to{' '}
            {isUpgrade ? 'upgrade' : 'change'} your plan
          </p>
        </div>

        {/* Error Message */}
        {error && (
          <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-red-700 flex items-start">
            <AlertCircle className="h-5 w-5 mr-2 flex-shrink-0 mt-0.5" />
            {error}
          </div>
        )}

        <div className="space-y-6">
          {/* Plan Summary */}
          <Card variant="elevated" padding="lg">
            <CardHeader>
              <CardTitle className="flex items-center">
                Plan Summary
                {preview?.is_upgrade && (
                  <span className="ml-2 inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-green-100 text-green-800">
                    <TrendingUp className="h-3 w-3 mr-1" />
                    Upgrade
                  </span>
                )}
                {preview?.is_downgrade && !preview?.is_cancellation && (
                  <span className="ml-2 inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-amber-100 text-amber-800">
                    <TrendingDown className="h-3 w-3 mr-1" />
                    Downgrade
                  </span>
                )}
                {preview?.is_cancellation && (
                  <span className="ml-2 inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-red-100 text-red-800">
                    Cancellation
                  </span>
                )}
              </CardTitle>
              <CardDescription>
                {isUpgrade ? 'Upgrading' : 'Changing'} from {currentPlanName} to{' '}
                {tierDisplayName}
              </CardDescription>
            </CardHeader>
            <CardContent>
              {isLoadingPreview ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="h-6 w-6 animate-spin text-neutral-400" />
                  <span className="ml-2 text-neutral-500">
                    Calculating billing details...
                  </span>
                </div>
              ) : (
                <div className="space-y-4">
                  {/* Warnings */}
                  {preview?.warnings && preview.warnings.length > 0 && (
                    <div className="space-y-2">
                      {preview.warnings.map((warning, index) => (
                        <div
                          key={index}
                          className="p-3 bg-amber-50 border border-amber-200 rounded-lg flex items-start text-amber-800 text-sm"
                        >
                          <AlertTriangle className="h-4 w-4 mr-2 flex-shrink-0 mt-0.5" />
                          {warning}
                        </div>
                      ))}
                    </div>
                  )}

                  {/* Current Plan Info (if changing from a paid plan) */}
                  {preview?.current_tier_name &&
                    preview.current_price_cents > 0 && (
                      <div className="bg-neutral-50 rounded-lg p-4 space-y-2">
                        <h4 className="text-sm font-medium text-neutral-700">
                          Current Plan
                        </h4>
                        <div className="flex justify-between items-center text-sm">
                          <span className="text-neutral-600">
                            {preview.current_tier_name}
                          </span>
                          <span className="text-neutral-900">
                            {formatPrice(preview.current_price_cents)}/
                            {preview.current_billing_cycle === 'annual'
                              ? 'year'
                              : 'month'}
                          </span>
                        </div>
                        {preview.days_remaining > 0 && (
                          <div className="flex items-center text-sm text-neutral-500">
                            <Calendar className="h-4 w-4 mr-1" />
                            {preview.days_remaining} days remaining in current
                            period
                          </div>
                        )}
                      </div>
                    )}

                  <div className="flex justify-between items-center py-3 border-b border-neutral-200">
                    <span className="text-neutral-600">New Plan</span>
                    <span className="font-semibold text-neutral-900">
                      {tierDisplayName}
                    </span>
                  </div>
                  <div className="flex justify-between items-center py-3 border-b border-neutral-200">
                    <span className="text-neutral-600">Billing Cycle</span>
                    <span className="font-semibold text-neutral-900 capitalize">
                      {billingCycle}
                    </span>
                  </div>
                  <div className="flex justify-between items-center py-3 border-b border-neutral-200">
                    <span className="text-neutral-600">
                      {billingCycle === 'annual' ? 'Annual' : 'Monthly'} Price
                    </span>
                    <span className="font-semibold text-neutral-900">
                      {isFree ? 'Free' : formatPrice(priceCents)}
                      {!isFree && (
                        <span className="text-sm font-normal text-neutral-500">
                          /{billingCycle === 'annual' ? 'year' : 'month'}
                        </span>
                      )}
                    </span>
                  </div>

                  {/* Proration Details */}
                  {preview &&
                    !isFree &&
                    preview.proration_credit_cents !== 0 && (
                      <div className="flex justify-between items-center py-3 border-b border-neutral-200">
                        <span className="text-neutral-600">
                          Proration Credit
                          <span className="block text-xs text-neutral-400">
                            Unused time on current plan
                          </span>
                        </span>
                        <span className="font-semibold text-green-600">
                          -{formatPrice(preview.proration_credit_cents)}
                        </span>
                      </div>
                    )}

                  {/* Amount Due Today */}
                  {preview && !isFree && (
                    <div className="flex justify-between items-center py-3 bg-primary-50 rounded-lg px-4 -mx-4">
                      <span className="font-medium text-neutral-900">
                        Due Today
                      </span>
                      <span className="text-2xl font-bold text-primary-700">
                        {formatPrice(preview.amount_due_today_cents)}
                      </span>
                    </div>
                  )}

                  {/* Next Billing Info */}
                  {preview?.next_billing_date && !isFree && (
                    <div className="flex justify-between items-center py-3 text-sm">
                      <span className="text-neutral-500 flex items-center">
                        <Calendar className="h-4 w-4 mr-1" />
                        Next billing date
                      </span>
                      <span className="text-neutral-700">
                        {new Date(preview.next_billing_date).toLocaleDateString(
                          'en-US',
                          {
                            month: 'long',
                            day: 'numeric',
                            year: 'numeric',
                          }
                        )}
                        {' · '}
                        {formatPrice(preview.next_billing_amount_cents)}
                      </span>
                    </div>
                  )}

                  {billingCycle === 'annual' && !isFree && (
                    <div className="bg-green-50 border border-green-200 rounded-lg p-3 text-green-700 text-sm">
                      <Check className="h-4 w-4 inline mr-2" />
                      You're saving 17% with annual billing
                    </div>
                  )}
                </div>
              )}
            </CardContent>
          </Card>

          {/* Payment Method Selection - Only for paid plans */}
          {!isFree && (
            <Card variant="elevated" padding="lg">
              <CardHeader>
                <CardTitle>Payment Method</CardTitle>
                <CardDescription>
                  Select how you'd like to pay for your subscription
                </CardDescription>
              </CardHeader>
              <CardContent>
                {isLoadingMethods ? (
                  <div className="flex items-center justify-center py-8">
                    <Loader2 className="h-6 w-6 animate-spin text-neutral-400" />
                  </div>
                ) : paymentMethods.length === 0 && !isAddingCard ? (
                  <div className="text-center py-6">
                    <CreditCard className="h-12 w-12 mx-auto mb-4 text-neutral-300" />
                    <p className="text-neutral-500 mb-4">
                      No payment methods on file
                    </p>
                    {isStripeAvailable && (
                      <Button
                        variant="primary"
                        onClick={() => setIsAddingCard(true)}
                      >
                        <Plus className="h-4 w-4 mr-2" />
                        Add Payment Method
                      </Button>
                    )}
                  </div>
                ) : (
                  <div className="space-y-3">
                    {paymentMethods.map((pm) => (
                      <label
                        key={pm.id}
                        className={`flex items-center p-4 border rounded-lg cursor-pointer transition-colors ${
                          selectedPaymentMethodId === pm.id
                            ? 'border-primary-500 bg-primary-50'
                            : 'border-neutral-200 hover:border-neutral-300'
                        }`}
                      >
                        <input
                          type="radio"
                          name="paymentMethod"
                          value={pm.id}
                          checked={selectedPaymentMethodId === pm.id}
                          onChange={() => setSelectedPaymentMethodId(pm.id)}
                          className="h-4 w-4 text-primary-600 focus:ring-primary-500"
                        />
                        <div className="ml-4 flex items-center">
                          <div className="h-8 w-12 bg-neutral-100 rounded flex items-center justify-center mr-3">
                            <CreditCard className="h-5 w-5 text-neutral-500" />
                          </div>
                          <div>
                            <p className="font-medium text-neutral-900">
                              {pm.card_brand.charAt(0).toUpperCase() +
                                pm.card_brand.slice(1)}{' '}
                              ending in {pm.card_last_four}
                            </p>
                            <p className="text-sm text-neutral-500">
                              Expires{' '}
                              {pm.card_exp_month.toString().padStart(2, '0')}/
                              {pm.card_exp_year}
                            </p>
                          </div>
                        </div>
                        {pm.is_default && (
                          <span className="ml-auto text-xs text-primary-600 font-medium">
                            Default
                          </span>
                        )}
                      </label>
                    ))}

                    {!isAddingCard && isStripeAvailable && (
                      <button
                        onClick={() => setIsAddingCard(true)}
                        className="w-full flex items-center justify-center p-4 border-2 border-dashed border-neutral-300 rounded-lg text-neutral-600 hover:border-neutral-400 hover:text-neutral-700 transition-colors"
                      >
                        <Plus className="h-4 w-4 mr-2" />
                        Add New Payment Method
                      </button>
                    )}
                  </div>
                )}

                {isAddingCard && isStripeAvailable && currentTenant && (
                  <div className="mt-4">
                    <AddCardForm
                      tenantId={currentTenant.id}
                      onSuccess={handleCardAdded}
                      onCancel={() => setIsAddingCard(false)}
                    />
                  </div>
                )}
              </CardContent>
            </Card>
          )}

          {/* Security Note */}
          <div className="flex items-start space-x-3 text-sm text-neutral-500">
            <Shield className="h-5 w-5 flex-shrink-0 mt-0.5" />
            <p>
              Your payment is secured with 256-bit SSL encryption. We never
              store your full card number on our servers.
            </p>
          </div>

          {/* Confirm Button */}
          <div className="flex justify-end space-x-4">
            <Button
              variant="outline"
              onClick={() =>
                navigate('/settings/organization/membership/change')
              }
              disabled={isConfirming}
            >
              Cancel
            </Button>
            <Button
              variant="primary"
              size="lg"
              onClick={handleConfirm}
              disabled={
                isConfirming ||
                isLoadingPreview ||
                (!isFree && !selectedPaymentMethodId)
              }
            >
              {isConfirming ? (
                <>
                  <Loader2 className="h-4 w-4 mr-2 animate-spin" />
                  Processing...
                </>
              ) : (
                <>
                  Confirm{' '}
                  {preview?.is_cancellation
                    ? 'Cancellation'
                    : isUpgrade
                      ? 'Upgrade'
                      : 'Change'}
                  {!isFree &&
                    preview &&
                    ` - ${formatPrice(preview.amount_due_today_cents)}`}
                  {!isFree && !preview && ` - ${formatPrice(priceCents)}`}
                </>
              )}
            </Button>
          </div>
        </div>
      </div>
    </DashboardLayout>
  );
}

export default ConfirmMembershipPage;

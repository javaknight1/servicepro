import { useEffect, useState } from 'react';
import {
  CreditCard,
  Plus,
  Trash2,
  Star,
  History,
  AlertCircle,
  Loader2,
} from 'lucide-react';
import { useStripe, useElements, CardElement } from '@stripe/react-stripe-js';
import {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  Button,
  Badge,
} from '@components/shared';
import { useBillingStore, useTenantStore } from '@store';
import { useStripeContext } from '@/providers/StripeProvider';
import { billingApi } from '@/services';
import { formatCardBrand } from '@/types/payment';

// Separate component for the card input form that uses Stripe hooks
// This component only renders when Elements provider is available
interface AddCardFormProps {
  tenantId: string;
  isFirstCard: boolean;
  onSuccess: () => void;
  onCancel: () => void;
}

function AddCardForm({
  tenantId,
  isFirstCard,
  onSuccess,
  onCancel,
}: AddCardFormProps) {
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

      // Add the payment method to our backend
      await addPaymentMethod(
        tenantId,
        setupIntent.payment_method as string,
        isFirstCard // Set as default if it's the first card
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
    <div className="mt-4 p-4 border border-neutral-200 rounded-lg space-y-4">
      <h4 className="font-medium text-neutral-900">Add New Card</h4>

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

      <div className="flex justify-end space-x-3">
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
            'Add Card'
          )}
        </Button>
      </div>
    </div>
  );
}

export function BillingSection() {
  const { currentTenant } = useTenantStore();
  const { isAvailable: isStripeAvailable, isLoading: isStripeLoading } =
    useStripeContext();
  const {
    paymentMethods,
    isLoadingMethods,
    methodsError,
    billingEvents,
    isLoadingHistory,
    historyTotal,
    loadPaymentMethods,
    setDefaultPaymentMethod,
    deletePaymentMethod,
    loadBillingHistory,
    isStripeEnabled,
    loadConfig,
  } = useBillingStore();

  const [isAddingCard, setIsAddingCard] = useState(false);
  const [showHistory, setShowHistory] = useState(false);

  useEffect(() => {
    loadConfig();
  }, [loadConfig]);

  useEffect(() => {
    if (currentTenant?.id && isStripeEnabled) {
      loadPaymentMethods(currentTenant.id);
    }
  }, [currentTenant?.id, isStripeEnabled, loadPaymentMethods]);

  const handleSetDefault = async (paymentMethodId: string) => {
    if (!currentTenant?.id) return;
    try {
      await setDefaultPaymentMethod(currentTenant.id, paymentMethodId);
    } catch {
      // Error is handled in the store
    }
  };

  const handleDelete = async (paymentMethodId: string) => {
    if (!currentTenant?.id) return;
    if (!confirm('Are you sure you want to delete this payment method?'))
      return;
    try {
      await deletePaymentMethod(currentTenant.id, paymentMethodId);
    } catch {
      // Error is handled in the store
    }
  };

  const handleShowHistory = () => {
    if (!showHistory && currentTenant?.id) {
      loadBillingHistory(currentTenant.id);
    }
    setShowHistory(!showHistory);
  };

  // Show loading while checking Stripe availability
  if (isStripeLoading) {
    return (
      <Card variant="elevated" padding="lg">
        <CardHeader>
          <CardTitle>Payment Method</CardTitle>
          <CardDescription>
            Manage your billing and payment information
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center justify-center py-8">
            <Loader2 className="h-6 w-6 animate-spin text-neutral-400" />
          </div>
        </CardContent>
      </Card>
    );
  }

  if (!isStripeEnabled || !isStripeAvailable) {
    return (
      <Card variant="elevated" padding="lg">
        <CardHeader>
          <CardTitle>Payment Method</CardTitle>
          <CardDescription>
            Manage your billing and payment information
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="text-center py-8 text-neutral-500">
            <CreditCard className="h-12 w-12 mx-auto mb-4 text-neutral-300" />
            <p>Payment processing is not yet configured.</p>
          </div>
        </CardContent>
      </Card>
    );
  }

  return (
    <div className="space-y-6">
      {/* Payment Methods */}
      <Card variant="elevated" padding="lg">
        <CardHeader>
          <CardTitle>Payment Methods</CardTitle>
          <CardDescription>Manage your saved payment methods</CardDescription>
        </CardHeader>
        <CardContent>
          {methodsError && (
            <div className="mb-4 p-3 bg-red-50 border border-red-200 rounded-lg flex items-center text-red-700">
              <AlertCircle className="h-4 w-4 mr-2" />
              {methodsError}
            </div>
          )}

          {isLoadingMethods ? (
            <div className="flex items-center justify-center py-8">
              <Loader2 className="h-6 w-6 animate-spin text-neutral-400" />
            </div>
          ) : paymentMethods.length === 0 && !isAddingCard ? (
            <div className="text-center py-8">
              <CreditCard className="h-12 w-12 mx-auto mb-4 text-neutral-300" />
              <p className="text-neutral-500 mb-4">No payment methods saved</p>
              <Button variant="primary" onClick={() => setIsAddingCard(true)}>
                <Plus className="h-4 w-4 mr-2" />
                Add Payment Method
              </Button>
            </div>
          ) : (
            <div className="space-y-4">
              {paymentMethods.map((pm) => (
                <div
                  key={pm.id}
                  className="flex items-center justify-between p-4 border border-neutral-200 rounded-lg"
                >
                  <div className="flex items-center space-x-4">
                    <div className="h-10 w-16 bg-neutral-100 rounded flex items-center justify-center">
                      <CreditCard className="h-6 w-6 text-neutral-500" />
                    </div>
                    <div>
                      <div className="flex items-center space-x-2">
                        <p className="font-medium text-neutral-900">
                          {formatCardBrand(pm.card_brand)} ending in{' '}
                          {pm.card_last_four}
                        </p>
                        {pm.is_default && (
                          <Badge variant="success" size="sm">
                            <Star className="h-3 w-3 mr-1" />
                            Default
                          </Badge>
                        )}
                      </div>
                      <p className="text-sm text-neutral-500">
                        Expires {pm.card_exp_month.toString().padStart(2, '0')}/
                        {pm.card_exp_year}
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center space-x-2">
                    {!pm.is_default && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleSetDefault(pm.id)}
                      >
                        Set Default
                      </Button>
                    )}
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => handleDelete(pm.id)}
                      className="text-red-600 hover:text-red-700 hover:bg-red-50"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </div>
                </div>
              ))}

              {!isAddingCard && (
                <Button
                  variant="outline"
                  onClick={() => setIsAddingCard(true)}
                  className="w-full"
                >
                  <Plus className="h-4 w-4 mr-2" />
                  Add Another Card
                </Button>
              )}
            </div>
          )}

          {/* Add Card Form - Only rendered when Stripe is available */}
          {isAddingCard && currentTenant?.id && (
            <AddCardForm
              tenantId={currentTenant.id}
              isFirstCard={paymentMethods.length === 0}
              onSuccess={() => setIsAddingCard(false)}
              onCancel={() => setIsAddingCard(false)}
            />
          )}
        </CardContent>
      </Card>

      {/* Billing History */}
      <Card variant="elevated" padding="lg">
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Billing History</CardTitle>
              <CardDescription>
                View your past invoices and payments
              </CardDescription>
            </div>
            <Button variant="outline" size="sm" onClick={handleShowHistory}>
              <History className="h-4 w-4 mr-2" />
              {showHistory ? 'Hide' : 'Show'} History
            </Button>
          </div>
        </CardHeader>

        {showHistory && (
          <CardContent>
            {isLoadingHistory ? (
              <div className="flex items-center justify-center py-8">
                <Loader2 className="h-6 w-6 animate-spin text-neutral-400" />
              </div>
            ) : billingEvents.length === 0 ? (
              <div className="text-center py-8 text-neutral-500">
                No billing history yet
              </div>
            ) : (
              <div className="space-y-3">
                {billingEvents.map((event) => (
                  <div
                    key={event.id}
                    className="flex items-center justify-between p-3 bg-neutral-50 rounded-lg"
                  >
                    <div>
                      <p className="font-medium text-neutral-900">
                        {event.description || event.event_type}
                      </p>
                      <p className="text-sm text-neutral-500">
                        {new Date(event.created_at).toLocaleDateString()}
                      </p>
                    </div>
                    <div className="flex items-center space-x-3">
                      {event.amount_cents && (
                        <span className="font-medium text-neutral-900">
                          ${(event.amount_cents / 100).toFixed(2)}
                        </span>
                      )}
                      <Badge
                        variant={
                          event.status === 'succeeded'
                            ? 'success'
                            : event.status === 'failed'
                              ? 'error'
                              : 'warning'
                        }
                        size="sm"
                      >
                        {event.status}
                      </Badge>
                      {event.stripe_invoice_url && (
                        <a
                          href={event.stripe_invoice_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="text-sm text-primary-600 hover:underline"
                        >
                          View
                        </a>
                      )}
                    </div>
                  </div>
                ))}

                {historyTotal > billingEvents.length && (
                  <p className="text-sm text-neutral-500 text-center pt-2">
                    Showing {billingEvents.length} of {historyTotal} events
                  </p>
                )}
              </div>
            )}
          </CardContent>
        )}
      </Card>
    </div>
  );
}

export default BillingSection;

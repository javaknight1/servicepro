import { useState } from 'react';
import type {
  SubscriptionManagerProps,
  BillingCycle,
} from '../../types/subscription';
import {
  formatPrice,
  formatSubscriptionStatus,
  formatBillingCycle,
  formatDate,
  getStatusColor,
  getPriceForCycle,
} from '../../types/subscription';
import { cn } from '../../utils/cn';
import { PlanSelector } from './PlanSelector';
import { CancellationFlow } from './CancellationFlow';

export function SubscriptionManager({
  subscription,
  plans,
  onChangePlan,
  onChangeBillingCycle,
  onCancel,
  onReactivate,
  isLoading = false,
  error,
  className,
}: SubscriptionManagerProps) {
  const [showPlanSelector, setShowPlanSelector] = useState(false);
  const [showCancellation, setShowCancellation] = useState(false);
  const [selectedPlanId, setSelectedPlanId] = useState<string | undefined>(
    subscription?.plan.id
  );
  const [selectedCycle, setSelectedCycle] = useState<BillingCycle>(
    subscription?.billingCycle || 'monthly'
  );

  const handleSelectPlan = (planId: string) => {
    setSelectedPlanId(planId);
  };

  const handleChangePlan = () => {
    if (selectedPlanId && onChangePlan) {
      onChangePlan(selectedPlanId);
      setShowPlanSelector(false);
    }
  };

  const handleChangeCycle = (cycle: BillingCycle) => {
    setSelectedCycle(cycle);
    if (onChangeBillingCycle) {
      onChangeBillingCycle(cycle);
    }
  };

  if (isLoading) {
    return (
      <div className={cn('animate-pulse space-y-4', className)}>
        <div className="h-8 w-48 rounded bg-neutral-200" />
        <div className="h-32 rounded-lg bg-neutral-200" />
        <div className="h-24 rounded-lg bg-neutral-200" />
      </div>
    );
  }

  if (error) {
    return (
      <div
        className={cn(
          'rounded-lg border border-red-200 bg-red-50 p-6 text-center',
          className
        )}
      >
        <svg
          className="mx-auto h-10 w-10 text-red-500\"
          fill="none"
          stroke="currentColor"
          viewBox="0 0 24 24"
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            strokeWidth={2}
            d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"
          />
        </svg>
        <p className="mt-3 text-sm text-red-700">{error}</p>
      </div>
    );
  }

  // No subscription - show plan selector
  if (!subscription) {
    return (
      <div className={className}>
        <div className="mb-6">
          <h2 className="text-xl font-semibold text-neutral-900">
            Choose a Plan
          </h2>
          <p className="mt-1 text-sm text-neutral-600">
            Select the plan that best fits your needs
          </p>
        </div>
        <PlanSelector
          plans={plans}
          selectedPlanId={selectedPlanId}
          billingCycle={selectedCycle}
          onSelectPlan={handleSelectPlan}
          onChangeBillingCycle={handleChangeCycle}
        />
      </div>
    );
  }

  // Show cancellation flow
  if (showCancellation) {
    return (
      <CancellationFlow
        subscription={subscription}
        onCancel={(_reason, _feedback, _immediate) => {
          if (onCancel) {
            onCancel();
          }
          setShowCancellation(false);
        }}
        onKeepSubscription={() => setShowCancellation(false)}
        className={className}
      />
    );
  }

  // Show plan selector
  if (showPlanSelector) {
    return (
      <div className={className}>
        <div className="mb-6 flex items-center justify-between">
          <div>
            <h2 className="text-xl font-semibold text-neutral-900">
              Change Plan
            </h2>
            <p className="mt-1 text-sm text-neutral-600">
              Select a new plan. Changes will be prorated.
            </p>
          </div>
          <button
            type="button"
            onClick={() => setShowPlanSelector(false)}
            className="rounded-lg p-2 text-neutral-400 hover:bg-neutral-100 hover:text-neutral-600"
          >
            <svg
              className="h-5 w-5"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M6 18L18 6M6 6l12 12"
              />
            </svg>
          </button>
        </div>

        <PlanSelector
          plans={plans}
          currentPlanId={subscription.plan.id}
          selectedPlanId={selectedPlanId}
          billingCycle={selectedCycle}
          onSelectPlan={handleSelectPlan}
          onChangeBillingCycle={handleChangeCycle}
        />

        <div className="mt-6 flex gap-3">
          <button
            type="button"
            onClick={handleChangePlan}
            disabled={
              !selectedPlanId || selectedPlanId === subscription.plan.id
            }
            className={cn(
              'flex-1 rounded-lg px-4 py-2.5 text-sm font-medium',
              'bg-primary-600 text-white hover:bg-primary-700',
              'disabled:bg-neutral-300 disabled:cursor-not-allowed'
            )}
          >
            Confirm Plan Change
          </button>
          <button
            type="button"
            onClick={() => setShowPlanSelector(false)}
            className="rounded-lg border border-neutral-300 bg-white px-4 py-2.5 text-sm font-medium text-neutral-700 hover:bg-neutral-50"
          >
            Cancel
          </button>
        </div>
      </div>
    );
  }

  // Main subscription view
  const isTrialing = subscription.status === 'trialing';
  const isCanceled = subscription.status === 'canceled';
  const isCancelScheduled = subscription.cancelAtPeriodEnd;
  const isPastDue = subscription.status === 'past_due';

  return (
    <div className={cn('space-y-6', className)}>
      {/* Current Plan Card */}
      <div className="rounded-lg border border-neutral-200 bg-white p-6">
        <div className="flex items-start justify-between">
          <div>
            <div className="flex items-center gap-3">
              <h3 className="text-lg font-semibold text-neutral-900">
                {subscription.plan.name}
              </h3>
              <span
                className={cn(
                  'rounded-full px-2.5 py-0.5 text-xs font-medium',
                  getStatusColor(subscription.status)
                )}
              >
                {formatSubscriptionStatus(subscription.status)}
              </span>
            </div>
            <p className="mt-1 text-sm text-neutral-600">
              {subscription.plan.description}
            </p>
          </div>
          <div className="text-right">
            <p className="text-2xl font-bold text-neutral-900">
              {formatPrice(
                getPriceForCycle(subscription.plan, subscription.billingCycle),
                subscription.plan.currency
              )}
            </p>
            <p className="text-sm text-neutral-500">
              per{' '}
              {subscription.billingCycle === 'monthly'
                ? 'month'
                : subscription.billingCycle === 'quarterly'
                  ? 'quarter'
                  : 'year'}
            </p>
          </div>
        </div>

        {/* Status Messages */}
        {isTrialing && subscription.trialDaysRemaining > 0 && (
          <div className="mt-4 rounded-lg bg-blue-50 px-4 py-3">
            <div className="flex items-center gap-2">
              <svg
                className="h-5 w-5 text-blue-600"
                fill="currentColor"
                viewBox="0 0 20 20"
              >
                <path
                  fillRule="evenodd"
                  d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                  clipRule="evenodd"
                />
              </svg>
              <p className="text-sm font-medium text-blue-800">
                {subscription.trialDaysRemaining} days left in your trial
              </p>
            </div>
            {subscription.trialEnd && (
              <p className="mt-1 text-sm text-blue-700">
                Trial ends on {formatDate(subscription.trialEnd)}
              </p>
            )}
          </div>
        )}

        {isCancelScheduled && !isCanceled && (
          <div className="mt-4 rounded-lg bg-amber-50 px-4 py-3">
            <div className="flex items-center gap-2">
              <svg
                className="h-5 w-5 text-amber-600"
                fill="currentColor"
                viewBox="0 0 20 20"
              >
                <path
                  fillRule="evenodd"
                  d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 11-2 0 1 1 0 012 0zm-1-8a1 1 0 00-1 1v3a1 1 0 002 0V6a1 1 0 00-1-1z"
                  clipRule="evenodd"
                />
              </svg>
              <p className="text-sm font-medium text-amber-800">
                Subscription will end on{' '}
                {formatDate(subscription.currentPeriodEnd)}
              </p>
            </div>
            {onReactivate && (
              <button
                type="button"
                onClick={onReactivate}
                className="mt-2 text-sm font-medium text-amber-700 hover:text-amber-800 underline"
              >
                Cancel cancellation and keep subscription
              </button>
            )}
          </div>
        )}

        {isPastDue && (
          <div className="mt-4 rounded-lg bg-red-50 px-4 py-3">
            <div className="flex items-center gap-2">
              <svg
                className="h-5 w-5 text-red-600"
                fill="currentColor"
                viewBox="0 0 20 20"
              >
                <path
                  fillRule="evenodd"
                  d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                  clipRule="evenodd"
                />
              </svg>
              <p className="text-sm font-medium text-red-800">
                Payment failed. Please update your payment method.
              </p>
            </div>
          </div>
        )}

        {/* Billing Info */}
        {!isCanceled && (
          <div className="mt-6 grid grid-cols-2 gap-4 border-t border-neutral-200 pt-4">
            <div>
              <p className="text-sm text-neutral-500">Billing cycle</p>
              <p className="font-medium text-neutral-900">
                {formatBillingCycle(subscription.billingCycle)}
              </p>
            </div>
            <div>
              <p className="text-sm text-neutral-500">
                {isCancelScheduled ? 'Ends on' : 'Next billing date'}
              </p>
              <p className="font-medium text-neutral-900">
                {formatDate(subscription.currentPeriodEnd)}
              </p>
            </div>
          </div>
        )}
      </div>

      {/* Features */}
      <div className="rounded-lg border border-neutral-200 bg-white p-6">
        <h4 className="font-medium text-neutral-900">Included Features</h4>
        <ul className="mt-4 space-y-3">
          {subscription.plan.features.map((feature) => (
            <li key={feature.id} className="flex items-center gap-3">
              {feature.isEnabled ? (
                <svg
                  className="h-5 w-5 text-green-500"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path
                    fillRule="evenodd"
                    d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                    clipRule="evenodd"
                  />
                </svg>
              ) : (
                <svg
                  className="h-5 w-5 text-neutral-300"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path
                    fillRule="evenodd"
                    d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                    clipRule="evenodd"
                  />
                </svg>
              )}
              <span
                className={cn(
                  'text-sm',
                  feature.isEnabled ? 'text-neutral-900' : 'text-neutral-400'
                )}
              >
                {feature.featureName}
                {feature.featureType === 'numeric' && feature.numericValue && (
                  <span className="font-medium"> ({feature.numericValue})</span>
                )}
                {feature.featureType === 'text' && feature.value && (
                  <span className="font-medium"> - {feature.value}</span>
                )}
              </span>
            </li>
          ))}
        </ul>
      </div>

      {/* Actions */}
      {!isCanceled && (
        <div className="flex flex-wrap gap-3">
          {onChangePlan && (
            <button
              type="button"
              onClick={() => setShowPlanSelector(true)}
              className="rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700"
            >
              Change Plan
            </button>
          )}

          {onCancel && !isCancelScheduled && (
            <button
              type="button"
              onClick={() => setShowCancellation(true)}
              className="rounded-lg border border-neutral-300 bg-white px-4 py-2 text-sm font-medium text-neutral-700 hover:bg-neutral-50"
            >
              Cancel Subscription
            </button>
          )}
        </div>
      )}

      {/* Canceled state */}
      {isCanceled && (
        <div className="rounded-lg border border-neutral-200 bg-neutral-50 p-6 text-center">
          <svg
            className="mx-auto h-12 w-12 text-neutral-400"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              strokeWidth={2}
              d="M9.172 16.172a4 4 0 015.656 0M9 10h.01M15 10h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
            />
          </svg>
          <h4 className="mt-4 font-medium text-neutral-900">
            Subscription Canceled
          </h4>
          <p className="mt-2 text-sm text-neutral-600">
            Your subscription has been canceled. Subscribe again to regain
            access.
          </p>
          {onChangePlan && (
            <button
              type="button"
              onClick={() => setShowPlanSelector(true)}
              className="mt-4 rounded-lg bg-primary-600 px-4 py-2 text-sm font-medium text-white hover:bg-primary-700"
            >
              Subscribe Again
            </button>
          )}
        </div>
      )}
    </div>
  );
}

import { useState } from 'react';
import type {
  CancellationFlowProps,
  CancellationReason,
} from '../../types/subscription';
import {
  formatDate,
  formatPrice,
  getPriceForCycle,
} from '../../types/subscription';
import { cn } from '../../utils/cn';

// Cancellation reasons with labels
const CANCELLATION_REASONS: {
  value: CancellationReason;
  label: string;
  description: string;
}[] = [
  {
    value: 'user_requested',
    label: "I don't need it anymore",
    description: 'The service no longer fits my needs',
  },
  {
    value: 'payment_failed',
    label: "It's too expensive",
    description: 'The pricing is not within my budget',
  },
  {
    value: 'other',
    label: 'Missing features',
    description: "I need features that aren't available",
  },
  {
    value: 'other',
    label: 'Switching to competitor',
    description: "I'm moving to a different service",
  },
  {
    value: 'other',
    label: 'Other reason',
    description: "I'll explain in the feedback",
  },
];

// Step indicator
function StepIndicator({
  currentStep,
  totalSteps,
}: {
  currentStep: number;
  totalSteps: number;
}) {
  return (
    <div className="flex items-center justify-center gap-2 mb-6">
      {Array.from({ length: totalSteps }).map((_, index) => (
        <div
          key={index}
          className={cn(
            'h-2 w-2 rounded-full transition-colors',
            index < currentStep ? 'bg-primary-600' : 'bg-neutral-200'
          )}
        />
      ))}
    </div>
  );
}

export function CancellationFlow({
  subscription,
  onCancel,
  onKeepSubscription,
  isProcessing = false,
  className,
}: CancellationFlowProps) {
  const [step, setStep] = useState(1);
  const [selectedReason, setSelectedReason] =
    useState<CancellationReason | null>(null);
  const [feedback, setFeedback] = useState('');
  const [cancelImmediately, setCancelImmediately] = useState(false);

  const totalSteps = 3;

  const handleBack = () => {
    if (step > 1) {
      setStep(step - 1);
    }
  };

  const handleNext = () => {
    if (step < totalSteps) {
      setStep(step + 1);
    }
  };

  const handleConfirmCancel = () => {
    if (selectedReason) {
      onCancel(selectedReason, feedback || undefined, cancelImmediately);
    }
  };

  // Step 1: Why are you leaving?
  const renderStep1 = () => (
    <div>
      <h3 className="text-lg font-semibold text-neutral-900 text-center">
        We're sorry to see you go
      </h3>
      <p className="mt-2 text-center text-sm text-neutral-600">
        Please let us know why you're cancelling so we can improve.
      </p>

      <div className="mt-6 space-y-3">
        {CANCELLATION_REASONS.map((reason, index) => (
          <button
            key={index}
            type="button"
            onClick={() => setSelectedReason(reason.value)}
            className={cn(
              'w-full rounded-lg border p-4 text-left transition-colors',
              selectedReason === reason.value &&
                CANCELLATION_REASONS[index] ===
                  CANCELLATION_REASONS.find((r) => r.value === selectedReason)
                ? 'border-primary-500 bg-primary-50'
                : 'border-neutral-200 hover:border-neutral-300'
            )}
          >
            <p className="font-medium text-neutral-900">{reason.label}</p>
            <p className="mt-1 text-sm text-neutral-500">
              {reason.description}
            </p>
          </button>
        ))}
      </div>

      <div className="mt-6 flex gap-3">
        <button
          type="button"
          onClick={handleNext}
          disabled={!selectedReason}
          className={cn(
            'flex-1 rounded-lg px-4 py-2.5 text-sm font-medium',
            'bg-primary-600 text-white hover:bg-primary-700',
            'disabled:bg-neutral-300 disabled:cursor-not-allowed'
          )}
        >
          Continue
        </button>
        <button
          type="button"
          onClick={onKeepSubscription}
          className="rounded-lg border border-neutral-300 bg-white px-4 py-2.5 text-sm font-medium text-neutral-700 hover:bg-neutral-50"
        >
          Never mind
        </button>
      </div>
    </div>
  );

  // Step 2: Feedback
  const renderStep2 = () => (
    <div>
      <h3 className="text-lg font-semibold text-neutral-900 text-center">
        Any additional feedback?
      </h3>
      <p className="mt-2 text-center text-sm text-neutral-600">
        Your feedback helps us improve our service for everyone.
      </p>

      <div className="mt-6">
        <textarea
          value={feedback}
          onChange={(e) => setFeedback(e.target.value)}
          placeholder="Tell us more about your experience..."
          rows={4}
          className={cn(
            'w-full rounded-lg border border-neutral-300 px-4 py-3 text-sm',
            'placeholder:text-neutral-400',
            'focus:border-primary-500 focus:outline-none focus:ring-1 focus:ring-primary-500'
          )}
        />
      </div>

      <div className="mt-6 flex gap-3">
        <button
          type="button"
          onClick={handleBack}
          className="rounded-lg border border-neutral-300 bg-white px-4 py-2.5 text-sm font-medium text-neutral-700 hover:bg-neutral-50"
        >
          Back
        </button>
        <button
          type="button"
          onClick={handleNext}
          className="flex-1 rounded-lg bg-primary-600 px-4 py-2.5 text-sm font-medium text-white hover:bg-primary-700"
        >
          Continue
        </button>
      </div>
    </div>
  );

  // Step 3: Confirmation
  const renderStep3 = () => (
    <div>
      <h3 className="text-lg font-semibold text-neutral-900 text-center">
        Confirm Cancellation
      </h3>

      {/* What you'll lose */}
      <div className="mt-6 rounded-lg border border-amber-200 bg-amber-50 p-4">
        <h4 className="font-medium text-amber-800">
          What you'll lose access to:
        </h4>
        <ul className="mt-2 space-y-2">
          {subscription.plan.features
            .filter((f) => f.isEnabled)
            .slice(0, 4)
            .map((feature) => (
              <li
                key={feature.id}
                className="flex items-center gap-2 text-sm text-amber-700"
              >
                <svg
                  className="h-4 w-4"
                  fill="currentColor"
                  viewBox="0 0 20 20"
                >
                  <path
                    fillRule="evenodd"
                    d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z"
                    clipRule="evenodd"
                  />
                </svg>
                {feature.featureName}
              </li>
            ))}
        </ul>
      </div>

      {/* Cancellation timing */}
      <div className="mt-6 space-y-3">
        <label className="flex items-start gap-3 rounded-lg border border-neutral-200 p-4 cursor-pointer hover:bg-neutral-50">
          <input
            type="radio"
            name="cancelTiming"
            checked={!cancelImmediately}
            onChange={() => setCancelImmediately(false)}
            className="mt-0.5"
          />
          <div>
            <p className="font-medium text-neutral-900">
              Cancel at end of billing period
            </p>
            <p className="text-sm text-neutral-600">
              Keep access until {formatDate(subscription.currentPeriodEnd)}
            </p>
          </div>
        </label>

        <label className="flex items-start gap-3 rounded-lg border border-neutral-200 p-4 cursor-pointer hover:bg-neutral-50">
          <input
            type="radio"
            name="cancelTiming"
            checked={cancelImmediately}
            onChange={() => setCancelImmediately(true)}
            className="mt-0.5"
          />
          <div>
            <p className="font-medium text-neutral-900">Cancel immediately</p>
            <p className="text-sm text-neutral-600">
              Lose access right away (no refund)
            </p>
          </div>
        </label>
      </div>

      {/* Summary */}
      <div className="mt-6 rounded-lg bg-neutral-50 p-4">
        <div className="flex justify-between text-sm">
          <span className="text-neutral-600">Current plan</span>
          <span className="font-medium text-neutral-900">
            {subscription.plan.name}
          </span>
        </div>
        <div className="mt-2 flex justify-between text-sm">
          <span className="text-neutral-600">Monthly cost</span>
          <span className="font-medium text-neutral-900">
            {formatPrice(
              getPriceForCycle(subscription.plan, subscription.billingCycle),
              subscription.plan.currency
            )}
          </span>
        </div>
        <div className="mt-2 flex justify-between text-sm">
          <span className="text-neutral-600">
            {cancelImmediately ? 'Access ends' : 'Access until'}
          </span>
          <span className="font-medium text-neutral-900">
            {cancelImmediately
              ? 'Immediately'
              : formatDate(subscription.currentPeriodEnd)}
          </span>
        </div>
      </div>

      {/* Actions */}
      <div className="mt-6 flex gap-3">
        <button
          type="button"
          onClick={handleBack}
          disabled={isProcessing}
          className="rounded-lg border border-neutral-300 bg-white px-4 py-2.5 text-sm font-medium text-neutral-700 hover:bg-neutral-50 disabled:opacity-50"
        >
          Back
        </button>
        <button
          type="button"
          onClick={handleConfirmCancel}
          disabled={isProcessing}
          className={cn(
            'flex-1 rounded-lg px-4 py-2.5 text-sm font-medium',
            'bg-red-600 text-white hover:bg-red-700',
            'disabled:opacity-50 disabled:cursor-not-allowed'
          )}
        >
          {isProcessing ? (
            <span className="flex items-center justify-center gap-2">
              <svg
                className="h-4 w-4 animate-spin"
                fill="none"
                viewBox="0 0 24 24"
              >
                <circle
                  className="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  strokeWidth="4"
                />
                <path
                  className="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                />
              </svg>
              Cancelling...
            </span>
          ) : (
            'Confirm Cancellation'
          )}
        </button>
      </div>

      {/* Keep subscription option */}
      <div className="mt-4 text-center">
        <button
          type="button"
          onClick={onKeepSubscription}
          disabled={isProcessing}
          className="text-sm font-medium text-primary-600 hover:text-primary-700 disabled:opacity-50"
        >
          I changed my mind - keep my subscription
        </button>
      </div>
    </div>
  );

  return (
    <div className={cn('max-w-lg mx-auto', className)}>
      {/* Header */}
      <div className="mb-6 rounded-lg border border-neutral-200 bg-white p-6">
        <StepIndicator currentStep={step} totalSteps={totalSteps} />

        {step === 1 && renderStep1()}
        {step === 2 && renderStep2()}
        {step === 3 && renderStep3()}
      </div>

      {/* Special offer (optional - could be shown based on conditions) */}
      {step === 3 && (
        <div className="rounded-lg border border-green-200 bg-green-50 p-6 text-center">
          <h4 className="font-semibold text-green-800">
            Wait! Special offer just for you
          </h4>
          <p className="mt-2 text-sm text-green-700">
            Get 20% off for the next 3 months if you stay!
          </p>
          <button
            type="button"
            onClick={onKeepSubscription}
            className="mt-4 rounded-lg bg-green-600 px-6 py-2 text-sm font-medium text-white hover:bg-green-700"
          >
            Claim Offer & Stay
          </button>
        </div>
      )}
    </div>
  );
}

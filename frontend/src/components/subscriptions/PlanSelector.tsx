import type {
  PlanSelectorProps,
  BillingCycle,
  Plan,
} from '../../types/subscription';
import {
  formatPrice,
  formatBillingCycleShort,
  getPriceForCycle,
  calculateSavings,
} from '../../types/subscription';
import { cn } from '../../utils/cn';

// Billing cycle toggle component
function BillingCycleToggle({
  value,
  onChange,
  disabled,
}: {
  value: BillingCycle;
  onChange: (cycle: BillingCycle) => void;
  disabled?: boolean;
}) {
  return (
    <div className="flex items-center justify-center gap-4">
      <button
        type="button"
        onClick={() => onChange('monthly')}
        disabled={disabled}
        className={cn(
          'rounded-lg px-4 py-2 text-sm font-medium transition-colors',
          value === 'monthly'
            ? 'bg-primary-600 text-white'
            : 'bg-neutral-100 text-neutral-600 hover:bg-neutral-200',
          disabled && 'opacity-50 cursor-not-allowed'
        )}
      >
        Monthly
      </button>
      <button
        type="button"
        onClick={() => onChange('annual')}
        disabled={disabled}
        className={cn(
          'rounded-lg px-4 py-2 text-sm font-medium transition-colors relative',
          value === 'annual'
            ? 'bg-primary-600 text-white'
            : 'bg-neutral-100 text-neutral-600 hover:bg-neutral-200',
          disabled && 'opacity-50 cursor-not-allowed'
        )}
      >
        Annual
        <span className="absolute -top-2 -right-2 rounded-full bg-green-500 px-1.5 py-0.5 text-[10px] font-bold text-white">
          Save 17%
        </span>
      </button>
    </div>
  );
}

// Plan card component
function PlanCard({
  plan,
  billingCycle,
  isSelected,
  isCurrent,
  isHighlighted,
  onSelect,
  disabled,
}: {
  plan: Plan;
  billingCycle: BillingCycle;
  isSelected: boolean;
  isCurrent: boolean;
  isHighlighted: boolean;
  onSelect: () => void;
  disabled?: boolean;
}) {
  const price = getPriceForCycle(plan, billingCycle);
  const savings = calculateSavings(plan);
  const isEnterprise = plan.tier === 'enterprise';
  const isFree = plan.tier === 'free';

  return (
    <div
      className={cn(
        'relative rounded-xl border-2 p-6 transition-all',
        isSelected
          ? 'border-primary-500 bg-primary-50 ring-2 ring-primary-500'
          : isHighlighted
            ? 'border-primary-300 bg-white'
            : 'border-neutral-200 bg-white hover:border-neutral-300',
        isCurrent && 'ring-2 ring-primary-200',
        disabled && 'opacity-50 cursor-not-allowed'
      )}
    >
      {/* Popular badge */}
      {isHighlighted && (
        <span className="absolute -top-3 left-1/2 -translate-x-1/2 rounded-full bg-primary-600 px-3 py-1 text-xs font-semibold text-white">
          Most Popular
        </span>
      )}

      {/* Current badge */}
      {isCurrent && (
        <span className="absolute right-4 top-4 rounded-full bg-primary-100 px-2.5 py-0.5 text-xs font-medium text-primary-700">
          Current
        </span>
      )}

      {/* Plan name and description */}
      <h3 className="text-lg font-semibold text-neutral-900">{plan.name}</h3>
      <p className="mt-1 text-sm text-neutral-600">{plan.description}</p>

      {/* Price */}
      <div className="mt-4">
        {isEnterprise ? (
          <div>
            <span className="text-3xl font-bold text-neutral-900">Custom</span>
            <p className="mt-1 text-sm text-neutral-500">
              Contact us for pricing
            </p>
          </div>
        ) : isFree ? (
          <div>
            <span className="text-3xl font-bold text-neutral-900">$0</span>
            <span className="text-neutral-500">
              {formatBillingCycleShort(billingCycle)}
            </span>
          </div>
        ) : (
          <div>
            <span className="text-3xl font-bold text-neutral-900">
              {formatPrice(price, plan.currency)}
            </span>
            <span className="text-neutral-500">
              {formatBillingCycleShort(billingCycle)}
            </span>
            {billingCycle === 'annual' && savings.annualPercent > 0 && (
              <p className="mt-1 text-sm text-green-600">
                Save {formatPrice(savings.annual, plan.currency)} per year
              </p>
            )}
          </div>
        )}
      </div>

      {/* Features */}
      <ul className="mt-6 space-y-3">
        {plan.features.slice(0, 6).map((feature) => (
          <li key={feature.id} className="flex items-start gap-3">
            {feature.isEnabled ? (
              <svg
                className="h-5 w-5 shrink-0 text-green-500"
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
                className="h-5 w-5 shrink-0 text-neutral-300"
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
                feature.isEnabled ? 'text-neutral-700' : 'text-neutral-400'
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
        {plan.features.length > 6 && (
          <li className="text-sm text-neutral-500">
            + {plan.features.length - 6} more features
          </li>
        )}
      </ul>

      {/* Trial info */}
      {plan.trialDays > 0 && (
        <p className="mt-4 text-sm text-neutral-500">
          {plan.trialDays}-day free trial
        </p>
      )}

      {/* Select button */}
      <button
        type="button"
        onClick={onSelect}
        disabled={disabled || isCurrent}
        className={cn(
          'mt-6 w-full rounded-lg px-4 py-2.5 text-sm font-medium transition-colors',
          isSelected || isHighlighted
            ? 'bg-primary-600 text-white hover:bg-primary-700'
            : 'bg-neutral-100 text-neutral-700 hover:bg-neutral-200',
          (disabled || isCurrent) && 'opacity-50 cursor-not-allowed'
        )}
      >
        {isCurrent
          ? 'Current Plan'
          : isEnterprise
            ? 'Contact Sales'
            : isSelected
              ? 'Selected'
              : 'Select Plan'}
      </button>
    </div>
  );
}

export function PlanSelector({
  plans,
  currentPlanId,
  selectedPlanId,
  billingCycle,
  onSelectPlan,
  onChangeBillingCycle,
  showComparison = false,
  highlightedPlanId,
  disabled = false,
  className,
}: PlanSelectorProps) {
  // Sort plans by sort order
  const sortedPlans = [...plans].sort((a, b) => a.sortOrder - b.sortOrder);

  // Auto-highlight professional plan if not specified
  const highlighted =
    highlightedPlanId || sortedPlans.find((p) => p.tier === 'professional')?.id;

  const handleSelectPlan = (plan: Plan) => {
    if (!disabled && onSelectPlan && plan.id !== currentPlanId) {
      onSelectPlan(plan.id);
    }
  };

  return (
    <div className={className}>
      {/* Billing cycle toggle */}
      {onChangeBillingCycle && (
        <div className="mb-8">
          <BillingCycleToggle
            value={billingCycle}
            onChange={onChangeBillingCycle}
            disabled={disabled}
          />
        </div>
      )}

      {/* Plan cards */}
      <div
        className={cn(
          'grid gap-6',
          sortedPlans.length <= 3
            ? 'grid-cols-1 md:grid-cols-3'
            : 'grid-cols-1 md:grid-cols-2 lg:grid-cols-4'
        )}
      >
        {sortedPlans.map((plan) => (
          <PlanCard
            key={plan.id}
            plan={plan}
            billingCycle={billingCycle}
            isSelected={selectedPlanId === plan.id}
            isCurrent={currentPlanId === plan.id}
            isHighlighted={highlighted === plan.id}
            onSelect={() => handleSelectPlan(plan)}
            disabled={disabled}
          />
        ))}
      </div>

      {/* Feature comparison table */}
      {showComparison && sortedPlans.length > 1 && (
        <div className="mt-12">
          <h4 className="mb-6 text-center text-lg font-semibold text-neutral-900">
            Compare All Features
          </h4>
          <div className="overflow-x-auto">
            <table className="w-full border-collapse">
              <thead>
                <tr>
                  <th className="border-b border-neutral-200 px-4 py-3 text-left text-sm font-medium text-neutral-600">
                    Feature
                  </th>
                  {sortedPlans.map((plan) => (
                    <th
                      key={plan.id}
                      className={cn(
                        'border-b border-neutral-200 px-4 py-3 text-center text-sm font-medium',
                        highlighted === plan.id
                          ? 'bg-primary-50 text-primary-700'
                          : 'text-neutral-600'
                      )}
                    >
                      {plan.name}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {/* Get all unique feature keys */}
                {Array.from(
                  new Set(
                    sortedPlans.flatMap((p) =>
                      p.features.map((f) => f.featureKey)
                    )
                  )
                ).map((featureKey) => {
                  const feature = sortedPlans
                    .find((p) =>
                      p.features.some((f) => f.featureKey === featureKey)
                    )
                    ?.features.find((f) => f.featureKey === featureKey);
                  if (!feature) return null;

                  return (
                    <tr
                      key={featureKey}
                      className="border-b border-neutral-100"
                    >
                      <td className="px-4 py-3 text-sm text-neutral-700">
                        {feature.featureName}
                      </td>
                      {sortedPlans.map((plan) => {
                        const planFeature = plan.features.find(
                          (f) => f.featureKey === featureKey
                        );
                        return (
                          <td
                            key={plan.id}
                            className={cn(
                              'px-4 py-3 text-center text-sm',
                              highlighted === plan.id ? 'bg-primary-50' : ''
                            )}
                          >
                            {!planFeature || !planFeature.isEnabled ? (
                              <svg
                                className="mx-auto h-5 w-5 text-neutral-300"
                                fill="currentColor"
                                viewBox="0 0 20 20"
                              >
                                <path
                                  fillRule="evenodd"
                                  d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z"
                                  clipRule="evenodd"
                                />
                              </svg>
                            ) : planFeature.featureType === 'boolean' ? (
                              <svg
                                className="mx-auto h-5 w-5 text-green-500"
                                fill="currentColor"
                                viewBox="0 0 20 20"
                              >
                                <path
                                  fillRule="evenodd"
                                  d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                                  clipRule="evenodd"
                                />
                              </svg>
                            ) : planFeature.featureType === 'numeric' &&
                              planFeature.numericValue ? (
                              <span className="font-medium text-neutral-900">
                                {planFeature.numericValue}
                              </span>
                            ) : planFeature.value ? (
                              <span className="text-neutral-700">
                                {planFeature.value}
                              </span>
                            ) : (
                              <svg
                                className="mx-auto h-5 w-5 text-green-500"
                                fill="currentColor"
                                viewBox="0 0 20 20"
                              >
                                <path
                                  fillRule="evenodd"
                                  d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                                  clipRule="evenodd"
                                />
                              </svg>
                            )}
                          </td>
                        );
                      })}
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </div>
      )}
    </div>
  );
}

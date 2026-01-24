import { useEffect, useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import { Button } from '@components/shared';
import {
  TierCard,
  TierComparisonTable,
  BillingToggle,
} from '@components/membership';
import { useMembershipStore, useAuthStore } from '@store';
import type { BillingCycle } from '@/types/membership';

export function PricingPage() {
  const navigate = useNavigate();
  const { isAuthenticated } = useAuthStore();
  const { tiers, isLoading, error, loadTiers } = useMembershipStore();

  const [billingCycle, setBillingCycle] = useState<BillingCycle>('monthly');

  useEffect(() => {
    loadTiers();
  }, [loadTiers]);

  return (
    <div className="min-h-screen bg-neutral-50">
      {/* Header */}
      <header className="bg-white border-b border-neutral-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center space-x-4">
              <Link
                to={isAuthenticated ? '/dashboard' : '/'}
                className="text-neutral-600 hover:text-neutral-900"
              >
                <ArrowLeft className="h-5 w-5" />
              </Link>
              <span className="text-xl font-bold text-neutral-900">
                ServicePro
              </span>
            </div>
            {!isAuthenticated && (
              <div className="flex items-center space-x-4">
                <Button variant="ghost" onClick={() => navigate('/login')}>
                  Sign In
                </Button>
                <Button variant="primary" onClick={() => navigate('/register')}>
                  Get Started
                </Button>
              </div>
            )}
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="text-center mb-8">
          <h1 className="text-4xl font-bold text-neutral-900 mb-4">
            Simple, Transparent Pricing
          </h1>
          <p className="text-xl text-neutral-600 max-w-2xl mx-auto">
            Choose the plan that best fits your business. Upgrade or downgrade
            at any time.
          </p>
        </div>

        {/* Billing Toggle */}
        <div className="flex justify-center mb-12">
          <BillingToggle value={billingCycle} onChange={setBillingCycle} />
        </div>

        {isLoading ? (
          <div className="text-center py-12">
            <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-primary-600 mx-auto"></div>
            <p className="text-neutral-600 mt-4">Loading plans...</p>
          </div>
        ) : error ? (
          <div className="text-center py-12">
            <p className="text-red-600 mb-4">{error}</p>
            <Button variant="outline" onClick={() => loadTiers()}>
              Try Again
            </Button>
          </div>
        ) : (
          <>
            {/* Tier Cards (Mobile-friendly) - View only, no select buttons */}
            <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-16">
              {tiers.map((tier) => (
                <TierCard
                  key={tier.id}
                  tier={tier}
                  highlighted={tier.name === 'basic'}
                  showSelectButton={false}
                  billingCycle={billingCycle}
                />
              ))}
            </div>

            {/* Feature Comparison Table (Desktop) - View only, no select buttons */}
            <div className="hidden lg:block">
              <h2 className="text-2xl font-bold text-neutral-900 text-center mb-8">
                Compare Plans
              </h2>
              <TierComparisonTable
                tiers={tiers}
                showSelectButtons={false}
                billingCycle={billingCycle}
              />
            </div>

            {/* FAQ Section */}
            <div className="mt-16">
              <h2 className="text-2xl font-bold text-neutral-900 text-center mb-8">
                Frequently Asked Questions
              </h2>
              <div className="max-w-3xl mx-auto space-y-6">
                <FaqItem
                  question="Can I change my plan at any time?"
                  answer="Yes! You can upgrade or downgrade your plan at any time. Changes take effect immediately."
                />
                <FaqItem
                  question="What happens to my data if I downgrade?"
                  answer="Your data is always safe. If you exceed the limits of a lower tier, you'll be prompted to remove some items or upgrade back."
                />
                <FaqItem
                  question="Do you offer a free trial?"
                  answer="Yes! All paid plans come with a 14-day free trial. No credit card required to start."
                />
                <FaqItem
                  question="What payment methods do you accept?"
                  answer="We accept all major credit cards, including Visa, MasterCard, and American Express."
                />
              </div>
            </div>
          </>
        )}
      </main>

      {/* Footer */}
      <footer className="bg-white border-t border-neutral-200 py-8 mt-16">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 text-center text-sm text-neutral-500">
          <p>
            &copy; {new Date().getFullYear()} ServicePro. All rights reserved.
          </p>
        </div>
      </footer>
    </div>
  );
}

interface FaqItemProps {
  question: string;
  answer: string;
}

function FaqItem({ question, answer }: FaqItemProps) {
  return (
    <div className="bg-white rounded-lg shadow-sm border border-neutral-200 p-6">
      <h3 className="text-lg font-medium text-neutral-900 mb-2">{question}</h3>
      <p className="text-neutral-600">{answer}</p>
    </div>
  );
}

export default PricingPage;

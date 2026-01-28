import { Link } from 'react-router-dom';
import { ArrowLeft, CheckCircle } from 'lucide-react';
import { Button } from '@components/shared';
import { Footer } from '@components/layout';

export function FeaturesPage() {
  return (
    <div className="min-h-screen bg-neutral-50 flex flex-col">
      {/* Header */}
      <header className="bg-white border-b border-neutral-200">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
          <div className="flex items-center justify-between h-16">
            <div className="flex items-center space-x-4">
              <Link to="/" className="text-neutral-600 hover:text-neutral-900">
                <ArrowLeft className="h-5 w-5" />
              </Link>
              <Link to="/" className="text-xl font-bold text-neutral-900">
                ServicePro
              </Link>
            </div>
            <div className="flex items-center space-x-4">
              <Link to="/login">
                <Button variant="ghost">Sign In</Button>
              </Link>
              <Link to="/register">
                <Button variant="primary">Get Started</Button>
              </Link>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="text-center mb-12">
          <h1 className="text-4xl font-bold text-neutral-900 mb-4">
            Powerful Features for Your Business
          </h1>
          <p className="text-xl text-neutral-600 max-w-2xl mx-auto">
            Everything you need to manage your service business efficiently.
          </p>
        </div>

        {/* Feature Grid */}
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8">
          <FeatureCard
            title="Customer Management"
            description="Keep track of all your customers, their contact information, service history, and preferences in one place."
          />
          <FeatureCard
            title="Job Scheduling"
            description="Schedule and assign jobs to your team with an intuitive calendar interface and real-time updates."
          />
          <FeatureCard
            title="Quotes & Estimates"
            description="Create professional quotes quickly and convert them to jobs with a single click."
          />
          <FeatureCard
            title="Invoicing"
            description="Generate and send invoices automatically. Track payments and manage your cash flow effortlessly."
          />
          <FeatureCard
            title="Team Management"
            description="Manage your team members, assign roles, and control access permissions."
          />
          <FeatureCard
            title="Reports & Analytics"
            description="Gain insights into your business with detailed reports on revenue, customers, and team performance."
          />
        </div>

        {/* CTA Section */}
        <div className="mt-16 text-center">
          <h2 className="text-2xl font-bold text-neutral-900 mb-4">
            Ready to get started?
          </h2>
          <p className="text-neutral-600 mb-8">
            Join thousands of businesses already using ServicePro.
          </p>
          <div className="flex justify-center space-x-4">
            <Link to="/register">
              <Button variant="primary" size="lg">
                Start Free Trial
              </Button>
            </Link>
            <Link to="/pricing">
              <Button variant="outline" size="lg">
                View Pricing
              </Button>
            </Link>
          </div>
        </div>
      </main>

      <Footer />
    </div>
  );
}

interface FeatureCardProps {
  title: string;
  description: string;
}

function FeatureCard({ title, description }: FeatureCardProps) {
  return (
    <div className="bg-white rounded-lg shadow-sm border border-neutral-200 p-6">
      <div className="flex items-start space-x-3">
        <CheckCircle className="h-6 w-6 text-primary-600 flex-shrink-0 mt-0.5" />
        <div>
          <h3 className="text-lg font-semibold text-neutral-900 mb-2">
            {title}
          </h3>
          <p className="text-neutral-600">{description}</p>
        </div>
      </div>
    </div>
  );
}

export default FeaturesPage;

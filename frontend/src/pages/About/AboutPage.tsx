import { Link } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import { Button } from '@components/shared';
import { Footer } from '@components/layout';

export function AboutPage() {
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
      <main className="flex-1 max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="text-center mb-12">
          <h1 className="text-4xl font-bold text-neutral-900 mb-4">
            About ServicePro
          </h1>
          <p className="text-xl text-neutral-600">
            Building the future of service business management.
          </p>
        </div>

        <div className="prose prose-neutral max-w-none">
          <section className="mb-12">
            <h2 className="text-2xl font-bold text-neutral-900 mb-4">
              Our Mission
            </h2>
            <p className="text-neutral-600 leading-relaxed">
              ServicePro was founded with a simple mission: to help service
              businesses operate more efficiently and deliver better experiences
              to their customers. We believe that technology should make running
              a business easier, not harder.
            </p>
          </section>

          <section className="mb-12">
            <h2 className="text-2xl font-bold text-neutral-900 mb-4">
              Our Story
            </h2>
            <p className="text-neutral-600 leading-relaxed mb-4">
              {/* TODO: Add your company story here */}
              We started ServicePro after experiencing firsthand the challenges
              of managing a growing service business. Juggling customer
              information, scheduling jobs, tracking invoices, and managing a
              team across multiple tools was overwhelming.
            </p>
            <p className="text-neutral-600 leading-relaxed">
              That's why we built ServicePro - an all-in-one platform that
              brings everything together in one place. Today, we're proud to
              serve businesses of all sizes, from solo operators to large teams.
            </p>
          </section>

          <section className="mb-12">
            <h2 className="text-2xl font-bold text-neutral-900 mb-4">
              Our Values
            </h2>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <ValueCard
                title="Simplicity"
                description="We believe powerful software doesn't have to be complicated. We design with simplicity in mind."
              />
              <ValueCard
                title="Reliability"
                description="Your business depends on us. We take that responsibility seriously and strive for 99.9% uptime."
              />
              <ValueCard
                title="Customer Focus"
                description="We listen to our customers and continuously improve based on their feedback."
              />
              <ValueCard
                title="Innovation"
                description="We're always looking for new ways to help our customers succeed."
              />
            </div>
          </section>
        </div>

        {/* CTA Section */}
        <div className="mt-16 text-center bg-white rounded-lg shadow-sm border border-neutral-200 p-8">
          <h2 className="text-2xl font-bold text-neutral-900 mb-4">
            Ready to transform your business?
          </h2>
          <p className="text-neutral-600 mb-8">
            Join thousands of service businesses already using ServicePro.
          </p>
          <Link to="/register">
            <Button variant="primary" size="lg">
              Start Free Trial
            </Button>
          </Link>
        </div>
      </main>

      <Footer />
    </div>
  );
}

interface ValueCardProps {
  title: string;
  description: string;
}

function ValueCard({ title, description }: ValueCardProps) {
  return (
    <div className="bg-white rounded-lg shadow-sm border border-neutral-200 p-6">
      <h3 className="text-lg font-semibold text-neutral-900 mb-2">{title}</h3>
      <p className="text-neutral-600">{description}</p>
    </div>
  );
}

export default AboutPage;

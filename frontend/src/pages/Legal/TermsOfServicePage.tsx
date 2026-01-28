import { Link } from 'react-router-dom';
import { ArrowLeft } from 'lucide-react';
import { Button } from '@components/shared';
import { Footer } from '@components/layout';

export function TermsOfServicePage() {
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
        <div className="bg-white rounded-lg shadow-sm border border-neutral-200 p-8 md:p-12">
          <h1 className="text-4xl font-bold text-neutral-900 mb-2">
            Terms of Service
          </h1>
          <p className="text-neutral-500 mb-8">
            Last updated: {new Date().toLocaleDateString()}
          </p>

          <div className="prose prose-neutral max-w-none">
            {/* TODO: Replace with your actual terms of service content */}
            <section className="mb-8">
              <h2 className="text-2xl font-bold text-neutral-900 mb-4">
                1. Acceptance of Terms
              </h2>
              <p className="text-neutral-600 leading-relaxed">
                By accessing or using ServicePro, you agree to be bound by these
                Terms of Service and all applicable laws and regulations. If you
                do not agree with any of these terms, you are prohibited from
                using or accessing this service.
              </p>
            </section>

            <section className="mb-8">
              <h2 className="text-2xl font-bold text-neutral-900 mb-4">
                2. Description of Service
              </h2>
              <p className="text-neutral-600 leading-relaxed">
                ServicePro provides a cloud-based platform for managing service
                businesses, including customer management, job scheduling,
                quoting, invoicing, and reporting features.
              </p>
            </section>

            <section className="mb-8">
              <h2 className="text-2xl font-bold text-neutral-900 mb-4">
                3. Account Registration
              </h2>
              <p className="text-neutral-600 leading-relaxed mb-4">
                To use our service, you must:
              </p>
              <ul className="list-disc list-inside text-neutral-600 space-y-2">
                <li>Create an account with accurate information</li>
                <li>Maintain the security of your account credentials</li>
                <li>Be at least 18 years old or have parental consent</li>
                <li>
                  Accept responsibility for all activities under your account
                </li>
              </ul>
            </section>

            <section className="mb-8">
              <h2 className="text-2xl font-bold text-neutral-900 mb-4">
                4. Acceptable Use
              </h2>
              <p className="text-neutral-600 leading-relaxed mb-4">
                You agree not to:
              </p>
              <ul className="list-disc list-inside text-neutral-600 space-y-2">
                <li>Use the service for any unlawful purpose</li>
                <li>
                  Attempt to gain unauthorized access to any part of the service
                </li>
                <li>Interfere with or disrupt the service or servers</li>
                <li>Upload malicious code or content</li>
                <li>Violate any applicable laws or regulations</li>
              </ul>
            </section>

            <section className="mb-8">
              <h2 className="text-2xl font-bold text-neutral-900 mb-4">
                5. Payment Terms
              </h2>
              <p className="text-neutral-600 leading-relaxed">
                Paid plans are billed in advance on a monthly or annual basis.
                All fees are non-refundable except as required by law. We
                reserve the right to change our pricing with 30 days notice.
              </p>
            </section>

            <section className="mb-8">
              <h2 className="text-2xl font-bold text-neutral-900 mb-4">
                6. Termination
              </h2>
              <p className="text-neutral-600 leading-relaxed">
                We may terminate or suspend your account at any time for
                violations of these terms. You may cancel your account at any
                time through your account settings.
              </p>
            </section>

            <section className="mb-8">
              <h2 className="text-2xl font-bold text-neutral-900 mb-4">
                7. Limitation of Liability
              </h2>
              <p className="text-neutral-600 leading-relaxed">
                To the maximum extent permitted by law, ServicePro shall not be
                liable for any indirect, incidental, special, consequential, or
                punitive damages resulting from your use of the service.
              </p>
            </section>

            <section className="mb-8">
              <h2 className="text-2xl font-bold text-neutral-900 mb-4">
                8. Changes to Terms
              </h2>
              <p className="text-neutral-600 leading-relaxed">
                We reserve the right to modify these terms at any time. We will
                notify users of any material changes via email or through the
                service.
              </p>
            </section>

            <section className="mb-8">
              <h2 className="text-2xl font-bold text-neutral-900 mb-4">
                9. Contact Us
              </h2>
              <p className="text-neutral-600 leading-relaxed">
                If you have questions about these Terms of Service, please
                contact us at{' '}
                <a
                  href="mailto:legal@servicepro.com"
                  className="text-primary-600 hover:underline"
                >
                  legal@servicepro.com
                </a>
                .
              </p>
            </section>
          </div>
        </div>
      </main>

      <Footer />
    </div>
  );
}

export default TermsOfServicePage;

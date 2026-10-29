import { Link } from 'react-router';
import { ArrowLeft } from 'lucide-react';
import { Button } from '@components/shared';
import { Footer, RequestAccessLink } from '@components/layout';
import { LEGAL_DRAFT_DATE, SUPPORT_EMAIL } from '@/config/support';

export function PrivacyPolicyPage() {
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
              <RequestAccessLink label="Request access" />
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="flex-1 max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="bg-white rounded-lg shadow-xs border border-neutral-200 p-8 md:p-12">
          <h1 className="text-4xl font-bold text-neutral-900 mb-2">
            Privacy Policy
          </h1>
          <div
            role="note"
            className="mb-8 p-4 bg-yellow-50 border border-yellow-200 rounded-lg text-yellow-800"
          >
            <strong>Draft — pending legal review.</strong> This document is a
            draft for the invite-only preview and is not yet final. Draft dated{' '}
            {LEGAL_DRAFT_DATE}.
          </div>

          <div className="prose prose-neutral max-w-none">
            <section className="mb-8">
              <h2 className="text-2xl font-bold text-neutral-900 mb-4">
                1. Introduction
              </h2>
              <p className="text-neutral-600 leading-relaxed">
                ServicePro ("we", "our", or "us") is committed to protecting
                your privacy. This Privacy Policy explains how we collect, use,
                disclose, and safeguard your information when you use our
                service.
              </p>
            </section>

            <section className="mb-8">
              <h2 className="text-2xl font-bold text-neutral-900 mb-4">
                2. Information We Collect
              </h2>
              <p className="text-neutral-600 leading-relaxed mb-4">
                We may collect information about you in a variety of ways,
                including:
              </p>
              <ul className="list-disc list-inside text-neutral-600 space-y-2">
                <li>
                  Personal data you provide when registering for an account
                </li>
                <li>
                  Information about your customers that you enter into our
                  system
                </li>
                <li>Usage data about how you interact with our service</li>
                <li>Device and browser information</li>
              </ul>
            </section>

            <section className="mb-8">
              <h2 className="text-2xl font-bold text-neutral-900 mb-4">
                3. How We Use Your Information
              </h2>
              <p className="text-neutral-600 leading-relaxed mb-4">
                We use the information we collect to:
              </p>
              <ul className="list-disc list-inside text-neutral-600 space-y-2">
                <li>Provide and maintain our service</li>
                <li>Process transactions and send related information</li>
                <li>Send administrative information and updates</li>
                <li>Respond to inquiries and provide customer support</li>
                <li>Improve our services and develop new features</li>
              </ul>
            </section>

            <section className="mb-8">
              <h2 className="text-2xl font-bold text-neutral-900 mb-4">
                4. Data Security
              </h2>
              <p className="text-neutral-600 leading-relaxed">
                We implement appropriate technical and organizational security
                measures to protect your personal information. However, no
                method of transmission over the Internet or electronic storage
                is 100% secure.
              </p>
            </section>

            <section className="mb-8">
              <h2 className="text-2xl font-bold text-neutral-900 mb-4">
                5. Your Rights
              </h2>
              <p className="text-neutral-600 leading-relaxed mb-4">
                Depending on your location, you may have certain rights
                regarding your personal information, including:
              </p>
              <ul className="list-disc list-inside text-neutral-600 space-y-2">
                <li>The right to access your personal data</li>
                <li>The right to correct inaccurate data</li>
                <li>The right to request deletion of your data</li>
                <li>The right to data portability</li>
              </ul>
            </section>

            <section className="mb-8">
              <h2 className="text-2xl font-bold text-neutral-900 mb-4">
                6. Contact Us
              </h2>
              <p className="text-neutral-600 leading-relaxed">
                If you have questions about this Privacy Policy, please contact
                us at{' '}
                <a
                  href={`mailto:${SUPPORT_EMAIL}`}
                  className="text-primary-600 hover:underline"
                >
                  {SUPPORT_EMAIL}
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

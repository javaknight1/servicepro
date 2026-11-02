import { Link } from 'react-router';
import {
  Check,
  ClipboardCheck,
  CreditCard,
  Landmark,
  ReceiptText,
  Repeat,
} from 'lucide-react';
import { MainLayout, RequestAccessLink } from '@components/layout';
import { BETA_ACCESS_MAILTO } from '@/config/support';

/**
 * Public pricing appears once T210 ships the Solo / Team / Pro plans for real
 * (D10). Until then the section stays hidden (D11 honesty gate, T255).
 */
const SHOW_PRICING = false;

const SAMPLE_INVOICE_LINES = [
  {
    site: 'Harborview Lofts',
    detail: 'Nightly cleaning, Mon–Fri · 22 visits',
    amount: '$2,400.00',
  },
  {
    site: 'Riverside Commons',
    detail: 'Cleaning 3x weekly · 13 visits',
    amount: '$1,800.00',
  },
  {
    site: 'Elm Street Plaza',
    detail: 'Nightly cleaning, Mon–Fri · 22 visits',
    amount: '$1,350.00',
  },
  {
    site: 'Short North Tower',
    detail: 'Nightly cleaning + weekend porter · 31 visits',
    amount: '$3,150.00',
  },
  {
    site: 'Short North Tower',
    detail: 'Carpet extraction, lobby · WO-2291, approved by Karen Doyle',
    amount: '$450.00',
  },
];

const STEPS = [
  {
    icon: Repeat,
    title: 'Recurring visits, scheduled once',
    body: "Set up a service agreement per site: nightly Mon–Fri, 3x a week, weekend porter. Visits land on your dispatch board and each tech's phone automatically.",
  },
  {
    icon: ClipboardCheck,
    title: 'Proof the work was done',
    body: 'Techs work a site checklist, take photos and get a sign-off from the site manager. The service report goes to the customer before they ask.',
  },
  {
    icon: ReceiptText,
    title: 'One invoice per customer',
    body: 'Roll every site and approved extra into one monthly invoice with the PO on it. Customers pay by bank or card; reminders follow up for you.',
  },
];

const PLANS = [
  {
    name: 'Solo',
    price: 39,
    users: '1 user',
    features: [
      'Customers, sites and agreements',
      'Scheduling and the tech mobile app',
      'Consolidated invoices and online payments',
      'QuickBooks Online sync',
    ],
  },
  {
    name: 'Team',
    price: 99,
    users: 'Unlimited users',
    highlight: true,
    features: [
      'Everything in Solo',
      'Dispatch board for your whole crew',
      'Roles: owner, office, technician',
      'Checklists, photos and sign-off',
    ],
  },
  {
    name: 'Pro',
    price: 199,
    users: 'Unlimited users',
    features: [
      'Everything in Team',
      'Multiple business entities',
      'Separate payouts and books per entity',
      'Advanced reporting and exports',
    ],
  },
];

export function LandingPage() {
  return (
    <MainLayout>
      {/* Hero */}
      <section className="bg-linear-to-b from-primary-50 to-white">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 py-16 lg:py-20 grid lg:grid-cols-2 gap-12 items-center">
          <div>
            <span className="inline-block rounded-full border border-primary-200 bg-primary-50 px-3 py-1 text-xs font-medium text-primary-700">
              Invite-only beta · spring 2027
            </span>
            <h1 className="mt-5 text-5xl md:text-6xl font-bold tracking-tight text-neutral-900">
              Get paid for commercial service work.
            </h1>
            <p className="mt-6 text-lg text-neutral-600 max-w-xl">
              Schedule the recurring visits, prove every one was done, and send
              each multi-site customer one consolidated invoice they can pay by
              bank transfer.
            </p>
            <div className="mt-8 flex flex-wrap gap-3">
              <RequestAccessLink size="lg" />
              <a
                href="#how-it-works"
                className="inline-flex items-center rounded-lg border border-primary-600 px-5 py-2.5 text-base font-medium text-primary-600 hover:bg-primary-50"
              >
                See how it works
              </a>
            </div>
            <p className="mt-4 text-sm text-neutral-500 max-w-md">
              Built for commercial cleaning and facility-maintenance companies
              with 1 to 10 technicians.
            </p>
            <p className="mt-3 text-sm text-neutral-600">
              Have an invite code?{' '}
              <Link
                to="/register"
                className="text-primary-600 hover:text-primary-700 font-medium"
              >
                Create your account
              </Link>
            </p>
          </div>

          {/* Example consolidated invoice */}
          <div className="relative">
            <div className="rounded-2xl border border-neutral-200 bg-white p-6 shadow-lg">
              <div className="flex items-start justify-between">
                <div>
                  <p className="text-sm text-neutral-500">
                    Example invoice{' '}
                    <span className="font-mono text-primary-600">INV-1042</span>{' '}
                    · October 2026
                  </p>
                  <p className="mt-1 text-lg font-semibold text-neutral-900">
                    Harborview Property Management
                  </p>
                  <p className="text-sm text-neutral-500">
                    4 sites · PO 7731-HV · Net 30, due Dec 1
                  </p>
                </div>
                <span className="rounded-full bg-primary-50 px-2.5 py-0.5 text-xs font-medium text-primary-700">
                  Sent
                </span>
              </div>
              <ul className="mt-4 divide-y divide-neutral-100">
                {SAMPLE_INVOICE_LINES.map((line) => (
                  <li
                    key={line.detail}
                    className="flex items-start justify-between gap-4 py-2.5"
                  >
                    <div>
                      <p className="text-sm font-medium text-neutral-900">
                        {line.site}
                      </p>
                      <p className="text-xs text-neutral-500">{line.detail}</p>
                    </div>
                    <p className="text-sm tabular-nums text-neutral-900">
                      {line.amount}
                    </p>
                  </li>
                ))}
              </ul>
              <div className="mt-2 flex items-center justify-between border-t border-neutral-200 pt-3">
                <p className="text-sm text-neutral-600">
                  Total incl. Ohio sales tax 8.0%
                </p>
                <p className="text-xl font-bold tabular-nums text-neutral-900">
                  $9,882.00
                </p>
              </div>
            </div>
            <div className="absolute -bottom-8 left-4 flex items-center gap-3 rounded-xl border border-neutral-200 bg-white px-4 py-3 shadow-md">
              <span className="flex h-9 w-9 items-center justify-center rounded-lg bg-success-50 text-success-600">
                <Landmark className="h-5 w-5" aria-hidden="true" />
              </span>
              <div>
                <p className="text-sm font-medium text-neutral-900">
                  Pay by bank
                </p>
                <p className="text-xs text-neutral-500">
                  Fee to you on $9,882.00: $10.00
                </p>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* How it works */}
      <section id="how-it-works" className="bg-white py-20">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
          <h2 className="max-w-2xl text-3xl md:text-4xl font-bold text-neutral-900">
            From the nightly visit to the money in your account.
          </h2>
          <div className="mt-12 grid gap-10 md:grid-cols-3">
            {STEPS.map(({ icon: Icon, title, body }) => (
              <div key={title}>
                <Icon className="h-8 w-8 text-primary-600" aria-hidden="true" />
                <h3 className="mt-4 text-lg font-semibold text-neutral-900">
                  {title}
                </h3>
                <p className="mt-2 text-neutral-600">{body}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {SHOW_PRICING && (
        <section id="pricing" className="bg-neutral-50 py-20">
          <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8">
            <h2 className="text-3xl md:text-4xl font-bold text-neutral-900">
              Pricing
            </h2>
            <p className="mt-2 text-neutral-600">
              Month to month, no contract. Export all your data anytime.
            </p>
            <div className="mt-10 grid gap-6 md:grid-cols-3">
              {PLANS.map((plan) => (
                <div
                  key={plan.name}
                  className={`rounded-2xl border bg-white p-6 ${plan.highlight ? 'border-primary-600 ring-1 ring-primary-600' : 'border-neutral-200'}`}
                >
                  <p className="text-lg font-semibold text-neutral-900">
                    {plan.name}
                  </p>
                  <p className="mt-3 text-4xl font-bold text-neutral-900">
                    ${plan.price}
                    <span className="text-base font-normal text-neutral-500">
                      /mo
                    </span>
                  </p>
                  <p className="text-sm text-neutral-600">{plan.users}</p>
                  <ul className="mt-5 space-y-2">
                    {plan.features.map((f) => (
                      <li
                        key={f}
                        className="flex gap-2 text-sm text-neutral-700"
                      >
                        <Check
                          className="h-4 w-4 shrink-0 text-primary-600"
                          aria-hidden="true"
                        />
                        {f}
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
            </div>
            <div className="mt-6 grid gap-4 rounded-2xl border border-neutral-200 bg-white p-6 md:grid-cols-2">
              <div className="flex gap-4">
                <span className="flex h-10 w-10 items-center justify-center rounded-lg bg-primary-50 text-primary-600">
                  <CreditCard className="h-5 w-5" aria-hidden="true" />
                </span>
                <div>
                  <p className="font-medium text-neutral-900">Card payments</p>
                  <p className="text-sm text-neutral-600">
                    Stripe&apos;s rate + 0.4%
                  </p>
                </div>
              </div>
              <div>
                <p className="font-medium text-neutral-900">
                  Bank transfers (ACH)
                </p>
                <p className="text-sm text-neutral-600">
                  1%, capped at $10 per payment. A $5,000 invoice costs you $10.
                </p>
              </div>
            </div>
          </div>
        </section>
      )}

      {/* Beta CTA */}
      <section className="bg-primary-600 py-14">
        <div className="max-w-6xl mx-auto px-4 sm:px-6 lg:px-8 flex flex-col gap-6 md:flex-row md:items-center md:justify-between">
          <div>
            <h2 className="text-2xl md:text-3xl font-bold text-white">
              The beta opens in spring 2027.
            </h2>
            <p className="mt-2 text-lg text-primary-100">
              We&apos;re onboarding a small group of commercial service teams in
              an invite-only beta first.
            </p>
          </div>
          <a
            href={BETA_ACCESS_MAILTO}
            className="inline-flex items-center justify-center rounded-lg bg-white px-6 py-3 text-base font-medium text-primary-600 shadow-sm hover:bg-primary-50"
          >
            Request beta access
          </a>
        </div>
      </section>
    </MainLayout>
  );
}

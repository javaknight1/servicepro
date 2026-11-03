import { useEffect, useState } from 'react';
import { useParams } from 'react-router';
import { CreditCard, Landmark, Loader2 } from 'lucide-react';
import {
  payInvoiceUrl,
  publicInvoiceApi,
} from '@services/publicInvoiceService';
import type {
  PayMethod,
  PublicInvoice,
  PublicPayMethod,
} from '@/types/publicInvoice';
import { formatDollars } from '@/utils/money';
import { getApiErrorInfo } from '@/utils/error';

/** "2026-12-01" → a local date (no UTC shift). */
function parseDay(value: string): Date | null {
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(value);
  if (!m) return null;
  return new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]));
}

function formatDay(value: string, options: Intl.DateTimeFormatOptions) {
  const d = parseDay(value);
  return d ? d.toLocaleDateString('en-US', options) : '';
}

const METHOD_LABEL: Record<PayMethod, string> = {
  bank: 'Pay by bank',
  card: 'Pay by card',
};

interface MethodOptionProps {
  option: PublicPayMethod;
  selected: boolean;
  recommended: boolean;
  businessName: string;
  currency: string;
  onSelect: () => void;
}

function MethodOption({
  option,
  selected,
  recommended,
  businessName,
  currency,
  onSelect,
}: MethodOptionProps) {
  const Icon = option.method === 'bank' ? Landmark : CreditCard;
  const fee = formatDollars(option.business_fee, { currency });
  return (
    <label
      className={`flex gap-3 rounded-2xl border bg-white p-4 cursor-pointer transition-colors ${
        selected
          ? 'border-2 border-primary-600 p-[15px]'
          : 'border-neutral-200 hover:border-neutral-300'
      }`}
    >
      <input
        type="radio"
        name="pay-method"
        value={option.method}
        checked={selected}
        onChange={onSelect}
        className="mt-1 h-5 w-5 shrink-0 accent-primary-600"
      />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center gap-2">
          <Icon
            className="h-5 w-5 text-neutral-700"
            aria-hidden="true"
            strokeWidth={1.75}
          />
          <span className="text-lg font-semibold text-neutral-900">
            {METHOD_LABEL[option.method]}
          </span>
          {recommended && (
            <span className="ml-auto rounded-full border border-success-200 bg-success-50 px-2.5 py-0.5 text-xs font-medium text-success-700">
              Recommended
            </span>
          )}
        </div>
        {selected && (
          <>
            <p className="mt-1.5 text-neutral-600">
              {option.method === 'bank'
                ? 'No fee to you. Connect your bank or enter account details.'
                : 'No fee to you. Visa, Mastercard, American Express and more.'}
            </p>
            <p className="mt-1.5 text-sm text-neutral-500">
              {fee} {option.method} fee paid by {businessName}
            </p>
          </>
        )}
      </div>
    </label>
  );
}

function Notice({ title, children }: { title: string; children?: string }) {
  return (
    <div className="rounded-2xl border border-neutral-200 bg-white p-5 text-center">
      <p className="font-semibold text-neutral-900">{title}</p>
      {children && <p className="mt-1 text-neutral-600">{children}</p>}
    </div>
  );
}

/**
 * Public pay page (T170, mockup 17): the emailed "Pay now" link lands here.
 * The customer picks bank (recommended, cheaper for the business) or card,
 * then the backend creates the Checkout session for that method. No login.
 *
 * Plain state rather than React Query: a customer's invoice must not end up
 * in the persisted query cache of whatever browser opens the link.
 */
export function PayInvoicePage() {
  const { token = '' } = useParams<{ token: string }>();
  const [invoice, setInvoice] = useState<PublicInvoice | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [method, setMethod] = useState<PayMethod | null>(null);

  useEffect(() => {
    let cancelled = false;
    publicInvoiceApi
      .getDetails(token)
      .then(({ data }) => {
        if (cancelled) return;
        setInvoice(data);
        setMethod(data.pay_methods[0]?.method ?? null);
      })
      .catch((err: unknown) => {
        if (cancelled) return;
        setError(
          getApiErrorInfo(err).status === 404
            ? "This payment link isn't valid. Please check the link in your email or contact the business."
            : "We couldn't load this invoice. Please try again in a moment."
        );
      });
    return () => {
      cancelled = true;
    };
  }, [token]);

  if (error) {
    return (
      <PageShell>
        <Notice title="Payment link unavailable">{error}</Notice>
      </PageShell>
    );
  }
  if (!invoice) {
    return (
      <PageShell>
        <div
          className="flex justify-center py-24 text-neutral-500"
          role="status"
          aria-label="Loading invoice"
        >
          <Loader2 className="h-6 w-6 animate-spin" aria-hidden="true" />
        </div>
      </PageShell>
    );
  }

  const currency = invoice.currency || 'USD';
  const amountDue = formatDollars(invoice.amount_due, { currency });
  const period = formatDay(invoice.issue_date, {
    month: 'long',
    year: 'numeric',
  });
  const dueDate = formatDay(invoice.due_date, {
    month: 'long',
    day: 'numeric',
    year: 'numeric',
  });
  const selected = invoice.pay_methods.find((m) => m.method === method);
  const isPaid = invoice.status === 'paid';

  return (
    <PageShell>
      <header className="flex items-center gap-4">
        <div
          className="flex h-14 w-14 shrink-0 items-center justify-center rounded-xl bg-primary-600 text-xl font-bold text-white"
          aria-hidden="true"
        >
          {invoice.business_name.trim().charAt(0).toUpperCase() || '?'}
        </div>
        <div className="min-w-0">
          <h1 className="text-xl font-semibold text-neutral-900">
            {invoice.business_name}
          </h1>
          {invoice.business_location && (
            <p className="text-neutral-500">{invoice.business_location}</p>
          )}
        </div>
      </header>

      <section className="mt-6 rounded-2xl border border-neutral-200 bg-white p-5">
        <p className="text-neutral-500">
          Invoice {invoice.invoice_number}
          {period && ` · ${period}`}
        </p>
        <p className="mt-1 text-4xl font-bold tracking-tight text-neutral-900">
          {amountDue}
        </p>
        {dueDate && !isPaid && (
          <p className="mt-1 text-neutral-700">Due {dueDate}</p>
        )}
        <dl className="mt-4 space-y-2 border-t border-neutral-200 pt-4 text-sm">
          {invoice.customer_name && (
            <div className="flex justify-between gap-4">
              <dt className="text-neutral-500">Billed to</dt>
              <dd className="text-right font-medium text-neutral-900">
                {invoice.customer_name}
              </dd>
            </div>
          )}
          {invoice.po_number && (
            <div className="flex justify-between gap-4">
              <dt className="text-neutral-500">PO number</dt>
              <dd className="text-right font-mono text-neutral-900">
                {invoice.po_number}
              </dd>
            </div>
          )}
        </dl>
      </section>

      {isPaid ? (
        <div className="mt-6">
          <Notice title="This invoice is paid">Thank you!</Notice>
        </div>
      ) : invoice.pay_methods.length === 0 ? (
        <div className="mt-6">
          <Notice title="Online payment isn't available for this invoice">
            {`Please contact ${invoice.business_name} to pay.`}
          </Notice>
        </div>
      ) : (
        <>
          <fieldset className="mt-6">
            <legend className="mb-3 font-semibold text-neutral-900">
              How would you like to pay?
            </legend>
            <div className="space-y-3">
              {invoice.pay_methods.map((option) => (
                <MethodOption
                  key={option.method}
                  option={option}
                  selected={option.method === method}
                  recommended={
                    option.method === 'bank' && invoice.pay_methods.length > 1
                  }
                  businessName={invoice.business_name}
                  currency={currency}
                  onSelect={() => setMethod(option.method)}
                />
              ))}
            </div>
          </fieldset>
          {selected && (
            <a
              href={payInvoiceUrl(token, selected.method)}
              className="mt-8 block w-full rounded-2xl bg-primary-600 px-4 py-4 text-center text-lg font-semibold text-white hover:bg-primary-700 focus:outline-none focus-visible:ring-2 focus-visible:ring-primary-500 focus-visible:ring-offset-2"
            >
              Pay {amountDue} by {selected.method}
            </a>
          )}
        </>
      )}
    </PageShell>
  );
}

function PageShell({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-neutral-50">
      <main className="mx-auto flex min-h-screen w-full max-w-md flex-col px-4 py-8">
        <div className="flex-1">{children}</div>
        <p className="mt-6 text-center text-sm text-neutral-500">
          Secure payment by Stripe · Powered by ServicePro
        </p>
      </main>
    </div>
  );
}

import { InfoCard } from '@components/shared';
import type { Customer } from '@services/customerService';
import { formatDollars, toAmount } from '@/utils/money';
import {
  ReceiptText,
  FileText,
  UserCheck,
  ShieldCheck,
  UserRound,
} from 'lucide-react';

interface BillingRulesCardsProps {
  customer: Customer;
}

/** Formats a decimal NTE limit: whole dollars when it has no cents. */
function formatLimit(limit: string): string {
  const amount = toAmount(limit);
  return formatDollars(amount, { wholeUnits: Number.isInteger(amount) });
}

/** The billing rules row of mockup 06 (T187), plus the main contact. */
export function BillingRulesCards({ customer }: BillingRulesCardsProps) {
  const contactName =
    `${customer.first_name} ${customer.last_name}`.trim() || '—';
  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-5">
      <InfoCard
        icon={<ReceiptText />}
        label="Billing"
        value={
          customer.consolidated_billing ? 'Consolidated monthly' : 'Per job'
        }
        detail={
          customer.consolidated_billing
            ? 'One invoice, a line per site'
            : 'An invoice for each job'
        }
      />
      <InfoCard
        icon={<FileText />}
        label="Purchase orders"
        value={customer.po_required ? 'PO required' : 'PO optional'}
        detail={
          customer.default_po_number
            ? `Current PO ${customer.default_po_number}`
            : 'No current PO'
        }
      />
      <InfoCard
        icon={<UserCheck />}
        label="Approver"
        value={customer.billing_approver_name || 'No approver'}
        detail={customer.billing_approver_email || 'Add one for extra work'}
      />
      <InfoCard
        icon={<ShieldCheck />}
        label="Extra work limit"
        value={
          customer.nte_limit
            ? `${formatLimit(customer.nte_limit)} not-to-exceed`
            : 'No limit'
        }
        detail={
          customer.nte_limit
            ? 'Per extra work order'
            : 'Extra work needs no approval'
        }
      />
      <InfoCard
        icon={<UserRound />}
        label="Contact"
        value={contactName}
        detail={customer.email || customer.phone_primary || 'No email'}
      />
    </div>
  );
}

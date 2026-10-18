/** Methods offered for a manually recorded payment (free text in the API). */
export const MANUAL_PAYMENT_METHODS = [
  { value: 'check', label: 'Check' },
  { value: 'cash', label: 'Cash' },
  { value: 'bank_transfer', label: 'Bank transfer / ACH' },
  { value: 'card', label: 'Card (outside ServicePro)' },
  { value: 'other', label: 'Other' },
] as const;

import type { Meta, StoryObj } from '@storybook/react';
import { PaymentSummary } from '../PaymentSummary';
import type { PaymentSummaryData } from '../../../types/payment';

const meta: Meta<typeof PaymentSummary> = {
  title: 'Payments/PaymentSummary',
  component: PaymentSummary,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component:
          'Displays a summary of the payment including line items, subtotal, tax, discounts, and total.',
      },
    },
  },
  argTypes: {
    showLineItems: {
      control: 'boolean',
      description: 'Whether to show individual line items',
    },
    isLoading: {
      control: 'boolean',
      description: 'Show loading skeleton',
    },
    className: {
      control: 'text',
      description: 'Additional CSS classes',
    },
  },
};

export default meta;
type Story = StoryObj<typeof PaymentSummary>;

const basicSummary: PaymentSummaryData = {
  lineItems: [
    {
      id: '1',
      description: 'Pro Subscription - Monthly',
      quantity: 1,
      unitPrice: 2999,
      totalPrice: 2999,
    },
  ],
  subtotal: 2999,
  tax: 240,
  taxRate: 8,
  total: 3239,
  currency: 'usd',
};

export const Default: Story = {
  args: {
    summary: basicSummary,
    showLineItems: true,
  },
};

export const WithDiscount: Story = {
  args: {
    summary: {
      ...basicSummary,
      discount: 500,
      discountCode: 'SAVE15',
      total: 2739,
    },
    showLineItems: true,
  },
};

export const WithDiscountNoCode: Story = {
  args: {
    summary: {
      ...basicSummary,
      discount: 300,
      total: 2939,
    },
    showLineItems: true,
  },
  parameters: {
    docs: {
      description: {
        story: 'Discount without a promo code',
      },
    },
  },
};

export const MultipleLineItems: Story = {
  args: {
    summary: {
      lineItems: [
        {
          id: '1',
          description: 'Pro Subscription - Monthly',
          quantity: 1,
          unitPrice: 2999,
          totalPrice: 2999,
        },
        {
          id: '2',
          description: 'Analytics Add-on',
          quantity: 1,
          unitPrice: 999,
          totalPrice: 999,
        },
        {
          id: '3',
          description: 'Priority Support',
          quantity: 1,
          unitPrice: 1499,
          totalPrice: 1499,
        },
        {
          id: '4',
          description: 'API Access',
          quantity: 1,
          unitPrice: 499,
          totalPrice: 499,
        },
      ],
      subtotal: 5996,
      tax: 480,
      taxRate: 8,
      total: 6476,
      currency: 'usd',
    },
    showLineItems: true,
  },
};

export const WithQuantities: Story = {
  args: {
    summary: {
      lineItems: [
        {
          id: '1',
          description: 'Team Seat',
          quantity: 5,
          unitPrice: 1000,
          totalPrice: 5000,
        },
        {
          id: '2',
          description: 'Storage Add-on (100GB)',
          quantity: 3,
          unitPrice: 500,
          totalPrice: 1500,
        },
      ],
      subtotal: 6500,
      tax: 520,
      taxRate: 8,
      total: 7020,
      currency: 'usd',
    },
    showLineItems: true,
  },
  parameters: {
    docs: {
      description: {
        story: 'Line items with quantities greater than 1 show unit price',
      },
    },
  },
};

export const HiddenLineItems: Story = {
  args: {
    summary: basicSummary,
    showLineItems: false,
  },
  parameters: {
    docs: {
      description: {
        story: 'Compact view without line item details',
      },
    },
  },
};

export const Loading: Story = {
  args: {
    summary: basicSummary,
    isLoading: true,
  },
};

export const NoTaxRate: Story = {
  args: {
    summary: {
      ...basicSummary,
      taxRate: undefined,
    },
    showLineItems: true,
  },
  parameters: {
    docs: {
      description: {
        story: 'Tax without percentage displayed',
      },
    },
  },
};

export const EuroCurrency: Story = {
  args: {
    summary: {
      ...basicSummary,
      currency: 'eur',
    },
    showLineItems: true,
  },
};

export const GBPCurrency: Story = {
  args: {
    summary: {
      ...basicSummary,
      currency: 'gbp',
    },
    showLineItems: true,
  },
};

export const LargeAmount: Story = {
  args: {
    summary: {
      lineItems: [
        {
          id: '1',
          description: 'Enterprise Annual Plan',
          quantity: 1,
          unitPrice: 999900,
          totalPrice: 999900,
        },
      ],
      subtotal: 999900,
      tax: 79992,
      taxRate: 8,
      discount: 100000,
      discountCode: 'ENTERPRISE',
      total: 979892,
      currency: 'usd',
    },
    showLineItems: true,
  },
};

export const ZeroTax: Story = {
  args: {
    summary: {
      ...basicSummary,
      tax: 0,
      taxRate: 0,
      total: 2999,
    },
    showLineItems: true,
  },
  parameters: {
    docs: {
      description: {
        story: 'Tax-exempt transaction',
      },
    },
  },
};

export const CompleteExample: Story = {
  args: {
    summary: {
      lineItems: [
        {
          id: '1',
          description: 'Business Plan - Annual',
          quantity: 1,
          unitPrice: 29900,
          totalPrice: 29900,
        },
        {
          id: '2',
          description: 'Additional Team Members',
          quantity: 4,
          unitPrice: 1000,
          totalPrice: 4000,
        },
        {
          id: '3',
          description: 'Premium Support Package',
          quantity: 1,
          unitPrice: 5000,
          totalPrice: 5000,
        },
      ],
      subtotal: 38900,
      tax: 3112,
      taxRate: 8,
      discount: 5000,
      discountCode: 'ANNUAL20',
      total: 37012,
      currency: 'usd',
    },
    showLineItems: true,
  },
};

import type { Meta, StoryObj } from '@storybook/react';
import { action } from '@storybook/addon-actions';
import { PaymentResult } from '../PaymentResult';
import type { PaymentResultData } from '../../../types/payment';

const meta: Meta<typeof PaymentResult> = {
  title: 'Payments/PaymentResult',
  component: PaymentResult,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component:
          'Displays the result of a payment attempt with appropriate styling, messaging, and action buttons.',
      },
    },
  },
  argTypes: {
    onRetry: {
      action: 'retry',
      description: 'Callback when user clicks retry button (failed state)',
    },
    onClose: {
      action: 'close',
      description: 'Callback when user clicks continue/close button',
    },
    successMessage: {
      control: 'text',
      description: 'Custom success message',
    },
    failureMessage: {
      control: 'text',
      description: 'Custom failure message',
    },
    className: {
      control: 'text',
      description: 'Additional CSS classes',
    },
  },
};

export default meta;
type Story = StoryObj<typeof PaymentResult>;

const successResult: PaymentResultData = {
  status: 'succeeded',
  paymentIntentId: 'pi_3NhGQVGswQrCr0Xb1234abcd',
  amount: {
    amount: 9999,
    currency: 'usd',
    displayAmount: '$99.99',
  },
  last4: '4242',
  brand: 'visa',
};

const failedResult: PaymentResultData = {
  status: 'failed',
  error: 'Your card was declined. Please try a different payment method.',
};

export const Success: Story = {
  args: {
    result: successResult,
    onClose: action('continue'),
  },
};

export const SuccessWithReceipt: Story = {
  args: {
    result: {
      ...successResult,
      receiptUrl: 'https://pay.stripe.com/receipts/example',
    },
    onClose: action('continue'),
  },
  parameters: {
    docs: {
      description: {
        story: 'Success state with a downloadable receipt link',
      },
    },
  },
};

export const SuccessCustomMessage: Story = {
  args: {
    result: successResult,
    successMessage:
      'Thank you for your purchase! Your subscription is now active.',
    onClose: action('continue'),
  },
};

export const SuccessMastercard: Story = {
  args: {
    result: {
      ...successResult,
      last4: '5678',
      brand: 'mastercard',
    },
    onClose: action('continue'),
  },
};

export const SuccessAmex: Story = {
  args: {
    result: {
      ...successResult,
      last4: '1234',
      brand: 'amex',
    },
    onClose: action('continue'),
  },
};

export const Failed: Story = {
  args: {
    result: failedResult,
    onRetry: action('retry'),
    onClose: action('close'),
  },
};

export const FailedInsufficientFunds: Story = {
  args: {
    result: {
      status: 'failed',
      error: 'Your card has insufficient funds. Please try a different card.',
    },
    onRetry: action('retry'),
    onClose: action('close'),
  },
};

export const FailedExpiredCard: Story = {
  args: {
    result: {
      status: 'failed',
      error: 'Your card has expired. Please update your card details.',
    },
    onRetry: action('retry'),
    onClose: action('close'),
  },
};

export const FailedCustomMessage: Story = {
  args: {
    result: failedResult,
    failureMessage:
      'Oops! Something went wrong with your payment. Our team has been notified.',
    onRetry: action('retry'),
    onClose: action('close'),
  },
};

export const FailedNoRetry: Story = {
  args: {
    result: failedResult,
    onClose: action('close'),
  },
  parameters: {
    docs: {
      description: {
        story: 'Failed state without retry option',
      },
    },
  },
};

export const Processing: Story = {
  args: {
    result: {
      status: 'processing',
      paymentIntentId: 'pi_processing123',
    },
  },
  parameters: {
    docs: {
      description: {
        story: 'Payment is being processed (async payment methods)',
      },
    },
  },
};

export const RequiresAction: Story = {
  args: {
    result: {
      status: 'requires_action',
      paymentIntentId: 'pi_requires_action123',
    },
    onClose: action('close'),
  },
  parameters: {
    docs: {
      description: {
        story: 'Additional authentication required (3D Secure, etc.)',
      },
    },
  },
};

export const Canceled: Story = {
  args: {
    result: {
      status: 'canceled',
    },
    onClose: action('close'),
  },
};

export const SuccessLargeAmount: Story = {
  args: {
    result: {
      ...successResult,
      amount: {
        amount: 1250000,
        currency: 'usd',
        displayAmount: '$12,500.00',
      },
    },
    onClose: action('continue'),
  },
};

export const SuccessEuro: Story = {
  args: {
    result: {
      ...successResult,
      amount: {
        amount: 4999,
        currency: 'eur',
        displayAmount: '€49.99',
      },
    },
    onClose: action('continue'),
  },
};

export const MobileView: Story = {
  args: {
    result: successResult,
    onClose: action('continue'),
  },
  parameters: {
    viewport: {
      defaultViewport: 'mobile1',
    },
    docs: {
      description: {
        story: 'Responsive layout on mobile devices',
      },
    },
  },
};

export const FailedWithStripeError: Story = {
  args: {
    result: {
      status: 'failed',
      error: {
        type: 'card_error',
        code: 'card_declined',
        message: 'Your card was declined.',
        decline_code: 'generic_decline',
      } as any,
    },
    onRetry: action('retry'),
    onClose: action('close'),
  },
  parameters: {
    docs: {
      description: {
        story: 'Failed with a Stripe error object',
      },
    },
  },
};

export const FailedWithErrorObject: Story = {
  args: {
    result: {
      status: 'failed',
      error: new Error('Network connection lost'),
    },
    onRetry: action('retry'),
    onClose: action('close'),
  },
  parameters: {
    docs: {
      description: {
        story: 'Failed with a JavaScript Error object',
      },
    },
  },
};

import type { Meta, StoryObj } from '@storybook/react';
import { useState } from 'react';
import { BillingAddress } from '../BillingAddress';
import type { BillingAddressData } from '../../../types/payment';
import { DEFAULT_BILLING_ADDRESS } from '../../../types/payment';

const meta: Meta<typeof BillingAddress> = {
  title: 'Payments/BillingAddress',
  component: BillingAddress,
  tags: ['autodocs'],
  parameters: {
    layout: 'padded',
    docs: {
      description: {
        component:
          'A billing address form component with validation support and accessibility features.',
      },
    },
  },
  argTypes: {
    disabled: {
      control: 'boolean',
      description: 'Disable all form fields',
    },
    className: {
      control: 'text',
      description: 'Additional CSS classes',
    },
  },
};

export default meta;
type Story = StoryObj<typeof BillingAddress>;

// Wrapper component to handle state
function BillingAddressWrapper(
  props: Partial<React.ComponentProps<typeof BillingAddress>>
) {
  const [address, setAddress] = useState<BillingAddressData>(
    props.value || DEFAULT_BILLING_ADDRESS
  );

  return <BillingAddress value={address} onChange={setAddress} {...props} />;
}

export const Default: Story = {
  render: (args) => <BillingAddressWrapper {...args} />,
  args: {},
};

export const WithPrefilledData: Story = {
  render: (args) => <BillingAddressWrapper {...args} />,
  args: {
    value: {
      name: 'John Doe',
      email: 'john@example.com',
      phone: '+1 (555) 123-4567',
      line1: '123 Main Street',
      line2: 'Apt 4B',
      city: 'San Francisco',
      state: 'CA',
      postalCode: '94102',
      country: 'US',
    },
  },
};

export const WithErrors: Story = {
  render: (args) => <BillingAddressWrapper {...args} />,
  args: {
    errors: {
      name: 'Name is required',
      email: 'Please enter a valid email address',
      line1: 'Street address is required',
      city: 'City is required',
      state: 'State is required',
      postalCode: 'Please enter a valid ZIP code',
    },
  },
};

export const Disabled: Story = {
  render: (args) => <BillingAddressWrapper {...args} />,
  args: {
    disabled: true,
    value: {
      name: 'John Doe',
      email: 'john@example.com',
      phone: '+1 (555) 123-4567',
      line1: '123 Main Street',
      line2: 'Apt 4B',
      city: 'San Francisco',
      state: 'CA',
      postalCode: '94102',
      country: 'US',
    },
  },
};

export const NonUSCountry: Story = {
  render: (args) => <BillingAddressWrapper {...args} />,
  args: {
    value: {
      ...DEFAULT_BILLING_ADDRESS,
      country: 'GB',
      state: '',
    },
  },
  parameters: {
    docs: {
      description: {
        story:
          'For non-US countries, the state field becomes a text input instead of a dropdown.',
      },
    },
  },
};

export const PartiallyFilled: Story = {
  render: (args) => <BillingAddressWrapper {...args} />,
  args: {
    value: {
      ...DEFAULT_BILLING_ADDRESS,
      name: 'Jane Smith',
      email: 'jane@company.com',
    },
    errors: {
      line1: 'Street address is required',
      city: 'City is required',
    },
  },
};

export const MobileView: Story = {
  render: (args) => <BillingAddressWrapper {...args} />,
  args: {
    value: DEFAULT_BILLING_ADDRESS,
  },
  parameters: {
    viewport: {
      defaultViewport: 'mobile1',
    },
    docs: {
      description: {
        story: 'The form is responsive and stacks fields on mobile viewports.',
      },
    },
  },
};

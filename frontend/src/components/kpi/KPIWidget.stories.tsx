import type { Meta, StoryObj } from '@storybook/react';
import KPIWidget from './KPIWidget';
import KPIGrid from './KPIGrid';
import { KPIData } from '../../types/kpi';

const meta: Meta<typeof KPIWidget> = {
  title: 'Components/KPI/KPIWidget',
  component: KPIWidget,
  parameters: {
    layout: 'centered',
    docs: {
      description: {
        component: `
A flexible KPI (Key Performance Indicator) widget component for displaying metrics
with support for multiple formats, trends, targets, and real-time updates.

## Features
- Multiple display formats (currency, percentage, compact, duration)
- Trend indicators with color coding
- Target progress visualization
- Sparkline mini-charts
- Loading and error states
- Real-time update capability
- Responsive sizing
- Keyboard accessible
        `,
      },
    },
  },
  tags: ['autodocs'],
  argTypes: {
    size: {
      control: 'select',
      options: ['sm', 'md', 'lg', 'xl'],
      description: 'Size variant of the widget',
    },
    showTrend: {
      control: 'boolean',
      description: 'Show/hide trend indicator',
    },
    showTarget: {
      control: 'boolean',
      description: 'Show/hide target progress bar',
    },
    showSparkline: {
      control: 'boolean',
      description: 'Show/hide sparkline chart',
    },
    loading: {
      control: 'boolean',
      description: 'Show loading skeleton',
    },
    animate: {
      control: 'boolean',
      description: 'Animate value changes',
    },
    compact: {
      control: 'boolean',
      description: 'Hide label for compact display',
    },
    realTime: {
      control: 'boolean',
      description: 'Show real-time indicator',
    },
  },
};

export default meta;
type Story = StoryObj<typeof KPIWidget>;

// Sample KPI data
const revenueKPI: KPIData = {
  id: 'revenue',
  label: 'Total Revenue',
  value: 125000,
  previousValue: 100000,
  target: 150000,
  format: 'currency',
  currency: 'USD',
  description: 'Total revenue for the current month',
  status: 'success',
  icon: '\ud83d\udcb0',
  lastUpdated: new Date().toISOString(),
};

const conversionKPI: KPIData = {
  id: 'conversion',
  label: 'Conversion Rate',
  value: 3.45,
  previousValue: 2.8,
  format: 'percentage',
  decimals: 2,
  description: 'Visitor to customer conversion rate',
  status: 'success',
  icon: '\ud83d\udcc8',
};

const customersKPI: KPIData = {
  id: 'customers',
  label: 'Active Customers',
  value: 12500,
  previousValue: 11800,
  format: 'compact',
  description: 'Total active customers',
  icon: '\ud83d\udc65',
};

const responseTimeKPI: KPIData = {
  id: 'response-time',
  label: 'Avg Response Time',
  value: 125,
  previousValue: 145,
  format: 'duration',
  description: 'Average customer support response time',
  status: 'info',
  icon: '\u23f1\ufe0f',
};

const sparklineData = [85, 92, 88, 95, 91, 98, 94, 102, 99, 108, 105, 115, 125];

// Stories
export const Default: Story = {
  args: {
    data: revenueKPI,
    showTrend: true,
  },
};

export const WithAllFeatures: Story = {
  args: {
    data: {
      ...revenueKPI,
      target: 150000,
    },
    showTrend: true,
    showTarget: true,
    showSparkline: true,
    sparklineData,
    realTime: true,
  },
};

export const CurrencyFormat: Story = {
  args: {
    data: revenueKPI,
    showTrend: true,
  },
};

export const PercentageFormat: Story = {
  args: {
    data: conversionKPI,
    showTrend: true,
  },
};

export const CompactFormat: Story = {
  args: {
    data: customersKPI,
    showTrend: true,
  },
};

export const DurationFormat: Story = {
  args: {
    data: responseTimeKPI,
    showTrend: true,
  },
};

export const WithSparkline: Story = {
  args: {
    data: revenueKPI,
    showSparkline: true,
    sparklineData,
  },
};

export const WithTargetProgress: Story = {
  args: {
    data: {
      ...revenueKPI,
      target: 150000,
    },
    showTarget: true,
    showTrend: true,
  },
};

export const PositiveTrend: Story = {
  args: {
    data: {
      ...revenueKPI,
      value: 150000,
      previousValue: 100000,
    },
    showTrend: true,
  },
};

export const NegativeTrend: Story = {
  args: {
    data: {
      ...revenueKPI,
      value: 80000,
      previousValue: 100000,
      status: 'error',
    },
    showTrend: true,
  },
};

export const NoTrend: Story = {
  args: {
    data: {
      ...revenueKPI,
      previousValue: undefined,
    },
    showTrend: true,
  },
};

export const Loading: Story = {
  args: {
    data: revenueKPI,
    loading: true,
  },
};

export const Error: Story = {
  args: {
    data: revenueKPI,
    error: 'Failed to load KPI data. Please try again.',
  },
};

export const SmallSize: Story = {
  args: {
    data: revenueKPI,
    size: 'sm',
    showTrend: true,
  },
};

export const MediumSize: Story = {
  args: {
    data: revenueKPI,
    size: 'md',
    showTrend: true,
  },
};

export const LargeSize: Story = {
  args: {
    data: revenueKPI,
    size: 'lg',
    showTrend: true,
  },
};

export const ExtraLargeSize: Story = {
  args: {
    data: revenueKPI,
    size: 'xl',
    showTrend: true,
  },
};

export const Clickable: Story = {
  args: {
    data: revenueKPI,
    showTrend: true,
    onClick: () => alert('KPI clicked!'),
  },
};

export const SuccessStatus: Story = {
  args: {
    data: { ...revenueKPI, status: 'success' },
    showTrend: true,
  },
};

export const WarningStatus: Story = {
  args: {
    data: {
      ...revenueKPI,
      value: 95000,
      status: 'warning',
    },
    showTrend: true,
  },
};

export const ErrorStatus: Story = {
  args: {
    data: {
      ...revenueKPI,
      value: 75000,
      previousValue: 100000,
      status: 'error',
    },
    showTrend: true,
  },
};

export const CompactMode: Story = {
  args: {
    data: revenueKPI,
    compact: true,
    showTrend: true,
  },
};

export const RealTimeIndicator: Story = {
  args: {
    data: revenueKPI,
    realTime: true,
    showTrend: true,
  },
};

export const WithCustomIcon: Story = {
  args: {
    data: {
      ...revenueKPI,
      icon: '\ud83d\ude80',
    },
    showTrend: true,
  },
};

export const WithPrefixAndSuffix: Story = {
  args: {
    data: {
      id: 'custom',
      label: 'Orders',
      value: 1250,
      previousValue: 1100,
      format: 'number',
      prefix: '#',
      suffix: ' orders',
    },
    showTrend: true,
  },
};

// KPI Grid stories
export const Grid: StoryObj<typeof KPIGrid> = {
  render: () => {
    const kpis: KPIData[] = [
      {
        id: 'revenue',
        label: 'Revenue',
        value: 125000,
        previousValue: 100000,
        format: 'currency',
        currency: 'USD',
        status: 'success',
        icon: '\ud83d\udcb0',
      },
      {
        id: 'customers',
        label: 'Customers',
        value: 2450,
        previousValue: 2100,
        format: 'number',
        status: 'success',
        icon: '\ud83d\udc65',
      },
      {
        id: 'orders',
        label: 'Orders',
        value: 856,
        previousValue: 920,
        format: 'number',
        status: 'warning',
        icon: '\ud83d\udce6',
      },
      {
        id: 'conversion',
        label: 'Conversion',
        value: 3.45,
        previousValue: 2.8,
        format: 'percentage',
        decimals: 2,
        status: 'success',
        icon: '\ud83d\udcc8',
      },
    ];

    return (
      <div className="w-full max-w-4xl">
        <KPIGrid kpis={kpis} columns={4} showTrends gap="md" />
      </div>
    );
  },
  parameters: {
    layout: 'padded',
  },
};

export const GridLoading: StoryObj<typeof KPIGrid> = {
  render: () => (
    <div className="w-full max-w-4xl">
      <KPIGrid kpis={[]} columns={4} loading />
    </div>
  ),
  parameters: {
    layout: 'padded',
  },
};

export const GridEmpty: StoryObj<typeof KPIGrid> = {
  render: () => (
    <div className="w-full max-w-4xl">
      <KPIGrid kpis={[]} columns={4} />
    </div>
  ),
  parameters: {
    layout: 'padded',
  },
};

export const GridError: StoryObj<typeof KPIGrid> = {
  render: () => (
    <div className="w-full max-w-4xl">
      <KPIGrid kpis={[]} error="Failed to load dashboard KPIs" />
    </div>
  ),
  parameters: {
    layout: 'padded',
  },
};

// Dashboard example with multiple KPIs
export const DashboardExample: Story = {
  render: () => {
    const dashboardKPIs: KPIData[] = [
      {
        id: 'mrr',
        label: 'Monthly Revenue',
        value: 84500,
        previousValue: 78200,
        target: 100000,
        format: 'currency',
        currency: 'USD',
        status: 'success',
        icon: '\ud83d\udcb5',
        category: 'finance',
      },
      {
        id: 'arr',
        label: 'Annual Revenue',
        value: 1014000,
        previousValue: 938400,
        format: 'currency',
        currency: 'USD',
        status: 'success',
        icon: '\ud83d\udcca',
        category: 'finance',
      },
      {
        id: 'active-jobs',
        label: 'Active Jobs',
        value: 127,
        previousValue: 115,
        format: 'number',
        status: 'info',
        icon: '\ud83d\udee0\ufe0f',
        category: 'operations',
      },
      {
        id: 'completion-rate',
        label: 'Job Completion',
        value: 94.5,
        previousValue: 92.1,
        format: 'percentage',
        decimals: 1,
        status: 'success',
        icon: '\u2705',
        category: 'operations',
      },
      {
        id: 'avg-ticket',
        label: 'Avg Ticket Value',
        value: 285,
        previousValue: 268,
        format: 'currency',
        currency: 'USD',
        status: 'success',
        icon: '\ud83c\udfab',
        category: 'sales',
      },
      {
        id: 'new-customers',
        label: 'New Customers',
        value: 48,
        previousValue: 52,
        format: 'number',
        status: 'warning',
        icon: '\ud83c\udf1f',
        category: 'sales',
      },
      {
        id: 'churn-rate',
        label: 'Churn Rate',
        value: 2.3,
        previousValue: 2.8,
        format: 'percentage',
        decimals: 1,
        status: 'success',
        icon: '\ud83d\udcc9',
        category: 'customer',
      },
      {
        id: 'nps',
        label: 'NPS Score',
        value: 72,
        previousValue: 68,
        format: 'number',
        suffix: ' pts',
        status: 'success',
        icon: '\u2b50',
        category: 'customer',
      },
    ];

    return (
      <div className="w-full max-w-6xl p-4">
        <h2 className="text-2xl font-bold mb-6">Business Dashboard</h2>
        <KPIGrid kpis={dashboardKPIs} columns={4} showTrends gap="md" />
      </div>
    );
  },
  parameters: {
    layout: 'fullscreen',
    docs: {
      description: {
        story:
          'Example of a complete dashboard with multiple KPI widgets in a grid layout.',
      },
    },
  },
};

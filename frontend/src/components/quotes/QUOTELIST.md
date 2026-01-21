# QuoteList Component

A comprehensive quote list component with filtering, sorting, pagination, and quick actions.

## Features

- **Data Fetching**: Uses React Query for efficient server state management
- **Advanced Table**: Built with React Table (TanStack Table) for sorting and pagination
- **Filtering**: Status-based filtering (All, Draft, Sent, Accepted)
- **Pagination**: Support for 25, 50, or 100 items per page
- **Quick Actions**: Contextual menu for each quote (Edit, Delete, Send, Convert)
- **Loading States**: Animated loading spinner with status text
- **Error Handling**: User-friendly error messages with retry functionality
- **Responsive Design**: Mobile-friendly with Tailwind CSS
- **Accessibility**: ARIA labels and keyboard navigation support

## Installation

The component requires the following dependencies:

```bash
npm install @tanstack/react-query @tanstack/react-table axios
```

## Usage

### Basic Usage

```tsx
import { QuoteList } from './components/quotes';

function QuotesPage() {
  return (
    <div className="container mx-auto p-6">
      <h1 className="text-2xl font-bold mb-6">Quotes</h1>
      <QuoteList />
    </div>
  );
}
```

### With Action Handlers

```tsx
import { QuoteList } from './components/quotes';
import { Quote } from './types/quote';

function QuotesPage() {
  const handleEdit = (quote: Quote) => {
    // Navigate to edit page or open modal
    router.push(`/quotes/${quote.id}/edit`);
  };

  const handleDelete = async (quote: Quote) => {
    if (confirm(`Delete quote ${quote.quote_number}?`)) {
      await quoteService.deleteQuote(quote.id);
      // The list will automatically refetch
    }
  };

  const handleConvert = (quote: Quote) => {
    // Convert accepted quote to job
    router.push(`/jobs/create?quoteId=${quote.id}`);
  };

  return (
    <QuoteList
      onEdit={handleEdit}
      onDelete={handleDelete}
      onConvert={handleConvert}
    />
  );
}
```

### Customer-Specific Quotes

```tsx
import { QuoteList } from './components/quotes';

function CustomerQuotes({ customerId }: { customerId: string }) {
  return (
    <div>
      <h2 className="text-xl font-semibold mb-4">Customer Quotes</h2>
      <QuoteList customerId={customerId} />
    </div>
  );
}
```

### With React Query Provider

The component requires a React Query provider in your app:

```tsx
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
      staleTime: 5 * 60 * 1000, // 5 minutes
    },
  },
});

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <YourApp />
    </QueryClientProvider>
  );
}
```

## Props

| Prop         | Type                     | Required | Default | Description                             |
| ------------ | ------------------------ | -------- | ------- | --------------------------------------- |
| `onEdit`     | `(quote: Quote) => void` | No       | -       | Callback when edit action is clicked    |
| `onDelete`   | `(quote: Quote) => void` | No       | -       | Callback when delete action is clicked  |
| `onConvert`  | `(quote: Quote) => void` | No       | -       | Callback when convert action is clicked |
| `customerId` | `string`                 | No       | -       | Filter quotes by specific customer      |

## Features in Detail

### Filtering

The component provides quick filters for common quote statuses:

- **All**: Shows all quotes (no status filter)
- **Draft**: Shows quotes in draft status
- **Sent**: Shows quotes sent to customers
- **Accepted**: Shows quotes accepted by customers

Filters automatically reset pagination to page 1 when changed.

### Sorting

Columns support sorting by clicking on the column header. The component maintains sorting state and communicates it to the backend via `sort_by` and `sort_order` parameters.

Default sorting: `created_at` descending (newest first)

### Pagination

- **Page Size Options**: 25, 50, or 100 items per page
- **Navigation**: Previous/Next buttons and direct page number selection
- **Information**: Shows "Showing X to Y of Z results"
- **Smart Ellipsis**: Shows ellipsis for large page counts

### Quick Actions Menu

Each quote row has an actions menu button (three dots) that shows contextual actions:

**For Draft Quotes:**

- Edit Quote
- Send to Customer
- Delete Quote

**For Sent Quotes:**

- Edit Quote
- Delete Quote

**For Accepted Quotes:**

- Edit Quote
- Convert to Job (if `onConvert` is provided)
- Delete Quote

**For Rejected/Expired Quotes:**

- Edit Quote
- Delete Quote

### Status Badges

Status is displayed as colored badges:

- **Draft**: Gray badge
- **Sent**: Blue badge
- **Accepted**: Green badge
- **Rejected**: Red badge
- **Expired**: Yellow badge

### Loading States

The component shows a centered loading spinner with animated rotation and "Loading quotes..." text while data is being fetched.

### Error States

When an error occurs:

- Red error banner with icon
- Error message displayed
- "Try again" button to retry the request
- Maintains user's current filters

## Table Columns

| Column   | Description             | Format                                 |
| -------- | ----------------------- | -------------------------------------- |
| Quote #  | Quote number            | Monospace font for easy reading        |
| Customer | Customer name and email | Name in bold, email in small gray text |
| Date     | Quote creation date     | Format: "Jan 15, 2024"                 |
| Amount   | Total quote amount      | Currency format: "$1,234.56"           |
| Status   | Quote status            | Colored badge                          |
| Actions  | Quick actions menu      | Three-dot menu button                  |

## API Integration

The component uses the `quoteService` for API calls:

```typescript
// Fetch quotes with filters
const response = await quoteService.getQuotes({
  customer_id: 'cust-123',
  status: 'draft',
  page: 1,
  page_size: 25,
  sort_by: 'created_at',
  sort_order: 'desc',
});

// Send quote to customer
await quoteService.sendQuote(quoteId);
```

### Expected API Response

```typescript
{
  "quotes": [
    {
      "id": "uuid",
      "quote_number": "Q-2024-001",
      "customer_id": "uuid",
      "customer_name": "John Doe",
      "customer_email": "john@example.com",
      "status": "draft",
      "subtotal": "1000.00",
      "tax_rate": "0.0825",
      "tax_amount": "82.50",
      "total": "1082.50",
      "valid_until": "2024-12-31T23:59:59Z",
      "notes": "",
      "items": [...],
      "created_at": "2024-01-15T10:00:00Z",
      "updated_at": "2024-01-15T10:00:00Z"
    }
  ],
  "pagination": {
    "page": 1,
    "page_size": 25,
    "total": 100,
    "total_pages": 4
  }
}
```

## React Query Integration

The component uses React Query with the following configuration:

- **Query Key**: `['quotes', filters]` - Automatically refetches when filters change
- **Keep Previous Data**: Maintains previous results while fetching new data
- **Retry**: Disabled (uses default React Query config)
- **Refetch**: Automatic refetch after mutations

### Manual Refetch

```tsx
const queryClient = useQueryClient();

// Refetch quotes after creating/updating/deleting
await queryClient.invalidateQueries(['quotes']);
```

## Styling

The component uses Tailwind CSS for styling with the following design system:

### Colors

- Primary: Blue (600)
- Success: Green (100, 800)
- Error: Red (100, 800)
- Warning: Yellow (100, 800)
- Neutral: Gray (50, 100, 200, 500, 700, 900)

### Responsive Breakpoints

- Mobile: Default (< 640px)
- Desktop: `sm:` breakpoint (≥ 640px)

## Accessibility

The component follows accessibility best practices:

- **ARIA Labels**: All interactive elements have proper labels
- **Keyboard Navigation**: Full keyboard support for all actions
- **Screen Readers**: Status messages and loading states announced
- **Focus Management**: Clear focus indicators on all interactive elements
- **Semantic HTML**: Proper table structure with thead/tbody

## Performance Considerations

- **React Query Caching**: Queries are cached and shared across components
- **Keep Previous Data**: Smooth transitions when changing pages/filters
- **Memoization**: Column definitions are memoized to prevent re-renders
- **Virtual Scrolling**: Not needed for current page sizes (consider if using 1000+ items)

## Testing

The component includes comprehensive unit tests:

```bash
# Run tests
npm test QuoteList.test.tsx

# Run with coverage
npm test -- --coverage QuoteList.test.tsx
```

### Test Coverage

- Loading states
- Error states and retry
- Empty states
- Quote rendering with all fields
- Status filtering (All, Draft, Sent, Accepted)
- Pagination navigation
- Page size changes
- Quick actions menu
- Customer filtering
- Menu open/close behavior

## Examples

### Complete Page Implementation

```tsx
import { useState } from 'react';
import { QuoteList } from './components/quotes';
import { Quote } from './types/quote';
import { Modal } from './components/ui';

function QuotesPage() {
  const [editingQuote, setEditingQuote] = useState<Quote | null>(null);

  const handleEdit = (quote: Quote) => {
    setEditingQuote(quote);
  };

  const handleDelete = async (quote: Quote) => {
    if (!confirm(`Delete quote ${quote.quote_number}?`)) return;

    try {
      await quoteService.deleteQuote(quote.id);
      // React Query will automatically refetch
    } catch (error) {
      alert('Failed to delete quote');
    }
  };

  const handleConvert = async (quote: Quote) => {
    try {
      // Create job from quote
      const job = await jobService.createFromQuote(quote.id);
      router.push(`/jobs/${job.id}`);
    } catch (error) {
      alert('Failed to convert quote');
    }
  };

  return (
    <div className="container mx-auto p-6">
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-2xl font-bold">Quotes</h1>
        <button
          onClick={() => router.push('/quotes/create')}
          className="rounded-md bg-blue-600 px-4 py-2 text-white hover:bg-blue-700"
        >
          New Quote
        </button>
      </div>

      <QuoteList
        onEdit={handleEdit}
        onDelete={handleDelete}
        onConvert={handleConvert}
      />

      {editingQuote && (
        <Modal onClose={() => setEditingQuote(null)}>
          <QuoteEditForm
            quote={editingQuote}
            onSave={() => setEditingQuote(null)}
          />
        </Modal>
      )}
    </div>
  );
}
```

### With Statistics Dashboard

```tsx
import { QuoteList } from './components/quotes';
import { useQuery } from '@tanstack/react-query';
import { quoteService } from './services/quoteService';

function QuotesDashboard() {
  const { data: stats } = useQuery({
    queryKey: ['quote-stats'],
    queryFn: () => quoteService.getStats(),
  });

  return (
    <div className="space-y-6">
      {/* Statistics Cards */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-4">
        <div className="rounded-lg bg-white p-4 shadow">
          <div className="text-sm text-gray-500">Total Quotes</div>
          <div className="text-2xl font-bold">{stats?.total || 0}</div>
        </div>
        <div className="rounded-lg bg-white p-4 shadow">
          <div className="text-sm text-gray-500">Draft</div>
          <div className="text-2xl font-bold">{stats?.draft || 0}</div>
        </div>
        <div className="rounded-lg bg-white p-4 shadow">
          <div className="text-sm text-gray-500">Sent</div>
          <div className="text-2xl font-bold">{stats?.sent || 0}</div>
        </div>
        <div className="rounded-lg bg-white p-4 shadow">
          <div className="text-sm text-gray-500">Accepted</div>
          <div className="text-2xl font-bold text-green-600">
            {stats?.accepted || 0}
          </div>
        </div>
      </div>

      {/* Quote List */}
      <QuoteList />
    </div>
  );
}
```

## Browser Support

- Chrome/Edge: Latest 2 versions
- Firefox: Latest 2 versions
- Safari: Latest 2 versions
- Mobile Safari: iOS 12+
- Chrome Mobile: Latest

## License

MIT

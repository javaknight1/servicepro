# QuoteDetail Component

A comprehensive quote creation and editing component with form management, real-time calculations, PDF generation, and auto-save functionality.

## Features

- **Form Management**: React Hook Form with Zod validation
- **Real-time Calculations**: Automatic subtotal, tax, and total calculations
- **Auto-save**: Debounced auto-save with visual feedback
- **PDF Generation**: Generate, preview, and download quote PDFs
- **Line Item Management**: Integrated LineItemList component
- **Tax Calculation**: Support for tax rates and exemptions
- **Validation**: Comprehensive field validation with error messages
- **Responsive Design**: Mobile-friendly with Tailwind CSS

## Installation

Required dependencies:

```bash
npm install react-hook-form @hookform/resolvers zod @tanstack/react-query @react-pdf/renderer
```

## Usage

### Creating a New Quote

```tsx
import { QuoteDetail } from './components/quotes';

function NewQuotePage() {
  const handleSave = (quote: Quote) => {
    console.log('Quote saved:', quote);
    router.push(`/quotes/${quote.id}`);
  };

  const handleCancel = () => {
    router.push('/quotes');
  };

  return (
    <div className="container mx-auto p-6">
      <QuoteDetail
        onSave={handleSave}
        onCancel={handleCancel}
        enableAutoSave={false} // Disabled for new quotes
      />
    </div>
  );
}
```

### Editing an Existing Quote

```tsx
import { QuoteDetail } from './components/quotes';
import { useParams } from 'react-router-dom';

function EditQuotePage() {
  const { id } = useParams();

  const handleSave = (quote: Quote) => {
    console.log('Quote updated:', quote);
  };

  return (
    <div className="container mx-auto p-6">
      <QuoteDetail
        quoteId={id}
        onSave={handleSave}
        enableAutoSave={true} // Auto-save enabled for existing quotes
      />
    </div>
  );
}
```

### With React Query Provider

The component requires a React Query provider:

```tsx
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';

const queryClient = new QueryClient();

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <YourApp />
    </QueryClientProvider>
  );
}
```

## Props

| Prop             | Type                     | Required | Default | Description                                             |
| ---------------- | ------------------------ | -------- | ------- | ------------------------------------------------------- |
| `quoteId`        | `string`                 | No       | -       | ID of quote to edit. If not provided, creates new quote |
| `onSave`         | `(quote: Quote) => void` | No       | -       | Callback when quote is saved                            |
| `onCancel`       | `(void) => void`         | No       | -       | Callback when cancel button is clicked                  |
| `enableAutoSave` | `boolean`                | No       | `true`  | Enable/disable auto-save functionality                  |

## Form Sections

### 1. Customer Information

Required fields:

- **Customer Name** (required)
- **Email** (required, validated)
- **Phone** (optional, format validated)
- **Address** (optional)
- **City** (optional)
- **State** (optional, 2-letter code)
- **ZIP Code** (optional, format validated)

Example:

```tsx
// Customer data
{
  customer_name: "John Doe",
  customer_email: "john@example.com",
  customer_phone: "(555) 123-4567",
  customer_address: "123 Main St",
  customer_city: "Los Angeles",
  customer_state: "CA",
  customer_zip: "90210"
}
```

### 2. Line Items

Uses the integrated `LineItemList` component:

- Add, edit, delete line items
- Drag-and-drop reordering
- Real-time total calculation
- Minimum 1 line item required
- Maximum 100 line items

Each line item has:

- Description (required, max 500 chars)
- Quantity (required, positive number)
- Unit Price (required, non-negative)
- Total (calculated automatically)

### 3. Tax Information

- **Tax ZIP Code**: For automatic tax rate lookup
- **Tax Rate**: Decimal format (e.g., 0.0825 for 8.25%)
- **Tax Exemption Type**: None, Non-Profit, Government, Resale, Manufacturing, Educational
- **Exemption Number**: Required if exemption type is selected

Tax exemption validation:

- **Non-Profit**: EIN format (12-3456789)
- **Government**: Any format
- **Resale**: Alphanumeric, 6-15 characters
- **Manufacturing/Educational**: Non-empty string

### 4. Terms & Conditions

- **Valid Until** (required): Future date
- **Payment Terms** (optional): e.g., "Net 30"
- **Delivery Timeline** (optional): e.g., "2-3 weeks"
- **Warranty Information** (optional): Max 500 characters

### 5. Notes

- Optional additional notes or special instructions
- Maximum 2000 characters
- Displayed on PDF quote

## Real-time Calculations

The component automatically calculates:

### Subtotal

Sum of all line item totals:

```
subtotal = Σ(quantity × unit_price)
```

### Tax Amount

Based on tax rate and exemption status:

```
tax_amount = exemption ? 0 : subtotal × tax_rate
```

### Total

```
total = subtotal + tax_amount
```

Calculations update immediately when:

- Line items change
- Tax rate changes
- Exemption type changes

## Auto-save

Auto-save features:

- **Debounced**: 2-second delay after last change
- **Visual Feedback**: Shows "Saving..." or "Last saved: X ago"
- **Only for Existing Quotes**: Auto-save only works for quotes with an ID
- **Error Handling**: Displays errors if save fails

Auto-save status display:

```tsx
{
  autoSaveStatus.isSaving ? (
    <span>Saving...</span>
  ) : (
    <span>Last saved: {formatLastSaved(autoSaveStatus.lastSaved)}</span>
  );
}
```

Time formats:

- "Just now" - Less than 10 seconds
- "X seconds ago" - 10-59 seconds
- "1 minute ago" - Exactly 1 minute
- "X minutes ago" - 2-59 minutes
- "12:34 PM" - 1+ hours

## PDF Generation

### Download PDF

```tsx
<button onClick={handleDownloadPDF}>Download PDF</button>
```

The PDF includes:

- Company header (configurable)
- Quote number and date
- Customer information
- Line items table
- Subtotal, tax, total
- Notes
- Terms and validity date

### Preview PDF

Opens PDF in new browser tab:

```tsx
<button onClick={handlePreviewPDF}>Preview PDF</button>
```

### PDF Styling

The PDF uses professional styling:

- Company logo and branding
- Clean table layout
- Proper formatting for currency
- Page footer with validity date

### Customization

Company information can be customized:

```tsx
const companyInfo = {
  name: 'ServicePro',
  address: '123 Business St, City, ST 12345',
  phone: '(555) 123-4567',
  email: 'info@servicepro.com',
};

await downloadQuotePDF(quote, companyInfo);
```

## Validation

### Field Validation Rules

**Customer Name:**

- Required
- Max 200 characters

**Email:**

- Required
- Valid email format
- Max 255 characters

**Phone:**

- Optional
- Format: digits, spaces, dashes, parentheses, plus signs
- Min 10 characters
- Max 20 characters

**State:**

- Optional
- Exactly 2 uppercase letters
- Example: "CA", "NY"

**ZIP Code:**

- Optional
- Format: 12345 or 12345-6789

**Line Items:**

- Minimum 1 item
- Maximum 100 items
- Each item validated individually

**Valid Until:**

- Required
- Must be future date

**Tax Exemption:**

- Exemption number required if type is not "none"

### Error Display

Errors are displayed inline below each field:

```tsx
{
  errors.customer_name && (
    <p className="mt-1 text-sm text-red-600">{errors.customer_name.message}</p>
  );
}
```

Form-level errors (e.g., save failures) are displayed at the top:

```tsx
<div className="rounded-md bg-red-50 p-4">
  <h3>Failed to save quote</h3>
  <p>{error.message}</p>
</div>
```

## API Integration

### Creating a Quote

```typescript
POST /api/v1/quotes
Content-Type: application/json

{
  "customer_name": "John Doe",
  "customer_email": "john@example.com",
  "items": [
    {
      "description": "Service",
      "quantity": 1,
      "unit_price": 100,
      "total": 100
    }
  ],
  "tax_rate": "0.0825",
  "valid_until": "2024-12-31T23:59:59Z"
}
```

Response:

```typescript
{
  "id": "uuid",
  "quote_number": "Q-2024-001",
  "customer_name": "John Doe",
  // ... all other fields
  "created_at": "2024-01-15T10:00:00Z",
  "updated_at": "2024-01-15T10:00:00Z"
}
```

### Updating a Quote

```typescript
PUT /api/v1/quotes/:id
Content-Type: application/json

{
  // Updated fields
}
```

### Sending to Customer

```typescript
POST /api/v1/quotes/:id/send
```

Changes quote status from `draft` to `sent`.

## State Management

The component uses React Hook Form for state management:

```tsx
const {
  register, // Register input fields
  control, // For controlled components (LineItemList)
  handleSubmit, // Form submission handler
  watch, // Watch field values for calculations
  setValue, // Programmatically set values
  formState, // Form state (errors, isDirty, etc.)
  reset, // Reset form to default/loaded values
} = useForm<QuoteFormData>({
  resolver: zodResolver(quoteFormSchema),
  defaultValues: getDefaultQuoteValues(),
});
```

### Watching Values

```tsx
const items = watch('items');
const taxRate = watch('tax_rate');
const exemptionType = watch('tax_exemption_type');
```

### Setting Values

```tsx
setValue('subtotal', subtotal.toFixed(2), {
  shouldValidate: false,
  shouldDirty: true,
});
```

## Custom Hooks

### useQuoteCalculations

Real-time calculation hook:

```tsx
const { subtotal, taxAmount, total } = useQuoteCalculations(watch, setValue);
```

Returns:

- `subtotal`: Formatted string
- `taxAmount`: Formatted string
- `total`: Formatted string

### useAutoSave

Auto-save hook with debouncing:

```tsx
const autoSaveStatus = useAutoSave(
  watch,
  async (data) => {
    await quoteService.updateQuote(quoteId, data);
  },
  {
    enabled: true,
    delay: 2000,
    onSaveSuccess: (data) => console.log('Saved:', data),
    onSaveError: (error) => console.error('Error:', error),
  }
);
```

Returns:

- `isSaving`: boolean
- `lastSaved`: Date | null
- `error`: Error | null

## Styling

The component uses Tailwind CSS with a professional design:

### Color Scheme

- Primary: Blue (600/700)
- Success: Green (600/700)
- Error: Red (50/600/800)
- Neutral: Gray (100/200/300/500/700/900)

### Layout

- **Desktop**: 3-column grid (2 cols main content, 1 col sidebar)
- **Mobile**: Single column, stacked layout

### Form Elements

- Rounded borders
- Shadow on white backgrounds
- Focus states with blue ring
- Proper spacing and padding

## Accessibility

- **Form Labels**: All inputs have associated labels
- **Error Messages**: Linked to inputs via aria-describedby
- **Keyboard Navigation**: Full keyboard support
- **Focus Management**: Proper focus indicators
- **Screen Readers**: Semantic HTML and ARIA labels

## Performance

### Optimizations

1. **Debounced Auto-save**: Prevents excessive API calls
2. **Memoized Calculations**: Only recalculate when dependencies change
3. **Controlled Re-renders**: React Hook Form minimizes re-renders
4. **Lazy PDF Generation**: PDFs generated on-demand, not on every render

### Best Practices

- Use `watch()` selectively to avoid unnecessary re-renders
- Memoize callbacks with `useCallback`
- Use `Controller` for complex components
- Implement proper loading and error states

## Examples

### Complete Implementation

```tsx
import { QuoteDetail } from './components/quotes';
import { useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

function QuotePage() {
  const navigate = useNavigate();
  const { id } = useParams();
  const [showSuccess, setShowSuccess] = useState(false);

  const handleSave = (quote: Quote) => {
    setShowSuccess(true);
    setTimeout(() => {
      navigate(`/quotes/${quote.id}`);
    }, 2000);
  };

  const handleCancel = () => {
    if (confirm('Discard changes?')) {
      navigate('/quotes');
    }
  };

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="container mx-auto p-6">
        {showSuccess && (
          <div className="mb-4 rounded-md bg-green-50 p-4">
            <p className="text-green-800">Quote saved successfully!</p>
          </div>
        )}

        <QuoteDetail
          quoteId={id}
          onSave={handleSave}
          onCancel={handleCancel}
          enableAutoSave={!!id}
        />
      </div>
    </div>
  );
}
```

### With Custom Validation

```tsx
import { QuoteFormData, quoteFormSchema } from './validation';
import { z } from 'zod';

// Extend validation schema
const customSchema = quoteFormSchema.refine(
  (data) => {
    // Custom validation: minimum quote value
    const total = parseFloat(data.total || '0');
    return total >= 100;
  },
  {
    message: 'Quote total must be at least $100',
    path: ['total'],
  }
);

// Use custom schema in form
const form = useForm<QuoteFormData>({
  resolver: zodResolver(customSchema),
});
```

### With Status Workflow

```tsx
function QuoteWorkflow({ quoteId }: { quoteId: string }) {
  const { data: quote } = useQuery(['quote', quoteId]);
  const [canEdit, setCanEdit] = useState(false);

  useEffect(() => {
    // Only allow editing draft or sent quotes
    setCanEdit(quote?.status === 'draft' || quote?.status === 'sent');
  }, [quote]);

  if (!canEdit) {
    return <div>This quote cannot be edited</div>;
  }

  return <QuoteDetail quoteId={quoteId} />;
}
```

## Troubleshooting

### Auto-save not working

Check:

1. `enableAutoSave` prop is `true`
2. `quoteId` is provided (auto-save only works for existing quotes)
3. Form has been modified (`isDirty` is true)
4. No validation errors preventing save

### Calculations not updating

Check:

1. Form fields are registered with `register()` or `Controller`
2. `watch()` is used to observe field changes
3. Dependencies in `useMemo` are correct

### PDF generation fails

Check:

1. @react-pdf/renderer is installed
2. Quote data is complete and valid
3. No circular references in quote object
4. Browser supports blob URLs

### Validation errors not showing

Check:

1. Form submitted with `handleSubmit(onSubmit)`
2. Zod schema matches form data structure
3. Error messages rendered in JSX
4. Field names match schema exactly

## Browser Support

- Chrome/Edge: Latest 2 versions
- Firefox: Latest 2 versions
- Safari: Latest 2 versions
- Mobile browsers: iOS 12+, Android 9+

## License

MIT

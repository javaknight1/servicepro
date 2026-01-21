# Line Item Management Components

A comprehensive set of React components for managing quote line items with TypeScript, Tailwind CSS, and full validation.

## Components

### LineItemList

Main container component that orchestrates all line item functionality.

**Features:**

- Add/edit/delete items
- Drag-and-drop reordering (HTML5 native)
- Bulk selection and actions
- Real-time total calculation
- Responsive design

**Usage:**

```tsx
import { LineItemList } from './components/quotes';

function QuoteForm() {
  const [items, setItems] = useState<LineItem[]>([]);
  const [taxRate, setTaxRate] = useState(0.0825);

  return (
    <LineItemList
      items={items}
      taxRate={taxRate}
      onChange={setItems}
      isEditable={true}
      showForm={true}
      showTotals={true}
      showBulkActions={true}
    />
  );
}
```

**Props:**

- `items: LineItem[]` - Array of line items
- `taxRate: number` - Tax rate as decimal (e.g., 0.0825 for 8.25%)
- `onChange: (items: LineItem[]) => void` - Callback when items change
- `isEditable?: boolean` - Enable/disable editing (default: true)
- `showForm?: boolean` - Show add item form (default: true)
- `showTotals?: boolean` - Show total calculator (default: true)
- `showBulkActions?: boolean` - Show bulk action controls (default: true)

---

### LineItemRow

Individual row component with inline editing.

**Features:**

- Inline edit mode
- Real-time validation
- Delete confirmation
- Drag handle for reordering
- Checkbox for bulk selection
- Quick duplicate action

**Usage:**

```tsx
<LineItemRow
  item={item}
  index={0}
  onUpdate={(id, updates) => handleUpdate(id, updates)}
  onDelete={(id) => handleDelete(id)}
  onDuplicate={(id) => handleDuplicate(id)}
  isSelected={false}
  onSelect={(id, selected) => handleSelect(id, selected)}
  isDragging={false}
  isEditable={true}
  dragHandleProps={dragProps}
/>
```

---

### LineItemForm

Form component for adding new line items.

**Features:**

- Real-time validation
- Keyboard shortcuts (Enter to submit, Esc to clear)
- Auto-calculate total preview
- Focus management
- Clear form after submit

**Usage:**

```tsx
<LineItemForm
  onAdd={(formData) => handleAddItem(formData)}
  onCancel={() => handleCancel()}
  initialData={{ description: '', quantity: 1, unit_price: 0 }}
  autoFocus={true}
  showCancel={false}
  buttonText="Add Item"
/>
```

**Keyboard Shortcuts:**

- `Enter` - Submit form
- `Esc` - Clear form or cancel

---

### TotalCalculator

Displays calculated totals with optional breakdown.

**Features:**

- Real-time calculation
- Automatic formatting
- Optional item breakdown
- Empty state handling
- Responsive layout

**Usage:**

```tsx
<TotalCalculator
  items={items}
  taxRate={0.0825}
  className="lg:w-96"
  showBreakdown={true}
/>
```

**Calculations:**

- Subtotal = Sum of (quantity × unit_price) for all items
- Tax Amount = Subtotal × tax_rate
- Total = Subtotal + Tax Amount

---

## TypeScript Types

### LineItem

```typescript
interface LineItem {
  id: string;
  quote_id?: string;
  description: string;
  quantity: number;
  unit_price: number;
  total: number;
  sort_order: number;
}
```

### LineItemFormData

```typescript
interface LineItemFormData {
  description: string;
  quantity: number | string;
  unit_price: number | string;
  sort_order?: number;
}
```

### LineItemValidationErrors

```typescript
interface LineItemValidationErrors {
  description?: string;
  quantity?: string;
  unit_price?: string;
}
```

---

## Validation Rules

### Description

- Required
- Maximum 500 characters
- Must not be empty after trimming

### Quantity

- Required
- Must be a valid number
- Must be greater than 0
- Maximum value: 999,999

### Unit Price

- Required
- Must be a valid number
- Cannot be negative
- Maximum value: 999,999

---

## Drag and Drop

The components use HTML5 native drag-and-drop API for reordering items:

1. Click and hold the drag handle (☰ icon)
2. Drag the row to the desired position
3. Release to reorder

Items are automatically renumbered with sequential `sort_order` values.

---

## Bulk Actions

### Selection

- Click individual checkboxes to select items
- Click header checkbox to select/deselect all
- Selected count is displayed in the toolbar

### Available Actions

- **Duplicate** - Creates copies of selected items with "(Copy)" suffix
- **Delete** - Removes selected items (with confirmation)

---

## Styling

All components use Tailwind CSS with a consistent design system:

**Colors:**

- Primary: Blue (buttons, links, focus states)
- Success: Green (duplicate actions)
- Danger: Red (delete actions, errors)
- Neutral: Gray (borders, backgrounds, text)

**Responsive Breakpoints:**

- Mobile-first design
- Grid layout adjusts for tablet/desktop
- Optimized for touch and mouse input

---

## Testing

Comprehensive test coverage using Vitest and React Testing Library:

```bash
# Run all tests
npm test

# Run tests for specific component
npm test TotalCalculator
npm test LineItemForm
npm test LineItemList

# Run tests in watch mode
npm test -- --watch

# Generate coverage report
npm test -- --coverage
```

**Test Files:**

- `__tests__/TotalCalculator.test.tsx` - 10 test cases
- `__tests__/LineItemForm.test.tsx` - 20 test cases
- `__tests__/LineItemList.test.tsx` - 19 test cases

**Coverage:**

- Unit tests for all components
- Integration tests for user workflows
- Edge case validation
- Accessibility testing

---

## Accessibility

All components follow WCAG 2.1 Level AA guidelines:

- Semantic HTML elements
- ARIA labels and roles
- Keyboard navigation support
- Focus management
- Screen reader friendly
- Sufficient color contrast
- Touch-friendly hit targets (44×44px minimum)

---

## Performance Optimizations

- `useMemo` for expensive calculations
- `useCallback` for stable function references
- Debounced validation
- Optimized re-renders
- Lazy state updates

---

## Browser Support

- Chrome/Edge (latest 2 versions)
- Firefox (latest 2 versions)
- Safari (latest 2 versions)
- Mobile browsers (iOS Safari, Chrome Mobile)

---

## Example: Complete Quote Form

```tsx
import React, { useState } from 'react';
import { LineItemList } from './components/quotes';
import type { LineItem } from './types/quote';

function QuoteEditor() {
  const [items, setItems] = useState<LineItem[]>([]);
  const [taxRate, setTaxRate] = useState(0.0825);

  const handleItemsChange = (newItems: LineItem[]) => {
    setItems(newItems);
    console.log('Items updated:', newItems);
  };

  return (
    <div className="max-w-7xl mx-auto px-4 py-8">
      <h1 className="text-3xl font-bold mb-8">Create Quote</h1>

      {/* Tax Rate Input */}
      <div className="mb-6">
        <label className="block text-sm font-medium text-gray-700 mb-2">
          Tax Rate (%)
        </label>
        <input
          type="number"
          value={taxRate * 100}
          onChange={(e) => setTaxRate(Number(e.target.value) / 100)}
          step="0.01"
          min="0"
          max="100"
          className="px-3 py-2 border border-gray-300 rounded-md"
        />
      </div>

      {/* Line Items */}
      <LineItemList
        items={items}
        taxRate={taxRate}
        onChange={handleItemsChange}
      />

      {/* Submit Button */}
      <div className="mt-8 flex justify-end">
        <button
          onClick={() => console.log('Submit quote', { items, taxRate })}
          className="px-6 py-3 bg-blue-600 text-white rounded-md hover:bg-blue-700"
        >
          Create Quote
        </button>
      </div>
    </div>
  );
}
```

---

## Contributing

When adding features or fixing bugs:

1. Add TypeScript types
2. Write unit tests
3. Update documentation
4. Follow existing code style
5. Test accessibility
6. Verify responsive design

---

## License

MIT

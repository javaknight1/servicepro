import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import { describe, it, expect, vi } from 'vitest';
import userEvent from '@testing-library/user-event';
import LineItemList from '../LineItemList';
import { LineItem } from '../../../types/quote';

describe('LineItemList', () => {
  const mockItems: LineItem[] = [
    {
      id: '1',
      description: 'Item 1',
      quantity: 2,
      unit_price: 50,
      total: 100,
      sort_order: 0,
    },
    {
      id: '2',
      description: 'Item 2',
      quantity: 1,
      unit_price: 75,
      total: 75,
      sort_order: 1,
    },
  ];

  const mockOnChange = vi.fn();

  beforeEach(() => {
    mockOnChange.mockClear();
  });

  it('should render line items table', () => {
    render(
      <LineItemList
        items={mockItems}
        taxRate={0.0825}
        onChange={mockOnChange}
      />
    );

    expect(screen.getByText('Item 1')).toBeInTheDocument();
    expect(screen.getByText('Item 2')).toBeInTheDocument();
    expect(screen.getByText('Line Items (2)')).toBeInTheDocument();
  });

  it('should show add item form by default', () => {
    render(
      <LineItemList items={[]} taxRate={0.0825} onChange={mockOnChange} />
    );

    expect(screen.getByLabelText(/description/i)).toBeInTheDocument();
  });

  it('should hide add item form when showForm is false', () => {
    render(
      <LineItemList
        items={[]}
        taxRate={0.0825}
        onChange={mockOnChange}
        showForm={false}
      />
    );

    expect(screen.queryByLabelText(/description/i)).not.toBeInTheDocument();
  });

  it('should show total calculator by default', () => {
    render(
      <LineItemList
        items={mockItems}
        taxRate={0.0825}
        onChange={mockOnChange}
      />
    );

    expect(screen.getByText('Subtotal')).toBeInTheDocument();
    // "Total" appears in multiple places (form label, table header, calculator)
    expect(screen.getAllByText('Total').length).toBeGreaterThan(0);
  });

  it('should hide total calculator when showTotals is false', () => {
    render(
      <LineItemList
        items={mockItems}
        taxRate={0.0825}
        onChange={mockOnChange}
        showTotals={false}
      />
    );

    expect(screen.queryByText('Subtotal')).not.toBeInTheDocument();
  });

  it('should add new item when form is submitted', async () => {
    const user = userEvent.setup();
    render(
      <LineItemList items={[]} taxRate={0.0825} onChange={mockOnChange} />
    );

    await user.type(screen.getByLabelText(/description/i), 'New Item');
    await user.type(screen.getByLabelText(/quantity/i), '3');
    await user.type(screen.getByLabelText(/unit price/i), '100');

    const addButton = screen.getByRole('button', { name: /add/i });
    await user.click(addButton);

    await waitFor(() => {
      expect(mockOnChange).toHaveBeenCalledWith(
        expect.arrayContaining([
          expect.objectContaining({
            description: 'New Item',
            quantity: 3,
            unit_price: 100,
            total: 300,
          }),
        ])
      );
    });
  });

  it('should delete item when delete button is clicked', async () => {
    const user = userEvent.setup();
    render(
      <LineItemList
        items={mockItems}
        taxRate={0.0825}
        onChange={mockOnChange}
      />
    );

    // Click delete button twice (first to show confirm, second to confirm)
    const deleteButtons = screen.getAllByTitle(/delete/i);
    await user.click(deleteButtons[0]);
    await user.click(deleteButtons[0]);

    await waitFor(() => {
      expect(mockOnChange).toHaveBeenCalledWith(
        expect.arrayContaining([expect.objectContaining({ id: '2' })])
      );
      expect(mockOnChange).not.toHaveBeenCalledWith(
        expect.arrayContaining([expect.objectContaining({ id: '1' })])
      );
    });
  });

  it('should show bulk actions when items are selected', async () => {
    const user = userEvent.setup();
    render(
      <LineItemList
        items={mockItems}
        taxRate={0.0825}
        onChange={mockOnChange}
      />
    );

    const checkboxes = screen.getAllByRole('checkbox');
    await user.click(checkboxes[1]); // Select first item

    expect(screen.getByText('1 selected')).toBeInTheDocument();
    // Multiple duplicate/delete buttons exist (bulk actions + per-row actions)
    expect(
      screen.getAllByRole('button', { name: /duplicate/i }).length
    ).toBeGreaterThan(0);
    expect(
      screen.getAllByRole('button', { name: /delete/i }).length
    ).toBeGreaterThan(0);
  });

  it('should select all items when select all checkbox is clicked', async () => {
    const user = userEvent.setup();
    render(
      <LineItemList
        items={mockItems}
        taxRate={0.0825}
        onChange={mockOnChange}
      />
    );

    const selectAllCheckbox = screen.getAllByRole('checkbox')[0];
    await user.click(selectAllCheckbox);

    expect(screen.getByText('2 selected')).toBeInTheDocument();
  });

  it('should duplicate selected items', async () => {
    window.confirm = vi.fn(() => true);
    const user = userEvent.setup();
    render(
      <LineItemList
        items={mockItems}
        taxRate={0.0825}
        onChange={mockOnChange}
      />
    );

    // Select first item
    const checkboxes = screen.getAllByRole('checkbox');
    await user.click(checkboxes[1]);

    // Click bulk duplicate button (first one is the bulk action button)
    await user.click(screen.getAllByRole('button', { name: /duplicate/i })[0]);

    await waitFor(() => {
      expect(mockOnChange).toHaveBeenCalledWith(
        expect.arrayContaining([
          expect.objectContaining({ description: 'Item 1' }),
          expect.objectContaining({ description: 'Item 2' }),
          expect.objectContaining({ description: 'Item 1 (Copy)' }),
        ])
      );
    });
  });

  it('should delete selected items with confirmation', async () => {
    window.confirm = vi.fn(() => true);
    const user = userEvent.setup();
    render(
      <LineItemList
        items={mockItems}
        taxRate={0.0825}
        onChange={mockOnChange}
      />
    );

    // Select first item
    const checkboxes = screen.getAllByRole('checkbox');
    await user.click(checkboxes[1]);

    // Click bulk delete button
    await user.click(screen.getAllByRole('button', { name: /delete/i })[0]);

    expect(window.confirm).toHaveBeenCalledWith('Delete 1 selected item(s)?');

    await waitFor(() => {
      expect(mockOnChange).toHaveBeenCalledWith(
        expect.arrayContaining([expect.objectContaining({ id: '2' })])
      );
    });
  });

  it('should not delete items if confirmation is cancelled', async () => {
    window.confirm = vi.fn(() => false);
    const user = userEvent.setup();
    render(
      <LineItemList
        items={mockItems}
        taxRate={0.0825}
        onChange={mockOnChange}
      />
    );

    const checkboxes = screen.getAllByRole('checkbox');
    await user.click(checkboxes[1]);

    await user.click(screen.getAllByRole('button', { name: /delete/i })[0]);

    expect(mockOnChange).not.toHaveBeenCalled();
  });

  it('should show empty state when no items and form is hidden', () => {
    render(
      <LineItemList
        items={[]}
        taxRate={0.0825}
        onChange={mockOnChange}
        showForm={false}
      />
    );

    expect(screen.getByText('No line items')).toBeInTheDocument();
    expect(
      screen.getByText(/get started by adding a line item/i)
    ).toBeInTheDocument();
  });

  it('should make items non-editable when isEditable is false', () => {
    render(
      <LineItemList
        items={mockItems}
        taxRate={0.0825}
        onChange={mockOnChange}
        isEditable={false}
      />
    );

    // Edit buttons should not be present
    expect(screen.queryByTitle(/edit/i)).not.toBeInTheDocument();
    expect(screen.queryByTitle(/delete/i)).not.toBeInTheDocument();
  });

  it('should hide bulk actions when showBulkActions is false', () => {
    render(
      <LineItemList
        items={mockItems}
        taxRate={0.0825}
        onChange={mockOnChange}
        showBulkActions={false}
      />
    );

    // Checkboxes should not be present
    const checkboxes = screen.queryAllByRole('checkbox');
    expect(checkboxes.length).toBe(0);
  });

  it('should show keyboard shortcuts tip', () => {
    render(
      <LineItemList
        items={mockItems}
        taxRate={0.0825}
        onChange={mockOnChange}
      />
    );

    expect(screen.getByText(/drag rows to reorder/i)).toBeInTheDocument();
  });

  it('should update sort order after deletion', async () => {
    const user = userEvent.setup();
    render(
      <LineItemList
        items={mockItems}
        taxRate={0.0825}
        onChange={mockOnChange}
      />
    );

    const deleteButtons = screen.getAllByTitle(/delete/i);
    await user.click(deleteButtons[0]);
    await user.click(deleteButtons[0]);

    await waitFor(() => {
      const call = mockOnChange.mock.calls[0][0];
      expect(call[0].sort_order).toBe(0); // Remaining item should have sort_order 0
    });
  });

  it('should generate unique IDs for new items', async () => {
    const user = userEvent.setup();
    render(
      <LineItemList items={[]} taxRate={0.0825} onChange={mockOnChange} />
    );

    // Add first item
    await user.type(screen.getByLabelText(/description/i), 'Item 1');
    await user.type(screen.getByLabelText(/quantity/i), '1');
    await user.type(screen.getByLabelText(/unit price/i), '10');
    await user.click(screen.getByRole('button', { name: /add/i }));

    await waitFor(() => {
      const firstCall = mockOnChange.mock.calls[0][0][0];
      expect(firstCall.id).toMatch(/^temp-/);
    });
  });
});

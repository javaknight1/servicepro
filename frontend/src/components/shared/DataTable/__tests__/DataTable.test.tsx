import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import React from 'react';
import { DataTable, Column } from '../DataTable';

interface TestData {
  id: string;
  name: string;
  status: string;
  count: number;
}

const mockData: TestData[] = [
  { id: '1', name: 'Alice', status: 'active', count: 10 },
  { id: '2', name: 'Bob', status: 'inactive', count: 5 },
  { id: '3', name: 'Charlie', status: 'active', count: 15 },
];

const mockColumns: Column<TestData>[] = [
  { key: 'name', header: 'Name', sortable: true },
  { key: 'status', header: 'Status' },
  { key: 'count', header: 'Count', sortable: true },
];

describe('DataTable', () => {
  describe('rendering', () => {
    it('should render table headers', () => {
      render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
        />
      );

      expect(screen.getByText('Name')).toBeInTheDocument();
      expect(screen.getByText('Status')).toBeInTheDocument();
      expect(screen.getByText('Count')).toBeInTheDocument();
    });

    it('should render data rows', () => {
      render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
        />
      );

      expect(screen.getByText('Alice')).toBeInTheDocument();
      expect(screen.getByText('Bob')).toBeInTheDocument();
      expect(screen.getByText('Charlie')).toBeInTheDocument();
    });

    it('should render cell values', () => {
      render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
        />
      );

      expect(screen.getByText('10')).toBeInTheDocument();
      expect(screen.getByText('5')).toBeInTheDocument();
      expect(screen.getByText('15')).toBeInTheDocument();
    });

    it('should apply custom className', () => {
      const { container } = render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
          className="custom-table-class"
        />
      );

      expect(container.firstChild).toHaveClass('custom-table-class');
    });
  });

  describe('empty state', () => {
    it('should show empty message when data is empty', () => {
      render(
        <DataTable
          data={[]}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
        />
      );

      expect(screen.getByText('No data available')).toBeInTheDocument();
    });

    it('should show custom empty message', () => {
      render(
        <DataTable
          data={[]}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
          emptyMessage="No records found"
        />
      );

      expect(screen.getByText('No records found')).toBeInTheDocument();
    });
  });

  describe('loading state', () => {
    it('should show loading indicator when isLoading is true', () => {
      render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
          isLoading
        />
      );

      expect(screen.getByText('Loading...')).toBeInTheDocument();
    });

    it('should not render data when loading', () => {
      render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
          isLoading
        />
      );

      expect(screen.queryByText('Alice')).not.toBeInTheDocument();
    });
  });

  describe('row click', () => {
    it('should call onRowClick when row is clicked', () => {
      const handleRowClick = vi.fn();

      render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
          onRowClick={handleRowClick}
        />
      );

      fireEvent.click(screen.getByText('Alice'));

      expect(handleRowClick).toHaveBeenCalledWith(mockData[0]);
    });

    it('should not throw when clicking row without onRowClick handler', () => {
      render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
        />
      );

      expect(() => {
        fireEvent.click(screen.getByText('Alice'));
      }).not.toThrow();
    });
  });

  describe('local sorting', () => {
    it('should sort data when clicking sortable column', () => {
      const { container } = render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
        />
      );

      // Click on Name header to sort
      fireEvent.click(screen.getByText('Name'));

      // Data should be sorted alphabetically
      const rows = container.querySelectorAll('tbody tr');
      expect(rows[0].textContent).toContain('Alice');
    });

    it('should toggle sort order when clicking same column twice', () => {
      const { container } = render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
        />
      );

      // Click twice to toggle
      fireEvent.click(screen.getByText('Name'));
      fireEvent.click(screen.getByText('Name'));

      // Data should be sorted in reverse order
      const rows = container.querySelectorAll('tbody tr');
      expect(rows[0].textContent).toContain('Charlie');
    });

    it('should not sort non-sortable columns', () => {
      const handleClick = vi.fn();
      const columnsWithNonSortable: Column<TestData>[] = [
        { key: 'name', header: 'Name' },
        { key: 'status', header: 'Status' },
      ];

      render(
        <DataTable
          data={mockData}
          columns={columnsWithNonSortable}
          keyExtractor={(row) => row.id}
        />
      );

      // Click on Status header (not sortable)
      fireEvent.click(screen.getByText('Status'));

      // Since it's not sortable, nothing should change
      expect(handleClick).not.toHaveBeenCalled();
    });
  });

  describe('controlled sorting', () => {
    it('should call onSort with correct parameters', () => {
      const handleSort = vi.fn();

      render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
          sorting={{
            sortBy: '',
            sortOrder: 'asc',
            onSort: handleSort,
          }}
        />
      );

      fireEvent.click(screen.getByText('Name'));

      expect(handleSort).toHaveBeenCalledWith('name', 'asc');
    });

    it('should toggle sort order in controlled mode', () => {
      const handleSort = vi.fn();

      render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
          sorting={{
            sortBy: 'name',
            sortOrder: 'asc',
            onSort: handleSort,
          }}
        />
      );

      fireEvent.click(screen.getByText('Name'));

      expect(handleSort).toHaveBeenCalledWith('name', 'desc');
    });
  });

  describe('pagination', () => {
    it('should render pagination controls', () => {
      render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
          pagination={{
            page: 1,
            pageSize: 10,
            total: 100,
            onPageChange: vi.fn(),
          }}
        />
      );

      expect(screen.getByText('Page 1 of 10')).toBeInTheDocument();
    });

    it('should show correct result count', () => {
      render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
          pagination={{
            page: 1,
            pageSize: 10,
            total: 25,
            onPageChange: vi.fn(),
          }}
        />
      );

      expect(
        screen.getByText(/Showing 1 to 10 of 25 results/)
      ).toBeInTheDocument();
    });

    it('should call onPageChange when clicking next', () => {
      const handlePageChange = vi.fn();

      render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
          pagination={{
            page: 1,
            pageSize: 10,
            total: 25,
            onPageChange: handlePageChange,
          }}
        />
      );

      // Find and click next button (ChevronRight)
      const buttons = screen.getAllByRole('button');
      const nextButton = buttons.find(
        (btn) => btn.className.includes('hover:bg-neutral-200') && !btn.disabled
      );

      if (nextButton) {
        fireEvent.click(nextButton);
      }

      // Get all buttons, next should be the second one
      const navButtons = screen.getAllByRole('button');
      fireEvent.click(navButtons[1]); // Click next button

      expect(handlePageChange).toHaveBeenCalledWith(2);
    });

    it('should call onPageChange when clicking previous', () => {
      const handlePageChange = vi.fn();

      render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
          pagination={{
            page: 2,
            pageSize: 10,
            total: 25,
            onPageChange: handlePageChange,
          }}
        />
      );

      // Get all buttons, previous should be the first one
      const navButtons = screen.getAllByRole('button');
      fireEvent.click(navButtons[0]); // Click previous button

      expect(handlePageChange).toHaveBeenCalledWith(1);
    });

    it('should disable previous button on first page', () => {
      render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
          pagination={{
            page: 1,
            pageSize: 10,
            total: 25,
            onPageChange: vi.fn(),
          }}
        />
      );

      const buttons = screen.getAllByRole('button');
      expect(buttons[0]).toBeDisabled();
    });

    it('should disable next button on last page', () => {
      render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
          pagination={{
            page: 3,
            pageSize: 10,
            total: 25,
            onPageChange: vi.fn(),
          }}
        />
      );

      const buttons = screen.getAllByRole('button');
      expect(buttons[1]).toBeDisabled();
    });

    it('should not render pagination when only one page', () => {
      render(
        <DataTable
          data={mockData}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
          pagination={{
            page: 1,
            pageSize: 10,
            total: 5,
            onPageChange: vi.fn(),
          }}
        />
      );

      expect(screen.queryByText(/Page/)).not.toBeInTheDocument();
    });
  });

  describe('custom render', () => {
    it('should use custom render function for columns', () => {
      const columnsWithRender: Column<TestData>[] = [
        { key: 'name', header: 'Name' },
        {
          key: 'status',
          header: 'Status',
          render: (value) => (
            <span data-testid="custom-render">
              {String(value).toUpperCase()}
            </span>
          ),
        },
      ];

      render(
        <DataTable
          data={mockData}
          columns={columnsWithRender}
          keyExtractor={(row) => row.id}
        />
      );

      expect(screen.getAllByTestId('custom-render')[0].textContent).toBe(
        'ACTIVE'
      );
    });
  });

  describe('column className', () => {
    it('should apply column className to header', () => {
      const columnsWithClass: Column<TestData>[] = [
        { key: 'name', header: 'Name', className: 'custom-column-class' },
      ];

      const { container } = render(
        <DataTable
          data={mockData}
          columns={columnsWithClass}
          keyExtractor={(row) => row.id}
        />
      );

      const header = container.querySelector('th.custom-column-class');
      expect(header).toBeInTheDocument();
    });

    it('should apply column className to cells', () => {
      const columnsWithClass: Column<TestData>[] = [
        { key: 'name', header: 'Name', className: 'custom-cell-class' },
      ];

      const { container } = render(
        <DataTable
          data={mockData}
          columns={columnsWithClass}
          keyExtractor={(row) => row.id}
        />
      );

      const cells = container.querySelectorAll('td.custom-cell-class');
      expect(cells.length).toBe(3); // One for each row
    });
  });

  describe('null/undefined values', () => {
    it('should display dash for null values', () => {
      const dataWithNull = [
        { id: '1', name: null, status: 'active', count: 10 },
      ];

      render(
        <DataTable
          data={dataWithNull as unknown as TestData[]}
          columns={mockColumns}
          keyExtractor={(row) => row.id}
        />
      );

      expect(screen.getByText('-')).toBeInTheDocument();
    });
  });
});

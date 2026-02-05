import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { DashboardLayout } from '../DashboardLayout';
import { useSidebarStore } from '@store/sidebarStore';

// Mock the child components since they have their own dependencies
vi.mock('../Sidebar', () => ({
  Sidebar: () => <div data-testid="sidebar">Sidebar</div>,
}));

vi.mock('../Header', () => ({
  Header: () => <div data-testid="header">Header</div>,
}));

// Mock the sidebar store
vi.mock('@store/sidebarStore', () => ({
  useSidebarStore: vi.fn(),
}));

describe('DashboardLayout', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('should render sidebar, header, and children', () => {
    vi.mocked(useSidebarStore).mockReturnValue({
      isCollapsed: false,
    } as ReturnType<typeof useSidebarStore>);

    render(
      <BrowserRouter>
        <DashboardLayout>
          <div>Dashboard Content</div>
        </DashboardLayout>
      </BrowserRouter>
    );

    expect(screen.getByTestId('sidebar')).toBeInTheDocument();
    expect(screen.getByTestId('header')).toBeInTheDocument();
    expect(screen.getByText('Dashboard Content')).toBeInTheDocument();
  });

  it('should render with collapsed sidebar', () => {
    vi.mocked(useSidebarStore).mockReturnValue({
      isCollapsed: true,
    } as ReturnType<typeof useSidebarStore>);

    const { container } = render(
      <BrowserRouter>
        <DashboardLayout>
          <div>Content</div>
        </DashboardLayout>
      </BrowserRouter>
    );

    expect(container.querySelector('main')).toBeInTheDocument();
  });
});

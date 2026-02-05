import { describe, it, expect, beforeEach, vi } from 'vitest';
import { act } from '@testing-library/react';
import { useSidebarStore } from '../sidebarStore';

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: vi.fn(() => {
      store = {};
    }),
  };
})();

Object.defineProperty(window, 'localStorage', {
  value: localStorageMock,
});

describe('sidebarStore', () => {
  beforeEach(() => {
    // Reset the store state before each test
    act(() => {
      useSidebarStore.setState({
        isCollapsed: false,
        isMobileOpen: false,
      });
    });
    localStorageMock.clear();
  });

  describe('initial state', () => {
    it('should have correct initial values', () => {
      const state = useSidebarStore.getState();

      expect(state.isCollapsed).toBe(false);
      expect(state.isMobileOpen).toBe(false);
    });
  });

  describe('setCollapsed', () => {
    it('should set collapsed to true', () => {
      act(() => {
        useSidebarStore.getState().setCollapsed(true);
      });

      expect(useSidebarStore.getState().isCollapsed).toBe(true);
    });

    it('should set collapsed to false', () => {
      act(() => {
        useSidebarStore.getState().setCollapsed(true);
        useSidebarStore.getState().setCollapsed(false);
      });

      expect(useSidebarStore.getState().isCollapsed).toBe(false);
    });
  });

  describe('toggleCollapsed', () => {
    it('should toggle from false to true', () => {
      expect(useSidebarStore.getState().isCollapsed).toBe(false);

      act(() => {
        useSidebarStore.getState().toggleCollapsed();
      });

      expect(useSidebarStore.getState().isCollapsed).toBe(true);
    });

    it('should toggle from true to false', () => {
      act(() => {
        useSidebarStore.getState().setCollapsed(true);
      });

      expect(useSidebarStore.getState().isCollapsed).toBe(true);

      act(() => {
        useSidebarStore.getState().toggleCollapsed();
      });

      expect(useSidebarStore.getState().isCollapsed).toBe(false);
    });
  });

  describe('setMobileOpen', () => {
    it('should set mobile open to true', () => {
      act(() => {
        useSidebarStore.getState().setMobileOpen(true);
      });

      expect(useSidebarStore.getState().isMobileOpen).toBe(true);
    });

    it('should set mobile open to false', () => {
      act(() => {
        useSidebarStore.getState().setMobileOpen(true);
        useSidebarStore.getState().setMobileOpen(false);
      });

      expect(useSidebarStore.getState().isMobileOpen).toBe(false);
    });
  });

  describe('toggleMobileOpen', () => {
    it('should toggle from false to true', () => {
      expect(useSidebarStore.getState().isMobileOpen).toBe(false);

      act(() => {
        useSidebarStore.getState().toggleMobileOpen();
      });

      expect(useSidebarStore.getState().isMobileOpen).toBe(true);
    });

    it('should toggle from true to false', () => {
      act(() => {
        useSidebarStore.getState().setMobileOpen(true);
      });

      expect(useSidebarStore.getState().isMobileOpen).toBe(true);

      act(() => {
        useSidebarStore.getState().toggleMobileOpen();
      });

      expect(useSidebarStore.getState().isMobileOpen).toBe(false);
    });
  });

  describe('state independence', () => {
    it('should not affect isCollapsed when changing isMobileOpen', () => {
      act(() => {
        useSidebarStore.getState().setCollapsed(true);
        useSidebarStore.getState().setMobileOpen(true);
      });

      expect(useSidebarStore.getState().isCollapsed).toBe(true);
      expect(useSidebarStore.getState().isMobileOpen).toBe(true);

      act(() => {
        useSidebarStore.getState().setMobileOpen(false);
      });

      expect(useSidebarStore.getState().isCollapsed).toBe(true);
      expect(useSidebarStore.getState().isMobileOpen).toBe(false);
    });
  });
});

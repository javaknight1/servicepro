import { describe, it, expect } from 'vitest';
import {
  CACHE_TIMES,
  QUERY_CONFIG,
  ENDPOINT_CACHE_POLICIES,
  SW_CACHE_CONFIG,
  LOCAL_CACHE_CONFIG,
  CACHE_INVALIDATION_RULES,
  getCachePolicy,
  getInvalidationKeys,
  isPersistentKey,
  getCustomExpiry,
} from '../cache-config';

describe('cache-config', () => {
  describe('CACHE_TIMES', () => {
    it('should have SHORT time of 1 minute', () => {
      expect(CACHE_TIMES.SHORT).toBe(60000);
    });

    it('should have DEFAULT time of 5 minutes', () => {
      expect(CACHE_TIMES.DEFAULT).toBe(300000);
    });

    it('should have MEDIUM time of 15 minutes', () => {
      expect(CACHE_TIMES.MEDIUM).toBe(900000);
    });

    it('should have LONG time of 30 minutes', () => {
      expect(CACHE_TIMES.LONG).toBe(1800000);
    });

    it('should have EXTENDED time of 1 hour', () => {
      expect(CACHE_TIMES.EXTENDED).toBe(3600000);
    });

    it('should have PERMANENT time of Infinity', () => {
      expect(CACHE_TIMES.PERMANENT).toBe(Infinity);
    });
  });

  describe('QUERY_CONFIG', () => {
    it('should have default query options', () => {
      expect(QUERY_CONFIG.defaultOptions.queries).toBeDefined();
      expect(QUERY_CONFIG.defaultOptions.queries.staleTime).toBe(
        CACHE_TIMES.DEFAULT
      );
      expect(QUERY_CONFIG.defaultOptions.queries.gcTime).toBe(CACHE_TIMES.LONG);
      expect(QUERY_CONFIG.defaultOptions.queries.retry).toBe(3);
    });

    it('should have retry delay function', () => {
      const retryDelay = QUERY_CONFIG.defaultOptions.queries.retryDelay;
      expect(retryDelay(0)).toBe(1000);
      expect(retryDelay(1)).toBe(2000);
      expect(retryDelay(2)).toBe(4000);
      expect(retryDelay(5)).toBe(30000); // capped at 30000
    });

    it('should have default mutation options', () => {
      expect(QUERY_CONFIG.defaultOptions.mutations).toBeDefined();
      expect(QUERY_CONFIG.defaultOptions.mutations.retry).toBe(1);
      expect(QUERY_CONFIG.defaultOptions.mutations.networkMode).toBe('online');
    });
  });

  describe('ENDPOINT_CACHE_POLICIES', () => {
    it('should have auth.currentUser policy', () => {
      const policy = ENDPOINT_CACHE_POLICIES['auth.currentUser'];
      expect(policy).toBeDefined();
      expect(policy.staleTime).toBe(CACHE_TIMES.SHORT);
    });

    it('should have customers.list policy', () => {
      const policy = ENDPOINT_CACHE_POLICIES['customers.list'];
      expect(policy).toBeDefined();
      expect(policy.staleTime).toBe(CACHE_TIMES.DEFAULT);
    });

    it('should have jobs.schedule with refetchInterval', () => {
      const policy = ENDPOINT_CACHE_POLICIES['jobs.schedule'];
      expect(policy.refetchInterval).toBe(60000);
    });

    it('should have dashboard.summary with refetchInterval', () => {
      const policy = ENDPOINT_CACHE_POLICIES['dashboard.summary'];
      expect(policy.refetchInterval).toBe(120000);
    });

    it('should have settings.general with long cache', () => {
      const policy = ENDPOINT_CACHE_POLICIES['settings.general'];
      expect(policy.staleTime).toBe(CACHE_TIMES.LONG);
      expect(policy.refetchOnWindowFocus).toBe(false);
    });
  });

  describe('SW_CACHE_CONFIG', () => {
    it('should have cache names defined', () => {
      expect(SW_CACHE_CONFIG.cacheNames.static).toBe('servicepro-static-v1');
      expect(SW_CACHE_CONFIG.cacheNames.api).toBe('servicepro-api-v1');
      expect(SW_CACHE_CONFIG.cacheNames.images).toBe('servicepro-images-v1');
      expect(SW_CACHE_CONFIG.cacheNames.fonts).toBe('servicepro-fonts-v1');
    });

    it('should have static assets config', () => {
      expect(SW_CACHE_CONFIG.staticAssets.maxEntries).toBe(100);
    });

    it('should have api responses config', () => {
      expect(SW_CACHE_CONFIG.apiResponses.maxEntries).toBe(200);
    });
  });

  describe('LOCAL_CACHE_CONFIG', () => {
    it('should have prefix defined', () => {
      expect(LOCAL_CACHE_CONFIG.prefix).toBe('sp_cache_');
    });

    it('should have default expiry', () => {
      expect(LOCAL_CACHE_CONFIG.defaultExpiry).toBe(CACHE_TIMES.DEFAULT);
    });

    it('should have max item size', () => {
      expect(LOCAL_CACHE_CONFIG.maxItemSize).toBe(512000);
    });

    it('should have persistent keys', () => {
      expect(LOCAL_CACHE_CONFIG.persistentKeys).toContain('theme');
      expect(LOCAL_CACHE_CONFIG.persistentKeys).toContain('locale');
      expect(LOCAL_CACHE_CONFIG.persistentKeys).toContain('preferences');
    });

    it('should have custom expiry settings', () => {
      expect(LOCAL_CACHE_CONFIG.customExpiry['user.preferences']).toBe(
        CACHE_TIMES.EXTENDED
      );
      expect(LOCAL_CACHE_CONFIG.customExpiry['app.version']).toBe(
        CACHE_TIMES.PERMANENT
      );
    });
  });

  describe('CACHE_INVALIDATION_RULES', () => {
    it('should invalidate customer caches on customer mutation', () => {
      const rules = CACHE_INVALIDATION_RULES['customers.mutate'];
      expect(rules).toContain('customers.list');
      expect(rules).toContain('customers.search');
      expect(rules).toContain('dashboard.summary');
    });

    it('should invalidate job caches on job mutation', () => {
      const rules = CACHE_INVALIDATION_RULES['jobs.mutate'];
      expect(rules).toContain('jobs.list');
      expect(rules).toContain('jobs.schedule');
      expect(rules).toContain('dashboard.metrics');
    });

    it('should invalidate invoice caches on payment mutation', () => {
      const rules = CACHE_INVALIDATION_RULES['payments.mutate'];
      expect(rules).toContain('invoices.list');
      expect(rules).toContain('invoices.detail');
      expect(rules).toContain('dashboard.revenue');
    });
  });

  describe('getCachePolicy', () => {
    it('should return policy for known endpoint', () => {
      const policy = getCachePolicy('auth.currentUser');
      expect(policy.staleTime).toBe(CACHE_TIMES.SHORT);
      expect(policy.gcTime).toBe(CACHE_TIMES.DEFAULT);
    });

    it('should return default policy for unknown endpoint', () => {
      const policy = getCachePolicy('unknown.endpoint');
      expect(policy.staleTime).toBe(CACHE_TIMES.DEFAULT);
      expect(policy.gcTime).toBe(CACHE_TIMES.LONG);
      expect(policy.refetchOnWindowFocus).toBe(true);
    });

    it('should return policy with refetchInterval', () => {
      const policy = getCachePolicy('jobs.schedule');
      expect(policy.refetchInterval).toBe(60000);
    });
  });

  describe('getInvalidationKeys', () => {
    it('should return invalidation keys for known mutation', () => {
      const keys = getInvalidationKeys('customers.mutate');
      expect(keys).toContain('customers.list');
      expect(keys.length).toBeGreaterThan(0);
    });

    it('should return empty array for unknown mutation', () => {
      const keys = getInvalidationKeys('unknown.mutate');
      expect(keys).toEqual([]);
    });

    it('should return multiple keys for jobs mutation', () => {
      const keys = getInvalidationKeys('jobs.mutate');
      expect(keys.length).toBe(4);
    });
  });

  describe('isPersistentKey', () => {
    it('should return true for theme key', () => {
      expect(isPersistentKey('user-theme')).toBe(true);
      expect(isPersistentKey('app-theme-dark')).toBe(true);
    });

    it('should return true for locale key', () => {
      expect(isPersistentKey('user-locale')).toBe(true);
    });

    it('should return true for preferences key', () => {
      expect(isPersistentKey('user-preferences')).toBe(true);
    });

    it('should return false for non-persistent key', () => {
      expect(isPersistentKey('auth-token')).toBe(false);
      expect(isPersistentKey('cache-data')).toBe(false);
    });
  });

  describe('getCustomExpiry', () => {
    it('should return custom expiry for user.preferences pattern', () => {
      const expiry = getCustomExpiry('user.preferences');
      expect(expiry).toBe(CACHE_TIMES.EXTENDED);
    });

    it('should return custom expiry for app.version pattern', () => {
      const expiry = getCustomExpiry('app.version');
      expect(expiry).toBe(CACHE_TIMES.PERMANENT);
    });

    it('should return custom expiry for feature.flags pattern', () => {
      const expiry = getCustomExpiry('feature.flags');
      expect(expiry).toBe(CACHE_TIMES.LONG);
    });

    it('should return default expiry for unknown pattern', () => {
      const expiry = getCustomExpiry('unknown.key');
      expect(expiry).toBe(LOCAL_CACHE_CONFIG.defaultExpiry);
    });

    it('should match partial patterns', () => {
      const expiry = getCustomExpiry('my-user.preferences-key');
      expect(expiry).toBe(CACHE_TIMES.EXTENDED);
    });
  });
});

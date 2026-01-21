/**
 * =============================================================================
 * Local Storage Cache Utility
 * =============================================================================
 * Provides a type-safe cache wrapper with expiration support
 */

import {
  LOCAL_CACHE_CONFIG,
  getCustomExpiry,
  isPersistentKey,
} from '../config/cache-config';

// =============================================================================
// Types
// =============================================================================

interface CacheEntry<T> {
  data: T;
  timestamp: number;
  expiry: number;
  version: number;
}

interface CacheStats {
  hits: number;
  misses: number;
  size: number;
  itemCount: number;
}

type CacheEventType = 'set' | 'get' | 'delete' | 'clear' | 'expire';

interface CacheEvent {
  type: CacheEventType;
  key: string;
  timestamp: number;
}

// =============================================================================
// Cache Class
// =============================================================================

class LocalStorageCache {
  private prefix: string;
  private version: number;
  private stats: CacheStats;
  private listeners: Map<string, Set<(data: unknown) => void>>;
  private eventLog: CacheEvent[];

  constructor() {
    this.prefix = LOCAL_CACHE_CONFIG.prefix;
    this.version = 1;
    this.stats = { hits: 0, misses: 0, size: 0, itemCount: 0 };
    this.listeners = new Map();
    this.eventLog = [];

    this.calculateStats();
  }

  // ===========================================================================
  // Core Methods
  // ===========================================================================

  /**
   * Get an item from cache
   */
  get<T>(key: string): T | null {
    const fullKey = this.getFullKey(key);

    try {
      const raw = localStorage.getItem(fullKey);

      if (!raw) {
        this.stats.misses++;
        this.logEvent('get', key);
        return null;
      }

      const entry: CacheEntry<T> = JSON.parse(raw);

      // Check expiration
      if (this.isExpired(entry)) {
        this.delete(key);
        this.stats.misses++;
        this.logEvent('expire', key);
        return null;
      }

      // Check version
      if (entry.version !== this.version) {
        this.delete(key);
        this.stats.misses++;
        return null;
      }

      this.stats.hits++;
      this.logEvent('get', key);
      return entry.data;
    } catch (error) {
      console.warn(`Cache get error for ${key}:`, error);
      this.delete(key);
      this.stats.misses++;
      return null;
    }
  }

  /**
   * Set an item in cache
   */
  set<T>(key: string, data: T, expiryMs?: number): boolean {
    const fullKey = this.getFullKey(key);
    const expiry = expiryMs ?? getCustomExpiry(key);

    const entry: CacheEntry<T> = {
      data,
      timestamp: Date.now(),
      expiry,
      version: this.version,
    };

    try {
      const serialized = JSON.stringify(entry);

      // Check item size
      if (serialized.length > LOCAL_CACHE_CONFIG.maxItemSize) {
        console.warn(
          `Cache item too large: ${key} (${serialized.length} chars)`
        );
        return false;
      }

      // Check if we need to make room
      this.ensureSpace(serialized.length);

      localStorage.setItem(fullKey, serialized);
      this.logEvent('set', key);
      this.notifyListeners(key, data);
      this.calculateStats();

      return true;
    } catch (error) {
      console.warn(`Cache set error for ${key}:`, error);

      // Handle quota exceeded
      if (this.isQuotaError(error)) {
        this.evictOldest(5);

        // Retry once
        try {
          localStorage.setItem(fullKey, JSON.stringify(entry));
          return true;
        } catch {
          return false;
        }
      }

      return false;
    }
  }

  /**
   * Delete an item from cache
   */
  delete(key: string): boolean {
    const fullKey = this.getFullKey(key);

    try {
      localStorage.removeItem(fullKey);
      this.logEvent('delete', key);
      this.notifyListeners(key, null);
      this.calculateStats();
      return true;
    } catch (error) {
      console.warn(`Cache delete error for ${key}:`, error);
      return false;
    }
  }

  /**
   * Check if a key exists and is valid
   */
  has(key: string): boolean {
    return this.get(key) !== null;
  }

  /**
   * Get item with metadata
   */
  getWithMetadata<T>(
    key: string
  ): { data: T; age: number; ttl: number } | null {
    const fullKey = this.getFullKey(key);

    try {
      const raw = localStorage.getItem(fullKey);
      if (!raw) return null;

      const entry: CacheEntry<T> = JSON.parse(raw);

      if (this.isExpired(entry)) {
        this.delete(key);
        return null;
      }

      const age = Date.now() - entry.timestamp;
      const ttl = entry.expiry === Infinity ? Infinity : entry.expiry - age;

      return { data: entry.data, age, ttl };
    } catch {
      return null;
    }
  }

  // ===========================================================================
  // Batch Operations
  // ===========================================================================

  /**
   * Get multiple items
   */
  getMany<T>(keys: string[]): Map<string, T | null> {
    const results = new Map<string, T | null>();

    for (const key of keys) {
      results.set(key, this.get<T>(key));
    }

    return results;
  }

  /**
   * Set multiple items
   */
  setMany<T>(items: Array<{ key: string; data: T; expiry?: number }>): void {
    for (const { key, data, expiry } of items) {
      this.set(key, data, expiry);
    }
  }

  /**
   * Delete multiple items
   */
  deleteMany(keys: string[]): void {
    for (const key of keys) {
      this.delete(key);
    }
  }

  // ===========================================================================
  // Pattern Operations
  // ===========================================================================

  /**
   * Get all keys matching a pattern
   */
  getKeys(pattern?: string): string[] {
    const keys: string[] = [];

    for (let i = 0; i < localStorage.length; i++) {
      const fullKey = localStorage.key(i);

      if (fullKey?.startsWith(this.prefix)) {
        const key = fullKey.slice(this.prefix.length);

        if (!pattern || key.includes(pattern)) {
          keys.push(key);
        }
      }
    }

    return keys;
  }

  /**
   * Delete all keys matching a pattern
   */
  deletePattern(pattern: string): number {
    const keys = this.getKeys(pattern);

    for (const key of keys) {
      this.delete(key);
    }

    return keys.length;
  }

  /**
   * Invalidate cache based on mutation type
   */
  invalidate(patterns: string[]): void {
    for (const pattern of patterns) {
      this.deletePattern(pattern);
    }
  }

  // ===========================================================================
  // Cache Management
  // ===========================================================================

  /**
   * Clear all cache (except persistent keys)
   */
  clear(includePersistent = false): void {
    const keys = this.getKeys();

    for (const key of keys) {
      if (includePersistent || !isPersistentKey(key)) {
        this.delete(key);
      }
    }

    this.logEvent('clear', '*');
    this.calculateStats();
  }

  /**
   * Clear expired entries
   */
  clearExpired(): number {
    let cleared = 0;
    const keys = this.getKeys();

    for (const key of keys) {
      const fullKey = this.getFullKey(key);

      try {
        const raw = localStorage.getItem(fullKey);
        if (!raw) continue;

        const entry: CacheEntry<unknown> = JSON.parse(raw);

        if (this.isExpired(entry)) {
          this.delete(key);
          cleared++;
        }
      } catch {
        this.delete(key);
        cleared++;
      }
    }

    return cleared;
  }

  /**
   * Evict oldest entries
   */
  evictOldest(count: number): void {
    const entries: Array<{ key: string; timestamp: number }> = [];
    const keys = this.getKeys();

    for (const key of keys) {
      if (isPersistentKey(key)) continue;

      const fullKey = this.getFullKey(key);

      try {
        const raw = localStorage.getItem(fullKey);
        if (!raw) continue;

        const entry: CacheEntry<unknown> = JSON.parse(raw);
        entries.push({ key, timestamp: entry.timestamp });
      } catch {
        // Skip invalid entries
      }
    }

    // Sort by oldest first
    entries.sort((a, b) => a.timestamp - b.timestamp);

    // Delete oldest entries
    for (let i = 0; i < Math.min(count, entries.length); i++) {
      this.delete(entries[i].key);
    }
  }

  // ===========================================================================
  // Listeners
  // ===========================================================================

  /**
   * Subscribe to changes for a key
   */
  subscribe<T>(key: string, callback: (data: T | null) => void): () => void {
    if (!this.listeners.has(key)) {
      this.listeners.set(key, new Set());
    }

    this.listeners.get(key)!.add(callback as (data: unknown) => void);

    // Return unsubscribe function
    return () => {
      this.listeners.get(key)?.delete(callback as (data: unknown) => void);
    };
  }

  /**
   * Notify listeners of changes
   */
  private notifyListeners(key: string, data: unknown): void {
    const callbacks = this.listeners.get(key);

    if (callbacks) {
      callbacks.forEach((callback) => callback(data));
    }
  }

  // ===========================================================================
  // Stats & Debugging
  // ===========================================================================

  /**
   * Get cache statistics
   */
  getStats(): CacheStats & { hitRate: number } {
    const total = this.stats.hits + this.stats.misses;
    const hitRate = total > 0 ? (this.stats.hits / total) * 100 : 0;

    return { ...this.stats, hitRate };
  }

  /**
   * Get recent events
   */
  getEventLog(limit = 100): CacheEvent[] {
    return this.eventLog.slice(-limit);
  }

  /**
   * Calculate current cache stats
   */
  private calculateStats(): void {
    let size = 0;
    let itemCount = 0;

    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i);

      if (key?.startsWith(this.prefix)) {
        const value = localStorage.getItem(key);

        if (value) {
          size += value.length;
          itemCount++;
        }
      }
    }

    this.stats.size = size;
    this.stats.itemCount = itemCount;
  }

  // ===========================================================================
  // Helpers
  // ===========================================================================

  private getFullKey(key: string): string {
    return `${this.prefix}${key}`;
  }

  private isExpired(entry: CacheEntry<unknown>): boolean {
    if (entry.expiry === Infinity) return false;
    return Date.now() - entry.timestamp > entry.expiry;
  }

  private isQuotaError(error: unknown): boolean {
    return (
      error instanceof DOMException &&
      (error.code === 22 || // Legacy quota exceeded
        error.code === 1014 || // Firefox
        error.name === 'QuotaExceededError' ||
        error.name === 'NS_ERROR_DOM_QUOTA_REACHED')
    );
  }

  private ensureSpace(requiredSize: number): void {
    const currentSize = this.stats.size;
    const maxSize = LOCAL_CACHE_CONFIG.maxTotalSize;

    if (currentSize + requiredSize > maxSize) {
      // Clear expired first
      this.clearExpired();

      // If still not enough, evict oldest
      if (this.stats.size + requiredSize > maxSize) {
        const toEvict = Math.ceil(
          (this.stats.size + requiredSize - maxSize) / 10000
        );
        this.evictOldest(Math.max(toEvict, 5));
      }
    }
  }

  private logEvent(type: CacheEventType, key: string): void {
    this.eventLog.push({
      type,
      key,
      timestamp: Date.now(),
    });

    // Keep event log bounded
    if (this.eventLog.length > 1000) {
      this.eventLog = this.eventLog.slice(-500);
    }
  }
}

// =============================================================================
// Singleton Instance
// =============================================================================

export const cache = new LocalStorageCache();

// =============================================================================
// Utility Functions
// =============================================================================

/**
 * Create a cached function wrapper
 */
export function withCache<T, Args extends unknown[]>(
  fn: (...args: Args) => Promise<T>,
  keyFn: (...args: Args) => string,
  expiryMs?: number
): (...args: Args) => Promise<T> {
  return async (...args: Args): Promise<T> => {
    const key = keyFn(...args);
    const cached = cache.get<T>(key);

    if (cached !== null) {
      return cached;
    }

    const result = await fn(...args);
    cache.set(key, result, expiryMs);

    return result;
  };
}

/**
 * Decorator for caching method results
 */
export function cached(keyPrefix: string, expiryMs?: number) {
  return function <T>(
    _target: unknown,
    propertyKey: string,
    descriptor: PropertyDescriptor
  ) {
    const originalMethod = descriptor.value;

    descriptor.value = async function (...args: unknown[]) {
      const key = `${keyPrefix}.${propertyKey}.${JSON.stringify(args)}`;
      const cachedValue = cache.get<T>(key);

      if (cachedValue !== null) {
        return cachedValue;
      }

      const result = await originalMethod.apply(this, args);
      cache.set(key, result, expiryMs);

      return result;
    };

    return descriptor;
  };
}

/**
 * Session storage cache (cleared on tab close)
 */
export const sessionCache = {
  get<T>(key: string): T | null {
    try {
      const raw = sessionStorage.getItem(`sp_session_${key}`);
      return raw ? JSON.parse(raw) : null;
    } catch {
      return null;
    }
  },

  set<T>(key: string, data: T): boolean {
    try {
      sessionStorage.setItem(`sp_session_${key}`, JSON.stringify(data));
      return true;
    } catch {
      return false;
    }
  },

  delete(key: string): void {
    sessionStorage.removeItem(`sp_session_${key}`);
  },

  clear(): void {
    const keysToRemove: string[] = [];

    for (let i = 0; i < sessionStorage.length; i++) {
      const key = sessionStorage.key(i);
      if (key?.startsWith('sp_session_')) {
        keysToRemove.push(key);
      }
    }

    keysToRemove.forEach((key) => sessionStorage.removeItem(key));
  },
};

// =============================================================================
// Export Types
// =============================================================================

export type { CacheEntry, CacheStats, CacheEvent, CacheEventType };

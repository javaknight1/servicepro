/**
 * Connection Manager - Advanced connection management with health monitoring
 */

import {
  ConnectionConfig,
  ConnectionState,
  ConnectionStatus,
  ConnectionMetrics,
  WebSocketEventType,
  WebSocketEventHandler,
  calculateBackoff,
} from '../../types/realtime';
import {
  WebSocketServiceImpl,
  createWebSocketService,
} from './WebSocketService';

// ============================================
// Types
// ============================================

export interface ConnectionHealth {
  isHealthy: boolean;
  status: ConnectionStatus;
  latency: number | null;
  lastCheck: number;
  consecutiveFailures: number;
  score: number; // 0-100
}

export interface ConnectionManagerConfig {
  /** Primary WebSocket URL */
  primaryUrl: string;
  /** Fallback URLs for failover */
  fallbackUrls?: string[];
  /** Authentication token */
  token?: string;
  /** Enable automatic failover */
  autoFailover?: boolean;
  /** Health check interval in ms */
  healthCheckInterval?: number;
  /** Maximum latency before considered unhealthy */
  maxLatency?: number;
  /** Number of failures before failover */
  failoverThreshold?: number;
  /** Enable connection pooling */
  enablePooling?: boolean;
  /** Maximum connections in pool */
  maxPoolSize?: number;
  /** Enable debug logging */
  debug?: boolean;
}

export interface ConnectionPool {
  connections: Map<string, WebSocketServiceImpl>;
  activeConnection: string | null;
  health: Map<string, ConnectionHealth>;
}

type ConnectionEventType =
  | 'connected'
  | 'disconnected'
  | 'failover'
  | 'health_check'
  | 'recovery'
  | 'error';

type ConnectionEventHandler = (event: {
  type: ConnectionEventType;
  data: unknown;
  timestamp: number;
}) => void;

// ============================================
// Connection Manager
// ============================================

export class ConnectionManager {
  private config: Required<ConnectionManagerConfig>;
  private pool: ConnectionPool;
  private eventHandlers: Map<ConnectionEventType, Set<ConnectionEventHandler>> =
    new Map();

  // Health monitoring
  private healthCheckTimer: NodeJS.Timeout | null = null;
  private healthHistory: Map<string, number[]> = new Map();

  // Failover state
  private currentUrlIndex = 0;
  private failoverInProgress = false;
  private lastFailoverTime = 0;
  private failoverCooldown = 30000; // 30 seconds

  // Recovery tracking
  private recoveryAttempts = 0;
  private maxRecoveryAttempts = 5;
  private recoveryTimer: NodeJS.Timeout | null = null;

  constructor(config: ConnectionManagerConfig) {
    this.config = {
      primaryUrl: config.primaryUrl,
      fallbackUrls: config.fallbackUrls || [],
      token: config.token || '',
      autoFailover: config.autoFailover ?? true,
      healthCheckInterval: config.healthCheckInterval || 30000,
      maxLatency: config.maxLatency || 5000,
      failoverThreshold: config.failoverThreshold || 3,
      enablePooling: config.enablePooling ?? false,
      maxPoolSize: config.maxPoolSize || 3,
      debug: config.debug ?? false,
    };

    this.pool = {
      connections: new Map(),
      activeConnection: null,
      health: new Map(),
    };
  }

  // ============================================
  // Connection Lifecycle
  // ============================================

  /**
   * Initialize and connect to the primary WebSocket
   */
  async connect(): Promise<void> {
    const url = this.getAllUrls()[this.currentUrlIndex];
    this.log(`Connecting to ${url}`);

    try {
      const service = await this.createConnection(url);
      this.pool.activeConnection = url;

      this.startHealthMonitoring();
      this.emitEvent('connected', { url });
    } catch (error) {
      this.log(`Connection failed: ${error}`);

      if (this.config.autoFailover && this.config.fallbackUrls.length > 0) {
        await this.failover();
      } else {
        throw error;
      }
    }
  }

  /**
   * Disconnect all connections
   */
  disconnect(): void {
    this.stopHealthMonitoring();
    this.stopRecovery();

    for (const [url, service] of this.pool.connections) {
      service.disconnect();
      this.log(`Disconnected from ${url}`);
    }

    this.pool.connections.clear();
    this.pool.activeConnection = null;
    this.pool.health.clear();

    this.emitEvent('disconnected', {});
  }

  /**
   * Get the active WebSocket service
   */
  getActiveConnection(): WebSocketServiceImpl | null {
    if (!this.pool.activeConnection) return null;
    return this.pool.connections.get(this.pool.activeConnection) || null;
  }

  /**
   * Check if connected
   */
  isConnected(): boolean {
    const active = this.getActiveConnection();
    return active?.isConnected() ?? false;
  }

  /**
   * Get connection state
   */
  getConnectionState(): ConnectionState | null {
    return this.getActiveConnection()?.getConnectionState() ?? null;
  }

  // ============================================
  // Health Monitoring
  // ============================================

  /**
   * Start health monitoring
   */
  private startHealthMonitoring(): void {
    if (this.healthCheckTimer) return;

    this.healthCheckTimer = setInterval(() => {
      this.performHealthCheck();
    }, this.config.healthCheckInterval);

    // Perform initial health check
    this.performHealthCheck();
  }

  /**
   * Stop health monitoring
   */
  private stopHealthMonitoring(): void {
    if (this.healthCheckTimer) {
      clearInterval(this.healthCheckTimer);
      this.healthCheckTimer = null;
    }
  }

  /**
   * Perform health check on active connection
   */
  private async performHealthCheck(): Promise<void> {
    const activeUrl = this.pool.activeConnection;
    if (!activeUrl) return;

    const service = this.pool.connections.get(activeUrl);
    if (!service) return;

    const health = this.pool.health.get(activeUrl) || this.createHealthRecord();

    try {
      if (service.isConnected()) {
        const latency = await service.ping();

        this.updateHealthRecord(activeUrl, {
          isHealthy: latency < this.config.maxLatency,
          latency,
          consecutiveFailures: 0,
          lastCheck: Date.now(),
        });

        // Track latency history for trend analysis
        this.trackLatency(activeUrl, latency);
      } else {
        this.updateHealthRecord(activeUrl, {
          isHealthy: false,
          consecutiveFailures: health.consecutiveFailures + 1,
          lastCheck: Date.now(),
        });
      }
    } catch (error) {
      this.log(`Health check failed: ${error}`);

      this.updateHealthRecord(activeUrl, {
        isHealthy: false,
        consecutiveFailures: health.consecutiveFailures + 1,
        lastCheck: Date.now(),
      });
    }

    // Check if failover is needed
    const currentHealth = this.pool.health.get(activeUrl)!;
    this.emitEvent('health_check', { url: activeUrl, health: currentHealth });

    if (
      !currentHealth.isHealthy &&
      currentHealth.consecutiveFailures >= this.config.failoverThreshold
    ) {
      this.log(`Health threshold exceeded, initiating failover`);
      await this.failover();
    }
  }

  /**
   * Get health status for a connection
   */
  getHealth(url?: string): ConnectionHealth | null {
    const targetUrl = url || this.pool.activeConnection;
    if (!targetUrl) return null;
    return this.pool.health.get(targetUrl) || null;
  }

  /**
   * Get health score (0-100)
   */
  private calculateHealthScore(health: ConnectionHealth): number {
    let score = 100;

    // Deduct for high latency
    if (health.latency) {
      const latencyPenalty = Math.min(
        50,
        (health.latency / this.config.maxLatency) * 50
      );
      score -= latencyPenalty;
    }

    // Deduct for failures
    score -= health.consecutiveFailures * 10;

    // Deduct if not healthy
    if (!health.isHealthy) {
      score -= 20;
    }

    return Math.max(0, Math.min(100, score));
  }

  // ============================================
  // Failover Logic
  // ============================================

  /**
   * Perform failover to next available URL
   */
  async failover(): Promise<void> {
    if (this.failoverInProgress) {
      this.log('Failover already in progress');
      return;
    }

    // Check cooldown
    if (Date.now() - this.lastFailoverTime < this.failoverCooldown) {
      this.log('Failover on cooldown');
      return;
    }

    this.failoverInProgress = true;
    this.lastFailoverTime = Date.now();

    const urls = this.getAllUrls();
    const startIndex = this.currentUrlIndex;

    this.log(`Starting failover from index ${startIndex}`);

    // Try each URL
    for (let i = 0; i < urls.length; i++) {
      this.currentUrlIndex = (startIndex + i + 1) % urls.length;
      const url = urls[this.currentUrlIndex];

      // Skip if we've already tried this URL recently
      const health = this.pool.health.get(url);
      if (
        health &&
        !health.isHealthy &&
        Date.now() - health.lastCheck < 60000
      ) {
        continue;
      }

      this.log(`Attempting failover to ${url}`);

      try {
        // Disconnect current if exists
        if (this.pool.activeConnection) {
          const current = this.pool.connections.get(this.pool.activeConnection);
          current?.disconnect();
        }

        const service = await this.createConnection(url);
        this.pool.activeConnection = url;

        this.emitEvent('failover', {
          from: urls[startIndex],
          to: url,
        });

        this.failoverInProgress = false;
        return;
      } catch (error) {
        this.log(`Failover to ${url} failed: ${error}`);
        this.updateHealthRecord(url, {
          isHealthy: false,
          consecutiveFailures:
            (this.pool.health.get(url)?.consecutiveFailures || 0) + 1,
          lastCheck: Date.now(),
        });
      }
    }

    this.failoverInProgress = false;
    this.log('All failover attempts failed, starting recovery');
    this.startRecovery();
  }

  // ============================================
  // Recovery Logic
  // ============================================

  /**
   * Start automatic recovery process
   */
  private startRecovery(): void {
    if (this.recoveryTimer) return;

    this.recoveryAttempts = 0;
    this.attemptRecovery();
  }

  /**
   * Stop recovery process
   */
  private stopRecovery(): void {
    if (this.recoveryTimer) {
      clearTimeout(this.recoveryTimer);
      this.recoveryTimer = null;
    }
    this.recoveryAttempts = 0;
  }

  /**
   * Attempt to recover connection
   */
  private async attemptRecovery(): Promise<void> {
    if (this.recoveryAttempts >= this.maxRecoveryAttempts) {
      this.log('Max recovery attempts reached');
      this.emitEvent('error', {
        error: new Error('Connection recovery failed'),
        attempts: this.recoveryAttempts,
      });
      return;
    }

    this.recoveryAttempts++;
    const delay = calculateBackoff(this.recoveryAttempts - 1, 5000, 60000, 2);

    this.log(`Recovery attempt ${this.recoveryAttempts} in ${delay}ms`);

    this.recoveryTimer = setTimeout(async () => {
      try {
        // Reset URL index to primary
        this.currentUrlIndex = 0;
        await this.connect();

        this.stopRecovery();
        this.emitEvent('recovery', { attempts: this.recoveryAttempts });
      } catch (error) {
        this.log(`Recovery attempt ${this.recoveryAttempts} failed`);
        this.attemptRecovery();
      }
    }, delay);
  }

  // ============================================
  // Connection Pool Management
  // ============================================

  /**
   * Create a new connection
   */
  private async createConnection(url: string): Promise<WebSocketServiceImpl> {
    // Check if connection already exists
    let service = this.pool.connections.get(url);

    if (service?.isConnected()) {
      return service;
    }

    // Create new connection
    const config: ConnectionConfig = {
      url,
      token: this.config.token,
      autoReconnect: true,
      debug: this.config.debug,
    };

    service = createWebSocketService(config);

    // Set up event forwarding
    service.on('close', () => {
      this.handleConnectionClose(url);
    });

    service.on('error', (event) => {
      this.handleConnectionError(url, event.data as Error);
    });

    await service.connect();

    // Store in pool
    this.pool.connections.set(url, service);
    this.pool.health.set(url, this.createHealthRecord());

    // Enforce pool size limit
    if (this.pool.connections.size > this.config.maxPoolSize) {
      this.prunePool();
    }

    return service;
  }

  /**
   * Handle connection close
   */
  private handleConnectionClose(url: string): void {
    this.log(`Connection closed: ${url}`);

    if (url === this.pool.activeConnection) {
      this.updateHealthRecord(url, {
        isHealthy: false,
        consecutiveFailures:
          (this.pool.health.get(url)?.consecutiveFailures || 0) + 1,
      });

      if (this.config.autoFailover) {
        this.failover();
      }
    }
  }

  /**
   * Handle connection error
   */
  private handleConnectionError(url: string, error: Error): void {
    this.log(`Connection error: ${url} - ${error.message}`);

    this.updateHealthRecord(url, {
      isHealthy: false,
      consecutiveFailures:
        (this.pool.health.get(url)?.consecutiveFailures || 0) + 1,
    });
  }

  /**
   * Remove excess connections from pool
   */
  private prunePool(): void {
    if (this.pool.connections.size <= this.config.maxPoolSize) return;

    // Sort by health score and remove lowest
    const entries = Array.from(this.pool.connections.entries())
      .filter(([url]) => url !== this.pool.activeConnection)
      .map(([url, service]) => ({
        url,
        service,
        score: this.pool.health.get(url)?.score || 0,
      }))
      .sort((a, b) => a.score - b.score);

    const toRemove = entries.slice(
      0,
      this.pool.connections.size - this.config.maxPoolSize
    );

    for (const { url, service } of toRemove) {
      service.disconnect();
      this.pool.connections.delete(url);
      this.pool.health.delete(url);
      this.log(`Pruned connection: ${url}`);
    }
  }

  // ============================================
  // Event Management
  // ============================================

  /**
   * Subscribe to connection events
   */
  on(event: ConnectionEventType, handler: ConnectionEventHandler): () => void {
    if (!this.eventHandlers.has(event)) {
      this.eventHandlers.set(event, new Set());
    }
    this.eventHandlers.get(event)!.add(handler);

    return () => this.off(event, handler);
  }

  /**
   * Unsubscribe from connection events
   */
  off(event: ConnectionEventType, handler: ConnectionEventHandler): void {
    this.eventHandlers.get(event)?.delete(handler);
  }

  private emitEvent(type: ConnectionEventType, data: unknown): void {
    const event = { type, data, timestamp: Date.now() };
    this.eventHandlers.get(type)?.forEach((handler) => {
      try {
        handler(event);
      } catch (error) {
        this.log(`Event handler error: ${error}`);
      }
    });
  }

  // ============================================
  // Token Management
  // ============================================

  /**
   * Update authentication token
   */
  setToken(token: string): void {
    this.config.token = token;

    // Update all connections
    for (const service of this.pool.connections.values()) {
      service.setToken(token);
    }
  }

  // ============================================
  // Metrics
  // ============================================

  /**
   * Get aggregated metrics from all connections
   */
  getMetrics(): ConnectionMetrics | null {
    const active = this.getActiveConnection();
    return active?.getMetrics() ?? null;
  }

  /**
   * Get all connection URLs and their health
   */
  getConnectionStatus(): Array<{
    url: string;
    isActive: boolean;
    health: ConnectionHealth | null;
  }> {
    const urls = this.getAllUrls();
    return urls.map((url) => ({
      url,
      isActive: url === this.pool.activeConnection,
      health: this.pool.health.get(url) || null,
    }));
  }

  // ============================================
  // Utilities
  // ============================================

  private getAllUrls(): string[] {
    return [this.config.primaryUrl, ...this.config.fallbackUrls];
  }

  private createHealthRecord(): ConnectionHealth {
    return {
      isHealthy: true,
      status: 'disconnected',
      latency: null,
      lastCheck: 0,
      consecutiveFailures: 0,
      score: 100,
    };
  }

  private updateHealthRecord(
    url: string,
    updates: Partial<ConnectionHealth>
  ): void {
    const current = this.pool.health.get(url) || this.createHealthRecord();
    const updated = { ...current, ...updates };
    updated.score = this.calculateHealthScore(updated);
    this.pool.health.set(url, updated);
  }

  private trackLatency(url: string, latency: number): void {
    if (!this.healthHistory.has(url)) {
      this.healthHistory.set(url, []);
    }

    const history = this.healthHistory.get(url)!;
    history.push(latency);

    // Keep last 100 samples
    if (history.length > 100) {
      history.shift();
    }
  }

  /**
   * Get average latency for a connection
   */
  getAverageLatency(url?: string): number | null {
    const targetUrl = url || this.pool.activeConnection;
    if (!targetUrl) return null;

    const history = this.healthHistory.get(targetUrl);
    if (!history || history.length === 0) return null;

    return history.reduce((a, b) => a + b, 0) / history.length;
  }

  private log(message: string): void {
    if (this.config.debug) {
      console.log(`[ConnectionManager] ${message}`);
    }
  }
}

// ============================================
// Singleton Factory
// ============================================

let managerInstance: ConnectionManager | null = null;

export const createConnectionManager = (
  config: ConnectionManagerConfig
): ConnectionManager => {
  if (managerInstance) {
    managerInstance.disconnect();
  }
  managerInstance = new ConnectionManager(config);
  return managerInstance;
};

export const getConnectionManager = (): ConnectionManager | null => {
  return managerInstance;
};

export default ConnectionManager;

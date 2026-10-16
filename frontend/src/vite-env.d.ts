/// <reference types="vite/client" />
/// <reference types="vite-plugin-pwa/client" />

declare module 'virtual:pwa-register' {
  export interface RegisterSWOptions {
    immediate?: boolean;
    onNeedRefresh?: () => void;
    onOfflineReady?: () => void;
    onRegistered?: (
      registration: ServiceWorkerRegistration | undefined
    ) => void;
    onRegisterError?: (error: Error) => void;
  }

  export function registerSW(
    options?: RegisterSWOptions
  ): (reloadPage?: boolean) => Promise<void>;
}

interface ImportMetaEnv {
  /** API root, default "/api" (same-origin). See src/config/endpoints.ts. */
  readonly VITE_API_URL?: string;
  /** WebSocket URL; unset disables real-time updates in production. */
  readonly VITE_WS_URL?: string;
  readonly VITE_APP_VERSION: string;
  readonly VITE_ENABLE_SW: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

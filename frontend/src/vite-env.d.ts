/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** API root, default "/api" (same-origin). See src/config/endpoints.ts. */
  readonly VITE_API_URL?: string;
  /** WebSocket URL; unset disables real-time updates in production. */
  readonly VITE_APP_VERSION: string;
  /** Public support address used by the Contact and legal pages. */
  readonly VITE_SUPPORT_EMAIL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

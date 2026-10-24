/// <reference types="vite/client" />

interface ImportMetaEnv {
  /** API root, default "/api" (same-origin). See src/config/endpoints.ts. */
  readonly VITE_API_URL?: string;
  /** Release version (root VERSION), baked in by release-deploy.yml. Unset
   * on local builds. Footer build stamp + Sentry release. */
  readonly VITE_APP_VERSION?: string;
  /** Commit the web build was cut from (github.sha), set by release-deploy.yml. */
  readonly VITE_GIT_SHA?: string;
  /** Public support address used by the Contact and legal pages. */
  readonly VITE_SUPPORT_EMAIL?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

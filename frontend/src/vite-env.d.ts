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
  /** Neon Auth URL; set = sign in with Neon Auth (T158). See src/config/auth.ts. */
  readonly VITE_NEON_AUTH_URL?: string;
  /** "true" shows "Continue with Google" (Neon Auth only). */
  readonly VITE_NEON_AUTH_GOOGLE?: string;
}

interface ImportMeta {
  readonly env: ImportMetaEnv;
}

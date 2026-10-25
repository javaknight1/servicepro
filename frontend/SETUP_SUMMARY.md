# Frontend Setup Summary

A short map of the ServicePro SPA as it stands (2026-10). For day-to-day
commands see [README.md](./README.md) and `make help` at the repo root.

## Stack

React 19 + TypeScript (strict), Vite 8, Tailwind CSS 4, React Router 7,
Zustand (client state) + TanStack React Query (server state), React Hook Form +
Zod, Axios, Headless UI, Lucide icons. Served in production by the Cloudflare
Worker `servicepro-web` at <https://servicepro.sharkfins.xyz>.

## Design System

- **Colors** (`@theme` in `src/index.css`, Tailwind v4 CSS-first config):
  **orange primary** (`primary-500` `#f97316`, `primary-600` `#ea580c` — the
  brand color) and **teal secondary** (`secondary-500` `#14b8a6`,
  `secondary-600` `#0d9488`), plus success, warning, error and neutral scales.
- **Typography**: Inter (`--font-sans`, `--font-heading` in `src/index.css`).
- **Shared components** (`src/components/shared/`): Avatar, Badge, Button,
  Card, DataTable, DropdownMenu, EmptyState, ErrorState, ImageCropModal, Input,
  Modal, StatsCard, TaxRatePicker. Follow these patterns for new UI.

## Pages and Routes (`src/routes/index.tsx`)

- **Public:** `/`, `/pricing`, `/features`, `/about`, `/contact`, `/privacy`,
  `/terms`, `/login`, `/register`, `/forgot-password`, `/reset-password`,
  `/verify-email`, `/invitations/accept`, public invoice payment
  success/cancelled pages.
- **App (signed in):** `/dashboard`, `/customers`, `/jobs` (+ `/jobs/calendar`),
  `/quotes`, `/invoices`, `/reports/{revenue,customers,ar-aging}`,
  `/team/{members,roles}`, `/settings`, `/settings/organization` (+ membership).
- `/unauthorized`, `/404`.

## Auth and API

- The API sets **httpOnly cookies**; the SPA never stores tokens. The axios
  instance (`src/services/api.ts`) uses `withCredentials`, sends
  `X-Requested-With: XMLHttpRequest` and does a single-flight refresh on 401.
- All calls are same-origin `/api` (Vite dev proxy locally → `localhost:8080`;
  the Worker in production).
- Neon Auth (managed Better Auth) sign-in screens are the next step of T158.

## Support Contact

`VITE_SUPPORT_EMAIL` (default `support@sharkfins.xyz`, `src/config/support.ts`).

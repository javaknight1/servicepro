# ServicePro Frontend

React SPA for ServicePro, built with TypeScript, Vite and Tailwind CSS, served
in production by the Cloudflare Worker `servicepro-web` at
<https://servicepro.sharkfins.xyz> (static assets + `/api/*` and `/version`
proxy to Cloud Run — see `worker/` and `wrangler.jsonc`).

## Tech Stack

- **Framework**: React 19 + TypeScript (strict)
- **Build Tool**: Vite 8
- **Styling**: Tailwind CSS 4 (CSS-first `@theme` tokens in `src/index.css`)
- **Routing**: React Router 7
- **State Management**: Zustand (client state) + TanStack React Query (server state)
- **Forms**: React Hook Form + Zod
- **HTTP Client**: Axios (`src/services/api.ts`, same-origin `/api`)
- **UI Components**: Headless UI + custom shared components
- **Icons**: Lucide React
- **Edge**: Cloudflare Worker (`worker/`, `wrangler.jsonc`)

## Project Structure

```
src/
├── components/
│   ├── shared/          # Reusable UI (Button, Input, Card, Modal, DataTable, TaxRatePicker, …)
│   ├── layout/          # MainLayout, DashboardLayout, Header, Sidebar, Protected/PublicRoute
│   └── calendar/, invoices/, jobs/, quotes/, reports/, settings/, tenants/, …
├── pages/               # Page components
├── services/            # API services (api.ts axios instance)
├── store/               # Zustand stores (auth, tenant, billing, membership, sidebar)
├── config/              # endpoints, cache config, support contact
├── types/               # TypeScript types (api.generated.ts from the OpenAPI spec)
├── hooks/               # Custom React hooks (React Query hooks)
├── utils/               # Utility functions
└── routes/              # Route configuration
worker/                  # Cloudflare Worker: /api/* + /version proxy
public/_headers          # CSP / HSTS / cache headers for static assets
```

## Getting Started

### Prerequisites

- Node.js 24 (`.nvmrc`)
- npm

### Installation

1. Install dependencies:

```bash
npm install
```

2. Copy the example environment file:

```bash
cp .env.example .env
```

3. Update the `.env` file with your configuration

### Development

Run the development server:

```bash
npm run dev
```

The app will be available at `http://localhost:3000`

### Building for Production

```bash
npm run build
```

Preview the production build:

```bash
npm run preview
```

## Design System

The application uses a comprehensive design system with:

- **Colors**: orange primary (`primary-600` = `#ea580c`), teal secondary
  (`secondary-600` = `#0d9488`), plus success, warning, error and neutral palettes
- **Typography**: Consistent font sizes, weights, and line heights
- **Spacing**: Standardized spacing scale
- **Components**: Reusable UI components following consistent patterns

### Using the Design System

All components use Tailwind CSS classes that reference the design tokens:

```tsx
// Using color palette
className = 'bg-primary-600 text-white';

// Using typography
className = 'text-lg font-semibold';

// Using spacing
className = 'px-4 py-2 mb-6';
```

## Shared Components

### Button

```tsx
import { Button } from '@components/shared';

<Button variant="primary" size="md" isLoading={false}>
  Click Me
</Button>;
```

Variants: `primary`, `secondary`, `outline`, `ghost`, `danger`
Sizes: `sm`, `md`, `lg`

### Input

```tsx
import { Input } from '@components/shared';

<Input label="Email" type="email" error="Invalid email" fullWidth />;
```

### Card

```tsx
import { Card, CardHeader, CardTitle, CardContent } from '@components/shared';

<Card variant="elevated" padding="lg">
  <CardHeader>
    <CardTitle>Title</CardTitle>
  </CardHeader>
  <CardContent>Content goes here</CardContent>
</Card>;
```

### Modal

```tsx
import { Modal } from '@components/shared';

<Modal
  isOpen={isOpen}
  onClose={() => setIsOpen(false)}
  title="Modal Title"
  size="md"
>
  Modal content
</Modal>;
```

## Authentication

The API sets httpOnly cookies on login; the SPA never stores tokens. The axios
client sends `X-Requested-With: XMLHttpRequest` (required on mutating cookie
requests) and refreshes the session on 401. Moving sign-in to Neon Auth
(managed Better Auth) is T158.

```tsx
import { useAuthStore } from '@store';

function MyComponent() {
  const { login, logout, isAuthenticated, user } = useAuthStore();

  // Login
  await login(email, password);

  // Logout
  logout();
}
```

## API Service

API calls go through the shared axios instance (cookies, CSRF header, refresh on 401):

```tsx
import { authApi, userApi } from '@services';

// Login
const response = await authApi.login({ email, password });

// Get current user
const user = await userApi.getCurrentUser();
```

## Routing

The app uses React Router 7 with protected routes (`src/routes/index.tsx`):

- **Public Routes**: Accessible by unauthenticated users (Login, Register, Landing)
- **Protected Routes**: Require authentication (dashboard, customers, jobs, quotes, invoices, reports, team, settings)

Protected routes automatically redirect to login if not authenticated.

## Form Validation

Forms use React Hook Form with Zod for validation:

```tsx
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';

const schema = z.object({
  email: z.string().email(),
  password: z.string().min(8),
});

const {
  register,
  handleSubmit,
  formState: { errors },
} = useForm({
  resolver: zodResolver(schema),
});
```

## Path Aliases

The project uses TypeScript path aliases for cleaner imports:

```tsx
import { Button } from '@components/shared';
import { useAuthStore } from '@store';
import { authApi } from '@services';
import { User } from '@types';
import { cn } from '@utils';
```

## Available Scripts

- `npm run dev` - Start development server
- `npm run build` - Build for production
- `npm run preview` - Preview production build
- `npm run lint` - Run ESLint
- `npm run type-check` - Run TypeScript type checking (app + worker)
- `npm run test` / `npm run test:run` - Vitest
- `npm run cf:dev` - Build and run the Worker locally (`wrangler dev`)
- `npm run generate:api` - Regenerate `src/types/api.generated.ts` (or `make docs` from the root)

From the repo root, prefer the `make frontend-*` targets (`make help`).

## Environment Variables

See `.env.example` (`VITE_*` values are baked in at build time). Key ones:

- `VITE_API_URL` - API base; empty means same-origin `/api` (the Vite dev
  proxy locally, the Worker in production)
- `VITE_SUPPORT_EMAIL` - support address shown in the app (default `support@sharkfins.xyz`)
- `VITE_ENV` - Environment (development, staging, production)

Worker settings live in `wrangler.jsonc` (`API_ORIGIN`) and Worker secrets
(`EDGE_PROXY_SECRET`).

## Contributing

1. Create a new branch for your feature
2. Follow the existing code style and component patterns
3. Use the shared components whenever possible
4. Ensure TypeScript types are properly defined
5. Test your changes before submitting

## License

Proprietary - All rights reserved

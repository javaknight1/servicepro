# ServicePro Frontend

Modern React frontend application for ServicePro platform built with TypeScript, Vite, and Tailwind CSS.

## Tech Stack

- **Framework**: React 18 + TypeScript
- **Build Tool**: Vite
- **Styling**: Tailwind CSS
- **Routing**: React Router v6
- **State Management**: Zustand
- **Forms**: React Hook Form + Zod
- **HTTP Client**: Axios
- **UI Components**: Headless UI + Custom Components
- **Icons**: Lucide React

## Project Structure

```
src/
├── components/
│   ├── shared/          # Reusable UI components
│   │   ├── Button/
│   │   ├── Input/
│   │   ├── Card/
│   │   └── Modal/
│   └── layout/          # Layout components
│       ├── Header.tsx
│       ├── Footer.tsx
│       ├── MainLayout.tsx
│       ├── ProtectedRoute.tsx
│       └── PublicRoute.tsx
├── pages/               # Page components
│   ├── Landing/
│   ├── Login/
│   ├── Register/
│   ├── Dashboard/
│   ├── Settings/
│   └── ...
├── services/            # API services
│   └── api.ts
├── store/               # Zustand stores
│   └── authStore.ts
├── theme/               # Design system
│   ├── colors.ts
│   ├── typography.ts
│   └── spacing.ts
├── types/               # TypeScript types
├── utils/               # Utility functions
├── hooks/               # Custom React hooks
└── routes/              # Route configuration
```

## Getting Started

### Prerequisites

- Node.js 18+
- npm or yarn

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

- **Colors**: Primary, secondary, success, warning, error, and neutral palettes
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

The app uses JWT-based authentication with automatic token refresh:

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

API calls are handled through Axios with automatic token injection:

```tsx
import { authApi, userApi } from '@services';

// Login
const response = await authApi.login({ email, password });

// Get current user
const user = await userApi.getCurrentUser();
```

## Routing

The app uses React Router v6 with protected routes:

- **Public Routes**: Accessible by unauthenticated users (Login, Register, Landing)
- **Protected Routes**: Require authentication (Dashboard, Settings)

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
- `npm run type-check` - Run TypeScript type checking

## Environment Variables

- `VITE_API_URL` - Backend API URL (default: http://localhost:8080/api)
- `VITE_ENV` - Environment (development, staging, production)

## Contributing

1. Create a new branch for your feature
2. Follow the existing code style and component patterns
3. Use the shared components whenever possible
4. Ensure TypeScript types are properly defined
5. Test your changes before submitting

## License

Proprietary - All rights reserved

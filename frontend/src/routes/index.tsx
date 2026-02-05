/**
 * =============================================================================
 * Application Routes
 * =============================================================================
 * Route configuration with code splitting and preloading
 */

import { useEffect } from 'react';
import { Routes, Route, Navigate, useLocation } from 'react-router-dom';
import { ProtectedRoute, PublicRoute } from '@components/layout';
/* eslint-disable react-refresh/only-export-components */
import {
  loadable,
  loadableWithRetry,
  DashboardSkeleton,
} from '@components/Loadable';
import {
  preloadRoute,
  preloadEagerRoutes,
  getPreloadHandler,
} from '@config/splitting';

// =============================================================================
// Lazy Loaded Pages
// =============================================================================

// Public pages
const LandingPage = loadable(
  async () => {
    const m = await import(
      /* webpackChunkName: "page-landing" */ '@pages/Landing'
    );
    return { default: m.LandingPage };
  },
  { chunkName: 'landing', delay: 0 }
);

const LoginPage = loadable(
  async () => {
    const m = await import(/* webpackChunkName: "page-auth" */ '@pages/Login');
    return { default: m.LoginPage };
  },
  { chunkName: 'login', delay: 100 }
);

const RegisterPage = loadable(
  () =>
    import(/* webpackChunkName: "page-auth" */ '@pages/Register').then((m) => ({
      default: m.RegisterPage,
    })),
  { chunkName: 'register', delay: 100 }
);

const ForgotPasswordPage = loadable(
  () =>
    import(/* webpackChunkName: "page-auth" */ '@pages/ForgotPassword').then(
      (m) => ({ default: m.ForgotPasswordPage })
    ),
  { chunkName: 'forgot-password', delay: 100 }
);

const ResetPasswordPage = loadable(
  () =>
    import(/* webpackChunkName: "page-auth" */ '@pages/ResetPassword').then(
      (m) => ({ default: m.ResetPasswordPage })
    ),
  { chunkName: 'reset-password', delay: 100 }
);

const VerifyEmailPage = loadable(
  () =>
    import(/* webpackChunkName: "page-auth" */ '@pages/VerifyEmail').then(
      (m) => ({ default: m.VerifyEmailPage })
    ),
  { chunkName: 'verify-email', delay: 100 }
);

const AcceptInvitationPage = loadable(
  () =>
    import(/* webpackChunkName: "page-auth" */ '@pages/AcceptInvitation').then(
      (m) => ({ default: m.AcceptInvitationPage })
    ),
  { chunkName: 'accept-invitation', delay: 100 }
);

// Protected pages with retry logic
const DashboardPage = loadableWithRetry(
  () =>
    import(/* webpackChunkName: "page-dashboard" */ '@pages/Dashboard').then(
      (m) => ({ default: m.DashboardPage })
    ),
  {
    chunkName: 'dashboard',
    delay: 0,
    fallback: <DashboardSkeleton />,
    timeout: 10000,
  },
  { retries: 3, delay: 1000 }
);

const SettingsPage = loadableWithRetry(
  () =>
    import(/* webpackChunkName: "page-settings" */ '@pages/Settings').then(
      (m) => ({ default: m.SettingsPage })
    ),
  { chunkName: 'settings', delay: 100 },
  { retries: 2 }
);

const OrgSettingsPage = loadable(
  () =>
    import(/* webpackChunkName: "page-settings" */ '@pages/Settings').then(
      (m) => ({ default: m.OrgSettingsPage })
    ),
  { chunkName: 'org-settings', delay: 100 }
);

const NewOrgPage = loadable(
  () =>
    import(/* webpackChunkName: "page-settings" */ '@pages/Settings').then(
      (m) => ({ default: m.NewOrgPage })
    ),
  { chunkName: 'new-org', delay: 100 }
);

const ChangeMembershipPage = loadable(
  () =>
    import(/* webpackChunkName: "page-settings" */ '@pages/Settings').then(
      (m) => ({ default: m.ChangeMembershipPage })
    ),
  { chunkName: 'change-membership', delay: 100 }
);

const ConfirmMembershipPage = loadable(
  () =>
    import(/* webpackChunkName: "page-settings" */ '@pages/Settings').then(
      (m) => ({ default: m.ConfirmMembershipPage })
    ),
  { chunkName: 'confirm-membership', delay: 100 }
);

const PricingPage = loadable(
  () =>
    import(/* webpackChunkName: "page-pricing" */ '@pages/Pricing').then(
      (m) => ({ default: m.PricingPage })
    ),
  { chunkName: 'pricing', delay: 100 }
);

const FeaturesPage = loadable(
  () =>
    import(/* webpackChunkName: "page-features" */ '@pages/Features').then(
      (m) => ({ default: m.FeaturesPage })
    ),
  { chunkName: 'features', delay: 100 }
);

const AboutPage = loadable(
  () =>
    import(/* webpackChunkName: "page-about" */ '@pages/About').then((m) => ({
      default: m.AboutPage,
    })),
  { chunkName: 'about', delay: 100 }
);

const ContactPage = loadable(
  () =>
    import(/* webpackChunkName: "page-contact" */ '@pages/Contact').then(
      (m) => ({ default: m.ContactPage })
    ),
  { chunkName: 'contact', delay: 100 }
);

const PrivacyPolicyPage = loadable(
  () =>
    import(/* webpackChunkName: "page-legal" */ '@pages/Legal').then((m) => ({
      default: m.PrivacyPolicyPage,
    })),
  { chunkName: 'privacy-policy', delay: 100 }
);

const TermsOfServicePage = loadable(
  () =>
    import(/* webpackChunkName: "page-legal" */ '@pages/Legal').then((m) => ({
      default: m.TermsOfServicePage,
    })),
  { chunkName: 'terms-of-service', delay: 100 }
);

// Module pages
const CustomersPage = loadable(
  () =>
    import(/* webpackChunkName: "page-customers" */ '@pages/Customers').then(
      (m) => ({ default: m.CustomersPage })
    ),
  { chunkName: 'customers', delay: 100 }
);

const CustomerDetailPage = loadable(
  () =>
    import(/* webpackChunkName: "page-customers" */ '@pages/Customers').then(
      (m) => ({ default: m.CustomerDetailPage })
    ),
  { chunkName: 'customer-detail', delay: 100 }
);

const JobsPage = loadable(
  () =>
    import(/* webpackChunkName: "page-jobs" */ '@pages/Jobs').then((m) => ({
      default: m.JobsPage,
    })),
  { chunkName: 'jobs', delay: 100 }
);

const QuotesPage = loadable(
  () =>
    import(/* webpackChunkName: "page-quotes" */ '@pages/Quotes').then((m) => ({
      default: m.QuotesPage,
    })),
  { chunkName: 'quotes', delay: 100 }
);

const InvoicesPage = loadable(
  () =>
    import(/* webpackChunkName: "page-invoices" */ '@pages/Invoices').then(
      (m) => ({ default: m.InvoicesPage })
    ),
  { chunkName: 'invoices', delay: 100 }
);

const JobDetailPage = loadable(
  () =>
    import(/* webpackChunkName: "page-jobs" */ '@pages/Jobs').then((m) => ({
      default: m.JobDetailPage,
    })),
  { chunkName: 'job-detail', delay: 100 }
);

const JobStatusHistoryPage = loadable(
  () =>
    import(/* webpackChunkName: "page-jobs" */ '@pages/Jobs').then((m) => ({
      default: m.JobStatusHistoryPage,
    })),
  { chunkName: 'job-status-history', delay: 100 }
);

const JobCalendarPage = loadable(
  () =>
    import(/* webpackChunkName: "page-jobs" */ '@pages/Jobs').then((m) => ({
      default: m.JobCalendarPage,
    })),
  { chunkName: 'job-calendar', delay: 100 }
);

const QuoteDetailPage = loadable(
  () =>
    import(/* webpackChunkName: "page-quotes" */ '@pages/Quotes').then((m) => ({
      default: m.QuoteDetailPage,
    })),
  { chunkName: 'quote-detail', delay: 100 }
);

const InvoiceDetailPage = loadable(
  () =>
    import(/* webpackChunkName: "page-invoices" */ '@pages/Invoices').then(
      (m) => ({ default: m.InvoiceDetailPage })
    ),
  { chunkName: 'invoice-detail', delay: 100 }
);

const PaymentSuccessPage = loadable(
  () =>
    import(/* webpackChunkName: "page-invoices" */ '@pages/Invoices').then(
      (m) => ({ default: m.PaymentSuccessPage })
    ),
  { chunkName: 'payment-success', delay: 100 }
);

const PaymentCancelledPage = loadable(
  () =>
    import(/* webpackChunkName: "page-invoices" */ '@pages/Invoices').then(
      (m) => ({ default: m.PaymentCancelledPage })
    ),
  { chunkName: 'payment-cancelled', delay: 100 }
);

// Team pages
const TeamMembersPage = loadable(
  () =>
    import(/* webpackChunkName: "page-team" */ '@pages/Team').then((m) => ({
      default: m.TeamMembersPage,
    })),
  { chunkName: 'team-members', delay: 100 }
);

const TeamRolesPage = loadable(
  () =>
    import(/* webpackChunkName: "page-team" */ '@pages/Team').then((m) => ({
      default: m.TeamRolesPage,
    })),
  { chunkName: 'team-roles', delay: 100 }
);

// Report pages
const RevenueReportPage = loadable(
  () =>
    import(/* webpackChunkName: "page-reports" */ '@pages/Reports').then(
      (m) => ({ default: m.RevenueReportPage })
    ),
  { chunkName: 'revenue-report', delay: 100 }
);

const CustomerReportPage = loadable(
  () =>
    import(/* webpackChunkName: "page-reports" */ '@pages/Reports').then(
      (m) => ({ default: m.CustomerReportPage })
    ),
  { chunkName: 'customer-report', delay: 100 }
);

const ARAgingReportPage = loadable(
  () =>
    import(/* webpackChunkName: "page-reports" */ '@pages/Reports').then(
      (m) => ({ default: m.ARAgingReportPage })
    ),
  { chunkName: 'ar-aging-report', delay: 100 }
);

// Error pages
const NotFoundPage = loadable(
  () =>
    import(/* webpackChunkName: "page-error" */ '@pages/NotFound').then(
      (m) => ({ default: m.NotFoundPage })
    ),
  { chunkName: 'not-found', delay: 0 }
);

const UnauthorizedPage = loadable(
  () =>
    import(/* webpackChunkName: "page-error" */ '@pages/Unauthorized').then(
      (m) => ({ default: m.UnauthorizedPage })
    ),
  { chunkName: 'unauthorized', delay: 0 }
);

// =============================================================================
// Route Preloading Hook
// =============================================================================

function useRoutePreloading() {
  const location = useLocation();

  useEffect(() => {
    // Preload high-priority routes on app mount
    preloadEagerRoutes();
  }, []);

  useEffect(() => {
    // Preload related routes based on current location
    const path = location.pathname;

    // Preload routes likely to be visited from current page
    if (path === '/' || path === '/login') {
      preloadRoute('dashboard', { delay: 500 });
      preloadRoute('register', { delay: 1000 });
    }

    if (path === '/dashboard') {
      preloadRoute('settings', { delay: 500 });
      preloadRoute('customers', { delay: 500 });
      preloadRoute('jobs', { delay: 500 });
      preloadRoute('revenueReport', { delay: 1000 });
    }

    if (path.startsWith('/reports')) {
      preloadRoute('revenueReport', { delay: 0 });
      preloadRoute('customerReport', { delay: 0 });
    }
  }, [location.pathname]);
}

// =============================================================================
// Preload Handlers (for link hover)
// =============================================================================

export const preloadHandlers = {
  login: getPreloadHandler('login'),
  register: getPreloadHandler('register'),
  dashboard: getPreloadHandler('dashboard'),
  customers: getPreloadHandler('customers'),
  jobs: getPreloadHandler('jobs'),
  quotes: getPreloadHandler('quotes'),
  invoices: getPreloadHandler('invoices'),
  settings: getPreloadHandler('settings'),
  revenueReport: getPreloadHandler('revenueReport'),
  customerReport: getPreloadHandler('customerReport'),
};

// =============================================================================
// App Routes Component
// =============================================================================

export function AppRoutes() {
  // Enable route preloading
  useRoutePreloading();

  return (
    <Routes>
      {/* Public routes - redirect to dashboard if authenticated */}
      <Route element={<PublicRoute />}>
        <Route path="/" element={<LandingPage />} />
        <Route path="/login" element={<LoginPage />} />
        <Route path="/register" element={<RegisterPage />} />
        <Route path="/forgot-password" element={<ForgotPasswordPage />} />
        <Route path="/reset-password" element={<ResetPasswordPage />} />
      </Route>

      {/* Public routes - accessible regardless of auth status */}
      <Route path="/verify-email" element={<VerifyEmailPage />} />
      <Route path="/unauthorized" element={<UnauthorizedPage />} />
      <Route path="/pricing" element={<PricingPage />} />
      <Route path="/features" element={<FeaturesPage />} />
      <Route path="/about" element={<AboutPage />} />
      <Route path="/contact" element={<ContactPage />} />
      <Route path="/privacy" element={<PrivacyPolicyPage />} />
      <Route path="/terms" element={<TermsOfServicePage />} />
      <Route path="/invitations/accept" element={<AcceptInvitationPage />} />

      {/* Protected routes - require authentication */}
      <Route element={<ProtectedRoute />}>
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/customers" element={<CustomersPage />} />
        <Route path="/customers/new" element={<CustomerDetailPage />} />
        <Route path="/customers/:id" element={<CustomerDetailPage />} />
        <Route path="/jobs" element={<JobsPage />} />
        <Route path="/jobs/calendar" element={<JobCalendarPage />} />
        <Route path="/jobs/new" element={<JobDetailPage />} />
        <Route path="/jobs/:id" element={<JobDetailPage />} />
        <Route path="/jobs/:id/history" element={<JobStatusHistoryPage />} />
        <Route path="/quotes" element={<QuotesPage />} />
        <Route path="/quotes/new" element={<QuoteDetailPage />} />
        <Route path="/quotes/:id" element={<QuoteDetailPage />} />
        <Route path="/invoices" element={<InvoicesPage />} />
        <Route path="/invoices/new" element={<InvoiceDetailPage />} />
        <Route
          path="/invoices/payment-success"
          element={<PaymentSuccessPage />}
        />
        <Route
          path="/invoices/payment-cancelled"
          element={<PaymentCancelledPage />}
        />
        <Route path="/invoices/:id" element={<InvoiceDetailPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="/settings/organization" element={<OrgSettingsPage />} />
        <Route path="/settings/organization/new" element={<NewOrgPage />} />
        <Route
          path="/settings/organization/membership/change"
          element={<ChangeMembershipPage />}
        />
        <Route
          path="/settings/organization/membership/confirm"
          element={<ConfirmMembershipPage />}
        />
        <Route path="/team/members" element={<TeamMembersPage />} />
        <Route path="/team/roles" element={<TeamRolesPage />} />
        <Route path="/reports/revenue" element={<RevenueReportPage />} />
        <Route path="/reports/customers" element={<CustomerReportPage />} />
        <Route path="/reports/ar-aging" element={<ARAgingReportPage />} />
      </Route>

      {/* 404 */}
      <Route path="/404" element={<NotFoundPage />} />
      <Route path="*" element={<Navigate to="/404" replace />} />
    </Routes>
  );
}

// =============================================================================
// Route Utilities
// =============================================================================

/**
 * Route paths for type-safe navigation
 */
export const routePaths = {
  home: '/',
  login: '/login',
  register: '/register',
  forgotPassword: '/forgot-password',
  resetPassword: '/reset-password',
  verifyEmail: '/verify-email',
  acceptInvitation: '/invitations/accept',
  dashboard: '/dashboard',
  customers: '/customers',
  customerNew: '/customers/new',
  customerDetail: (id: string) => `/customers/${id}`,
  jobs: '/jobs',
  jobCalendar: '/jobs/calendar',
  quotes: '/quotes',
  invoices: '/invoices',
  settings: '/settings',
  orgSettings: '/settings/organization',
  newOrg: '/settings/organization/new',
  changeMembership: '/settings/organization/membership/change',
  confirmMembership: '/settings/organization/membership/confirm',
  pricing: '/pricing',
  features: '/features',
  about: '/about',
  contact: '/contact',
  privacy: '/privacy',
  terms: '/terms',
  team: {
    members: '/team/members',
    roles: '/team/roles',
  },
  reports: {
    revenue: '/reports/revenue',
    customers: '/reports/customers',
    arAging: '/reports/ar-aging',
  },
  unauthorized: '/unauthorized',
  notFound: '/404',
} as const;

/**
 * Check if a path requires authentication
 */
export function isProtectedRoute(path: string): boolean {
  const protectedPaths = [
    '/dashboard',
    '/customers',
    '/jobs',
    '/quotes',
    '/invoices',
    '/settings',
    '/team',
    '/reports',
  ];

  return protectedPaths.some((p) => path.startsWith(p));
}

/**
 * Get the redirect path after login
 */
export function getPostLoginRedirect(from?: string): string {
  if (from && isProtectedRoute(from)) {
    return from;
  }
  return routePaths.dashboard;
}

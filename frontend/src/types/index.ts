export type {
  User,
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  RegisterResponse,
  RegisterOptions,
  PasswordResetRequestRequest,
  PasswordResetConfirmRequest,
  VerifyEmailRequest,
  ResendVerificationRequest,
  UpdateUserRequest,
  ErrorResponse,
  AuthState,
} from './auth';

export type {
  Permission,
  Role,
  RoleAssignment,
  RoleAuditLog,
  CreateRoleRequest,
  UpdateRoleRequest,
  AssignRoleRequest,
  BulkAssignRolesRequest,
  RoleFilters,
  AuditFilters,
} from './role';

export type {
  Customer,
  CustomerFilters,
  CustomerListResponse,
  CustomerStats,
  CustomerSegment,
  CustomerType,
  CustomerStatus,
} from './customer';
export { getCustomerStatusLabel, getCustomerTypeLabel } from './customer';

export type { Job, JobFilters, JobListResponse, JobStats } from './job';
export {
  JobStatus,
  JobPriority,
  getJobStatusLabel,
  getJobPriorityLabel,
  getJobTypeLabel,
  getNextStatusLabel,
} from './job';

export type {
  Invoice,
  InvoiceFilters,
  InvoiceListResponse,
  InvoiceStats,
  InvoiceLineItem,
} from './invoice';
export { InvoiceStatus, getInvoiceStatusLabel } from './invoice';

export type {
  Quote,
  QuoteRequest,
  QuoteListResponse,
  QuoteStats,
  LineItem,
  LineItemFormData,
} from './quote';
export { QuoteStatus } from './quote';

export type {
  Tenant,
  TenantSettings,
  TenantMember,
  CreateTenantRequest,
  UpdateTenantRequest,
  AddMemberRequest,
  UpdateMemberRoleRequest,
  TenantListResponse,
  TenantMembersResponse,
  TenantState,
} from './tenant';

/** The API's error envelope (T223, docs/api/errors.md). */
export interface ApiError {
  /** Stable snake_case machine code. */
  error: string;
  /** Human-readable, safe to show. */
  message?: string;
  /** Matches the X-Request-ID response header. */
  request_id?: string;
  /** Validation errors: request field -> problem. */
  fields?: Record<string, string>;
  /** Extra messages, e.g. per-row import errors. */
  details?: string[];
  statusCode?: number;
}

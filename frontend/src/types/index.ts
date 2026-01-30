export type {
  User,
  LoginRequest,
  LoginResponse,
  RegisterRequest,
  RegisterResponse,
  PasswordResetRequestRequest,
  PasswordResetConfirmRequest,
  VerifyEmailRequest,
  ResendVerificationRequest,
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
  getNextStatusLabel,
} from './job';

export type {
  Invoice,
  InvoiceFilters,
  InvoiceListResponse,
  InvoiceStats,
  InvoiceLineItem,
} from './invoice';
export {
  InvoiceStatus,
  getInvoiceStatusLabel,
  formatCurrency,
} from './invoice';

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

export interface ApiError {
  error: string;
  message?: string;
  statusCode?: number;
}

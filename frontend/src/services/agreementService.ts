import api from './api';
import type { JobType } from './jobService';

export type AgreementFrequency =
  'weekdays' | 'weekly' | 'times_per_week' | 'monthly';
export type AgreementStatus = 'active' | 'paused';
/** What `price_per_period` is the price of. */
export type BillingPeriod = 'month' | 'visit';

/** A service agreement: recurring work at one customer site (T167). */
export interface Agreement {
  id: string;
  customer_id: string;
  site_id: string;
  title: string;
  frequency: AgreementFrequency;
  /** 0 = Sunday … 6 = Saturday. */
  weekdays: number[];
  times_per_week?: number;
  day_of_month?: number;
  /** YYYY-MM-DD. */
  start_date: string;
  /** YYYY-MM-DD, inclusive. */
  end_date?: string;
  /** HH:MM in the tenant's time zone. */
  visit_start_time: string;
  visit_duration_minutes: number;
  /** API decimal, e.g. "2400.00". */
  price_per_period: string;
  billing_period: BillingPeriod;
  assigned_technician_id?: string;
  job_type: JobType;
  status: AgreementStatus;
  notes?: string;
  created_at: string;
  updated_at: string;
}

/** An agreement's frequency (models.AgreementScheduleInput). */
export interface AgreementSchedule {
  frequency: AgreementFrequency;
  weekdays?: number[];
  times_per_week?: number;
  day_of_month?: number;
}

/** Body of POST /agreements. */
export interface CreateAgreementRequest extends AgreementSchedule {
  customer_id: string;
  site_id: string;
  title: string;
  start_date: string;
  end_date?: string;
  visit_start_time?: string;
  visit_duration_minutes?: number;
  price_per_period: string;
  billing_period?: BillingPeriod;
  assigned_technician_id?: string;
  job_type?: JobType;
  notes?: string;
}

/** Body of PUT /agreements/{id}; omitted fields are left unchanged. */
export interface UpdateAgreementRequest {
  title?: string;
  /** Replaces the whole frequency. */
  schedule?: AgreementSchedule;
  start_date?: string;
  end_date?: string;
  clear_end_date?: boolean;
  visit_start_time?: string;
  visit_duration_minutes?: number;
  price_per_period?: string;
  billing_period?: BillingPeriod;
  assigned_technician_id?: string;
  clear_technician?: boolean;
  job_type?: JobType;
  notes?: string;
}

export interface AgreementFilters {
  customer_id?: string;
  site_id?: string;
  status?: AgreementStatus;
}

/** One upcoming visit (GET /agreements/{id}/preview). */
export interface AgreementVisitPreview {
  date: string;
  starts_at: string;
  ends_at: string;
}

export interface AgreementPreview {
  agreement_id: string;
  timezone: string;
  visits: AgreementVisitPreview[];
}

const basePath = '/v1/agreements';

/** Service agreements (T167). */
export const agreementService = {
  async list(filters: AgreementFilters = {}): Promise<Agreement[]> {
    const response = await api.get<{ agreements: Agreement[] }>(basePath, {
      params: filters,
    });
    return response.data.agreements ?? [];
  },

  async get(id: string): Promise<Agreement> {
    const response = await api.get<Agreement>(`${basePath}/${id}`);
    return response.data;
  },

  async create(data: CreateAgreementRequest): Promise<Agreement> {
    const response = await api.post<Agreement>(basePath, data);
    return response.data;
  },

  async update(id: string, data: UpdateAgreementRequest): Promise<Agreement> {
    const response = await api.put<Agreement>(`${basePath}/${id}`, data);
    return response.data;
  },

  async pause(id: string): Promise<Agreement> {
    const response = await api.post<Agreement>(`${basePath}/${id}/pause`);
    return response.data;
  },

  async resume(id: string): Promise<Agreement> {
    const response = await api.post<Agreement>(`${basePath}/${id}/resume`);
    return response.data;
  },

  async remove(id: string): Promise<void> {
    await api.delete(`${basePath}/${id}`);
  },

  async preview(id: string, days = 14): Promise<AgreementPreview> {
    const response = await api.get<AgreementPreview>(
      `${basePath}/${id}/preview`,
      { params: { days } }
    );
    return response.data;
  },
};

/** React Query keys for agreements. */
export const agreementKeys = {
  all: ['agreements'] as const,
  list: (filters: AgreementFilters = {}) =>
    ['agreements', 'list', filters] as const,
  detail: (id: string | null | undefined) =>
    ['agreements', 'detail', id ?? null] as const,
  preview: (id: string | null | undefined, days: number) =>
    ['agreements', 'preview', id ?? null, days] as const,
};

const DAY_SHORT = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
const DAY_LETTER = ['Su', 'M', 'Tu', 'W', 'Th', 'F', 'Sa'];

/** Default days for "N times a week" (mirrors models/agreement.go). */
const TIMES_PER_WEEK_DAYS: Record<number, number[]> = {
  1: [1],
  2: [1, 4],
  3: [1, 3, 5],
  4: [1, 2, 4, 5],
  5: [1, 2, 3, 4, 5],
  6: [1, 2, 3, 4, 5, 6],
  7: [0, 1, 2, 3, 4, 5, 6],
};

function ordinal(n: number): string {
  const s = ['th', 'st', 'nd', 'rd'];
  const v = n % 100;
  return `${n}${s[(v - 20) % 10] || s[v] || s[0]}`;
}

/** Monday-first order for display. */
function mondayFirst(days: number[]): number[] {
  return [...days].sort((a, b) => ((a + 6) % 7) - ((b + 6) % 7));
}

/** "Nightly, Mon–Fri", "3x weekly, M/W/F", "Weekly, Tue/Fri", "Monthly on the 15th". */
export function describeFrequency(
  a: Pick<
    Agreement,
    'frequency' | 'weekdays' | 'times_per_week' | 'day_of_month'
  >
): string {
  switch (a.frequency) {
    case 'weekdays':
      return 'Nightly, Mon–Fri';
    case 'weekly': {
      const days = mondayFirst(a.weekdays ?? []);
      return `Weekly, ${days.map((d) => DAY_SHORT[d]).join('/')}`;
    }
    case 'times_per_week': {
      const n = a.times_per_week ?? 0;
      const days = mondayFirst(
        a.weekdays?.length ? a.weekdays : (TIMES_PER_WEEK_DAYS[n] ?? [])
      );
      return `${n}x weekly, ${days.map((d) => DAY_LETTER[d]).join('/')}`;
    }
    case 'monthly':
      return a.day_of_month
        ? `Monthly on the ${ordinal(a.day_of_month)}`
        : 'Monthly';
  }
}

/** Approximate visits a month, for a monthly figure of per-visit prices. */
function visitsPerMonth(
  a: Pick<
    Agreement,
    'frequency' | 'weekdays' | 'times_per_week' | 'day_of_month'
  >
): number {
  const weeksPerMonth = 52 / 12;
  switch (a.frequency) {
    case 'weekdays':
      return 5 * weeksPerMonth;
    case 'weekly':
      return (a.weekdays?.length ?? 0) * weeksPerMonth;
    case 'times_per_week':
      return (a.times_per_week ?? 0) * weeksPerMonth;
    case 'monthly':
      return 1;
  }
}

/**
 * The agreement's price per month: the price itself for monthly billing,
 * else the per-visit price times the average visits a month (an estimate).
 */
export function monthlyAmount(a: Agreement): number {
  const price = Number.parseFloat(a.price_per_period) || 0;
  if (a.billing_period === 'month') return price;
  return Math.round(price * visitsPerMonth(a) * 100) / 100;
}

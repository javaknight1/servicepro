/**
 * Recurrence Type
 */
export enum RecurrenceType {
  NONE = 'none',
  DAILY = 'daily',
  WEEKLY = 'weekly',
  MONTHLY = 'monthly',
  YEARLY = 'yearly',
  CUSTOM = 'custom',
}

/**
 * Exception Type
 */
export enum ExceptionType {
  SKIP = 'skip',
  RESCHEDULE = 'reschedule',
  MODIFY = 'modify',
}

/**
 * Recurring Pattern Request
 */
export interface RecurringPatternRequest {
  title: string;
  description?: string;
  recurrenceType: RecurrenceType;
  startDate: string;
  endDate?: string;
  interval: number;
  daysOfWeek?: number[];
  dayOfMonth?: number;
  monthOfYear?: number;
  occurrences?: number;
  timeStart: string;
  timeEnd: string;
  assignedTechIds: string[];
  location?: string;
  jobTemplateId?: string;
}

/**
 * Recurring Pattern Response
 */
export interface RecurringPatternResponse {
  id: string;
  title: string;
  description: string;
  recurrenceType: RecurrenceType;
  recurrenceRule: string;
  startDate: string;
  endDate?: string;
  interval: number;
  daysOfWeek?: number[];
  dayOfMonth?: number;
  monthOfYear?: number;
  occurrences?: number;
  timeStart: string;
  timeEnd: string;
  duration: number;
  assignedTechIds: string[];
  location: string;
  jobTemplateId?: string;
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
}

/**
 * Generation Request
 */
export interface GenerationRequest {
  recurringScheduleId: string;
  startDate: string;
  endDate: string;
  skipHolidays: boolean;
  skipWeekends: boolean;
  maxOccurrences: number;
}

/**
 * Skipped Date
 */
export interface SkippedDate {
  date: string;
  reason: string;
}

/**
 * Generation Result
 */
export interface GenerationResult {
  generatedCount: number;
  skippedCount: number;
  schedules: any[];
  skippedDates?: SkippedDate[];
}

/**
 * Schedule Exception
 */
export interface ScheduleException {
  id: string;
  recurringPatternId: string;
  exceptionDate: string;
  exceptionType: ExceptionType;
  reason: string;
  rescheduledDate?: string;
  modifications?: string;
  createdBy: string;
  createdAt: string;
  updatedAt: string;
}

/**
 * Exception Request
 */
export interface ExceptionRequest {
  recurringPatternId: string;
  exceptionDate: string;
  exceptionType: ExceptionType;
  reason: string;
  rescheduledDate?: string;
  modifications?: string;
}

/**
 * Holiday
 */
export interface Holiday {
  id: string;
  name: string;
  date: string;
  country: string;
  isRecurring: boolean;
  createdAt: string;
  updatedAt: string;
}

/**
 * Holiday Request
 */
export interface HolidayRequest {
  name: string;
  date: string;
  country?: string;
  isRecurring: boolean;
}

/**
 * Recurring Form Props
 */
export interface RecurringFormProps {
  initialData?: RecurringPatternResponse;
  onSubmit: (data: RecurringPatternRequest) => void;
  onCancel?: () => void;
  isLoading?: boolean;
}

/**
 * Pattern Preview Props
 */
export interface PatternPreviewProps {
  pattern: RecurringPatternRequest | RecurringPatternResponse;
  previewMonths?: number;
  showSkipped?: boolean;
}

/**
 * Day of Week Constants
 */
export const DAYS_OF_WEEK = [
  { value: 0, label: 'Sunday', short: 'Sun' },
  { value: 1, label: 'Monday', short: 'Mon' },
  { value: 2, label: 'Tuesday', short: 'Tue' },
  { value: 3, label: 'Wednesday', short: 'Wed' },
  { value: 4, label: 'Thursday', short: 'Thu' },
  { value: 5, label: 'Friday', short: 'Fri' },
  { value: 6, label: 'Saturday', short: 'Sat' },
];

/**
 * Month Constants
 */
export const MONTHS = [
  { value: 1, label: 'January' },
  { value: 2, label: 'February' },
  { value: 3, label: 'March' },
  { value: 4, label: 'April' },
  { value: 5, label: 'May' },
  { value: 6, label: 'June' },
  { value: 7, label: 'July' },
  { value: 8, label: 'August' },
  { value: 9, label: 'September' },
  { value: 10, label: 'October' },
  { value: 11, label: 'November' },
  { value: 12, label: 'December' },
];

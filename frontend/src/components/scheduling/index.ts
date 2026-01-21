/**
 * Scheduling Components
 *
 * Real-time conflict detection and resolution for schedule management.
 */

export {
  ConflictChecker,
  default as ConflictCheckerDefault,
} from './ConflictChecker';
export { ConflictAlert } from './ConflictAlert';

export type {
  ConflictType,
  ConflictSeverity,
  ConflictDetail,
  ConflictCheckRequest,
  ConflictCheckResponse,
  ResolutionSuggestion,
  ResolutionStrategy,
  TimeSlotSuggestion,
  TechnicianSuggestion,
  SplitScheduleSuggestion,
  ScheduleSegment,
  ValidationError,
  ValidationResult,
  ConflictCheckerProps,
  ConflictAlertProps,
  ConflictingSchedule,
  TimeOverlapInfo,
} from './types';

export {
  ConflictType as ConflictTypeEnum,
  ConflictSeverity as ConflictSeverityEnum,
} from './types';

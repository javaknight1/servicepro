/**
 * Conflict Types
 * Matches backend ConflictType enum
 */
export enum ConflictType {
  TECHNICIAN_OVERLAP = 'technician_overlap',
  LOCATION_OVERLAP = 'location_overlap',
  EQUIPMENT_OVERLAP = 'equipment_overlap',
  TIME_OVERLAP = 'time_overlap',
  BUSINESS_HOURS = 'business_hours',
  WORKLOAD_EXCESS = 'workload_excess',
}

/**
 * Conflict Severity Levels
 * Matches backend ConflictSeverity enum
 */
export enum ConflictSeverity {
  LOW = 'low',
  MEDIUM = 'medium',
  HIGH = 'high',
  CRITICAL = 'critical',
}

/**
 * Conflicting Schedule Information
 */
export interface ConflictingSchedule {
  id: string;
  title: string;
  jobNumber?: string;
  startTime: string;
  endTime: string;
  location?: string;
}

/**
 * Time Overlap Information
 */
export interface TimeOverlapInfo {
  overlapStart: string;
  overlapEnd: string;
  overlapMinutes: number;
}

/**
 * Conflict Detail
 */
export interface ConflictDetail {
  conflictType: ConflictType;
  severity: ConflictSeverity;
  description: string;
  conflictingWith?: ConflictingSchedule;
  technicianId?: string;
  timeOverlap?: TimeOverlapInfo;
}

/**
 * Resolution Suggestion
 */
export interface ResolutionSuggestion {
  type: string;
  description: string;
  action: string;
  priority: number;
}

/**
 * Conflict Check Response
 */
export interface ConflictCheckResponse {
  hasConflicts: boolean;
  conflicts: ConflictDetail[];
  suggestions?: ResolutionSuggestion[];
}

/**
 * Conflict Check Request
 */
export interface ConflictCheckRequest {
  scheduleId?: string;
  jobId: string;
  startTime: string;
  endTime: string;
  assignedTechIds: string[];
  location?: string;
}

/**
 * Time Slot Suggestion
 */
export interface TimeSlotSuggestion {
  startTime: string;
  endTime: string;
  availableSlots: number;
  reason: string;
}

/**
 * Technician Suggestion
 */
export interface TechnicianSuggestion {
  technicianId: string;
  name: string;
  email: string;
  availability: string;
  currentWorkload: number;
  skills?: string[];
}

/**
 * Schedule Segment for Split Suggestions
 */
export interface ScheduleSegment {
  startTime: string;
  endTime: string;
  assignedTechIds: string[];
  notes?: string;
}

/**
 * Split Schedule Suggestion
 */
export interface SplitScheduleSuggestion {
  splitCount: number;
  segments: ScheduleSegment[];
  reason: string;
}

/**
 * Resolution Strategy
 */
export interface ResolutionStrategy {
  id: string;
  type: string;
  title: string;
  description: string;
  confidence: number;
  impactLevel: 'low' | 'medium' | 'high';
  autoApplicable: boolean;
  suggestion:
    | TimeSlotSuggestion[]
    | TechnicianSuggestion[]
    | SplitScheduleSuggestion
    | null;
  conflictsResolved: number;
}

/**
 * Validation Error
 */
export interface ValidationError {
  field?: string;
  code: string;
  message: string;
  severity: 'error' | 'warning';
}

/**
 * Validation Result
 */
export interface ValidationResult {
  isValid: boolean;
  errors?: ValidationError[];
  warnings?: ValidationError[];
}

/**
 * Conflict Checker Props
 */
export interface ConflictCheckerProps {
  scheduleId?: string;
  jobId: string;
  startTime: Date;
  endTime: Date;
  assignedTechIds: string[];
  location?: string;
  onConflictDetected?: (response: ConflictCheckResponse) => void;
  onConflictResolved?: () => void;
  autoCheck?: boolean;
  showSuggestions?: boolean;
}

/**
 * Conflict Alert Props
 */
export interface ConflictAlertProps {
  conflicts: ConflictDetail[];
  suggestions?: ResolutionSuggestion[];
  onDismiss?: () => void;
  onSelectSuggestion?: (suggestion: ResolutionSuggestion) => void;
  className?: string;
}

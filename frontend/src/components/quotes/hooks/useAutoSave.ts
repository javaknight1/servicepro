import { useEffect, useRef, useState, useCallback } from 'react';
import { UseFormWatch } from 'react-hook-form';
import { QuoteFormData } from '../validation';

export interface AutoSaveOptions {
  /** Debounce delay in milliseconds (default: 2000) */
  delay?: number;
  /** Callback when save starts */
  onSaveStart?: () => void;
  /** Callback when save completes successfully */
  onSaveSuccess?: (data: QuoteFormData) => void;
  /** Callback when save fails */
  onSaveError?: (error: Error) => void;
  /** Enable auto-save (default: true) */
  enabled?: boolean;
}

export interface AutoSaveStatus {
  isSaving: boolean;
  lastSaved: Date | null;
  error: Error | null;
}

/**
 * Hook for auto-saving form data with debouncing
 *
 * @param watch - React Hook Form watch function
 * @param onSave - Async function to save the data
 * @param options - Configuration options
 * @returns Auto-save status information
 */
export const useAutoSave = (
  watch: UseFormWatch<QuoteFormData>,
  onSave: (data: QuoteFormData) => Promise<void>,
  options: AutoSaveOptions = {}
): AutoSaveStatus => {
  const {
    delay = 2000,
    onSaveStart,
    onSaveSuccess,
    onSaveError,
    enabled = true,
  } = options;

  const [isSaving, setIsSaving] = useState(false);
  const [lastSaved, setLastSaved] = useState<Date | null>(null);
  const [error, setError] = useState<Error | null>(null);

  const saveTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const previousDataRef = useRef<string>('');
  const isMountedRef = useRef(false);

  // Watch all form fields
  const formData = watch();

  // Save function with error handling
  const performSave = useCallback(
    async (data: QuoteFormData) => {
      if (!enabled) return;

      setIsSaving(true);
      setError(null);
      onSaveStart?.();

      try {
        await onSave(data);
        setLastSaved(new Date());
        onSaveSuccess?.(data);
      } catch (err) {
        const error = err instanceof Error ? err : new Error('Save failed');
        setError(error);
        onSaveError?.(error);
      } finally {
        setIsSaving(false);
      }
    },
    [enabled, onSave, onSaveStart, onSaveSuccess, onSaveError]
  );

  // Debounced auto-save effect
  useEffect(() => {
    // Skip on initial mount
    if (!isMountedRef.current) {
      isMountedRef.current = true;
      previousDataRef.current = JSON.stringify(formData);
      return;
    }

    // Skip if auto-save is disabled
    if (!enabled) return;

    // Check if data has actually changed
    const currentData = JSON.stringify(formData);
    if (currentData === previousDataRef.current) {
      return;
    }

    // Clear existing timeout
    if (saveTimeoutRef.current) {
      clearTimeout(saveTimeoutRef.current);
    }

    // Set new timeout for debounced save
    saveTimeoutRef.current = setTimeout(() => {
      previousDataRef.current = currentData;
      performSave(formData);
    }, delay);

    // Cleanup timeout on unmount
    return () => {
      if (saveTimeoutRef.current) {
        clearTimeout(saveTimeoutRef.current);
      }
    };
  }, [formData, delay, enabled, performSave]);

  return {
    isSaving,
    lastSaved,
    error,
  };
};

/**
 * Hook for manual save with loading state
 */
export const useManualSave = (
  onSave: (data: QuoteFormData) => Promise<void>
) => {
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<Error | null>(null);

  const save = useCallback(
    async (data: QuoteFormData) => {
      setIsSaving(true);
      setError(null);

      try {
        await onSave(data);
        return true;
      } catch (err) {
        const error = err instanceof Error ? err : new Error('Save failed');
        setError(error);
        return false;
      } finally {
        setIsSaving(false);
      }
    },
    [onSave]
  );

  return {
    save,
    isSaving,
    error,
    clearError: () => setError(null),
  };
};

/**
 * Format last saved time for display
 */
export const formatLastSaved = (lastSaved: Date | null): string => {
  if (!lastSaved) return 'Never';

  const now = new Date();
  const diffMs = now.getTime() - lastSaved.getTime();
  const diffSecs = Math.floor(diffMs / 1000);
  const diffMins = Math.floor(diffSecs / 60);

  if (diffSecs < 10) return 'Just now';
  if (diffSecs < 60) return `${diffSecs} seconds ago`;
  if (diffMins === 1) return '1 minute ago';
  if (diffMins < 60) return `${diffMins} minutes ago`;

  return lastSaved.toLocaleTimeString('en-US', {
    hour: 'numeric',
    minute: '2-digit',
  });
};

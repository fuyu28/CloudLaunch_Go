/**
 * @fileoverview 汎用Zodバリデーションフック
 *
 * このフックは、任意のZodスキーマを使用したフォームバリデーション機能を提供します。
 */

import { useState, useCallback, useMemo } from "react";
import { ZodError } from "zod";

import type { ZodSchema } from "zod";

export type ValidationState<T> = {
  errors: Record<keyof T, string | undefined>;
  touchedFields: Set<keyof T>;
  isValid: boolean;
};

export type ZodValidationResult<T> = {
  getError: (field: keyof T) => string | undefined;
  hasError: (field: keyof T) => boolean;
  touch: (field: keyof T) => void;
  touchAll: () => void;
  resetTouched: () => void;
  validate: () => { isValid: boolean; errors: Record<keyof T, string> };
  canSubmit: boolean;
  state: ValidationState<T>;
};

export function useZodValidation<T extends Record<string, unknown>>(
  schema: ZodSchema<T>,
  data: T,
  options: {
    realtime?: boolean;
    /** タッチされていないフィールドのエラーも表示するか（デフォルト: false） */
    showUntouchedErrors?: boolean;
  } = {},
): ZodValidationResult<T> {
  const { showUntouchedErrors = false } = options;

  const [touchedFields, setTouchedFields] = useState<Set<keyof T>>(new Set());

  const validationResult = useMemo(() => {
    try {
      schema.parse(data);
      return { isValid: true, errors: {} as Record<keyof T, string> };
    } catch (error) {
      if (error instanceof ZodError) {
        const errors: Record<keyof T, string> = {} as Record<keyof T, string>;
        error.issues.forEach((issue) => {
          const fieldName = issue.path[0] as keyof T;
          if (fieldName) {
            errors[fieldName] = issue.message;
          }
        });
        return { isValid: false, errors };
      }
      return { isValid: false, errors: {} as Record<keyof T, string> };
    }
  }, [schema, data]);

  const state: ValidationState<T> = useMemo(() => {
    const displayErrors: Record<keyof T, string | undefined> = {} as Record<
      keyof T,
      string | undefined
    >;

    Object.keys(validationResult.errors).forEach((field) => {
      const fieldKey = field as keyof T;
      const shouldShow = showUntouchedErrors || touchedFields.has(fieldKey);
      displayErrors[fieldKey] = shouldShow ? validationResult.errors[fieldKey] : undefined;
    });

    return {
      errors: displayErrors,
      touchedFields,
      isValid: validationResult.isValid,
    };
  }, [validationResult, touchedFields, showUntouchedErrors]);

  const getError = useCallback(
    (field: keyof T): string | undefined => {
      return state.errors[field];
    },
    [state.errors],
  );

  const hasError = useCallback(
    (field: keyof T): boolean => {
      return !!state.errors[field];
    },
    [state.errors],
  );

  const touch = useCallback((field: keyof T) => {
    setTouchedFields((prev) => new Set([...prev, field]));
  }, []);

  const touchAll = useCallback(() => {
    const allFields = Object.keys(data) as (keyof T)[];
    setTouchedFields(new Set(allFields));
  }, [data]);

  const resetTouched = useCallback(() => {
    setTouchedFields(new Set());
  }, []);

  const validate = useCallback(() => {
    touchAll();
    return validationResult;
  }, [touchAll, validationResult]);

  const canSubmit = useMemo(() => {
    return validationResult.isValid;
  }, [validationResult.isValid]);

  return {
    getError,
    hasError,
    touch,
    touchAll,
    resetTouched,
    validate,
    canSubmit,
    state,
  };
}

export default useZodValidation;

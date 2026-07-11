/**
 * @fileoverview Zodベースの設定フォーム管理フック
 *
 * このフックは、設定ページのフォーム状態管理と操作を提供します。
 * Zodスキーマを使用して型安全なバリデーションを実現します。
 */

import { useState, useEffect, useMemo, useCallback } from "react";
import toast from "react-hot-toast";
import { ZodError } from "zod";

import { logger } from "@renderer/utils/logger";

import { credsSchema } from "@renderer/schemas/credentials";
import type { Creds } from "src/types/creds";
import type { ApiResult } from "src/types/result";

export type SettingsFormData = {
  bucketName: string;
  endpoint: string;
  region: string;
  accessKeyId: string;
  secretAccessKey: string;
};

export type SettingsValidationErrors = {
  bucketName?: string;
  endpoint?: string;
  region?: string;
  accessKeyId?: string;
  secretAccessKey?: string;
};

export type SettingsFormResult = {
  formData: SettingsFormData;
  updateField: (field: keyof SettingsFormData, value: string) => void;
  updateFormData: (data: Partial<SettingsFormData>) => void;
  canSubmit: boolean;
  isSaving: boolean;
  isLoading: boolean;
  isTesting: boolean;
  errors: SettingsValidationErrors;
  /** フィールドエラー（互換性のため） */
  fieldErrors: SettingsValidationErrors;
  fieldValidation: Record<keyof SettingsFormData, { isValid: boolean; message?: string }>;
  handleSave: () => Promise<void>;
  testConnection: () => Promise<void>;
  resetForm: () => void;
  validateField: (fieldName: keyof SettingsFormData) => string | undefined;
  isConnectionSuccessful: boolean | null;
};

export function useSettingsFormZod(): SettingsFormResult {
  const [formData, setFormData] = useState<SettingsFormData>({
    bucketName: "",
    endpoint: "",
    region: "",
    accessKeyId: "",
    secretAccessKey: "",
  });

  const [isSaving, setIsSaving] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [isTesting, setIsTesting] = useState(false);

  const [isConnectionSuccessful, setIsConnectionSuccessful] = useState<boolean | null>(null);
  const [lastTestedData, setLastTestedData] = useState<SettingsFormData | null>(null);

  const isDataChanged = useCallback(
    (data1: SettingsFormData, data2: SettingsFormData | null): boolean => {
      if (!data2) return true;
      return (
        data1.bucketName !== data2.bucketName ||
        data1.endpoint !== data2.endpoint ||
        data1.region !== data2.region ||
        data1.accessKeyId !== data2.accessKeyId ||
        data1.secretAccessKey !== data2.secretAccessKey
      );
    },
    [],
  );

  /**
   * 初期データの読み込み
   *
   * StrictMode の二重発火や、素早い unmount → remount のときに、
   * 遅い resolve が新しい入力を上書きしないよう cancelled フラグでガードする。
   */
  useEffect(() => {
    let cancelled = false;
    const loadInitialData = async (): Promise<void> => {
      try {
        setIsLoading(true);
        const result: ApiResult<Creds> = await window.api.credential.getCredential();

        if (cancelled) return;
        if (result.success && result.data) {
          setFormData({
            bucketName: result.data.bucketName || "",
            endpoint: result.data.endpoint || "",
            region: result.data.region || "",
            accessKeyId: result.data.accessKeyId || "",
            secretAccessKey: result.data.secretAccessKey || "",
          });
        }
      } catch (error) {
        if (cancelled) return;
        logger.error("初期データの読み込みに失敗しました:", {
          component: "useSettingsFormZod",
          function: "unknown",
          data: error,
        });
        toast.error("設定の読み込みに失敗しました");
      } finally {
        if (!cancelled) {
          setIsLoading(false);
        }
      }
    };

    loadInitialData();
    return () => {
      cancelled = true;
    };
  }, []);

  const updateField = useCallback(
    (field: keyof SettingsFormData, value: string) => {
      setFormData((prev) => {
        const newData = {
          ...prev,
          [field]: value,
        };

        // 入力が変わったあとの古い「接続成功」を無効化する。
        if (lastTestedData && isDataChanged(newData, lastTestedData)) {
          setIsConnectionSuccessful(null);
        }

        return newData;
      });
    },
    [lastTestedData, isDataChanged],
  );

  const updateFormData = useCallback(
    (data: Partial<SettingsFormData>) => {
      setFormData((prev) => {
        const newData = {
          ...prev,
          ...data,
        };

        // 入力が変わったあとの古い「接続成功」を無効化する。
        if (lastTestedData && isDataChanged(newData, lastTestedData)) {
          setIsConnectionSuccessful(null);
        }

        return newData;
      });
    },
    [lastTestedData, isDataChanged],
  );

  const validateField = useCallback(
    (fieldName: keyof SettingsFormData): string | undefined => {
      try {
        const fieldSchema = credsSchema.shape[fieldName];
        if (fieldSchema) {
          fieldSchema.parse(formData[fieldName]);
        }
        return undefined;
      } catch (error) {
        if (error instanceof ZodError) {
          return error.issues[0]?.message;
        }
        return "入力値が無効です";
      }
    },
    [formData],
  );

  const validateAllFields = useCallback(() => {
    try {
      credsSchema.parse(formData);
      return { isValid: true, errors: {} };
    } catch (error) {
      if (error instanceof ZodError) {
        const errorMap: Record<string, string> = {};
        error.issues.forEach((issue) => {
          const fieldName = issue.path[0];
          if (fieldName && typeof fieldName === "string") {
            errorMap[fieldName] = issue.message;
          }
        });
        return { isValid: false, errors: errorMap };
      }
      return { isValid: false, errors: { general: "バリデーションエラーが発生しました" } };
    }
  }, [formData]);

  const fieldValidation = useMemo(() => {
    const fieldNames: (keyof SettingsFormData)[] = [
      "bucketName",
      "endpoint",
      "region",
      "accessKeyId",
      "secretAccessKey",
    ];

    return fieldNames.reduce(
      (acc, fieldName) => {
        const errorMessage = validateField(fieldName);
        const isValid = !errorMessage;

        acc[fieldName] = {
          isValid,
          message: errorMessage,
        };
        return acc;
      },
      {} as Record<keyof SettingsFormData, { isValid: boolean; message?: string }>,
    );
  }, [validateField]);

  const errors = useMemo((): SettingsValidationErrors => {
    return {
      bucketName: fieldValidation.bucketName?.message,
      endpoint: fieldValidation.endpoint?.message,
      region: fieldValidation.region?.message,
      accessKeyId: fieldValidation.accessKeyId?.message,
      secretAccessKey: fieldValidation.secretAccessKey?.message,
    };
  }, [fieldValidation]);

  const canSubmit = useMemo(() => {
    const validationResult = validateAllFields();
    return validationResult.isValid && !isSaving && !isTesting;
  }, [validateAllFields, isSaving, isTesting]);

  const handleSave = useCallback(async () => {
    if (!canSubmit) {
      toast.error("入力内容に問題があります");
      return;
    }

    setIsSaving(true);
    try {
      // 未テストや入力変更後は保存前に接続テストを必須にする。
      const needsConnectionTest =
        isDataChanged(formData, lastTestedData) || isConnectionSuccessful !== true;

      if (needsConnectionTest) {
        const testResult: ApiResult = await window.api.credential.validateCredential(formData);

        if (!testResult.success) {
          toast.error(testResult.message || "接続テストに失敗しました。設定を確認してください。");
          setIsConnectionSuccessful(false);
          return;
        }

        setIsConnectionSuccessful(true);
        setLastTestedData({ ...formData });
      }

      const result: ApiResult = await window.api.credential.upsertCredential(formData);

      if (result.success) {
        toast.success("設定を保存しました");
      } else {
        toast.error(result.message || "設定の保存に失敗しました");
      }
    } catch (error) {
      logger.error("設定保存エラー:", {
        component: "useSettingsFormZod",
        function: "handleSave",
        data: error,
      });
      toast.error("設定の保存中にエラーが発生しました");
    } finally {
      setIsSaving(false);
    }
  }, [formData, canSubmit, isDataChanged, lastTestedData, isConnectionSuccessful]);

  /**
   * 手動接続テスト処理（明示的に実行する場合）
   */
  const testConnection = useCallback(async () => {
    if (!canSubmit) {
      toast.error("すべての項目を正しく入力してください");
      return;
    }

    setIsTesting(true);
    try {
      const result: ApiResult = await window.api.credential.validateCredential(formData);

      if (result.success) {
        toast.success("接続テストに成功しました");
        setIsConnectionSuccessful(true);
        setLastTestedData({ ...formData });
      } else {
        toast.error(result.message || "接続テストに失敗しました");
        setIsConnectionSuccessful(false);
      }
    } catch (error) {
      logger.error("接続テストエラー:", {
        component: "useSettingsFormZod",
        function: "testConnection",
        data: error,
      });
      toast.error("接続テスト中にエラーが発生しました");
      setIsConnectionSuccessful(false);
    } finally {
      setIsTesting(false);
    }
  }, [formData, canSubmit]);

  const resetForm = useCallback(() => {
    setFormData({
      bucketName: "",
      endpoint: "",
      region: "",
      accessKeyId: "",
      secretAccessKey: "",
    });
  }, []);

  return {
    formData,
    updateField,
    updateFormData,
    canSubmit,
    isSaving,
    isLoading,
    isTesting,
    errors,
    fieldErrors: errors, // 互換性のため
    fieldValidation,
    handleSave,
    testConnection,
    resetForm,
    validateField,
    isConnectionSuccessful,
  };
}

export default useSettingsFormZod;

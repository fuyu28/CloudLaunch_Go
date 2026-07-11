/**
 * @fileoverview Zodベースのゲームフォームバリデーションフック
 *
 * このフックは、ゲーム登録・編集フォームのバリデーション機能を提供します。
 * Zodスキーマを使用して型安全かつ保守可能なバリデーションを実現します。
 *
 * 主な機能：
 * - Zodスキーマベースの検証
 * - リアルタイムバリデーション
 * - エラーメッセージの自動生成
 * - 送信可能状態の判定
 *
 * 使用例：
 * ```tsx
 * const { canSubmit, errors, validateField } = useGameFormValidationZod(gameData)
 * ```
 */

import { useMemo, useState, useCallback, useEffect } from "react";
import { ZodError } from "zod";

import { gameFormSchema } from "@renderer/schemas/game";
import type { InputGameData } from "src/types/game";
import {
  validateExecutablePath,
  validateImagePath,
  validateSaveFolderPath,
} from "@renderer/utils/fileValidation";
import { logger } from "@renderer/utils/logger";

/**
 * バリデーションエラーの型定義
 */
export type ValidationErrors = {
  title?: string;
  publisher?: string;
  exePath?: string;
  imagePath?: string;
  saveFolderPath?: string;
};

/**
 * ゲームフォームバリデーションフックの戻り値
 */
export type GameFormValidationResult = {
  /** 送信可能かどうか */
  canSubmit: boolean;
  /** バリデーションエラー（タッチされたフィールドのみ表示） */
  errors: ValidationErrors;
  /** 特定フィールドのバリデーション実行 */
  validateField: (fieldName: keyof InputGameData) => string | undefined;
  /** 全フィールドのバリデーション実行 */
  validateAllFields: () => { isValid: boolean; errors: Record<string, string> };
  /** ファイル存在チェックを含む非同期バリデーション */
  validateAllFieldsWithFileCheck: () => Promise<{
    isValid: boolean;
    errors: Record<string, string>;
  }>;
  /** 必須フィールドがすべて入力されているかチェック */
  hasRequiredFields: boolean;
  /** 各フィールドの検証状態 */
  fieldValidation: Record<
    keyof InputGameData,
    { isValid: boolean; message?: string; shouldShowError: boolean }
  >;
  /** フィールドがタッチされたことを記録 */
  markFieldAsTouched: (fieldName: keyof InputGameData) => void;
  /** すべてのフィールドをタッチ済みとして設定（送信時に使用） */
  markAllFieldsAsTouched: () => void;
  /** タッチされたフィールドをリセット（モーダル開閉時に使用） */
  resetTouchedFields: () => void;
  /** 特定フィールドのファイル存在チェックを実行 */
  validateFileField: (fieldName: keyof InputGameData) => Promise<void>;
};

/**
 * Zodベースのゲームフォームバリデーションフック
 *
 * ゲーム登録・編集フォームのバリデーション機能を提供します。
 * Zodスキーマを使用して型安全なバリデーションを実現します。
 *
 * @param gameData 検証対象のゲームデータ
 * @returns バリデーション結果とヘルパー関数
 */
export function useGameFormValidationZod(gameData: InputGameData): GameFormValidationResult {
  const [touchedFields, setTouchedFields] = useState<Set<keyof InputGameData>>(new Set());

  const [fileCheckErrors, setFileCheckErrors] = useState<Record<string, string>>({});

  // ファイル存在チェックが debounce 待機中のフィールド集合。
  // debounce 期間中（500ms）に canSubmit が true になり submit が通ってしまう問題を避けるため、
  // pending が空になるまで送信を無効化する。
  const [pendingFileCheckCount, setPendingFileCheckCount] = useState<number>(0);

  /**
   * 特定フィールドのファイル存在チェックを実行
   * フィールドにアクセスがあったときにリアルタイムで実行
   */
  const validateFileField = useCallback(
    async (fieldName: keyof InputGameData) => {
      const fieldValue = gameData[fieldName] as string;

      if (!fieldValue || fieldValue.trim() === "") {
        setFileCheckErrors((prev) => {
          const newErrors = { ...prev };
          delete newErrors[fieldName];
          return newErrors;
        });
        return;
      }

      let hasError = false;
      let errorMessage = "";

      try {
        switch (fieldName) {
          case "exePath": {
            const isValidExe = await validateExecutablePath(fieldValue);
            if (!isValidExe) {
              hasError = true;
              errorMessage = "実行ファイルが存在しないか、無効なファイルです";
            }
            break;
          }
          case "imagePath": {
            const isValidImage = await validateImagePath(fieldValue);
            if (!isValidImage) {
              try {
                new URL(fieldValue);
              } catch {
                hasError = true;
                errorMessage = "画像ファイルが存在しないか、無効なファイルです";
              }
            }
            break;
          }
          case "saveFolderPath": {
            const isValidFolder = await validateSaveFolderPath(fieldValue);
            if (!isValidFolder) {
              hasError = true;
              errorMessage = "セーブフォルダが存在しないか、無効なフォルダです";
            }
            break;
          }
        }

        setFileCheckErrors((prev) => {
          const newErrors = { ...prev };
          if (hasError) {
            newErrors[fieldName] = errorMessage;
          } else {
            delete newErrors[fieldName];
          }
          return newErrors;
        });
      } catch (error) {
        logger.warn(`ファイル存在チェックエラー (${fieldName}):`, {
          component: "useGameFormValidationZod",
          function: "checkFileExists",
          error: error instanceof Error ? error : new Error(String(error)),
          data: { fieldName },
        });
      }
    },
    [gameData],
  );

  // gameDataの変更を監視してファイル存在チェックを自動実行
  useEffect(() => {
    const fileFields = ["exePath", "imagePath", "saveFolderPath"] as const;
    const timeoutIds: NodeJS.Timeout[] = [];
    let scheduledCount = 0;

    fileFields.forEach((fieldName) => {
      const fieldValue = gameData[fieldName] as string;
      if (fieldValue && fieldValue.trim() !== "") {
        // デバウンスされたファイル存在チェック（500ms後に実行）
        scheduledCount += 1;
        const timeoutId = setTimeout(async () => {
          try {
            await validateFileField(fieldName);
          } finally {
            // debounce 完了で pending カウントを1つ減らす。
            setPendingFileCheckCount((prev) => Math.max(0, prev - 1));
          }
        }, 500);
        timeoutIds.push(timeoutId);
      } else {
        setFileCheckErrors((prev) => {
          const newErrors = { ...prev };
          delete newErrors[fieldName];
          return newErrors;
        });
      }
    });

    // 予約したデバウンス数だけ pending を増やす。
    if (scheduledCount > 0) {
      setPendingFileCheckCount((prev) => prev + scheduledCount);
    }

    return () => {
      timeoutIds.forEach((timeoutId) => clearTimeout(timeoutId));
      // 未実行分の pending は取り消し扱いにする（新しい effect が改めて予約する）。
      if (scheduledCount > 0) {
        setPendingFileCheckCount((prev) => Math.max(0, prev - scheduledCount));
      }
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [gameData.exePath, gameData.imagePath, gameData.saveFolderPath, validateFileField]);

  const markFieldAsTouched = useCallback((fieldName: keyof InputGameData) => {
    setTouchedFields((prev) => new Set([...prev, fieldName]));
  }, []);

  const markAllFieldsAsTouched = useCallback(() => {
    setTouchedFields(new Set(["title", "publisher", "exePath", "imagePath", "saveFolderPath"]));
  }, []);

  const resetTouchedFields = useCallback(() => {
    setTouchedFields(new Set());
    setFileCheckErrors({});
  }, []);

  /**
   * Zodスキーマを使用したフィールドバリデーション
   * 個別フィールドの検証を実行し、エラーメッセージを返す
   * 全体スキーマを使用してrefineバリデーションも適用
   */
  const validateField = useCallback(
    (fieldName: keyof InputGameData): string | undefined => {
      try {
        gameFormSchema.parse(gameData);
        return undefined;
      } catch (error) {
        if (error instanceof ZodError) {
          const fieldError = error.issues.find((issue) => issue.path.includes(fieldName));
          return fieldError?.message;
        }
        return "入力値が無効です";
      }
    },
    [gameData],
  );

  /**
   * 全フィールドのバリデーション結果を取得
   * Zodスキーマの全体検証を実行
   */
  const validateAllFields = useCallback(() => {
    try {
      gameFormSchema.parse(gameData);
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
  }, [gameData]);

  /**
   * ファイル存在チェックを含む非同期バリデーション
   * フォーム送信時の最終バリデーションで使用
   */
  const validateAllFieldsWithFileCheck = useCallback(async () => {
    try {
      gameFormSchema.parse(gameData);

      const errors: Record<string, string> = {};

      if (gameData.exePath) {
        const isValidExe = await validateExecutablePath(gameData.exePath);
        if (!isValidExe) {
          errors.exePath = "実行ファイルが存在しないか、無効なファイルです";
        }
      }

      if (gameData.imagePath && gameData.imagePath.trim()) {
        const isValidImage = await validateImagePath(gameData.imagePath);
        if (!isValidImage) {
          try {
            new URL(gameData.imagePath);
          } catch {
            errors.imagePath = "画像ファイルが存在しないか、無効なファイルです";
          }
        }
      }

      if (gameData.saveFolderPath && gameData.saveFolderPath.trim()) {
        const isValidFolder = await validateSaveFolderPath(gameData.saveFolderPath);
        if (!isValidFolder) {
          errors.saveFolderPath = "セーブフォルダが存在しないか、無効なフォルダです";
        }
      }

      const hasErrors = Object.keys(errors).length > 0;
      setFileCheckErrors(errors);
      return { isValid: !hasErrors, errors };
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
  }, [gameData]);

  const fieldValidation = useMemo(() => {
    const fieldNames: (keyof InputGameData)[] = [
      "title",
      "publisher",
      "exePath",
      "imagePath",
      "saveFolderPath",
    ];

    return fieldNames.reduce(
      (acc, fieldName) => {
        const zodErrorMessage = validateField(fieldName);
        const fileCheckError = fileCheckErrors[fieldName];

        // Zodエラーまたはファイル存在チェックエラーのいずれかを使用
        const errorMessage = zodErrorMessage || fileCheckError;
        const isValid = !errorMessage;
        const shouldShowError = touchedFields.has(fieldName) && !!errorMessage;

        acc[fieldName] = {
          isValid,
          message: errorMessage,
          shouldShowError,
        };
        return acc;
      },
      {} as Record<
        keyof InputGameData,
        { isValid: boolean; message?: string; shouldShowError: boolean }
      >,
    );
  }, [touchedFields, validateField, fileCheckErrors]);

  const errors = useMemo((): ValidationErrors => {
    return {
      title: fieldValidation.title?.shouldShowError ? fieldValidation.title.message : undefined,
      publisher: fieldValidation.publisher?.shouldShowError
        ? fieldValidation.publisher.message
        : undefined,
      exePath: fieldValidation.exePath?.shouldShowError
        ? fieldValidation.exePath.message
        : undefined,
      imagePath: fieldValidation.imagePath?.shouldShowError
        ? fieldValidation.imagePath.message
        : undefined,
      saveFolderPath: fieldValidation.saveFolderPath?.shouldShowError
        ? fieldValidation.saveFolderPath.message
        : undefined,
    };
  }, [fieldValidation]);

  const hasRequiredFields = useMemo(() => {
    return (
      fieldValidation.title?.isValid &&
      fieldValidation.publisher?.isValid &&
      fieldValidation.exePath?.isValid
    );
  }, [fieldValidation]);

  // 送信可能状態の判定（Zodスキーマでの全体検証＋ファイル存在チェック）
  // debounce 待機中のファイルチェックが残っている間は false にしておき、
  // 「存在しないパス」を入力した直後に一瞬 canSubmit=true になり submit が通ってしまうのを防ぐ。
  const canSubmit = useMemo(() => {
    const validationResult = validateAllFields();
    const hasFileCheckErrors = Object.keys(fileCheckErrors).length > 0;
    const hasPendingFileChecks = pendingFileCheckCount > 0;
    return validationResult.isValid && !hasFileCheckErrors && !hasPendingFileChecks;
  }, [validateAllFields, fileCheckErrors, pendingFileCheckCount]);

  return {
    canSubmit,
    errors,
    validateField,
    validateAllFields,
    validateAllFieldsWithFileCheck,
    hasRequiredFields,
    fieldValidation,
    markFieldAsTouched,
    markAllFieldsAsTouched,
    resetTouchedFields,
    validateFileField,
  };
}

export default useGameFormValidationZod;

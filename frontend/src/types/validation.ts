/**
 * @fileoverview バリデーション関連型定義
 *
 * このファイルは、アプリケーション全体で使用されるバリデーション関連の型を定義します。
 */

export type ValidationResult = {
  isValid: boolean;
  message?: string;
};

export type ValidationResultMultiple = {
  isValid: boolean;
  errors: string[];
};

export type ValidationErrors<T extends string = string> = {
  [K in T]?: string;
};

export type GameFormValidationErrors = {
  title?: string;
  publisher?: string;
  exePath?: string;
  saveFolderPath?: string;
  imagePath?: string;
};

export type SettingsFormValidationErrors = {
  bucketName?: string;
  region?: string;
  endpoint?: string;
  accessKeyId?: string;
  secretAccessKey?: string;
};

export type ValidationFunction<T = unknown> = (value: T) => ValidationResult;

export type AsyncValidationFunction<T = unknown> = (value: T) => Promise<ValidationResult>;

export type FieldValidator<T = string> = {
  field: string;
  validator: ValidationFunction<T>;
  required?: boolean;
};

export type FormValidator<T extends Record<string, unknown>> = {
  data: T;
  validators: FieldValidator[];
};

export type ValidationConfig = {
  requiredMessage?: string;
  minLengthMessage?: string;
  maxLengthMessage?: string;
  urlMessage?: string;
};

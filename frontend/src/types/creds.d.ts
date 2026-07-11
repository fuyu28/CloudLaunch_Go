/**
 * @fileoverview 認証情報型定義
 *
 * S3/R2 クレデンシャルの TypeScript 型。
 */

export type Schema = {
  bucketName: string;
  region: string;
  endpoint: string;
  accessKeyId: string;
};

export type Creds = Schema & {
  secretAccessKey: string;
};

export type CredsContextType = {
  isValidCreds: boolean;
  creds: Creds | undefined;
  setIsValidCreds: (v: boolean) => void;
  reloadCreds: () => Promise<boolean>;
};

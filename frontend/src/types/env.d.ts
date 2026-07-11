/**
 * @fileoverview 環境変数型定義
 *
 * ProcessEnv にバケット／認証キー等の読み取り専用フィールドを宣言する。
 */

declare namespace NodeJS {
  interface ProcessEnv {
    readonly BUCKET_NAME: string;
    readonly REGION: string;
    readonly ENDPOINT: string;
    readonly ACCESS_KEY_ID: string;
    readonly SECRET_ACCESS_KEY: string;
  }
}

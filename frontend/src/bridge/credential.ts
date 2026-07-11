/**
 * @fileoverview S3 認証情報ブリッジ。
 *
 * OS 資格情報ストア経由の保存・検証。読み出し時に Secret は返さない。
 */

import {
  SaveCredential,
  LoadCredential,
  ValidateCredential,
  ValidateSavedCredential,
} from "../../wailsjs/go/app/App";
import { toApiResultVoid } from "./helpers";
import type { WindowApi } from "./types";

export function createCredentialBridge(): WindowApi["credential"] {
  return {
    upsertCredential: async (creds) => {
      const result = await SaveCredential("default", {
        BucketName: creds.bucketName,
        Region: creds.region,
        Endpoint: creds.endpoint,
        AccessKeyID: creds.accessKeyId,
        SecretAccessKey: creds.secretAccessKey,
      });
      return toApiResultVoid(result);
    },
    getCredential: async () => {
      const result = await LoadCredential("default");
      if (!result.success || !result.data) {
        return { success: false, message: result.error?.message ?? "認証情報がありません" };
      }
      return {
        success: true,
        data: {
          accessKeyId: result.data.AccessKeyID,
          // Secret は再表示しない。空のまま返し、未変更保存時はバックエンドが既存値を維持する。
          secretAccessKey: "",
          bucketName: result.data.BucketName ?? "",
          region: result.data.Region ?? "",
          endpoint: result.data.Endpoint ?? "",
        },
      };
    },
    validateCredential: async (creds) => {
      const result = await ValidateCredential({
        bucketName: creds.bucketName,
        region: creds.region,
        endpoint: creds.endpoint,
        accessKeyId: creds.accessKeyId,
        secretAccessKey: creds.secretAccessKey,
      });
      return toApiResultVoid(result);
    },
    validateSavedCredential: async () => toApiResultVoid(await ValidateSavedCredential("default")),
  };
}

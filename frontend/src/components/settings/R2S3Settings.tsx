/**
 * @fileoverview R2/S3設定コンポーネント
 *
 * クラウドストレージ（R2/S3）の設定を管理するコンポーネントです。
 */

import { useAtom } from "jotai";
import { FaCheck, FaSyncAlt, FaTimes } from "react-icons/fa";
import toast from "react-hot-toast";

import SettingsFormField from "./SettingsFormField";
import { SettingsToggle } from "./SettingsToggle";
import { useConnectionStatus } from "../../hooks/useConnectionStatus";
import { useOfflineMode } from "../../hooks/useOfflineMode";
import { useSettingsFormZod } from "../../hooks/useSettingsFormZod";
import { s3ForcePathStyleAtom, s3UseTLSAtom } from "../../state/settings";
import { getOfflineDisabledClasses } from "../../utils/offlineUtils";
import { logger } from "../../utils/logger";

export default function R2S3Settings(): React.JSX.Element {
  const {
    formData,
    updateField,
    canSubmit,
    isSaving,
    handleSave,
    fieldErrors,
    testConnection,
    isTesting,
    isConnectionSuccessful,
  } = useSettingsFormZod();
  const {
    status: connectionStatus,
    message: connectionMessage,
    check: checkConnection,
  } = useConnectionStatus();
  const { isOfflineMode, checkNetworkFeature } = useOfflineMode();
  const [s3ForcePathStyle, setS3ForcePathStyle] = useAtom(s3ForcePathStyleAtom);
  const [s3UseTLS, setS3UseTLS] = useAtom(s3UseTLSAtom);

  const handleForcePathStyleChange = async (enabled: boolean): Promise<void> => {
    const result = await window.api.settings.updateS3ForcePathStyle(enabled);
    if (!result.success) {
      logger.error("ForcePathStyle 更新エラー:", {
        component: "R2S3Settings",
        function: "handleForcePathStyleChange",
        data: result.message,
      });
      toast.error("path-style 設定の更新に失敗しました");
      return;
    }
    setS3ForcePathStyle(enabled);
  };

  const handleUseTLSChange = async (enabled: boolean): Promise<void> => {
    const result = await window.api.settings.updateS3UseTLS(enabled);
    if (!result.success) {
      logger.error("UseTLS 更新エラー:", {
        component: "R2S3Settings",
        function: "handleUseTLSChange",
        data: result.message,
      });
      toast.error("TLS 設定の更新に失敗しました");
      return;
    }
    setS3UseTLS(enabled);
  };

  const handleConnectionTest = (): void => {
    if (!checkNetworkFeature("接続テスト")) {
      return;
    }
    testConnection();
  };

  const handleSaveSettings = (): void => {
    if (!checkNetworkFeature("設定保存")) {
      return;
    }
    handleSave();
  };

  const handleStatusCheck = (): void => {
    if (!checkNetworkFeature("接続状態の確認")) {
      return;
    }
    checkConnection();
  };

  const disabledClasses = getOfflineDisabledClasses(isOfflineMode);
  const statusText = isOfflineMode
    ? "オフライン中"
    : connectionStatus === "loading"
      ? "確認中..."
      : connectionStatus === "success"
        ? "接続中"
        : "未接続";
  const statusColor = isOfflineMode
    ? "text-warning"
    : connectionStatus === "success"
      ? "text-success"
      : connectionStatus === "error"
        ? "text-error"
        : "text-base-content/70";

  return (
    <div className={disabledClasses}>
      <h2 className="text-xl font-semibold mb-2 flex items-center justify-between">
        クラウド（R2/S3）
        <div className="text-sm flex items-center space-x-1">
          {isOfflineMode ? (
            <span className="text-warning">オフラインモード</span>
          ) : (
            <>
              {(isTesting || isSaving) && <FaSyncAlt className="animate-spin text-base-content" />}
              {!isTesting && !isSaving && isConnectionSuccessful === true && (
                <FaCheck className="text-success" />
              )}
              {!isTesting && !isSaving && isConnectionSuccessful === false && (
                <FaTimes className="text-error" />
              )}
              <span className="text-base-content/80">
                {isTesting
                  ? "接続確認中..."
                  : isSaving
                    ? "保存中..."
                    : isConnectionSuccessful === true
                      ? "接続テスト成功"
                      : isConnectionSuccessful === false
                        ? "接続テスト失敗"
                        : ""}
              </span>
            </>
          )}
        </div>
      </h2>

      <div className="flex items-center justify-between rounded-lg border border-base-300 bg-base-200/40 px-4 py-3">
        <div className="text-sm">
          <span className="text-base-content/70">現在の接続状態:</span>
          <span className={`ml-2 font-medium ${statusColor}`}>{statusText}</span>
          {!isOfflineMode && connectionStatus === "error" && connectionMessage && (
            <span className="ml-2 text-xs text-base-content/60">{connectionMessage}</span>
          )}
        </div>
        {!isOfflineMode && (
          <button
            className="btn btn-xs btn-outline"
            onClick={handleStatusCheck}
            disabled={connectionStatus === "loading" || isTesting || isSaving}
          >
            再確認
          </button>
        )}
      </div>

      <div className="flex flex-col space-y-4 mt-4">
        <SettingsFormField
          label="Bucket Name"
          value={formData.bucketName}
          onChange={(value) => updateField("bucketName", value)}
          placeholder="バケット名を入力"
          required
          error={fieldErrors.bucketName}
          helpText="S3互換ストレージのバケット名"
        />

        <SettingsFormField
          label="Endpoint"
          value={formData.endpoint}
          onChange={(value) => updateField("endpoint", value)}
          placeholder="https://<アカウント>.r2.cloudflarestorage.com"
          required
          error={fieldErrors.endpoint}
          helpText="R2またはS3互換ストレージのエンドポイントURL"
        />

        <SettingsFormField
          label="Region"
          value={formData.region}
          onChange={(value) => updateField("region", value)}
          placeholder="auto"
          helpText="ストレージのリージョン（通常は auto で問題ありません）"
        />

        <SettingsFormField
          label="Access Key ID"
          value={formData.accessKeyId}
          onChange={(value) => updateField("accessKeyId", value)}
          placeholder="アクセスキーを入力"
          required
          error={fieldErrors.accessKeyId}
          helpText="ストレージアクセス用のアクセスキーID"
        />

        <SettingsFormField
          label="Secret Access Key"
          value={formData.secretAccessKey}
          onChange={(value) => updateField("secretAccessKey", value)}
          placeholder="シークレットアクセスキーを入力"
          type="password"
          required
          error={fieldErrors.secretAccessKey}
          helpText="ストレージアクセス用のシークレットキー"
        />
      </div>

      <div className="bg-base-200 p-4 rounded-lg space-y-4 mt-6">
        <h4 className="font-medium">接続詳細</h4>
        <SettingsToggle
          label="Force path-style"
          description="MinIO など path-style が必要なエンドポイント向け（仮想ホスト形式を使わない）"
          checked={s3ForcePathStyle}
          onChange={(value) => void handleForcePathStyleChange(value)}
          disabled={isOfflineMode}
        />
        <SettingsToggle
          label="TLS を使用"
          description="オフにすると HTTP で接続します（ローカル MinIO など）"
          checked={s3UseTLS}
          onChange={(value) => void handleUseTLSChange(value)}
          disabled={isOfflineMode}
        />
      </div>

      <div className="form-control mt-6 flex justify-end space-x-2">
        {!isOfflineMode && (
          <button
            className="btn btn-outline"
            onClick={handleConnectionTest}
            disabled={isTesting || isSaving || !canSubmit}
          >
            {isTesting ? "テスト中..." : "接続テスト"}
          </button>
        )}
        <button
          className="btn btn-primary"
          onClick={handleSaveSettings}
          disabled={!canSubmit || isSaving || isOfflineMode}
          title="保存時に自動で接続テストを実行します"
        >
          {isSaving ? "保存中..." : "保存"}
        </button>
      </div>

      {isOfflineMode && (
        <div className="mt-4 p-4 bg-warning/10 border border-warning/20 rounded-lg">
          <p className="text-sm text-warning">
            オフラインモードでは R2/S3 設定の変更や接続テストはできません。
            <br />
            設定を変更するには、「動作」タブからオフラインモードを無効にしてください。
          </p>
        </div>
      )}
    </div>
  );
}

import { useEffect, useState } from "react";

import { useScreenshotSettings } from "@renderer/hooks/useScreenshotSettings";

import { SettingsToggle } from "./SettingsToggle";
import { TabSectionHeader } from "./TabSectionHeader";

export default function ScreenshotSettingsTab(): React.JSX.Element {
  const {
    screenshotSyncEnabled,
    screenshotUploadJpeg,
    screenshotJpegQuality,
    screenshotClientOnly,
    screenshotLocalJpeg,
    screenshotHotkey,
    screenshotHotkeyNotify,
    isCapturingHotkey,
    setIsCapturingHotkey,
    handleScreenshotSyncEnabledChange,
    handleScreenshotUploadJpegChange,
    handleScreenshotJpegQualityChange,
    handleScreenshotClientOnlyChange,
    handleScreenshotLocalJpegChange,
    applyScreenshotHotkey,
    handleScreenshotHotkeyChange,
    handleScreenshotHotkeyNotifyChange,
  } = useScreenshotSettings();

  // ホットキー入力は uncontrolled 化する。atomWithStorage は入力途中の1文字ごとに
  // localStorage へ書き込まれ、失敗時にロールバックできないため、
  // 「編集は draftHotkey に保持 → onBlur で apply 成功時のみ atom を更新」の順序に統一する。
  const [draftHotkey, setDraftHotkey] = useState<string>(screenshotHotkey);

  // atom が他所（起動同期・キャプチャモードでの反映など）で書き換わったら、
  // 未フォーカスの draft も追従させる。
  useEffect(() => {
    setDraftHotkey(screenshotHotkey);
  }, [screenshotHotkey]);

  const jpegQualityEnabled = screenshotLocalJpeg || (screenshotSyncEnabled && screenshotUploadJpeg);

  return (
    <div className="space-y-6">
      <TabSectionHeader title="スクリーンショット" description="撮影・保存・クラウド同期" />

      <div className="bg-base-200 p-4 rounded-lg space-y-4">
        <h4 className="font-medium">撮影</h4>
        <SettingsToggle
          label="タイトルバーを除外"
          description="オンでクライアント領域のみを撮影します"
          checked={screenshotClientOnly}
          onChange={(value) => void handleScreenshotClientOnlyChange(value)}
        />

        <div>
          <div className="mb-2">
            <h4 className="font-medium">ホットキー</h4>
            <p className="text-sm text-base-content/70">例: Ctrl+Alt+S</p>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="text"
              className="input input-bordered input-sm flex-1"
              value={draftHotkey}
              onChange={(event) => setDraftHotkey(event.target.value)}
              onBlur={async (event) => {
                const nextValue = event.target.value;
                if (nextValue === screenshotHotkey) return;
                const ok = await applyScreenshotHotkey(nextValue, true);
                if (!ok) {
                  setDraftHotkey(screenshotHotkey);
                }
              }}
              readOnly={isCapturingHotkey}
            />
            <button className="btn btn-primary btn-sm" onClick={() => setIsCapturingHotkey(true)}>
              入力開始
            </button>
            <button
              className="btn btn-outline btn-sm"
              onClick={async () => {
                const ok = await handleScreenshotHotkeyChange(draftHotkey);
                if (!ok) {
                  setDraftHotkey(screenshotHotkey);
                }
              }}
              disabled={isCapturingHotkey}
            >
              適用
            </button>
          </div>
          {isCapturingHotkey && (
            <p className="text-xs text-base-content/60 mt-2">
              ホットキーを押してください（Escでキャンセル）
            </p>
          )}
        </div>

        <SettingsToggle
          label="ホットキー通知"
          description="押下時にWindows通知を表示します"
          checked={screenshotHotkeyNotify}
          onChange={(value) => void handleScreenshotHotkeyNotifyChange(value)}
        />
      </div>

      <div className="bg-base-200 p-4 rounded-lg space-y-4">
        <h4 className="font-medium">ローカル保存</h4>
        <SettingsToggle
          label="ローカル保存をJPEGにする"
          description="オンでPNGより容量を抑えて保存します"
          checked={screenshotLocalJpeg}
          onChange={(value) => void handleScreenshotLocalJpegChange(value)}
        />
      </div>

      <div className="bg-base-200 p-4 rounded-lg space-y-4">
        <h4 className="font-medium">クラウド</h4>
        <SettingsToggle
          label="クラウド同期"
          description="スクリーンショットをクラウドにアップロードします"
          checked={screenshotSyncEnabled}
          onChange={(value) => void handleScreenshotSyncEnabledChange(value)}
        />

        <SettingsToggle
          label="JPEGでアップロード"
          description="PNGより容量を抑えられます"
          checked={screenshotUploadJpeg}
          onChange={(value) => void handleScreenshotUploadJpegChange(value)}
          disabled={!screenshotSyncEnabled}
        />

        <div>
          <div className="flex items-center justify-between mb-2">
            <div>
              <h4 className="font-medium">JPEG品質</h4>
              <p className="text-sm text-base-content/70">
                ローカル保存 / アップロードの両方に適用（1-100）
              </p>
            </div>
            <span className="text-sm font-mono">{screenshotJpegQuality}</span>
          </div>
          <input
            type="range"
            min={1}
            max={100}
            value={screenshotJpegQuality}
            onChange={(event) => void handleScreenshotJpegQualityChange(Number(event.target.value))}
            className="range range-primary"
            disabled={!jpegQualityEnabled}
          />
        </div>
      </div>
    </div>
  );
}

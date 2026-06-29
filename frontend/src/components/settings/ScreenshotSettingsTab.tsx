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
    setScreenshotHotkey,
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

  return (
    <div className="space-y-6">
      <TabSectionHeader title="スクリーンショット" description="撮影データの同期と形式" />
      <div className="bg-base-200 p-4 rounded-lg space-y-4">
        <SettingsToggle
          label="クラウド同期"
          description="スクリーンショットをクラウドにアップロードします"
          checked={screenshotSyncEnabled}
          onChange={(value) => void handleScreenshotSyncEnabledChange(value)}
        />

        <SettingsToggle
          label="タイトルバーを除外"
          description="オンでクライアント領域のみを撮影します"
          checked={screenshotClientOnly}
          onChange={(value) => void handleScreenshotClientOnlyChange(value)}
        />

        <SettingsToggle
          label="ローカル保存をJPEGにする"
          description="オンでPNGより容量を抑えて保存します"
          checked={screenshotLocalJpeg}
          onChange={(value) => void handleScreenshotLocalJpegChange(value)}
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
                数値が高いほど画質は向上します（1-100）
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
            disabled={!screenshotSyncEnabled || !screenshotUploadJpeg}
          />
        </div>

        <div>
          <div className="mb-2">
            <h4 className="font-medium">ホットキー</h4>
            <p className="text-sm text-base-content/70">
              例: Ctrl+Alt+S（押すとSnipping Toolが起動します）
            </p>
          </div>
          <div className="flex items-center gap-2">
            <input
              type="text"
              className="input input-bordered input-sm flex-1"
              value={screenshotHotkey}
              onChange={(event) => setScreenshotHotkey(event.target.value)}
              onBlur={(event) => void applyScreenshotHotkey(event.target.value, false)}
              readOnly={isCapturingHotkey}
            />
            <button className="btn btn-primary btn-sm" onClick={() => setIsCapturingHotkey(true)}>
              入力開始
            </button>
            <button
              className="btn btn-outline btn-sm"
              onClick={() => void handleScreenshotHotkeyChange(screenshotHotkey)}
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
    </div>
  );
}

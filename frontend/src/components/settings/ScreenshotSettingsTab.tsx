import { useScreenshotSettings } from "@renderer/hooks/useScreenshotSettings";

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
      <div className="border-l-4 border-primary pl-4">
        <h3 className="text-lg font-semibold text-primary mb-1">スクリーンショット</h3>
        <p className="text-sm text-base-content/60">撮影データの同期と形式</p>
      </div>
      <div className="bg-base-200 p-4 rounded-lg space-y-4">
        <div className="flex items-center justify-between">
          <div>
            <h4 className="font-medium">クラウド同期</h4>
            <p className="text-sm text-base-content/70">
              スクリーンショットをクラウドにアップロードします
            </p>
          </div>
          <input
            type="checkbox"
            className="toggle toggle-primary"
            checked={screenshotSyncEnabled}
            onChange={(event) => void handleScreenshotSyncEnabledChange(event.target.checked)}
          />
        </div>

        <div className="flex items-center justify-between">
          <div>
            <h4 className="font-medium">タイトルバーを除外</h4>
            <p className="text-sm text-base-content/70">オンでクライアント領域のみを撮影します</p>
          </div>
          <input
            type="checkbox"
            className="toggle toggle-primary"
            checked={screenshotClientOnly}
            onChange={(event) => void handleScreenshotClientOnlyChange(event.target.checked)}
          />
        </div>

        <div className="flex items-center justify-between">
          <div>
            <h4 className="font-medium">ローカル保存をJPEGにする</h4>
            <p className="text-sm text-base-content/70">オンでPNGより容量を抑えて保存します</p>
          </div>
          <input
            type="checkbox"
            className="toggle toggle-primary"
            checked={screenshotLocalJpeg}
            onChange={(event) => void handleScreenshotLocalJpegChange(event.target.checked)}
          />
        </div>

        <div className="flex items-center justify-between">
          <div>
            <h4 className="font-medium">JPEGでアップロード</h4>
            <p className="text-sm text-base-content/70">PNGより容量を抑えられます</p>
          </div>
          <input
            type="checkbox"
            className="toggle toggle-primary"
            checked={screenshotUploadJpeg}
            onChange={(event) => void handleScreenshotUploadJpegChange(event.target.checked)}
            disabled={!screenshotSyncEnabled}
          />
        </div>

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

        <div className="flex items-center justify-between">
          <div>
            <h4 className="font-medium">ホットキー通知</h4>
            <p className="text-sm text-base-content/70">押下時にWindows通知を表示します</p>
          </div>
          <input
            type="checkbox"
            className="toggle toggle-primary"
            checked={screenshotHotkeyNotify}
            onChange={(event) => void handleScreenshotHotkeyNotifyChange(event.target.checked)}
          />
        </div>
      </div>
    </div>
  );
}

import { TabSectionHeader } from "./TabSectionHeader";

type BehaviorTabProps = {
  offlineMode: boolean;
  onOfflineModeChange: (enabled: boolean) => Promise<void>;
  autoTracking: boolean;
  onAutoTrackingChange: (enabled: boolean) => Promise<void>;
  transferConcurrency: number;
  onTransferConcurrencyInputChange: (value: number) => void;
  onTransferConcurrencyBlur: (value: number) => Promise<void>;
};

export default function BehaviorTab({
  offlineMode,
  onOfflineModeChange,
  autoTracking,
  onAutoTrackingChange,
  transferConcurrency,
  onTransferConcurrencyInputChange,
  onTransferConcurrencyBlur,
}: BehaviorTabProps): React.JSX.Element {
  return (
    <div className="space-y-6">
      <TabSectionHeader
        title="動作設定"
        description="アプリケーションの動作を設定"
        color="secondary"
      />
      <div className="bg-base-200 p-4 rounded-lg space-y-4">
        <div>
          <h4 className="font-medium mb-3">機能設定</h4>
          <div className="form-control mb-4">
            <label className="label cursor-pointer justify-start p-0">
              <input
                type="checkbox"
                className="toggle toggle-primary mr-3"
                checked={offlineMode}
                onChange={(e) => void onOfflineModeChange(e.target.checked)}
              />
              <div>
                <span className="label-text font-medium">オフラインモード</span>
                <p className="text-xs text-base-content/50 mt-1">
                  {offlineMode ? "クラウド機能が無効" : "すべての機能が利用可能"}
                </p>
              </div>
            </label>
          </div>

          <div className="form-control">
            <label className="label cursor-pointer justify-start p-0">
              <input
                type="checkbox"
                className="toggle toggle-primary mr-3"
                checked={autoTracking}
                onChange={(e) => void onAutoTrackingChange(e.target.checked)}
              />
              <div>
                <span className="label-text font-medium">自動ゲーム検出</span>
                <p className="text-xs text-base-content/50 mt-1">
                  {autoTracking ? "実行中ゲームを自動検出して監視開始" : "手動でのゲーム登録のみ"}
                </p>
              </div>
            </label>
          </div>

          <div className="form-control mt-4">
            <label className="label p-0 mb-2">
              <span className="label-text font-medium">同時転送数</span>
            </label>
            <div className="flex items-center gap-3">
              <input
                type="number"
                min={1}
                max={32}
                step={1}
                className="input input-bordered input-sm w-24"
                value={transferConcurrency}
                onChange={(e) => onTransferConcurrencyInputChange(Number(e.target.value))}
                onBlur={(e) => void onTransferConcurrencyBlur(Number(e.target.value))}
              />
              <span className="text-xs text-base-content/50">1〜32</span>
            </div>
            <p className="text-xs text-base-content/50 mt-2">
              アップロード/ダウンロード共通の同時転送数です
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

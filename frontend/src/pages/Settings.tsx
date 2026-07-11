/**
 * @fileoverview 設定ページ
 *
 * トップレベルのフラットなタブで各設定カテゴリを切り替える。
 */

import { useState } from "react";

import AppearanceTab from "@renderer/components/settings/AppearanceTab";
import BehaviorTab from "@renderer/components/settings/BehaviorTab";
import DefaultsTab from "@renderer/components/settings/DefaultsTab";
import R2S3Settings from "@renderer/components/settings/R2S3Settings";
import ScreenshotSettingsTab from "@renderer/components/settings/ScreenshotSettingsTab";
import SyncAndLogsTab from "@renderer/components/settings/SyncAndLogsTab";

type TabType = "appearance" | "behavior" | "defaults" | "screenshot" | "cloud" | "data";

const TABS: { id: TabType; label: string }[] = [
  { id: "appearance", label: "外観" },
  { id: "behavior", label: "動作" },
  { id: "defaults", label: "初期表示" },
  { id: "screenshot", label: "スクリーンショット" },
  { id: "cloud", label: "クラウド" },
  { id: "data", label: "データ・ログ" },
];

export default function Settings(): React.JSX.Element {
  const [activeTab, setActiveTab] = useState<TabType>("appearance");

  return (
    <div className="container mx-auto px-6 py-8">
      <h1 className="text-3xl font-bold mb-6">設定</h1>

      <div role="tablist" className="tabs tabs-lifted mb-6 overflow-x-auto">
        {TABS.map((tab) => (
          <button
            key={tab.id}
            role="tab"
            className={`tab tab-lifted ${activeTab === tab.id ? "tab-active" : ""}`}
            onClick={() => setActiveTab(tab.id)}
          >
            {tab.label}
          </button>
        ))}
      </div>

      <div className="bg-base-100 p-6 rounded-lg border border-base-200">
        {activeTab === "appearance" && <AppearanceTab />}
        {activeTab === "behavior" && <BehaviorTab />}
        {activeTab === "defaults" && <DefaultsTab />}
        {activeTab === "screenshot" && <ScreenshotSettingsTab />}
        {activeTab === "cloud" && <R2S3Settings />}
        {activeTab === "data" && <SyncAndLogsTab />}
      </div>
    </div>
  );
}

/**
 * @fileoverview 設定: 外観タブ
 *
 * DaisyUI テーマの選択と適用を行う。
 */

import { useAtom } from "jotai";

import { DAISYUI_THEMES } from "@renderer/constants/themes";
import { themeAtom, isChangingThemeAtom, changeThemeAtom } from "../../state/settings";
import { TabSectionHeader } from "./TabSectionHeader";

export default function AppearanceTab(): React.JSX.Element {
  const [currentTheme] = useAtom(themeAtom);
  const [isChangingTheme] = useAtom(isChangingThemeAtom);
  const [, changeTheme] = useAtom(changeThemeAtom);

  return (
    <div className="space-y-6">
      <TabSectionHeader title="外観設定" description="アプリケーションの見た目を設定" />
      <div className="bg-base-200 p-4 rounded-lg">
        <div className="mb-3">
          <h4 className="font-medium">テーマ</h4>
          <p className="text-sm text-base-content/70">外観テーマを選択</p>
        </div>
        <div className="form-control">
          <label className="label pb-1">
            <span className="label-text text-sm">現在: {currentTheme}</span>
          </label>
          <div className="flex items-center gap-2">
            <select
              className="select select-bordered select-sm"
              value={currentTheme}
              onChange={(e) => changeTheme(e.target.value as typeof currentTheme)}
              disabled={isChangingTheme}
            >
              {DAISYUI_THEMES.map((theme) => (
                <option key={theme} value={theme}>
                  {theme}
                </option>
              ))}
            </select>
            {isChangingTheme && <span className="loading loading-spinner loading-sm"></span>}
          </div>
        </div>
      </div>
    </div>
  );
}

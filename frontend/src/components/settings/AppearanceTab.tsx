import { useAtom } from "jotai";

import { DAISYUI_THEMES } from "@renderer/constants/themes";
import { themeAtom, isChangingThemeAtom, changeThemeAtom } from "../../state/settings";

export default function AppearanceTab(): React.JSX.Element {
  const [currentTheme] = useAtom(themeAtom);
  const [isChangingTheme] = useAtom(isChangingThemeAtom);
  const [, changeTheme] = useAtom(changeThemeAtom);

  return (
    <div className="space-y-6">
      <div className="border-l-4 border-primary pl-4">
        <h3 className="text-lg font-semibold text-primary mb-1">外観設定</h3>
        <p className="text-sm text-base-content/60">アプリケーションの見た目を設定</p>
      </div>
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

import { useAtom } from "jotai";
import toast from "react-hot-toast";

import {
  defaultSortOptionAtom,
  defaultFilterStateAtom,
  sortOptionLabels,
  filterStateLabels,
} from "../../state/settings";
import type { SortOption, FilterOption } from "src/types/menu";
import { TabSectionHeader } from "./TabSectionHeader";

export default function DefaultsTab(): React.JSX.Element {
  const [defaultSortOption, setDefaultSortOption] = useAtom(defaultSortOptionAtom);
  const [defaultFilterState, setDefaultFilterState] = useAtom(defaultFilterStateAtom);

  const handleSortChange = (newSortOption: SortOption): void => {
    setDefaultSortOption(newSortOption);
    toast.success(`デフォルトソート順を「${sortOptionLabels[newSortOption]}」に変更しました`);
  };

  const handleFilterChange = (newFilterState: FilterOption): void => {
    setDefaultFilterState(newFilterState);
    toast.success(`デフォルトフィルターを「${filterStateLabels[newFilterState]}」に変更しました`);
  };

  return (
    <div className="space-y-6">
      <TabSectionHeader
        title="デフォルト設定"
        description="ホーム画面の初期表示設定"
        color="accent"
      />

      <div className="grid gap-4 md:grid-cols-2">
        <div className="bg-base-200 p-4 rounded-lg">
          <div className="mb-3">
            <h4 className="font-medium">ソート順</h4>
            <p className="text-sm text-base-content/70">初期表示時のソート方法</p>
          </div>
          <div className="form-control">
            <div className="mb-2">
              <p className="text-xs text-base-content/60 mt-1">
                {`現在: ${sortOptionLabels[defaultSortOption]}`}
              </p>
            </div>
            <select
              className="select select-bordered select-sm"
              value={defaultSortOption}
              onChange={(e) => handleSortChange(e.target.value as SortOption)}
            >
              {Object.entries(sortOptionLabels).map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </select>
          </div>
        </div>

        <div className="bg-base-200 p-4 rounded-lg">
          <div className="mb-3">
            <h4 className="font-medium">フィルター</h4>
            <p className="text-sm text-base-content/70">初期表示時のフィルター状態</p>
          </div>
          <div className="form-control">
            <div className="mb-2">
              <p className="text-xs text-base-content/60 mt-1">
                {`現在: ${filterStateLabels[defaultFilterState]}`}
              </p>
            </div>
            <select
              className="select select-bordered select-sm"
              value={defaultFilterState}
              onChange={(e) => handleFilterChange(e.target.value as FilterOption)}
            >
              {Object.entries(filterStateLabels).map(([value, label]) => (
                <option key={value} value={value}>
                  {label}
                </option>
              ))}
            </select>
          </div>
        </div>
      </div>
    </div>
  );
}

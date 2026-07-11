/**
 * @fileoverview プロセス監視のデバッグページ
 */

import { useCallback, useEffect, useMemo, useState } from "react";

type ProcessSnapshotItem = {
  name: string;
  pid: number;
  cmd: string;
  normalizedName: string;
  normalizedCmd: string;
};

type ProcessSnapshot = {
  source: string;
  items: ProcessSnapshotItem[];
};

export default function DebugProcess(): React.JSX.Element {
  const [snapshot, setSnapshot] = useState<ProcessSnapshot>({ source: "none", items: [] });
  const [filter, setFilter] = useState("");
  const [isLoading, setIsLoading] = useState(false);

  const loadSnapshot = useCallback(async () => {
    setIsLoading(true);
    try {
      const data = await window.api.processMonitor.getProcessSnapshot();
      setSnapshot(data);
    } finally {
      setIsLoading(false);
    }
  }, []);

  useEffect(() => {
    loadSnapshot();
  }, [loadSnapshot]);

  const filteredItems = useMemo(() => {
    const keyword = filter.trim().toLowerCase();
    if (!keyword) return snapshot.items;
    return snapshot.items.filter((item) => {
      return (
        item.name.toLowerCase().includes(keyword) ||
        item.cmd.toLowerCase().includes(keyword) ||
        item.normalizedName.toLowerCase().includes(keyword) ||
        item.normalizedCmd.toLowerCase().includes(keyword)
      );
    });
  }, [filter, snapshot.items]);

  return (
    <div className="container mx-auto px-6 py-8">
      <div className="flex items-center justify-between mb-6">
        <div>
          <h1 className="text-3xl font-bold">プロセス監視デバッグ</h1>
          <p className="text-sm text-base-content/70">
            取得したプロセス一覧と正規化後の値を確認できます（source: {snapshot.source}）
          </p>
        </div>
        <button
          className={`btn btn-primary ${isLoading ? "btn-disabled" : ""}`}
          onClick={loadSnapshot}
        >
          {isLoading ? "取得中..." : "再取得"}
        </button>
      </div>

      <div className="mb-4">
        <input
          className="input input-bordered w-full"
          placeholder="フィルタ（exe名 / パス / 正規化後）"
          value={filter}
          onChange={(event) => setFilter(event.target.value)}
        />
      </div>

      <div className="overflow-x-auto bg-base-100 rounded-lg border border-base-200">
        <table className="table table-zebra">
          <thead>
            <tr>
              <th>PID</th>
              <th>名前</th>
              <th>Cmd</th>
              <th>正規化名</th>
              <th>正規化Cmd</th>
            </tr>
          </thead>
          <tbody>
            {filteredItems.length === 0 ? (
              <tr>
                <td colSpan={5} className="text-center text-sm text-base-content/60">
                  表示対象がありません
                </td>
              </tr>
            ) : (
              filteredItems.map((item) => (
                <tr key={`${item.pid}-${item.name}`}>
                  <td className="font-mono text-xs">{item.pid}</td>
                  <td className="text-sm">{item.name}</td>
                  <td className="text-xs break-all">{item.cmd}</td>
                  <td className="text-xs break-all">{item.normalizedName}</td>
                  <td className="text-xs break-all">{item.normalizedCmd}</td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

/**
 * @fileoverview 設定タブのセクション見出し
 *
 * 設定画面内の小見出し表示。
 */

type AccentColor = "primary" | "secondary" | "accent" | "info" | "success" | "warning" | "error";

type TabSectionHeaderProps = {
  title: string;
  description: string;
  /** デフォルトは primary。タブごとにアクセントカラーを変えるときに指定する。 */
  color?: AccentColor;
};

// Tailwind JIT は動的にクラス名を組み立てると拾えないため、明示的にマップ化する。
const borderClassByColor: Record<AccentColor, string> = {
  primary: "border-primary",
  secondary: "border-secondary",
  accent: "border-accent",
  info: "border-info",
  success: "border-success",
  warning: "border-warning",
  error: "border-error",
};

const textClassByColor: Record<AccentColor, string> = {
  primary: "text-primary",
  secondary: "text-secondary",
  accent: "text-accent",
  info: "text-info",
  success: "text-success",
  warning: "text-warning",
  error: "text-error",
};

export function TabSectionHeader({
  title,
  description,
  color = "primary",
}: TabSectionHeaderProps): React.JSX.Element {
  return (
    <div className={`border-l-4 ${borderClassByColor[color]} pl-4`}>
      <h3 className={`text-lg font-semibold ${textClassByColor[color]} mb-1`}>{title}</h3>
      <p className="text-sm text-base-content/60">{description}</p>
    </div>
  );
}

/**
 * @fileoverview 設定タブ共通のトグル行コンポーネント。
 *
 * 「label + description + toggle」のお決まりレイアウトを集約する。
 * 設定タブの各トグルは値とハンドラを差し替えるだけで描画できる。
 */

type SettingsToggleProps = {
  label: string;
  description: string;
  checked: boolean;
  onChange: (value: boolean) => void;
  disabled?: boolean;
};

export function SettingsToggle({
  label,
  description,
  checked,
  onChange,
  disabled,
}: SettingsToggleProps): React.JSX.Element {
  return (
    <div className="flex items-center justify-between">
      <div>
        <h4 className="font-medium">{label}</h4>
        <p className="text-sm text-base-content/70">{description}</p>
      </div>
      <input
        type="checkbox"
        className="toggle toggle-primary"
        checked={checked}
        onChange={(event) => onChange(event.target.checked)}
        disabled={disabled}
      />
    </div>
  );
}

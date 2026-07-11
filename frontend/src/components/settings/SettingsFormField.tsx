/**
 * @fileoverview 設定フォームフィールドコンポーネント
 *
 * このコンポーネントは、設定ページで使用される入力フィールドを統一的に提供します。
 */

export type SettingsFormFieldProps = {
  label: string;
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  type?: "text" | "password";
  disabled?: boolean;
  required?: boolean;
  error?: string;
  helpText?: string;
  labelWidth?: string;
};

export function SettingsFormField({
  label,
  value,
  onChange,
  placeholder = "",
  type = "text",
  disabled = false,
  required = false,
  error,
  helpText,
  labelWidth = "w-36",
}: SettingsFormFieldProps): React.JSX.Element {
  return (
    <div className="flex flex-col space-y-1">
      <div className="flex items-center">
        <span className={`${labelWidth} text-sm font-medium`}>
          {label}
          {required && <span className="text-error ml-1">*</span>}
        </span>
        <input
          type={type}
          className={`input input-bordered flex-1 ${error ? "input-error" : ""}`}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          placeholder={placeholder}
          disabled={disabled}
          required={required}
        />
      </div>

      {error && <div className="text-error text-sm ml-36">{error}</div>}

      {helpText && !error && <div className="text-base-content text-xs ml-36">{helpText}</div>}
    </div>
  );
}

export default SettingsFormField;

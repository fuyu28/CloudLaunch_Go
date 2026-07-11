/**
 * @fileoverview ファイル選択ボタンコンポーネント
 *
 * このコンポーネントは、ファイルやフォルダの選択操作を統一的に提供します。
 */

import { MESSAGES } from "@renderer/constants";

export type FileSelectButtonProps = {
  label: string;
  value: string;
  onChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
  onBrowse: () => void;
  disabled?: boolean;
  placeholder?: string;
  name?: string;
  required?: boolean;
  browseButtonText?: string;
  errorMessage?: string;
  onBlur?: () => void;
};

export function FileSelectButton({
  label,
  value,
  onChange,
  onBrowse,
  disabled = false,
  placeholder = "",
  name,
  required = false,
  browseButtonText = MESSAGES.UI.BROWSE,
  errorMessage,
  onBlur,
}: FileSelectButtonProps): React.JSX.Element {
  const inputId = name || label.replace(/\s+/g, "-").toLowerCase();

  return (
    <div>
      <label className="label" htmlFor={inputId}>
        <span className="label-text">{label}</span>
      </label>
      <div className="flex">
        <input
          type="text"
          id={inputId}
          name={name}
          value={value}
          onChange={onChange}
          onBlur={onBlur}
          className={`input input-bordered flex-1 ${errorMessage ? "input-error" : ""}`}
          placeholder={placeholder}
          required={required}
          disabled={disabled}
        />
        <button type="button" className="btn ml-2" onClick={onBrowse} disabled={disabled}>
          {browseButtonText}
        </button>
      </div>
      {errorMessage && (
        <div className="label">
          <span className="label-text-alt text-error">{errorMessage}</span>
        </div>
      )}
    </div>
  );
}

export default FileSelectButton;

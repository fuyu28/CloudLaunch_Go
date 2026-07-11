/**
 * @fileoverview ゲームフォームフィールドコンポーネント
 *
 * このコンポーネントは、ゲーム登録・編集フォームの入力フィールドを提供します。
 */

import { FileSelectButton } from "../common/FileSelectButton";
import type { InputGameData } from "src/types/game";
import type { GameFormValidationResult } from "../../hooks/useGameFormValidationZod";

export type GameFormFieldsProps = {
  gameData: InputGameData;
  onChange: (event: React.ChangeEvent<HTMLInputElement>) => void;
  onBrowseImage: () => void;
  onBrowseExe: () => void;
  onBrowseSaveFolder: () => void;
  disabled?: boolean;
  validation: GameFormValidationResult;
};

export function GameFormFields({
  gameData,
  onChange,
  onBrowseImage,
  onBrowseExe,
  onBrowseSaveFolder,
  disabled = false,
  validation,
}: GameFormFieldsProps): React.JSX.Element {
  return (
    <div className="space-y-4">
      <div>
        <label className="label" htmlFor="title">
          <span className="label-text">タイトル</span>
        </label>
        <input
          type="text"
          id="title"
          name="title"
          value={gameData.title}
          onChange={onChange}
          onBlur={() => validation.markFieldAsTouched("title")}
          className={`input input-bordered w-full ${validation.errors.title ? "input-error" : ""}`}
          required
          disabled={disabled}
        />
        {validation.errors.title && (
          <div className="label">
            <span className="label-text-alt text-error">{validation.errors.title}</span>
          </div>
        )}
      </div>

      <div>
        <label className="label" htmlFor="publisher">
          <span className="label-text">ブランド</span>
        </label>
        <input
          type="text"
          id="publisher"
          name="publisher"
          value={gameData.publisher}
          onChange={onChange}
          onBlur={() => validation.markFieldAsTouched("publisher")}
          className={`input input-bordered w-full ${validation.errors.publisher ? "input-error" : ""}`}
          required
          disabled={disabled}
        />
        {validation.errors.publisher && (
          <div className="label">
            <span className="label-text-alt text-error">{validation.errors.publisher}</span>
          </div>
        )}
      </div>

      <FileSelectButton
        label="サムネイル画像の場所"
        name="imagePath"
        value={gameData.imagePath || ""}
        onChange={onChange}
        onBrowse={onBrowseImage}
        onBlur={() => validation.markFieldAsTouched("imagePath")}
        disabled={disabled}
        placeholder="画像ファイルを選択してください"
        errorMessage={validation.errors.imagePath}
      />

      <FileSelectButton
        label="実行ファイルの場所"
        name="exePath"
        value={gameData.exePath}
        onChange={onChange}
        onBrowse={onBrowseExe}
        onBlur={() => validation.markFieldAsTouched("exePath")}
        disabled={disabled}
        placeholder="実行ファイルを選択してください"
        required
        errorMessage={validation.errors.exePath}
      />

      <FileSelectButton
        label="セーブデータフォルダの場所"
        name="saveFolderPath"
        value={gameData.saveFolderPath || ""}
        onChange={onChange}
        onBrowse={onBrowseSaveFolder}
        onBlur={() => validation.markFieldAsTouched("saveFolderPath")}
        disabled={disabled}
        placeholder="セーブデータフォルダを選択してください"
        errorMessage={validation.errors.saveFolderPath}
      />
    </div>
  );
}

export default GameFormFields;

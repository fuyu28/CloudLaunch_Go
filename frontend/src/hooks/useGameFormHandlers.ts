import { useCallback } from "react";

import { useFileSelection } from "@renderer/hooks/useFileSelection";

import type { InputGameData } from "src/types/game";
import type { GameFormValidationResult } from "./useGameFormValidationZod";

type GameFormValidationMethods = Pick<
  GameFormValidationResult,
  "markFieldAsTouched" | "validateFileField"
>;

type UseGameFormHandlersParams = {
  setGameData: React.Dispatch<React.SetStateAction<InputGameData>>;
  validation: GameFormValidationMethods;
};

const IMAGE_FILTERS = [{ name: "Image", extensions: ["png", "jpg", "jpeg", "gif"] }];
const EXECUTABLE_FILTERS = [{ name: "Executable", extensions: ["exe", "app"] }];

export function useGameFormHandlers({ setGameData, validation }: UseGameFormHandlersParams): {
  isBrowsing: boolean;
  browseImage: () => Promise<void>;
  browseExe: () => Promise<void>;
  browseSaveFolder: () => Promise<void>;
  handleChange: (e: React.ChangeEvent<HTMLInputElement>) => void;
} {
  const { isBrowsing, selectFile, selectFolder } = useFileSelection();

  const updateField = useCallback(
    (field: keyof InputGameData, value: string) => {
      setGameData((prev) => ({ ...prev, [field]: value }));
      validation.markFieldAsTouched(field);
    },
    [setGameData, validation],
  );

  const browseImage = useCallback(async () => {
    await selectFile(IMAGE_FILTERS, (filePath) => {
      updateField("imagePath", filePath);
      validation.validateFileField("imagePath");
    });
  }, [selectFile, updateField, validation]);

  const browseExe = useCallback(async () => {
    await selectFile(EXECUTABLE_FILTERS, (filePath) => {
      updateField("exePath", filePath);
      validation.validateFileField("exePath");
    });
  }, [selectFile, updateField, validation]);

  const browseSaveFolder = useCallback(async () => {
    await selectFolder((folderPath) => {
      updateField("saveFolderPath", folderPath);
      validation.validateFileField("saveFolderPath");
    });
  }, [selectFolder, updateField, validation]);

  const handleChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>): void => {
      const { name, value } = e.target;
      updateField(name as keyof InputGameData, value);
    },
    [updateField],
  );

  return {
    isBrowsing,
    browseImage,
    browseExe,
    browseSaveFolder,
    handleChange,
  };
}

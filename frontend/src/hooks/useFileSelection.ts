/**
 * @fileoverview ファイル選択フックユーティリティ
 *
 * このファイルは、ファイル・フォルダ選択の共通ロジックを提供します。
 */

import { useState, useCallback } from "react";

import { handleApiError } from "../utils/errorHandler";

type FileFilter = {
  name: string;
  extensions: string[];
};

export function useFileSelection(): {
  isBrowsing: boolean;
  selectFile: (filters: FileFilter[], onSuccess: (filePath: string) => void) => Promise<void>;
  selectFolder: (onSuccess: (folderPath: string) => void) => Promise<void>;
} {
  const [isBrowsing, setIsBrowsing] = useState(false);

  const selectFile = useCallback(
    async (filters: FileFilter[], onSuccess: (filePath: string) => void) => {
      setIsBrowsing(true);
      try {
        const result = await window.api.file.selectFile(filters);
        if (result.success && result.data !== undefined) {
          onSuccess(result.data);
        } else if (result.success) {
          // バックエンドのファイルダイアログはユーザキャンセル時に success:true, data:undefined を返す。
          // これをエラー扱いにするとキャンセル時にトーストが出てしまうため、何もせず抜ける。
          return;
        } else {
          handleApiError(
            {
              success: false,
              message: (result as { success: false; message: string }).message,
            },
            "ファイルの選択に失敗しました",
          );
        }
      } finally {
        setIsBrowsing(false);
      }
    },
    [],
  );

  const selectFolder = useCallback(async (onSuccess: (folderPath: string) => void) => {
    setIsBrowsing(true);
    try {
      const result = await window.api.file.selectFolder();
      if (result.success && result.data !== undefined) {
        onSuccess(result.data);
      } else if (result.success) {
        // フォルダダイアログもユーザキャンセル時は success:true, data:undefined を返す仕様。
        // エラートーストを出さず何もせず終了する。
        return;
      } else {
        handleApiError(
          {
            success: false,
            message: (result as { success: false; message: string }).message,
          },
          "フォルダの選択に失敗しました",
        );
      }
    } finally {
      setIsBrowsing(false);
    }
  }, []);

  return {
    isBrowsing,
    selectFile,
    selectFolder,
  };
}

/**
 * @fileoverview 画像読み込み管理用カスタムフック
 *
 * このフックは画像の読み込み状態を管理し、エラーハンドリングを行います。
 */

import { useEffect, useState } from "react";
import toast from "react-hot-toast";

import { logger } from "@renderer/utils/logger";

import type { ApiResult } from "src/types/result";

type ImageLoadState = {
  imageSrc?: string;
  isLoading: boolean;
  error?: string;
};

/**
 * NoImage SVGをbase64エンコードしたdata URL
 * 灰色の背景に "No Image" テキストが表示される
 */
const createNoImageDataUrl = (): string => {
  const svg = `
    <svg width="400" height="300" xmlns="http://www.w3.org/2000/svg">
      <rect width="100%" height="100%" fill="#f3f4f6"/>
      <text x="50%" y="50%" font-family="Arial, sans-serif" font-size="24" 
            fill="#9ca3af" text-anchor="middle" dominant-baseline="middle">
        No Image
      </text>
    </svg>
  `;
  return `data:image/svg+xml;base64,${btoa(svg)}`;
};

export const useImageLoader = (src: string): ImageLoadState => {
  const [state, setState] = useState<ImageLoadState>(() => ({
    imageSrc: undefined,
    isLoading: true,
    error: undefined,
  }));

  useEffect(() => {
    let mounted = true;

    const loadImage = async (): Promise<void> => {
      if (!mounted) return;

      setState((prev) => ({ ...prev, isLoading: true, error: undefined }));

      // 未設定画像はエラーにせず NoImage（トーストも出さない）。
      if (!src || src.trim() === "") {
        if (mounted) {
          setState({
            imageSrc: createNoImageDataUrl(),
            isLoading: false,
            error: undefined,
          });
        }
        return;
      }

      try {
        const result = await validateAndLoadImage(src);

        if (mounted) {
          if (result.success && result.data) {
            setState({
              imageSrc: result.data,
              isLoading: false,
              error: undefined,
            });
          } else {
            // 読み込み失敗は NoImage + トースト（サイレント失敗にしない）。
            const errorMessage = result.success ? "データが取得できませんでした" : result.message;
            logger.warn("画像読み込み失敗:", {
              component: "useImageLoader",
              function: "loadImage",
              data: { src, errorMessage },
            });
            if (errorMessage) {
              // Home 等で複数の壊れた画像がある場合にトーストが氾濫しないよう、
              // 画像パス由来の toastId で dedup する。
              toast.error(`画像読み込み失敗: ${errorMessage}`, {
                id: `image-load-failed:${src.trim()}`,
              });
            }
            setState({
              imageSrc: createNoImageDataUrl(),
              isLoading: false,
              error: errorMessage || "画像読み込みに失敗しました",
            });
          }
        }
      } catch (error) {
        if (mounted) {
          logger.error("Error loading image:", {
            component: "useImageLoader",
            function: "loadImage",
            error: error instanceof Error ? error : new Error(String(error)),
          });
          const errorMsg = error instanceof Error ? error.message : "不明なエラー";
          // 同一画像の失敗トーストを ID で dedup（連打しない）。
          toast.error(`画像読み込みエラー: ${errorMsg}`, {
            id: `image-load-failed:${src.trim()}`,
          });
          setState({
            imageSrc: createNoImageDataUrl(),
            isLoading: false,
            error: errorMsg,
          });
        }
      }
    };

    loadImage();
    return () => {
      mounted = false;
    };
  }, [src]);

  return state;
};

const validateAndLoadImage = async (src: string): Promise<ApiResult<string>> => {
  const isHttpUrl = src.startsWith("http://") || src.startsWith("https://");
  const isFileUrl = src.startsWith("file://");
  const isAbsolutePath = /^[A-Za-z]:\\/.test(src) || src.startsWith("/");

  if (!isHttpUrl && !isFileUrl && !isAbsolutePath) {
    return {
      success: false,
      message: `無効な画像パス形式: ${src}`,
    };
  }

  if (isHttpUrl) {
    try {
      new URL(src); // URL形式の検証
    } catch {
      return {
        success: false,
        message: `無効なURL形式: ${src}`,
      };
    }
  }

  const isLocal = isFileUrl || isAbsolutePath;

  try {
    if (isLocal) {
      const path = src.replace(/^file:\/\//, "");
      const result = (await window.api.loadImage.loadImageFromLocal(path)) as ApiResult<string>;
      return result;
    } else {
      const result = (await window.api.loadImage.loadImageFromWeb(src)) as ApiResult<string>;
      return result;
    }
  } catch (error) {
    return {
      success: false,
      message: error instanceof Error ? error.message : "不明なエラー",
    };
  }
};

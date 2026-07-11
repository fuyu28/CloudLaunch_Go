/**
 * @fileoverview 動的画像読み込みコンポーネント
 *
 * このコンポーネントは、ローカル画像とWeb画像の読み込みを統一的に処理し、
 * 画像が存在しない場合にNoImage画像を表示します。
 */

import { memo } from "react";

import { useImageLoader } from "@renderer/hooks/useImageLoader";

import type { ImgHTMLAttributes } from "react";

type DynamicImgProps = Omit<ImgHTMLAttributes<HTMLImageElement>, "src"> & {
  src: string; // 普通のURL or ローカルファイルパス（空文字列の場合はNoImage）
};

const DynamicImage = memo(function DynamicImage({
  src: originalSrc,
  ...imgProps
}: DynamicImgProps): React.JSX.Element {
  const { imageSrc, isLoading } = useImageLoader(originalSrc);

  if (isLoading && !imageSrc) {
    return (
      <div
        className="flex items-center justify-center bg-base-200 text-base-content"
        style={{
          width: imgProps.width || "100%",
          height: imgProps.height || "200px",
          ...imgProps.style,
        }}
      >
        <span className="text-sm">Loading...</span>
      </div>
    );
  }

  if (imageSrc) {
    return <img src={imageSrc} {...imgProps} />;
  }

  // フォールバック（通常は発生しない）
  return (
    <div
      className="flex items-center justify-center bg-gray-100 text-base-content"
      style={{
        width: imgProps.width || "100%",
        height: imgProps.height || "200px",
        ...imgProps.style,
      }}
    >
      <span className="text-sm">No Image</span>
    </div>
  );
});

export default DynamicImage;

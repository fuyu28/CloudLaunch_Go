/**
 * @fileoverview ベースモーダルコンポーネント
 *
 * このコンポーネントは、アプリケーション内で使用されるモーダルの基本構造を提供します。
 */

import React from "react";
import { RxCross1 } from "react-icons/rx";

export type ModalSize = "sm" | "md" | "lg" | "xl" | "full";

export type BaseModalProps = {
  isOpen: boolean;
  onClose: () => void;
  onClosed?: () => void;
  title?: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
  size?: ModalSize;
  showCloseButton?: boolean;
  /** モーダルのID（一意である必要があります） */
  id?: string;
  className?: string;
  closeOnClickOutside?: boolean;
  closeOnEscape?: boolean;
};

/**
 * モーダルサイズに対応するCSSクラスのマッピング
 */
const sizeClasses: Record<ModalSize, string> = {
  sm: "max-w-sm",
  md: "max-w-md",
  lg: "max-w-lg",
  xl: "max-w-xl",
  full: "max-w-full",
};

export function BaseModal({
  isOpen,
  onClose,
  onClosed,
  title,
  children,
  footer,
  size = "lg",
  showCloseButton = true,
  id = "base-modal",
  className = "",
  closeOnClickOutside = true,
  closeOnEscape = true,
}: BaseModalProps): React.JSX.Element {
  React.useEffect(() => {
    if (!closeOnEscape || !isOpen) return;

    const handleEscape = (event: KeyboardEvent): void => {
      if (event.key === "Escape") {
        onClose();
      }
    };

    document.addEventListener("keydown", handleEscape);
    return () => document.removeEventListener("keydown", handleEscape);
  }, [isOpen, onClose, closeOnEscape]);

  // モーダルが閉じられた後の処理
  // 初期マウント時（一度も開いていない状態）に onClosed が発火しないよう、
  // 一度でも isOpen=true になったかどうかを ref で保持しガードする。
  const hasBeenOpenedRef = React.useRef<boolean>(false);
  React.useEffect(() => {
    if (isOpen) {
      hasBeenOpenedRef.current = true;
      return undefined;
    }
    if (onClosed && hasBeenOpenedRef.current) {
      // 即 onClosed すると閉じアニメ中に親が unmount しチラつくため、DaisyUI 分待ってから。
      const timer = setTimeout(() => {
        onClosed();
      }, 300);

      return () => clearTimeout(timer);
    }
    return undefined;
  }, [isOpen, onClosed]);

  const handleBackdropClick = (event: React.MouseEvent): void => {
    if (closeOnClickOutside && event.target === event.currentTarget) {
      onClose();
    }
  };

  return (
    <>
      <input type="checkbox" id={id} className="modal-toggle" checked={isOpen} readOnly />
      <div className="modal cursor-pointer" onClick={handleBackdropClick}>
        <div
          className={`modal-box relative ${sizeClasses[size]} ${className}`}
          onClick={(e) => e.stopPropagation()}
        >
          {title && (
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-xl font-bold">{title}</h3>
              {showCloseButton && (
                <button
                  className="btn btn-sm btn-circle absolute right-2 top-2"
                  onClick={onClose}
                  type="button"
                  aria-label="モーダルを閉じる"
                >
                  <RxCross1 />
                </button>
              )}
            </div>
          )}

          {/* 閉じるボタン（タイトルなしの場合） */}
          {!title && showCloseButton && (
            <button
              className="btn btn-sm btn-circle absolute right-2 top-2"
              onClick={onClose}
              type="button"
              aria-label="モーダルを閉じる"
            >
              <RxCross1 />
            </button>
          )}

          <div className="modal-content">{children}</div>

          {footer && <div className="modal-action">{footer}</div>}
        </div>
      </div>
    </>
  );
}

export default BaseModal;

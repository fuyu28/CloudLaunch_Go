/**
 * @fileoverview 時間フォーマットフック
 *
 * このファイルは、秒数を人間が読みやすい形式に変換するフックを提供します。
 */

import { useMemo } from "react";

export type TimeFormatHook = {
  formatDuration: (seconds: number) => string;
  formatShort: (seconds: number) => string;
  formatSmart: (seconds: number) => string;
  formatDate: (date: Date | string | number | null | undefined) => string;
  /** 日付+時間フォーマット（例: "2025年7月7日(月) 11:11"） */
  formatDateWithTime: (date: Date | string | number | null | undefined) => string;
  /** 日付+時間+秒フォーマット（例: "2025年7月7日(月) 11:11:30"） */
  formatDateWithTimeSeconds: (date: Date | string | number | null | undefined) => string;
};

export function useTimeFormat(): TimeFormatHook {
  const normalizeDate = (value: Date | string | number | null | undefined): Date | null => {
    if (!value) return null;
    if (value instanceof Date) return Number.isNaN(value.getTime()) ? null : value;
    const parsed = new Date(value);
    return Number.isNaN(parsed.getTime()) ? null : parsed;
  };

  const formatDuration = useMemo(() => {
    return (seconds: number): string => {
      if (seconds <= 0) return "0秒";

      const hours = Math.floor(seconds / 3600);
      const minutes = Math.floor((seconds % 3600) / 60);
      const remainingSeconds = seconds % 60;

      const parts: string[] = [];

      if (hours > 0) {
        parts.push(`${hours}時間`);
      }
      if (minutes > 0) {
        parts.push(`${minutes}分`);
      }
      if (remainingSeconds > 0 || parts.length === 0) {
        parts.push(`${remainingSeconds}秒`);
      }

      return parts.join("");
    };
  }, []);

  const formatShort = useMemo(() => {
    return (seconds: number): string => {
      if (seconds <= 0) return "0:00";

      const hours = Math.floor(seconds / 3600);
      const minutes = Math.floor((seconds % 3600) / 60);
      const remainingSeconds = seconds % 60;

      if (hours > 0) {
        return `${hours}:${String(minutes).padStart(2, "0")}:${String(remainingSeconds).padStart(2, "0")}`;
      } else {
        return `${minutes}:${String(remainingSeconds).padStart(2, "0")}`;
      }
    };
  }, []);

  const formatSmart = useMemo(() => {
    return (seconds: number): string => {
      if (seconds <= 0) return "未プレイ";

      const hours = Math.floor(seconds / 3600);
      const minutes = Math.floor((seconds % 3600) / 60);

      if (hours > 0) {
        if (minutes > 0) {
          return `${hours}時間${minutes}分`;
        } else {
          return `${hours}時間`;
        }
      } else if (minutes > 0) {
        return `${minutes}分`;
      } else {
        return `${seconds}秒`;
      }
    };
  }, []);

  const formatDate = useMemo(() => {
    return (date: Date | string | number | null | undefined): string => {
      const normalized = normalizeDate(date);
      if (!normalized) {
        return "不明";
      }
      const year = normalized.getFullYear();
      const month = normalized.getMonth() + 1;
      const day = normalized.getDate();
      const dayOfWeek = ["日", "月", "火", "水", "木", "金", "土"][normalized.getDay()];

      return `${year}年${month}月${day}日(${dayOfWeek})`;
    };
  }, []);

  const formatDateWithTime = useMemo(() => {
    return (date: Date | string | number | null | undefined): string => {
      const normalized = normalizeDate(date);
      if (!normalized) {
        return "不明";
      }
      const year = normalized.getFullYear();
      const month = normalized.getMonth() + 1;
      const day = normalized.getDate();
      const dayOfWeek = ["日", "月", "火", "水", "木", "金", "土"][normalized.getDay()];
      const hours = normalized.getHours();
      const hoursPadded = String(hours).padStart(2, "0");
      const minutes = normalized.getMinutes();
      const minutesPadded = String(minutes).padStart(2, "0");

      return `${year}年${month}月${day}日(${dayOfWeek}) ${hoursPadded}:${minutesPadded}`;
    };
  }, []);

  const formatDateWithTimeSeconds = useMemo(() => {
    return (date: Date | string | number | null | undefined): string => {
      const normalized = normalizeDate(date);
      if (!normalized) {
        return "不明";
      }
      const year = normalized.getFullYear();
      const month = normalized.getMonth() + 1;
      const day = normalized.getDate();
      const dayOfWeek = ["日", "月", "火", "水", "木", "金", "土"][normalized.getDay()];
      const hours = normalized.getHours();
      const hoursPadded = String(hours).padStart(2, "0");
      const minutes = normalized.getMinutes();
      const minutesPadded = String(minutes).padStart(2, "0");
      const seconds = normalized.getSeconds();
      const secondsPadded = String(seconds).padStart(2, "0");

      return `${year}年${month}月${day}日(${dayOfWeek}) ${hoursPadded}:${minutesPadded}:${secondsPadded}`;
    };
  }, []);

  return {
    formatDuration,
    formatShort,
    formatSmart,
    formatDate,
    formatDateWithTime,
    formatDateWithTimeSeconds,
  };
}

export const timeUtils = {
  /**
   * 秒数を時、分、秒に分解
   * @param seconds 総秒数
   * @returns 時、分、秒のオブジェクト
   */
  parseSeconds: (seconds: number) => {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const remainingSeconds = seconds % 60;
    return { hours, minutes, seconds: remainingSeconds };
  },

  /**
   * 時、分、秒を秒数に変換
   * @param hours 時間
   * @param minutes 分
   * @param seconds 秒
   * @returns 総秒数
   */
  toSeconds: (hours: number, minutes: number, seconds: number): number => {
    return hours * 3600 + minutes * 60 + seconds;
  },

  /**
   * 詳細フォーマット（非hooks版）
   * @param seconds 秒数
   * @returns フォーマット済み文字列
   */
  formatDuration: (seconds: number): string => {
    if (seconds <= 0) return "0秒";

    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const remainingSeconds = seconds % 60;

    const parts: string[] = [];

    if (hours > 0) {
      parts.push(`${hours}時間`);
    }
    if (minutes > 0) {
      parts.push(`${minutes}分`);
    }
    if (remainingSeconds > 0 || parts.length === 0) {
      parts.push(`${remainingSeconds}秒`);
    }

    return parts.join("");
  },

  /**
   * スマートフォーマット（非hooks版）
   * @param seconds 秒数
   * @returns フォーマット済み文字列
   */
  formatSmart: (seconds: number): string => {
    if (seconds <= 0) return "未プレイ";

    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);

    if (hours > 0) {
      if (minutes > 0) {
        return `${hours}時間${minutes}分`;
      } else {
        return `${hours}時間`;
      }
    } else if (minutes > 0) {
      return `${minutes}分`;
    } else {
      return `${seconds}秒`;
    }
  },

  /**
   * 日付フォーマット（非hooks版）
   * @param date 日付
   * @returns フォーマット済み文字列（例: "2025年7月7日(月)"）
   */
  formatDate: (date: Date): string => {
    const year = date.getFullYear();
    const month = date.getMonth() + 1;
    const day = date.getDate();
    const dayOfWeek = ["日", "月", "火", "水", "木", "金", "土"][date.getDay()];

    return `${year}年${month}月${day}日(${dayOfWeek})`;
  },
};

export default useTimeFormat;

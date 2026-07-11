export type NormalizeHotkeyFailureReason =
  | "cancel"
  | "modifier-only"
  | "need-modifier"
  | "unsupported";

export type NormalizeHotkeyResult =
  | { ok: true; combo: string }
  | { ok: false; reason: NormalizeHotkeyFailureReason };

const NAMED_CODE_KEYS: Record<string, string> = {
  Insert: "Insert",
  Delete: "Delete",
  Home: "Home",
  End: "End",
  PageUp: "PageUp",
  PageDown: "PageDown",
  ScrollLock: "ScrollLock",
  Pause: "Pause",
  Space: "Space",
};

const NAMED_KEY_KEYS: Record<string, string> = {
  Insert: "Insert",
  Delete: "Delete",
  Home: "Home",
  End: "End",
  PageUp: "PageUp",
  PageDown: "PageDown",
  ScrollLock: "ScrollLock",
  Pause: "Pause",
  " ": "Space",
  Spacebar: "Space",
};

const BARE_ALLOWED = new Set([
  "Insert",
  "Delete",
  "Home",
  "End",
  "PageUp",
  "PageDown",
  "ScrollLock",
  "Pause",
  ...Array.from({ length: 11 }, (_, i) => `F${i + 1}`),
]);

function resolveMainKey(event: Pick<KeyboardEvent, "key" | "code">): string | null {
  const { code, key } = event;

  // PrintScreen / F12 は非対応
  if (code === "PrintScreen" || code === "F12" || key === "PrintScreen" || key === "F12") {
    return null;
  }

  const fCode = /^F([1-9]|1[01])$/.exec(code);
  if (fCode) {
    return `F${fCode[1]}`;
  }
  if (NAMED_CODE_KEYS[code]) {
    return NAMED_CODE_KEYS[code];
  }
  const keyMatch = /^Key([A-Z])$/.exec(code);
  if (keyMatch) {
    return keyMatch[1];
  }
  const digitMatch = /^Digit([0-9])$/.exec(code);
  if (digitMatch) {
    return digitMatch[1];
  }

  if (/^F([1-9]|1[01])$/.test(key)) {
    return key.toUpperCase();
  }
  if (NAMED_KEY_KEYS[key]) {
    return NAMED_KEY_KEYS[key];
  }
  if (key.length === 1 && /[A-Za-z0-9]/.test(key)) {
    return key.toUpperCase();
  }
  return null;
}

/**
 * キーダウンイベントからホットキー文字列を組み立てる。
 * Escape はキャンセル。英数字・Space は修飾必須、F1–F11 と名前付きキーは単体可。
 * PrintScreen / F12 は非対応。
 */
export function normalizeHotkeyFromEvent(
  event: Pick<KeyboardEvent, "key" | "code" | "ctrlKey" | "altKey" | "shiftKey" | "metaKey">,
): NormalizeHotkeyResult {
  if (event.key === "Escape") {
    return { ok: false, reason: "cancel" };
  }

  const modifiers: string[] = [];
  if (event.ctrlKey) modifiers.push("Ctrl");
  if (event.altKey) modifiers.push("Alt");
  if (event.shiftKey) modifiers.push("Shift");
  if (event.metaKey) modifiers.push("Win");

  if (
    event.key === "Control" ||
    event.key === "Alt" ||
    event.key === "Shift" ||
    event.key === "Meta"
  ) {
    return { ok: false, reason: "modifier-only" };
  }

  const mainKey = resolveMainKey(event);
  if (!mainKey) {
    return { ok: false, reason: "unsupported" };
  }

  if (modifiers.length === 0 && !BARE_ALLOWED.has(mainKey)) {
    return { ok: false, reason: "need-modifier" };
  }

  return { ok: true, combo: [...modifiers, mainKey].join("+") };
}

export function normalizeHotkeyFailureMessage(reason: NormalizeHotkeyFailureReason): string | null {
  switch (reason) {
    case "need-modifier":
      return "英数字や Space には Ctrl / Alt / Shift / Win のいずれかを付けてください";
    case "unsupported":
      return "このキーはホットキーに使えません（PrintScreen / F12 は非対応）";
    default:
      return null;
  }
}

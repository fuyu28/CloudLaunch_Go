import { describe, expect, it } from "vitest";

import { normalizeHotkeyFailureMessage, normalizeHotkeyFromEvent } from "../hotkeyNormalize";

function evt(
  partial: Partial<
    Pick<KeyboardEvent, "key" | "code" | "ctrlKey" | "altKey" | "shiftKey" | "metaKey">
  >,
): Pick<KeyboardEvent, "key" | "code" | "ctrlKey" | "altKey" | "shiftKey" | "metaKey"> {
  return {
    key: "",
    code: "",
    ctrlKey: false,
    altKey: false,
    shiftKey: false,
    metaKey: false,
    ...partial,
  };
}

describe("normalizeHotkeyFromEvent", () => {
  it("accepts modifier + letter via event.code", () => {
    expect(
      normalizeHotkeyFromEvent(evt({ key: "s", code: "KeyS", ctrlKey: true, altKey: true })),
    ).toEqual({
      ok: true,
      combo: "Ctrl+Alt+S",
    });
  });

  it("allows bare PrintScreen and F8", () => {
    expect(normalizeHotkeyFromEvent(evt({ key: "PrintScreen", code: "PrintScreen" }))).toEqual({
      ok: true,
      combo: "PrintScreen",
    });
    expect(normalizeHotkeyFromEvent(evt({ key: "F8", code: "F8" }))).toEqual({
      ok: true,
      combo: "F8",
    });
  });

  it("allows Shift+F8", () => {
    expect(normalizeHotkeyFromEvent(evt({ key: "F8", code: "F8", shiftKey: true }))).toEqual({
      ok: true,
      combo: "Shift+F8",
    });
  });

  it("requires modifier for letters and Space", () => {
    expect(normalizeHotkeyFromEvent(evt({ key: "a", code: "KeyA" }))).toEqual({
      ok: false,
      reason: "need-modifier",
    });
    expect(normalizeHotkeyFromEvent(evt({ key: " ", code: "Space" }))).toEqual({
      ok: false,
      reason: "need-modifier",
    });
    expect(normalizeHotkeyFromEvent(evt({ key: " ", code: "Space", ctrlKey: true }))).toEqual({
      ok: true,
      combo: "Ctrl+Space",
    });
  });

  it("rejects unsupported and modifier-only", () => {
    expect(
      normalizeHotkeyFromEvent(evt({ key: "ArrowUp", code: "ArrowUp", ctrlKey: true })),
    ).toEqual({
      ok: false,
      reason: "unsupported",
    });
    expect(
      normalizeHotkeyFromEvent(evt({ key: "Control", code: "ControlLeft", ctrlKey: true })),
    ).toEqual({
      ok: false,
      reason: "modifier-only",
    });
  });

  it("cancels on Escape", () => {
    expect(normalizeHotkeyFromEvent(evt({ key: "Escape", code: "Escape" }))).toEqual({
      ok: false,
      reason: "cancel",
    });
  });

  it("maps meta to Win", () => {
    expect(normalizeHotkeyFromEvent(evt({ key: "a", code: "KeyA", metaKey: true }))).toEqual({
      ok: true,
      combo: "Win+A",
    });
  });
});

describe("normalizeHotkeyFailureMessage", () => {
  it("returns user-facing hints", () => {
    expect(normalizeHotkeyFailureMessage("need-modifier")).toMatch(/Ctrl/);
    expect(normalizeHotkeyFailureMessage("unsupported")).toMatch(/使えません/);
    expect(normalizeHotkeyFailureMessage("cancel")).toBeNull();
  });
});

import { describe, it, expect } from "vitest";
import { renderHook } from "@testing-library/react";

import { useLatestRequestId } from "../useLatestRequestId";

describe("useLatestRequestId", () => {
  it("returns a stable object identity across re-renders", () => {
    const { result, rerender } = renderHook(() => useLatestRequestId());
    const first = result.current;
    rerender();
    expect(result.current).toBe(first);
    expect(result.current.reset).toBe(first.reset);
  });

  it("tracks latest request ids", () => {
    const { result } = renderHook(() => useLatestRequestId());
    const a = result.current.next();
    const b = result.current.next();
    expect(result.current.isLatest(a)).toBe(false);
    expect(result.current.isLatest(b)).toBe(true);
    result.current.reset();
    expect(result.current.isLatest(b)).toBe(false);
  });
});

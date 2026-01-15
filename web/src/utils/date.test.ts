import { describe, expect, it, vi } from "vitest";
import {
  formatDateDisplay,
  formatTimeDisplay,
  todayString,
  toLocalDateString
} from "./date";

describe("date utils", () => {
  it("formats local date string with zero padding", () => {
    const date = new Date(2024, 0, 2);
    expect(toLocalDateString(date)).toBe("2024-01-02");
  });

  it("returns today string from current system time", () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date(2024, 0, 2, 12, 0, 0));
    expect(todayString()).toBe("2024-01-02");
    vi.useRealTimers();
  });

  it("keeps raw value for invalid date display input", () => {
    expect(formatDateDisplay("not-a-date")).toBe("not-a-date");
  });

  it("formats date display using locale options", () => {
    const value = "2024-01-02";
    const parsed = new Date(`${value}T00:00:00`);
    const expected = new Intl.DateTimeFormat("zh-CN", {
      year: "numeric",
      month: "2-digit",
      day: "2-digit",
      weekday: "short"
    }).format(parsed);
    expect(formatDateDisplay(value)).toBe(expected);
  });

  it("returns placeholder for empty time display input", () => {
    expect(formatTimeDisplay(null)).toBe("—");
  });

  it("keeps raw value for invalid time display input", () => {
    expect(formatTimeDisplay("not-a-time")).toBe("not-a-time");
  });

  it("formats time display using locale options", () => {
    const parsed = new Date("2024-01-02T03:04:00.000Z");
    const expected = new Intl.DateTimeFormat("zh-CN", {
      hour: "2-digit",
      minute: "2-digit",
      month: "2-digit",
      day: "2-digit"
    }).format(parsed);
    expect(formatTimeDisplay(parsed.toISOString())).toBe(expected);
  });
});

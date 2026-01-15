import { describe, expect, it } from "vitest";
import { buildMonthCalendar } from "./calendar";
import { toLocalDateString } from "./date";

describe("calendar utils", () => {
  it("builds a month grid with correct day placement", () => {
    const baseDate = new Date(2024, 0, 15);
    const weeks = buildMonthCalendar(baseDate);
    const flattened = weeks.flat();

    for (const week of weeks) {
      expect(week).toHaveLength(7);
    }

    const days = flattened.filter((day) => day.dayNumber !== null);
    expect(days).toHaveLength(31);

    const first = flattened.find((day) => day.dayNumber !== null);
    expect(first).toBeDefined();
    expect(first?.dayNumber).toBe(1);
    expect(flattened.indexOf(first!)).toBe(new Date(2024, 0, 1).getDay());

    const last = [...flattened].reverse().find((day) => day.dayNumber !== null);
    expect(last).toBeDefined();
    expect(last?.dayNumber).toBe(31);
    expect(flattened.lastIndexOf(last!)).toBe(new Date(2024, 0, 31).getDay() + (weeks.length - 1) * 7);

    const dayTen = flattened.find((day) => day.dayNumber === 10);
    expect(dayTen).toBeDefined();
    expect(dayTen?.date).toBe(toLocalDateString(new Date(2024, 0, 10)));
  });
});

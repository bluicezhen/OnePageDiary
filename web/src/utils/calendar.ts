import { toLocalDateString } from "./date";

export interface CalendarDay {
  date: string | null;
  dayNumber: number | null;
}

export function buildMonthCalendar(baseDate: Date): CalendarDay[][] {
  const year = baseDate.getFullYear();
  const month = baseDate.getMonth();
  const firstDay = new Date(year, month, 1);
  const lastDay = new Date(year, month + 1, 0);

  const startWeekday = firstDay.getDay();
  const totalDays = lastDay.getDate();

  const weeks: CalendarDay[][] = [];
  let currentDay = 1 - startWeekday;

  while (currentDay <= totalDays) {
    const week: CalendarDay[] = [];
    for (let i = 0; i < 7; i += 1) {
      if (currentDay < 1 || currentDay > totalDays) {
        week.push({ date: null, dayNumber: null });
      } else {
        const date = new Date(year, month, currentDay);
        week.push({
          date: toLocalDateString(date),
          dayNumber: currentDay
        });
      }
      currentDay += 1;
    }
    weeks.push(week);
  }

  return weeks;
}

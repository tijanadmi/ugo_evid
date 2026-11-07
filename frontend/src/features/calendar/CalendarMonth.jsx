import { months, weekdays, iso } from "./calendarUtils";

export default function CalendarMonth({
  year,
  month,
  today,
  byDate,
  selectedDate,
  canWrite,
  onToggle,
}) {
  const first = (new Date(Date.UTC(year, month - 1, 1)).getUTCDay() + 6) % 7;
  const days = new Date(Date.UTC(year, month, 0)).getUTCDate();
  return (
    <>
      <div className="calendar-weekdays" aria-hidden="true">
        {weekdays.map((d, i) => (
          <span className={i > 4 ? "weekend" : ""} key={d}>
            {d}
          </span>
        ))}
      </div>
      <div
        className="calendar-month"
        aria-label={`${months[month - 1]} ${year}`}
      >
        {Array.from({ length: 42 }, (_, index) => {
          const n = index - first + 1;
          if (n < 1 || n > days)
            return (
              <div key={index} className="calendar-empty" aria-hidden="true" />
            );
          const key = iso(year, month, n),
            day = byDate.get(key),
            working = day?.radni === "D",
            isToday = key === iso(today.year, today.month, today.day);
          return (
            <button
              key={index}
              type="button"
              className={`calendar-day ${day ? (working ? "working" : "nonworking") : "missing"}${key === selectedDate ? " selected" : ""}${isToday ? " today" : ""}`}
              disabled={!canWrite || !day}
              aria-label={`${key}, ${day ? (working ? "radni dan" : "neradni dan") : "nije generisan"}${day?.opis ? ", " + day.opis : ""}`}
              aria-pressed={!!day && !working}
              aria-current={isToday ? "date" : undefined}
              title={
                day
                  ? `${day.opis || key} · ${day.potvrdjen === "D" ? "Potvrđen" : "Čeka potvrdu"} · Klik za promenu statusa`
                  : "Prvo generišite kalendar"
              }
              onClick={() => onToggle(day)}
            >
              <span className="calendar-day-top">
                <strong>{n}</strong>
                {day?.potvrdjen === "N" && (
                  <span
                    className="calendar-pending"
                    aria-label="Čeka potvrdu"
                  />
                )}
              </span>
              <span className="calendar-day-state">
                {day ? (working ? "Radni" : "Neradni") : "Nije generisan"}
              </span>
              <span className="calendar-day-description">{day?.opis}</span>
              {day?.izvor === "RUCNO" && (
                <span className="calendar-day-source">Ručna izmena</span>
              )}
            </button>
          );
        })}
      </div>
    </>
  );
}

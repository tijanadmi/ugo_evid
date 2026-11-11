import { usePermissions } from "../context/usePermissions";
import CalendarMonth from "../features/calendar/CalendarMonth";
import HolidayImportDialog from "../features/calendar/HolidayImportDialog";
import CalendarDaysEditor from "../features/calendar/CalendarDaysEditor";
import { months, iso, localToday } from "../features/calendar/calendarUtils";
import { useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import toast from "react-hot-toast";
import { useAuth } from "../context/useAuth";
import ConfirmAction from "../ui/ConfirmAction";
import { announceBusinessChange } from "../services/DataSync";
import "./WorkCalendar.css";

export default function WorkCalendar() {
  const { api, user } = useAuth(),
    client = useQueryClient();
  const [params, setParams] = useSearchParams();
  const today = localToday();
  const selectedYear = Number(params.get("godina")),
    selectedMonth = Number(params.get("mesec"));
  const year =
    Number.isInteger(selectedYear) &&
    selectedYear >= 1900 &&
    selectedYear <= 2200
      ? selectedYear
      : today.year;
  const month =
    Number.isInteger(selectedMonth) && selectedMonth >= 1 && selectedMonth <= 12
      ? selectedMonth
      : params.has("godina")
        ? 1
        : today.month;
  const [draft, setDraft] = useState(null),
    [action, setAction] = useState(null),
    [busy, setBusy] = useState(false),
    [blocked, setBlocked] = useState(false),
    [preview, setPreview] = useState(null),
    [cancelDraft, setCancelDraft] = useState(false),
    [selection, setSelection] = useState("");
  const pending = useRef(false),
    queryKey = ["work-calendar", user.username, year];
  const query = useQuery({
    queryKey,
    queryFn: ({ signal }) => api(`/radni_kalendar/${year}`, { signal }),
    retry: false,
    refetchOnWindowFocus: true,
  });
  const items = query.data?.items || [],
    byDate = new Map(items.map((d) => [d.datum, d])),
    total = (Date.UTC(year + 1, 0, 1) - Date.UTC(year, 0, 1)) / 86400000,
    approved = items.filter((d) => d.potvrdjen === "D").length;
  const permissions = usePermissions();
  const canWrite =
      permissions.calendarAdmin && !!query.data && !query.error && !busy,
    hasDialog = !!draft || !!action || !!preview;
  const selectedDate = selection.startsWith(iso(year, month, 1).slice(0, 7))
    ? selection
    : iso(year, month, 1);
  const selectedDay = byDate.get(selectedDate);
  function navigate(y, m) {
    if (y < 1900 || y > 2200 || pending.current || hasDialog) return;
    setParams({ godina: String(y), mesec: String(m) });
    setBlocked(false);
  }
  function shift(delta) {
    const d = new Date(Date.UTC(year, month - 1 + delta, 1));
    navigate(d.getUTCFullYear(), d.getUTCMonth() + 1);
  }
  function edit(day) {
    setBlocked(false);
    setDraft({
      action: "change",
      version: query.data.version,
      od: day?.datum || selectedDate,
      do: day?.datum || selectedDate,
      radni: day?.radni || "N",
      opis: day?.opis || "",
    });
  }
  async function submit(change, optimisticDay) {
    if (pending.current) return;
    pending.current = true;
    setBusy(true);
    const before = query.data;
    // Cancel an in-flight focus refresh before presenting the optimistic color.
    await client.cancelQueries({ queryKey });
    if (optimisticDay)
      client.setQueryData(queryKey, {
        ...before,
        items: before.items.map((d) =>
          d.datum === optimisticDay
            ? { ...d, radni: change.radni, potvrdjen: "N", izvor: "RUCNO" }
            : d,
        ),
      });
    try {
      await api(`/radni_kalendar/${year}`, { method: "POST", body: change });
      setDraft(null);
      setAction(null);
      setPreview(null);
      toast.success(
        change.action === "import"
          ? "Praznici su uvezeni. Proverite i potvrdite godinu."
          : "Kalendar je sačuvan.",
        { id: "calendar-save" },
      );
      announceBusinessChange(client);
      await query.refetch();
    } catch (e) {
      if (optimisticDay) client.setQueryData(queryKey, before);
      toast.error(e.message);
      if (e.status === 409) {
        setBlocked(true);
        setAction(null);
        setPreview(null);
        await query.refetch();
      }
    } finally {
      pending.current = false;
      setBusy(false);
    }
  }
  function toggle(day) {
    if (!canWrite || pending.current || !day) return;
    setSelection(day.datum);
    submit(
      {
        action: "change",
        version: query.data.version,
        od: day.datum,
        do: day.datum,
        radni: day.radni === "D" ? "N" : "D",
        opis: day.opis || "",
      },
      day.datum,
    );
  }
  async function download() {
    if (!canWrite || pending.current) return;
    pending.current = true;
    setBusy(true);
    const snapshot = query.data;
    try {
      const result = await api(`/radni_kalendar/${year}/praznici`);
      setPreview({
        version: snapshot.version,
        rows: result.items.map((h) => {
          const d = snapshot.items.find((x) => x.datum === h.datum);
          const protectedDay = d?.izvor === "RUCNO";
          return { ...h, protectedDay, selected: !protectedDay };
        }),
      });
    } catch (e) {
      toast.error(e.message);
    } finally {
      pending.current = false;
      setBusy(false);
    }
  }
  return (
    <section
      className="contract-card work-calendar"
      aria-label="Radni kalendar"
    >
      <header className="calendar-heading">
        <div>
          <h1>Radni kalendar</h1>
          <p>Klik na dan menja radni status i čuva izmenu.</p>
        </div>
        <label>
          Godina
          <input
            aria-label="Godina"
            type="number"
            min="1900"
            max="2200"
            value={year}
            disabled={busy || hasDialog}
            onChange={(e) => {
              const value = Number(e.target.value);
              if (value >= 1900 && value <= 2200) navigate(value, month);
            }}
          />
        </label>
      </header>
      <div className="calendar-tools">
        <button
          className="button"
          disabled={!canWrite}
          onClick={() =>
            submit({ action: "generate", version: query.data.version })
          }
        >
          Generiši kalendar
        </button>
        <button
          className="button"
          disabled={!canWrite || items.length !== total}
          onClick={download}
        >
          Preuzmi praznike za Srbiju
        </button>
        <button
          className="button"
          disabled={!canWrite || !items.length}
          onClick={() => edit(selectedDay)}
        >
          Izmeni dane
        </button>
        <button
          className="button primary"
          disabled={!canWrite || items.length !== total || approved === total}
          onClick={() =>
            setAction({ action: "approve", version: query.data.version })
          }
        >
          Potvrdi godinu
        </button>
      </div>
      <div className="calendar-month-nav">
        <button
          className="button"
          aria-label="Prethodni mesec"
          disabled={busy || hasDialog || (year === 1900 && month === 1)}
          onClick={() => shift(-1)}
        >
          ←
        </button>
        <label>
          <span className="sr-only">Mesec</span>
          <select
            aria-label="Mesec"
            disabled={busy || hasDialog}
            value={month}
            onChange={(e) => navigate(year, Number(e.target.value))}
          >
            {months.map((m, i) => (
              <option key={m} value={i + 1}>
                {m}
              </option>
            ))}
          </select>
        </label>
        <h2>{year}.</h2>
        <button
          className="button"
          aria-label="Sledeći mesec"
          disabled={busy || hasDialog || (year === 2200 && month === 12)}
          onClick={() => shift(1)}
        >
          →
        </button>
        <button
          className="button calendar-today"
          disabled={busy || hasDialog}
          onClick={() => navigate(today.year, today.month)}
        >
          Danas
        </button>
      </div>
      <div className="calendar-legend">
        <span>
          <i className="working" />
          Radni dan
        </span>
        <span>
          <i className="nonworking" />
          Neradni dan / vikend
        </span>
        <span>
          <i className="unapproved" />
          Čeka potvrdu
        </span>
      </div>
      {query.isPending ? (
        <p role="status">Učitavanje kalendara...</p>
      ) : query.error ? (
        <p role="alert">
          Kalendar nije dostupan.{" "}
          <button className="button" onClick={() => query.refetch()}>
            Osveži
          </button>
        </p>
      ) : (
        <>
          <p className="calendar-status" role="status">
            Generisano {items.length} od {total} dana. Potvrđeno {approved} od{" "}
            {total} dana.
            {busy
              ? " Čuvanje / preuzimanje…"
              : items.length < total
                ? " Generišite godinu da biste uređivali sve dane."
                : ""}
          </p>
          <CalendarMonth
            year={year}
            month={month}
            today={today}
            byDate={byDate}
            selectedDate={selectedDate}
            canWrite={canWrite}
            onToggle={toggle}
          />
          <footer className="calendar-footer">
            <span>
              <strong>{selectedDate}</strong> ·{" "}
              {selectedDay?.opis || "Bez opisa"}
            </span>
            <button
              className="button"
              disabled={!canWrite || !selectedDay}
              onClick={() => edit(selectedDay)}
            >
              Opis / korekcija
            </button>
          </footer>
        </>
      )}
      {action && (
        <ConfirmAction
          modal
          title={`Potvrditi kalendar za ${year}. godinu?`}
          confirmLabel="Potvrdi"
          disabled={busy}
          onCancel={() => setAction(null)}
          onConfirm={() => submit(action)}
        >
          <p>
            Potvrđujem da su provereni praznici i izuzetno radni dani. Sačuvani
            rokovi zahteva ostaju nepromenjeni.
          </p>
        </ConfirmAction>
      )}
      <HolidayImportDialog
        preview={preview}
        setPreview={setPreview}
        year={year}
        busy={busy}
        onSubmit={submit}
      />
      <CalendarDaysEditor
        draft={draft}
        setDraft={setDraft}
        year={year}
        busy={busy}
        blocked={blocked}
        cancelDraft={cancelDraft}
        setCancelDraft={setCancelDraft}
        onSubmit={submit}
      />
    </section>
  );
}

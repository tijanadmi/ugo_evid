import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../../context/useAuth";
import { formatDate } from "../../utils/contractFormatting";

export default function RequestDeadline({
  contractID,
  row,
  original,
  reminderDays,
}) {
  const { api, user } = useAuth();
  const saved =
    original?.rok &&
    original.datum_zahteva === row.datum_zahteva &&
    original.rok_radnih_dana === row.rok_radnih_dana;
  const valid =
    /^\d{4}-\d{2}-\d{2}$/.test(row.datum_zahteva || "") &&
    Number.isInteger(row.rok_radnih_dana) &&
    row.rok_radnih_dana > 0 &&
    row.rok_radnih_dana <= 3660;
  const query = useQuery({
    queryKey: [
      "request-deadline",
      user.username,
      contractID,
      row.datum_zahteva,
      row.rok_radnih_dana,
    ],
    queryFn: ({ signal }) =>
      api(
        `/ugo_evid/${contractID}/pracenje/rok?${new URLSearchParams({ datum_zahteva: row.datum_zahteva, rok_radnih_dana: row.rok_radnih_dana })}`,
        { signal },
      ),
    enabled: !!contractID && valid && !saved,
    retry: false,
  });
  const deadline = saved ? original.rok : query.data?.rok;
  const days = Number(reminderDays);
  let reminder = "";
  if (deadline && Number.isInteger(days) && days >= 0) {
    const date = new Date(`${deadline}T00:00:00Z`);
    date.setUTCDate(date.getUTCDate() - days);
    reminder = date.toISOString().slice(0, 10);
  }
  return (
    <div aria-live="polite">
      <p>
        <strong>
          Krajnji rok realizacije:{" "}
          {deadline
            ? formatDate(deadline)
            : valid
              ? "Obračun…"
              : "Unesite datum zahteva i rok u radnim danima."}
        </strong>
      </p>
      {!saved && query.error && (
        <p role="alert">
          {query.error.message}{" "}
          <button
            type="button"
            className="button"
            onClick={() => query.refetch()}
          >
            Ponovi obračun
          </button>
        </p>
      )}
      {reminder && (
        <p>
          Podsetnik od {formatDate(reminder)} — {days} kalendarskih dana pre
          roka, za sačuvan zahtev koji nije nacrt, otkazan ili realizovan. Broj
          dana podešava se u podacima o održavanju.
        </p>
      )}
      <p>
        Rok se računa od narednog dana prema potvrđenom radnom kalendaru.
        Stvarni datum realizacije unesite kada usluga bude izvršena. Zapisnik
        može biti potpisan nakon roka.
      </p>
    </div>
  );
}

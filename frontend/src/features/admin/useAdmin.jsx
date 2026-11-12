import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../../context/useAuth";

export function useAdminQuery(key, path, enabled = true) {
  const { api, user } = useAuth();
  return useQuery({
    queryKey: ["admin", user?.username, ...key],
    queryFn: ({ signal }) => api(path, { signal }),
    enabled: !!user && enabled,
    retry: false,
    refetchOnWindowFocus: true,
  });
}

export function AdminPagination({ page, total, onChange, busy }) {
  const pages = Math.max(1, Math.ceil(total / 20));
  return (
    <nav className="admin-pagination" aria-label="Stranice administracije">
      <span>
        Stranica {page} od {pages} · Ukupno {total}
      </span>
      <button
        className="button"
        disabled={busy || page === 1}
        onClick={() => onChange(page - 1)}
      >
        Prethodna
      </button>
      <button
        className="button"
        disabled={busy || page >= pages}
        onClick={() => onChange(page + 1)}
      >
        Sledeća
      </button>
    </nav>
  );
}

export function adminDate(value) {
  if (!value) return "—";
  const [date, time] = value.split("T");
  return `${date.split("-").reverse().join(".")} ${time || ""}`.trim();
}

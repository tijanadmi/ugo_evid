import { Link } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../../context/useAuth";
import { usePermissions } from "../../context/usePermissions";

export default function AdminLink() {
  const { api, user } = useAuth();
  const permissions = usePermissions();
  const query = useQuery({
    queryKey: ["admin", user?.username, "count"],
    queryFn: ({ signal }) => api("/admin/obavestenja?page_size=1", { signal }),
    enabled: !!permissions.admin,
    refetchInterval: 30000,
    retry: false,
  });
  if (!permissions.admin) return null;
  return (
    <Link
      className="button"
      to="/administracija/obavestenja"
      title="Administracija i otvorena obaveštenja"
    >
      Administracija{query.data?.total > 0 ? ` (${query.data.total})` : ""}
    </Link>
  );
}

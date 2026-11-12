import { useQuery } from "@tanstack/react-query";
import { useAuth } from "./useAuth";

export function usePermissions() {
  const { api, user } = useAuth();
  const query = useQuery({
    queryKey: ["permissions", user?.username],
    queryFn: ({ signal }) => api("/users/prava", { signal }),
    enabled: !!user,
    staleTime: 15000,
    refetchInterval: 30000,
    refetchOnWindowFocus: "always",
    retry: false,
  });
  // Fail closed if rights cannot be refreshed; the server checks every action too.
  const data = query.error ? null : query.data;
  const organizations = data?.organizations || [];
  return {
    ...query,
    organizations,
    writableOrganizations: organizations.filter((org) => org.can_write),
    canWrite: (orgID) =>
      organizations.some((org) => org.id === Number(orgID) && org.can_write),
    calendarAdmin: data?.calendar_admin === true,
    admin: data?.admin === true,
  };
}

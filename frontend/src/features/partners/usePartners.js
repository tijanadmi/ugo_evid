import { useQuery } from "@tanstack/react-query";
import { useAuth } from "../../context/useAuth";
import { getMyPartners } from "../../services/apiPartners";

export function usePartners(page, pageSize, naziv, org = 0) {
  const { user, api } = useAuth();
  return useQuery({
    queryKey: ["partners", user?.username, page, pageSize, naziv, org],
    queryFn: ({ signal }) =>
      getMyPartners(api, page, pageSize, naziv, signal, org),
    enabled: !!user,
    refetchOnWindowFocus: true,
    refetchInterval: 30000,
  });
}

import { useQuery } from '@tanstack/react-query';
import { useAuth } from '../../context/AuthContext';
import { getOrganizations } from '../../services/apiOrganizations';

export function useOrganizations() {
  const { api, user } = useAuth();
  return useQuery({
    queryKey: ['organizations', user?.username],
    queryFn: ({ signal }) => getOrganizations(api, signal),
    staleTime: 5 * 60 * 1000,
    enabled: !!user,
  });
}

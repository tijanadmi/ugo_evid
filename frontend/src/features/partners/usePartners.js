import { useQuery } from '@tanstack/react-query';
import { useAuth } from '../../context/AuthContext';
import { getMyPartners } from '../../services/apiPartners';

export function usePartners(page, pageSize) {
  const { user, api } = useAuth();
  return useQuery({
    queryKey: ['partners', user?.username, page, pageSize],
    queryFn: ({ signal }) => getMyPartners(api, page, pageSize, signal),
    enabled: !!user,
  });
}

import { useQuery } from '@tanstack/react-query';
import { useAuth } from '../../context/AuthContext';
import { getContracts } from '../../services/apiContracts';

export function useContracts({ status, page, pageSize, orgID }) {
  const { api, user } = useAuth();
  return useQuery({
    queryKey: ['contracts', user?.username, status, orgID, page, pageSize],
    queryFn: ({ signal }) => getContracts(api, { status, page, pageSize, orgID, signal }),
    enabled: !!user,
  });
}

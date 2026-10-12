import { useQuery } from '@tanstack/react-query';
import { useAuth } from '../../context/useAuth';
import { getContracts } from '../../services/apiContracts';

export function useContracts({ status, page, pageSize, orgID, supplier }) {
  const { api, user } = useAuth();
  return useQuery({
    queryKey: ['contracts', user?.username, status, orgID, supplier, page, pageSize],
    queryFn: ({ signal }) => getContracts(api, { status, page, pageSize, orgID, supplier, signal }),
    enabled: !!user,
  });
}

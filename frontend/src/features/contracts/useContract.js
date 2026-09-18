import { useQuery } from '@tanstack/react-query';
import { useAuth } from '../../context/AuthContext';
import { getContract } from '../../services/apiContracts';

export function useContract(id) {
  const { api, user } = useAuth();
  return useQuery({
    queryKey: ['contract', user?.username, id],
    queryFn: ({ signal }) => getContract(api, id, signal),
    enabled: !!user && Number.isSafeInteger(id) && id > 0,
  });
}

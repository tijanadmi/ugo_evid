import toast from 'react-hot-toast';
import { QueryClient, QueryCache } from '@tanstack/react-query';

export const queryClient = new QueryClient({
  queryCache: new QueryCache({ onError: error => toast.error(error.message, { id: 'operation-error' }) }),
  defaultOptions: {
    queries: {
      staleTime: 0,
      retry: false,
      refetchOnWindowFocus: false,
    },
    mutations: { retry: false },
  },
});

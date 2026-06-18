import { useEffect, type ReactElement } from 'react';
import { useAuthStore } from '@/stores';

export function ProtectedRoute({ children }: { children: ReactElement }) {
  const isAuthenticated = useAuthStore((state) => state.isAuthenticated);
  const managementKey = useAuthStore((state) => state.managementKey);
  const apiBase = useAuthStore((state) => state.apiBase);
  const restoreSession = useAuthStore((state) => state.restoreSession);

  useEffect(() => {
    if (!isAuthenticated && managementKey && apiBase) {
      restoreSession({
        expectedMode: 'external_panel',
        expectedPanelBase: apiBase,
      }).catch(() => {});
    }
  }, [apiBase, isAuthenticated, managementKey, restoreSession]);

  return children;
}
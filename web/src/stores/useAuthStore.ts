import { create } from 'zustand';

interface AuthState {
  isAuthenticated: boolean;
  managementKey: string;
  apiBase: string;
  connectionStatus: string;
  serverVersion: string;
  serverBuildDate: string;
  supportsPlugin: boolean;
  restoreSession: (opts?: any) => Promise<any>;
  logout: () => void;
}

export const useAuthStore = create<AuthState>(() => ({
  isAuthenticated: true,
  managementKey: '',
  apiBase: '',
  connectionStatus: 'connected',
  serverVersion: '',
  serverBuildDate: '',
  supportsPlugin: false,
  restoreSession: async () => ({}),
  logout: () => {},
}));

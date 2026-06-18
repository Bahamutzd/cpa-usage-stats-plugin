import { create } from 'zustand';
export const useConfigStore = create<any>(() => ({ config: null, fetchConfig: async () => {}, clearCache: () => {} }));

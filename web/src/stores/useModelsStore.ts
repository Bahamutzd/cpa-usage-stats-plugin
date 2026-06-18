import { create } from 'zustand';
export const useModelsStore = create<any>(() => ({ models: [], loading: false, fetchModels: async () => {} }));

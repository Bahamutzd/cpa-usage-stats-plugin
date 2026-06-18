import { apiClient } from './client';
type ApiKeyUsageResponse = any;

const API_KEY_USAGE_TIMEOUT_MS = 30 * 1000;

export const apiKeyUsageApi = {
  getUsage: () =>
    apiClient.get<ApiKeyUsageResponse>('/api-key-usage', {
      timeout: API_KEY_USAGE_TIMEOUT_MS,
    }),
};

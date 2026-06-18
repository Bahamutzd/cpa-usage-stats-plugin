import axios, { AxiosInstance, AxiosRequestConfig, AxiosResponse } from 'axios';

const REQUEST_TIMEOUT_MS = 30000;

class ApiClient {
  private instance: AxiosInstance;
  private apiBase: string = '';
  private managementKey: string = '';

  constructor() {
    this.instance = axios.create({ timeout: REQUEST_TIMEOUT_MS, headers: { 'Content-Type': 'application/json' } });
    this.instance.interceptors.request.use((config) => {
      config.baseURL = this.apiBase;
      if (this.managementKey) config.headers.Authorization = `Bearer ${this.managementKey}`;
      return config;
    });
  }

  setConfig(config: { apiBase: string; managementKey: string; timeout?: number }) {
    this.apiBase = (config.apiBase || '').replace(/\/+$/, '');
    this.managementKey = config.managementKey || '';
    if (config.timeout) this.instance.defaults.timeout = config.timeout;
  }

  async get<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T> { return (await this.instance.get<T>(url, config)).data; }
  async post<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> { return (await this.instance.post<T>(url, data, config)).data; }
  async put<T = unknown>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<T> { return (await this.instance.put<T>(url, data, config)).data; }
  async delete<T = unknown>(url: string, config?: AxiosRequestConfig): Promise<T> { return (await this.instance.delete<T>(url, config)).data; }
}

export const apiClient = new ApiClient();

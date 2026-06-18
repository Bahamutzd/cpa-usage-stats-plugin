const LS = { base: "cpaust.base", key: "cpaust.key" };

export function getConfig() {
  return {
    base: localStorage.getItem(LS.base) || "",
    key: localStorage.getItem(LS.key) || "",
  };
}

export function saveConfig(base: string, key: string) {
  localStorage.setItem(LS.base, base);
  localStorage.setItem(LS.key, key);
}

export async function api(path: string, opts: RequestInit = {}) {
  const cfg = getConfig();
  const base = cfg.base.replace(/\/+$/, "");
  const headers: Record<string, string> = {
    Authorization: "Bearer " + cfg.key,
    ...(opts.headers as Record<string, string>),
  };
  if (opts.body && typeof opts.body === "string") {
    headers["Content-Type"] = "application/json";
  }
  const res = await fetch(base + path, { ...opts, headers });
  const ct = res.headers.get("content-type") || "";
  if (res.status === 401) throw new Error("需要 Management Key（401）");
  if (!res.ok) {
    const txt = ct.includes("json")
      ? (await res.json().catch(() => ({}))).error
      : await res.text().catch(() => "");
    throw new Error(String(res.status) + " " + (txt || res.statusText));
  }
  if (ct.includes("application/x-ndjson") || ct.includes("octet-stream"))
    return await res.blob();
  return ct.includes("json") ? res.json() : res.text();
}

export function dayStartMS() {
  const d = new Date();
  d.setHours(0, 0, 0, 0);
  return d.getTime();
}
export const nowMS = () => Date.now();
export const h24 = () => nowMS() - 24 * 3600 * 1000;

export function fmt(n: number | null | undefined): string {
  return Number(n || 0).toLocaleString();
}
export function pct(n: number | null | undefined): string {
  if (n == null || isNaN(n)) return "—";
  return (n * 100).toFixed(1) + "%";
}
export function dur(n: number | null | undefined): string {
  if (n == null) return "—";
  return Math.round(n) + "ms";
}
export function esc(s: string): string {
  return String(s ?? "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;");
}

// Types
export interface DashboardSummary {
  generated_at_ms: number;
  window: { today_start_ms: number; now_ms: number; rolling_30m_start_ms: number };
  today: TodaySummary;
  rolling_30m: RollingSummary;
  top_models_today: TopModel[];
  recent_failures: RecentFailure[];
  token_mix?: TokenMixSegment[];
  channel_health?: ChannelHealth[];
  traffic_timeline?: TrafficPoint[];
  hourly_activity?: HourlyActivityPoint[];
}
export interface TodaySummary {
  total_calls: number; success_calls: number; failure_calls: number;
  success_rate: number; input_tokens: number; output_tokens: number;
  cached_tokens: number; cache_read_tokens: number; cache_creation_tokens: number;
  reasoning_tokens: number; total_tokens: number; total_cost: number;
  average_latency_ms: number | null; zero_token_calls: number;
}
export interface RollingSummary { rpm: number; tpm: number; total_calls: number; total_tokens: number; }
export interface TopModel { model: string; calls: number; tokens: number; success_rate: number; }
export interface RecentFailure { timestamp_ms: number; model: string; source?: string; source_hash?: string; fail_status_code?: number; fail_summary?: string; }
export interface TokenMixSegment { key: string; tokens: number; share: number; }
export interface ChannelHealth { auth_index: string; calls: number; failures: number; tokens: number; average_latency_ms: number | null; auth_label_snapshot?: string; }
export interface TrafficPoint { bucket_ms: number; calls: number; tokens: number; success: number; failure: number; }
export interface HourlyActivityPoint { bucket_ms: number; calls: number; tokens: number; intensity: number; }

export interface MonitoringAnalytics {
  generated_at_ms: number;
  summary?: MonitoringSummary;
  model_stats?: ModelStat[];
  timeline?: TimelineRow[];
  hourly_distribution?: HourlyRow[];
  channel_share?: ChannelShareRow[];
  failure_sources?: FailureSourceRow[];
  recent_failures?: MonitoringRecentFailure[];
}
export interface MonitoringSummary {
  total_calls: number; success_calls: number; failure_calls: number;
  success_rate: number; input_tokens: number; output_tokens: number;
  cached_tokens: number; cache_read_tokens: number; cache_creation_tokens: number;
  reasoning_tokens: number; total_tokens: number; total_cost: number;
  average_latency_ms: number | null; zero_token_calls: number;
  rpm_30m: number; tpm_30m: number;
}
export interface ModelStat {
  model: string; calls: number; success_calls: number; failure_calls: number;
  success_rate: number; total_tokens: number;
}
export interface TimelineRow { bucket_ms: number; label: string; calls: number; tokens: number; success: number; failure: number; }
export interface HourlyRow { hour: number; calls: number; tokens: number; }
export interface ChannelShareRow { auth_index: string; calls: number; failure: number; tokens: number; average_latency_ms: number | null; auth_label_snapshot?: string; }
export interface FailureSourceRow { source_hash: string; auth_index: string; calls: number; failure: number; source?: string; }
export interface MonitoringRecentFailure { timestamp_ms: number; model: string; source?: string; source_hash?: string; fail_status_code?: number; fail_summary?: string; }

export interface UsageEvent {
  event_hash: string; timestamp_ms: number; timestamp: string;
  model: string; auth_index: string; source: string;
  input_tokens: number; output_tokens: number; cache_read_tokens: number;
  total_tokens: number; latency_ms: number | null; failed: boolean;
  fail_status_code: number | null;
}
export interface UsagePayload { events: UsageEvent[]; total: number; }

export interface ApiKeyAlias { apiKeyHash: string; alias: string; updatedAtMs: number; }
export interface ApiKeyAliasesResponse { items: ApiKeyAlias[]; }

export interface ModelPrice {
  prompt: number; completion: number; cacheRead: number; cacheCreation: number;
  source?: string;
}
export interface ModelPricesResponse { prices: Record<string, ModelPrice>; }
export interface ModelPriceSyncResponse { imported: number; skipped: number; prices: Record<string, ModelPrice>; }
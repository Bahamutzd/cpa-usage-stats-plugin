import axios from 'axios';

const LS_KEY = 'cpaust.key';
const LS_BASE = 'cpaust.base';

function getKey() { try { return localStorage.getItem(LS_KEY) || ''; } catch { return ''; } }
function getBase() { try { return localStorage.getItem(LS_BASE) || ''; } catch { return ''; } }

export async function api(path: string, opts: RequestInit = {}) {
  const base = getBase().replace(/\/+$/, '');
  const headers: Record<string, string> = { Authorization: 'Bearer ' + getKey(), ...(opts.headers as Record<string, string>) };
  if (opts.body && typeof opts.body === 'string') headers['Content-Type'] = 'application/json';
  const res = await fetch(base + path, { ...opts, headers });
  if (!res.ok) throw new Error(String(res.status));
  const ct = res.headers.get('content-type') || '';
  if (ct.includes('application/x-ndjson')) return await res.blob();
  return res.json();
}

export function dayStartMS() { const d = new Date(); d.setHours(0,0,0,0); return d.getTime(); }
export const nowMS = () => Date.now();
export const fmt = (n: any) => Number(n||0).toLocaleString();
export const pct = (n: any) => n==null||isNaN(n) ? '—' : (n*100).toFixed(1)+'%';
export const dur = (n: any) => n==null ? '—' : Math.round(n)+'ms';
export const esc = (s: any) => String(s??'').replace(/[&<>"]/g, (c: string) => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;'} as any)[c] || c);
export const arr = (v: any) => v||[];

let priceCache: Record<string, any> | null = null;
export async function ensurePrices() {
  if (priceCache) return priceCache;
  try { const d = await api('/v0/management/model-prices') as any; priceCache = d?.prices || {}; } catch { priceCache = {}; }
  return priceCache!;
}
export function cost(model: string, tokens: number, mode: string) {
  if (!priceCache) return 0;
  const p = priceCache[model]; if (!p) return 0;
  const rate = mode === 'input' ? p.prompt : mode === 'output' ? p.completion : mode === 'cache_read' ? p.cacheRead : p.cacheCreation;
  return (tokens / 1_000_000) * rate;
}

export interface DashboardSummary { today: any; rolling_30m: any; top_models_today: any[]; recent_failures: any[]; channel_health?: any[]; token_mix?: any[]; }
export interface TodaySummary {}
export interface RollingSummary {}

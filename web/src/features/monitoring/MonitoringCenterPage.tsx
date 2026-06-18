import { useState, useEffect, useRef } from 'react';

const LS_KEY = 'cpaust.key'; const LS_BASE = 'cpaust.base';
const getKey = () => { try { return localStorage.getItem(LS_KEY) || ''; } catch { return ''; } };
const getBase = () => { try { return localStorage.getItem(LS_BASE) || ''; } catch { return ''; } };
async function api(path: string, opts: RequestInit = {}) {
  const base = getBase().replace(/\/+$/, '');
  const headers: Record<string, string> = { Authorization: 'Bearer ' + getKey(), ...(opts.headers as Record<string, string>) };
  if (opts.body && typeof opts.body === 'string') headers['Content-Type'] = 'application/json';
  const res = await fetch(base + path, { ...opts, headers });
  if (!res.ok) throw new Error(String(res.status));
  return res.json();
}
function fmt(n: any) { return Number(n || 0).toLocaleString(); }
function pct(n: any) { return n == null || isNaN(n) ? '—' : (n * 100).toFixed(1) + '%'; }
function esc(s: any) { return String(s ?? '').replace(/[&<>"]/g, (c: string) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' } as any)[c] || c); }
function arr(v: any) { return v || []; }

export function MonitoringCenterPage() {
  const [data, setData] = useState<any>(null);
  const [err, setErr] = useState('');
  const timer = useRef<any>();
  const load = async () => {
    try {
      const n = Date.now();
      const d = await api('/v0/management/monitoring/analytics', {
        method: 'POST',
        body: JSON.stringify({ from_ms: n - 86400000, to_ms: n, include: { summary: true, model_stats: true, timeline: true, channel_share: true, failure_sources: true, recent_failures: 20 }, granularity: 'hour' }),
      });
      setData(d); setErr('');
    } catch (e: any) { setErr(e.message); }
  };
  useEffect(() => { load(); timer.current = setInterval(load, 30000); return () => clearInterval(timer.current); }, []);

  if (err) return <div><div style={{ color: '#e0556a', padding: '1rem' }}>{err}</div></div>;
  if (!data) return <div><div style={{ padding: '2rem', color: '#888' }}>Loading...</div></div>;

  const s = data.summary || {};
  return <div>
    <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: '1rem', marginBottom: '1.5rem' }}>
      <C label="Total Requests" value={fmt(s.total_calls)} sub={fmt(s.success_calls) + ' ok / ' + fmt(s.failure_calls) + ' fail'} />
      <C label="Total Tokens" value={fmt(s.total_tokens)} sub={'In ' + fmt(s.input_tokens) + ' / Out ' + fmt(s.output_tokens)} />
      <C label="Success Rate" value={pct(s.success_rate)} sub={'30m RPM ' + fmt(s.rpm_30m)} />
      <C label="Avg Latency" value={fmt(s.average_latency_ms == null ? '—' : Math.round(s.average_latency_ms) + 'ms')} sub={fmt(s.zero_token_calls) + ' zero-token'} />
    </div>
    {arr(data.model_stats).length > 0 && <T title="Model Stats" cols={['Model', 'Calls', 'Success', 'Failure', 'Rate', 'Tokens']} rows={arr(data.model_stats).map((m: any) => [esc(m.model), fmt(m.calls), fmt(m.success_calls), fmt(m.failure_calls), pct(m.success_rate), fmt(m.total_tokens)])} />}
    {arr(data.recent_failures).length > 0 && <T title="Recent Failures" cols={['Time', 'Model', 'Source', 'Status', 'Summary']} rows={arr(data.recent_failures).map((f: any) => [new Date(f.timestamp_ms).toLocaleString(), esc(f.model), esc(f.source || ''), String(f.fail_status_code || '—'), esc((f.fail_summary || '').slice(0, 80))])} />}
  </div>;
}
function C({ label, value, sub }: any) {
  return <div style={{ border: '1px solid var(--border, #3a3a3a)', borderRadius: 8, padding: '1rem 1.2rem', background: 'var(--bg-card, #2a2a2a)' }}>
    <div style={{ color: '#888', fontSize: '0.75rem', fontWeight: 500, textTransform: 'uppercase', letterSpacing: '0.05em', marginBottom: '0.25rem' }}>{label}</div>
    <div style={{ fontSize: '1.5rem', fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}>{value}</div>
    <div style={{ color: '#888', fontSize: '0.78rem', marginTop: '0.3rem' }}>{sub}</div>
  </div>;
}
function T({ title, rows, cols }: any) {
  return <>
    <h2 style={{ fontSize: '0.9rem', fontWeight: 600, margin: '1.5rem 0 0.75rem' }}>{title}</h2>
    <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.85rem', fontVariantNumeric: 'tabular-nums' }}>
      <thead><tr>{cols.map((c: string, i: number) => <th key={i} style={{ padding: '0.5rem', color: '#888', fontSize: '0.75rem', textTransform: 'uppercase', textAlign: i > 0 ? 'right' : 'left', borderBottom: '2px solid var(--border, #3a3a3a)' }}>{c}</th>)}</tr></thead>
      <tbody>{rows.map((r: any[], i: number) => <tr key={i}>{r.map((v: any, j: number) => <td key={j} style={{ padding: '0.5rem', borderBottom: '1px solid var(--border, #3a3a3a)', textAlign: j > 0 ? 'right' : 'left' }}>{v}</td>)}</tr>)}</tbody>
    </table>
  </>;
}

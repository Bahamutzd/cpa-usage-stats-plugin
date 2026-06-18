import { useState, useEffect, useRef } from 'react';
import styles from './DashboardPage.module.scss';

const LS_KEY = 'cpaust.key';
const LS_BASE = 'cpaust.base';
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
function dur(n: any) { return n == null ? '—' : Math.round(n) + 'ms'; }
function esc(s: any) { return String(s ?? '').replace(/[&<>"]/g, (c: string) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' } as any)[c] || c); }
function arr(v: any) { return v || []; }

export function DashboardPage() {
  const [data, setData] = useState<any>(null);
  const [err, setErr] = useState('');
  const timer = useRef<any>();
  const load = async () => {
    try {
      const ds = new Date(); ds.setHours(0, 0, 0, 0);
      const d = await api('/v0/management/dashboard/summary?today_start_ms=' + ds.getTime() + '&now_ms=' + Date.now() + '&top_models=8&recent_failures=10');
      setData(d); setErr('');
    } catch (e: any) { setErr(e.message); }
  };
  useEffect(() => { load(); timer.current = setInterval(load, 30000); return () => clearInterval(timer.current); }, []);

  if (err) return <div className={styles.page}><div style={{ color: '#e0556a', padding: '1rem' }}>{err}</div></div>;
  if (!data) return <div className={styles.page}><div style={{ padding: '2rem', color: '#888' }}>Loading...</div></div>;

  const t = data.today || {};
  const r = data.rolling_30m || {};
  return (
    <div className={styles.page}>
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: '1rem', marginBottom: '1.5rem' }}>
        <C label="Requests" value={fmt(t.total_calls)} sub={fmt(t.success_calls) + ' ok / ' + fmt(t.failure_calls) + ' fail'} />
        <C label="Success Rate" value={pct(t.success_rate)} sub={'Failure ' + pct(1 - (t.success_rate || 0))} />
        <C label="Tokens" value={fmt(t.total_tokens)} sub={'In ' + fmt(t.input_tokens) + ' / Out ' + fmt(t.output_tokens)} />
        <C label="30m RPM/TPM" value={fmt(r.rpm) + ' / ' + fmt(r.tpm)} sub={fmt(r.total_calls) + ' req'} />
        <C label="Avg Latency" value={dur(t.average_latency_ms)} sub={fmt(t.zero_token_calls) + ' zero-token'} />
        <C label="Cache" value={fmt((t.cache_read_tokens || 0) + (t.cache_creation_tokens || 0))} sub={'R ' + fmt(t.cache_read_tokens) + ' / W ' + fmt(t.cache_creation_tokens)} />
      </div>
      {arr(data.top_models_today).length > 0 && <T title="Top Models" cols={['Model', 'Calls', 'Tokens', 'Success']} rows={arr(data.top_models_today).map((m: any) => [esc(m.model), fmt(m.calls), fmt(m.tokens), pct(m.success_rate)])} />}
      {arr(data.recent_failures).length > 0 && <T title="Recent Failures" cols={['Time', 'Model', 'Source', 'Status', 'Summary']} rows={arr(data.recent_failures).map((f: any) => [new Date(f.timestamp_ms).toLocaleString(), esc(f.model), esc(f.source || f.source_hash || ''), String(f.fail_status_code || '—'), esc((f.fail_summary || '').slice(0, 80))])} />}
    </div>
  );
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

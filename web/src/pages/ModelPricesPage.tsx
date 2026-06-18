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
export function ModelPricesPage() {
  const [prices, setPrices] = useState<Record<string, any>>({});
  const [err, setErr] = useState('');
  const [busy, setBusy] = useState('');
  const timer = useRef<any>();
  const load = async () => {
    try { const d = await api('/v0/management/model-prices'); setPrices(d?.prices || {}); setErr(''); } catch (e: any) { setErr(e.message); }
  };
  useEffect(() => { load(); timer.current = setInterval(load, 120000); return () => clearInterval(timer.current); }, []);

  const sync = async () => {
    setBusy('syncing');
    try { const r = await api('/v0/management/model-prices/sync', { method: 'POST' }); setPrices(r?.prices || {}); alert('Imported ' + (r?.imported || 0) + ', skipped ' + (r?.skipped || 0)); }
    catch (e: any) { alert('Sync failed: ' + e.message); }
    finally { setBusy(''); }
  };
  const save = async () => {
    setBusy('saving');
    try { await api('/v0/management/model-prices', { method: 'PUT', body: JSON.stringify({ prices }) }); alert('Saved'); }
    catch (e: any) { alert('Save failed: ' + e.message); }
    finally { setBusy(''); }
  };

  const ids = Object.keys(prices).sort();
  if (err) return <div style={{ color: '#e0556a', padding: '1rem' }}>{err}</div>;
  return <div>
    <div style={{ display: 'flex', gap: '0.5rem', marginBottom: '1rem' }}>
      <button onClick={sync} disabled={busy === 'syncing'} style={{ padding: '0.4rem 0.8rem', border: '1px solid var(--accent, #646cff)', borderRadius: 8, background: 'var(--accent, #646cff)', color: '#fff', cursor: 'pointer' }}>{busy === 'syncing' ? 'Syncing...' : 'Sync from LiteLLM'}</button>
      <button onClick={save} disabled={busy === 'saving'} style={{ padding: '0.4rem 0.8rem', border: '1px solid #3a3a3a', borderRadius: 8, background: 'transparent', color: 'inherit', cursor: 'pointer' }}>{busy === 'saving' ? 'Saving...' : 'Save'}</button>
      <span style={{ color: '#888', fontSize: '0.8rem', alignSelf: 'center' }}>USD / 1M token</span>
    </div>
    {ids.length === 0 ? <div style={{ color: '#888', padding: '2rem', textAlign: 'center' }}>No prices. Click Sync to pull from LiteLLM.</div> :
    <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.85rem' }}>
      <thead><tr><th style={{ padding: '0.5rem', color: '#888', borderBottom: '2px solid var(--border)' }}>Model</th><th style={{ padding: '0.5rem', color: '#888', textAlign: 'right', borderBottom: '2px solid var(--border)' }}>Prompt</th><th style={{ padding: '0.5rem', color: '#888', textAlign: 'right', borderBottom: '2px solid var(--border)' }}>Completion</th><th style={{ padding: '0.5rem', color: '#888', textAlign: 'right', borderBottom: '2px solid var(--border)' }}>Cache Read</th><th style={{ padding: '0.5rem', color: '#888', textAlign: 'right', borderBottom: '2px solid var(--border)' }}>Cache Create</th></tr></thead>
      <tbody>{ids.map((id: string) => { const p = prices[id]; return <tr key={id}><td style={{ padding: '0.5rem', borderBottom: '1px solid var(--border)' }}>{id}</td><td style={{ padding: '0.5rem', borderBottom: '1px solid var(--border)', textAlign: 'right' }}>{p.prompt || 0}</td><td style={{ padding: '0.5rem', borderBottom: '1px solid var(--border)', textAlign: 'right' }}>{p.completion || 0}</td><td style={{ padding: '0.5rem', borderBottom: '1px solid var(--border)', textAlign: 'right' }}>{p.cacheRead || 0}</td><td style={{ padding: '0.5rem', borderBottom: '1px solid var(--border)', textAlign: 'right' }}>{p.cacheCreation || 0}</td></tr>; })}</tbody>
    </table>}
  </div>;
}

import { useState, useEffect, useCallback, useRef } from "react";
import { api, clearPriceCache, ModelPrice } from "../services/api";

export default function Prices() {
  const [prices, setPrices] = useState<Record<string, ModelPrice>>({});
  const [err, setErr] = useState("");
  const [busy, setBusy] = useState("");
  const timer = useRef<ReturnType<typeof setInterval>>();

  const load = useCallback(async () => {
    try {
      const d: any = await api("/v0/management/model-prices");
      setPrices(d?.prices || {});
      setErr("");
    } catch (e: any) { setErr(e.message); }
  }, []);
  useEffect(() => { load(); timer.current = setInterval(load, 120000); return () => clearInterval(timer.current); }, [load]);

  const update = (id: string, f: string, v: string) => setPrices(p => ({ ...p, [id]: { ...p[id], [f]: parseFloat(v) || 0 } }));
  const remove = (id: string) => { const n = { ...prices }; delete n[id]; setPrices(n); };

  const sync = async () => {
    setBusy("syncing");
    try {
      const res: any = await api("/v0/management/model-prices/sync", { method: "POST" });
      setPrices(res?.prices || {});
      clearPriceCache();
      alert(`导入 ${res?.imported || 0}，跳过 ${res?.skipped || 0}`);
    } catch (e: any) { alert("同步失败：" + e.message); }
    finally { setBusy(""); }
  };
  const save = async () => {
    setBusy("saving");
    try {
      await api("/v0/management/model-prices", { method: "PUT", body: JSON.stringify({ prices }) });
      clearPriceCache();
      alert("已保存");
    } catch (e: any) { alert("保存失败：" + e.message); }
    finally { setBusy(""); }
  };

  const ids = Object.keys(prices).sort();
  if (err) return <div className="err">{err}</div>;
  return (
    <div>
      <div className="tools">
        <button className="primary" onClick={sync} disabled={busy === "syncing"}>{busy === "syncing" ? "同步中..." : "从 LiteLLM 同步"}</button>
        <button onClick={save} disabled={busy === "saving"}>{busy === "saving" ? "保存中..." : "保存（全量替换）"}</button>
        <span className="dim">单位 USD / 1M token</span>
      </div>
      {ids.length === 0 ? <div className="dim" style={{ padding: "2rem", textAlign: "center" }}>暂无价格</div> :
       <table>
         <thead><tr><th>模型</th><th className="num">Prompt</th><th className="num">Completion</th><th className="num">缓存读</th><th className="num">缓存建</th><th>来源</th><th></th></tr></thead>
         <tbody>
           {ids.map(id => { const p = prices[id]; return (
             <tr key={id}>
               <td>{id}</td>
               <td className="num"><input className="ed" value={p.prompt || 0} onChange={e => update(id, "prompt", e.target.value)} /></td>
               <td className="num"><input className="ed" value={p.completion || 0} onChange={e => update(id, "completion", e.target.value)} /></td>
               <td className="num"><input className="ed" value={p.cacheRead || 0} onChange={e => update(id, "cacheRead", e.target.value)} /></td>
               <td className="num"><input className="ed" value={p.cacheCreation || 0} onChange={e => update(id, "cacheCreation", e.target.value)} /></td>
               <td className="muted">{p.source || ""}</td>
               <td><button onClick={() => remove(id)}>删除</button></td>
             </tr>
           );})}
         </tbody>
       </table>}
    </div>
  );
}
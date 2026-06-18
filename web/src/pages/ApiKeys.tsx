import { useState, useEffect, useCallback, useRef } from "react";
import { api, clearPriceCache, ApiKeyAlias } from "../services/api";

export default function ApiKeys() {
  const [items, setItems] = useState<ApiKeyAlias[]>([]);
  const [err, setErr] = useState("");
  const [saving, setSaving] = useState(false);
  const timer = useRef<ReturnType<typeof setInterval>>();

  const load = useCallback(async () => {
    try {
      const d: any = await api("/v0/management/api-key-aliases");
      setItems((d?.items || []).map((x: ApiKeyAlias) => ({ ...x })));
      setErr("");
    } catch (e: any) { setErr(e.message); }
  }, []);
  useEffect(() => { load(); timer.current = setInterval(load, 60000); return () => clearInterval(timer.current); }, [load]);

  const add = () => setItems([...items, { apiKeyHash: "", alias: "", updatedAtMs: 0 }]);
  const remove = (i: number) => setItems(items.filter((_, idx) => idx !== i));
  const update = (i: number, f: "apiKeyHash" | "alias", v: string) => { const n = [...items]; n[i] = { ...n[i], [f]: v }; setItems(n); };

  const save = async () => {
    setSaving(true);
    try {
      await api("/v0/management/api-key-aliases", { method: "PUT", body: JSON.stringify({ items }) });
      clearPriceCache();
      load();
    } catch (e: any) { alert("保存失败：" + e.message); }
    finally { setSaving(false); }
  };

  if (err) return <div className="err">{err}</div>;
  return (
    <div>
      <div className="tools">
        <button onClick={add}>新增</button>
        <button className="primary" onClick={save} disabled={saving}>{saving ? "保存中..." : "保存（全量替换）"}</button>
        <span className="dim">删除单个请编辑后从列表移除再保存</span>
      </div>
      <table>
        <thead><tr><th style={{ width: "40%" }}>API Key Hash (SHA-256)</th><th style={{ width: "40%" }}>别名</th><th></th></tr></thead>
        <tbody>
          {items.map((x, i) => (
            <tr key={i}>
              <td><input className="ed" value={x.apiKeyHash} onChange={e => update(i, "apiKeyHash", e.target.value)} placeholder="64 位 hex" /></td>
              <td><input className="ed" value={x.alias} onChange={e => update(i, "alias", e.target.value)} /></td>
              <td><button onClick={() => remove(i)}>删除</button></td>
            </tr>
          ))}
          {items.length === 0 && <tr><td colSpan={3} className="dim" style={{ textAlign: "center", padding: "2rem" }}>暂无别名</td></tr>}
        </tbody>
      </table>
    </div>
  );
}
import { useState, useEffect, useCallback } from "react";
import { api, esc, ModelPrice } from "../services/api";

export default function Prices() {
  const [prices, setPrices] = useState<Record<string, ModelPrice>>({});
  const [err, setErr] = useState("");
  const [syncing, setSyncing] = useState(false);
  const [saving, setSaving] = useState(false);

  const load = useCallback(() => {
    setErr("");
    api("/v0/management/model-prices")
      .then((d: any) => setPrices(d.prices || {}))
      .catch((e) => setErr(e.message));
  }, []);
  useEffect(() => { load(); }, [load]);

  const update = (id: string, field: string, value: string) => {
    setPrices((prev) => ({
      ...prev,
      [id]: { ...prev[id], [field]: parseFloat(value) || 0 },
    }));
  };
  const remove = (id: string) => {
    setPrices((prev) => {
      const next = { ...prev };
      delete next[id];
      return next;
    });
  };

  const sync = async () => {
    setSyncing(true);
    try {
      const res: any = await api("/v0/management/model-prices/sync", { method: "POST" });
      setPrices(res.prices || {});
      alert(`导入 ${res.imported || 0}，跳过 ${res.skipped || 0}`);
    } catch (e: any) {
      alert("同步失败：" + e.message);
    } finally {
      setSyncing(false);
    }
  };

  const save = async () => {
    setSaving(true);
    try {
      await api("/v0/management/model-prices", {
        method: "PUT",
        body: JSON.stringify({ prices }),
      });
      alert("已保存");
    } catch (e: any) {
      alert("保存失败：" + e.message);
    } finally {
      setSaving(false);
    }
  };

  const ids = Object.keys(prices).sort();

  if (err) return <div className="err">{err}</div>;

  return (
    <div>
      <div className="tools">
        <button className="primary" onClick={sync} disabled={syncing}>
          {syncing ? "同步中..." : "从 LiteLLM 同步"}
        </button>
        <button onClick={save} disabled={saving}>
          {saving ? "保存中..." : "保存（全量替换）"}
        </button>
        <span className="dim">单位 USD / 1M token</span>
      </div>
      {ids.length === 0 ? (
        <div className="dim" style={{ padding: "2rem", textAlign: "center" }}>
          暂无价格，可点击「从 LiteLLM 同步」
        </div>
      ) : (
        <table>
          <thead>
            <tr>
              <th>模型</th>
              <th className="num">Prompt</th>
              <th className="num">Completion</th>
              <th className="num">缓存读</th>
              <th className="num">缓存建</th>
              <th>来源</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            {ids.map((id) => {
              const p = prices[id];
              return (
                <tr key={id}>
                  <td>{esc(id)}</td>
                  <td className="num">
                    <input
                      className="ed"
                      value={p.prompt || 0}
                      onChange={(e) => update(id, "prompt", e.target.value)}
                    />
                  </td>
                  <td className="num">
                    <input
                      className="ed"
                      value={p.completion || 0}
                      onChange={(e) => update(id, "completion", e.target.value)}
                    />
                  </td>
                  <td className="num">
                    <input
                      className="ed"
                      value={p.cacheRead || 0}
                      onChange={(e) => update(id, "cacheRead", e.target.value)}
                    />
                  </td>
                  <td className="num">
                    <input
                      className="ed"
                      value={p.cacheCreation || 0}
                      onChange={(e) => update(id, "cacheCreation", e.target.value)}
                    />
                  </td>
                  <td className="muted">{esc(p.source || "")}</td>
                  <td>
                    <button onClick={() => remove(id)}>删除</button>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      )}
    </div>
  );
}
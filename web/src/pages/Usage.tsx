import { useState, useEffect } from "react";
import { api, fmt, esc, UsagePayload } from "../services/api";

export default function Usage() {
  const [data, setData] = useState<UsagePayload | null>(null);
  const [err, setErr] = useState("");

  const load = () => {
    setErr("");
    api("/v0/management/usage?limit=200")
      .then((d) => setData(d as UsagePayload))
      .catch((e) => setErr(e.message));
  };
  useEffect(load, []);

  const handleExport = async () => {
    try {
      const blob = await api("/v0/management/usage/export") as Blob;
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "usage-events.jsonl";
      a.click();
      URL.revokeObjectURL(url);
    } catch (e: any) {
      alert(e.message);
    }
  };

  if (err) return <div className="err">{err}</div>;

  return (
    <div>
      <div className="tools">
        <button onClick={handleExport}>导出 JSONL</button>
        <span className="dim">最近 200 条{data ? ` · 共 ${fmt(data.total)} 条` : ""}</span>
      </div>
      {!data ? (
        <div className="loading">加载中...</div>
      ) : data.events.length === 0 ? (
        <div className="dim">暂无事件</div>
      ) : (
        <table>
          <thead>
            <tr>
              <th>时间</th><th>模型</th><th>渠道</th><th>来源</th>
              <th className="num">入</th><th className="num">出</th><th className="num">缓存</th>
              <th className="num">总</th><th className="num">延迟</th><th>状态</th>
            </tr>
          </thead>
          <tbody>
            {data.events.map((e) => (
              <tr key={e.event_hash}>
                <td className="muted">{new Date(e.timestamp_ms).toLocaleString()}</td>
                <td>{esc(e.model)}</td>
                <td>{esc(e.auth_index || "—")}</td>
                <td className="muted">{esc(e.source || "—")}</td>
                <td className="num">{fmt(e.input_tokens)}</td>
                <td className="num">{fmt(e.output_tokens)}</td>
                <td className="num">{fmt(e.cache_read_tokens)}</td>
                <td className="num">{fmt(e.total_tokens)}</td>
                <td className="num">{e.latency_ms == null ? "—" : e.latency_ms + "ms"}</td>
                <td>
                  {e.failed ? (
                    <span className="tag tag-red">{e.fail_status_code || "失败"}</span>
                  ) : (
                    <span className="tag tag-green">ok</span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}
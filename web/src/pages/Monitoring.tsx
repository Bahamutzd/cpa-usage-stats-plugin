import { useState, useEffect, useRef } from "react";
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  BarChart, Bar,
} from "recharts";
import {
  api, h24, nowMS, fmt, pct, dur, esc, arr,
  ensurePrices, cost, MonitoringAnalytics,
} from "../services/api";

export default function Monitoring() {
  const [data, setData] = useState<MonitoringAnalytics | null>(null);
  const [err, setErr] = useState("");
  const timer = useRef<ReturnType<typeof setInterval>>();

  const load = async () => {
    try {
      await ensurePrices();
      const d = await api("/v0/management/monitoring/analytics", {
        method: "POST",
        body: JSON.stringify({
          from_ms: h24(), to_ms: nowMS(),
          include: {
            summary: true, model_stats: true, timeline: true,
            hourly_distribution: true, channel_share: true,
            failure_sources: true, recent_failures: 20,
          },
          granularity: "hour",
        }),
      }) as MonitoringAnalytics;
      setData(d);
      setErr("");
    } catch (e: any) { setErr(e.message); }
  };
  useEffect(() => { load(); timer.current = setInterval(load, 30000); return () => clearInterval(timer.current); }, []);

  if (err) return <div className="err">{err}</div>;
  if (!data) return <div className="loading">加载中...</div>;

  const s = data.summary;

  return (
    <div>
      {s && (
        <div className="cards">
          <Card label="总请求" value={fmt(s.total_calls)} sub={`成功 ${fmt(s.success_calls)} / 失败 ${fmt(s.failure_calls)}`} />
          <Card label="总 Token" value={fmt(s.total_tokens)} sub={`入 ${fmt(s.input_tokens)} / 出 ${fmt(s.output_tokens)}`} />
          <Card label="成功率" value={pct(s.success_rate)} sub={`30m RPM ${fmt(s.rpm_30m)}`} />
          <Card label="平均延迟" value={dur(s.average_latency_ms)} sub={`零 token 调用 ${fmt(s.zero_token_calls)}`} />
        </div>
      )}

      {/* Timeline chart */}
      {arr(data.timeline).length > 2 && (
        <div className="chart-box">
          <h3>24h 时间线</h3>
          <ResponsiveContainer width="100%" height={240}>
            <LineChart data={arr(data.timeline).map(p => ({ ...p, time: new Date(p.bucket_ms).toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" }) }))}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
              <XAxis dataKey="time" tick={{ fontSize: 11, fill: "var(--text-muted)" }} />
              <YAxis tick={{ fontSize: 11, fill: "var(--text-muted)" }} />
              <Tooltip contentStyle={{ background: "var(--bg-card)", border: "1px solid var(--border)", borderRadius: 8, fontSize: 12 }} />
              <Line type="monotone" dataKey="calls" stroke="#2563eb" strokeWidth={2} dot={false} name="请求" />
              <Line type="monotone" dataKey="failure" stroke="#dc2626" strokeWidth={1.5} dot={false} name="失败" />
            </LineChart>
          </ResponsiveContainer>
        </div>
      )}

      <div className="split">
        {/* Model stats bar */}
        {arr(data.model_stats).length > 0 && (
          <div className="chart-box">
            <h3>模型统计</h3>
            <ResponsiveContainer width="100%" height={Math.max(200, arr(data.model_stats).length * 28 + 40)}>
              <BarChart data={arr(data.model_stats).map(m => ({ model: m.model, calls: m.calls }))} layout="vertical" margin={{ left: 100 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                <XAxis type="number" tick={{ fontSize: 11, fill: "var(--text-muted)" }} />
                <YAxis type="category" dataKey="model" tick={{ fontSize: 11, fill: "var(--text-muted)" }} width={100} />
                <Tooltip contentStyle={{ background: "var(--bg-card)", border: "1px solid var(--border)", borderRadius: 8, fontSize: 12 }} />
                <Bar dataKey="calls" fill="#2563eb" radius={[0, 4, 4, 0]} />
              </BarChart>
            </ResponsiveContainer>
            <table style={{ marginTop: "0.5rem" }}>
              <thead><tr><th>模型</th><th className="num">调用</th><th className="num">成功</th><th className="num">失败</th><th className="num">Token</th><th className="num">成本</th></tr></thead>
              <tbody>
                {arr(data.model_stats).map((m, i) => (
                  <tr key={i}>
                    <td>{esc(m.model)}</td>
                    <td className="num">{fmt(m.calls)}</td>
                    <td className="num">{fmt(m.success_calls)}</td>
                    <td className="num">{fmt(m.failure_calls)}</td>
                    <td className="num">{fmt(m.total_tokens)}</td>
                    <td className="num">${cost(m.model, m.total_tokens, "input").toFixed(4)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {/* Channel share */}
        {arr(data.channel_share).length > 0 && (
          <div>
            <h3>渠道</h3>
            <table>
              <thead><tr><th>渠道</th><th className="num">调用</th><th className="num">失败率</th><th className="num">Token</th></tr></thead>
              <tbody>
                {arr(data.channel_share).map((c, i) => (
                  <tr key={i}>
                    <td>{esc(c.auth_index)} {esc(c.auth_label_snapshot || "")}</td>
                    <td className="num">{fmt(c.calls)}</td>
                    <td className="num"><span className={c.failure > 0 ? "tag tag-red" : "tag tag-green"}>{pct(c.failure / Math.max(c.calls, 1))}</span></td>
                    <td className="num">{fmt(c.tokens)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Recent failures */}
      {arr(data.recent_failures).length > 0 && (
        <>
          <h3>最近失败</h3>
          <table>
            <thead><tr><th>时间</th><th>模型</th><th>来源</th><th className="num">状态码</th><th>摘要</th></tr></thead>
            <tbody>
              {arr(data.recent_failures).map((f, i) => (
                <tr key={i}>
                  <td className="muted">{new Date(f.timestamp_ms).toLocaleString()}</td>
                  <td>{esc(f.model)}</td>
                  <td className="muted">{esc(f.source || f.source_hash)}</td>
                  <td className="num">{f.fail_status_code || "—"}</td>
                  <td>{esc(f.fail_summary || "").slice(0, 80)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </>
      )}
    </div>
  );
}

function Card({ label, value, sub }: { label: string; value: string; sub: string }) {
  return <div className="card"><div className="k">{label}</div><div className="v">{value}</div><div className="sub">{sub}</div></div>;
}
import { useState, useEffect, useRef } from "react";
import {
  LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
  PieChart, Pie, Cell, BarChart, Bar,
} from "recharts";
import {
  api, dayStartMS, nowMS, fmt, pct, dur, esc, arr,
  ensurePrices, cost, DashboardSummary, TodaySummary, RollingSummary,
} from "../services/api";

const COLORS = ["#2563eb", "#16a34a", "#ea580c", "#8b5cf6", "#dc2626", "#0891b2"];

export default function Dashboard() {
  const [data, setData] = useState<DashboardSummary | null>(null);
  const [err, setErr] = useState("");
  const timer = useRef<ReturnType<typeof setInterval>>();

  const load = async () => {
    try {
      await ensurePrices();
      const d = await api(
        `/v0/management/dashboard/summary?today_start_ms=${dayStartMS()}&now_ms=${nowMS()}&top_models=8&recent_failures=10`
      ) as DashboardSummary;
      setData(d);
      setErr("");
    } catch (e: any) { setErr(e.message); }
  };
  useEffect(() => { load(); timer.current = setInterval(load, 30000); return () => clearInterval(timer.current); }, []);

  if (err) return <div className="err">{err}</div>;
  if (!data) return <div className="loading">加载中...</div>;

  const t = data.today || ({} as TodaySummary);
  const r = data.rolling_30m || ({} as RollingSummary);

  return (
    <div>
      {/* Summary cards */}
      <div className="cards">
        <Card label="今日请求" value={fmt(t.total_calls)} sub={`成功 ${fmt(t.success_calls)} · 失败 ${fmt(t.failure_calls)}`} />
        <Card label="成功率" value={pct(t.success_rate)} sub={`失败率 ${pct(1 - (t.success_rate || 0))}`} />
        <Card label="今日 Token" value={fmt(t.total_tokens)} sub={`入 ${fmt(t.input_tokens)} / 出 ${fmt(t.output_tokens)}`} />
        <Card label="30m RPM / TPM" value={`${fmt(r.rpm)} / ${fmt(r.tpm)}`} sub={`30m 请求 ${fmt(r.total_calls)}`} />
        <Card label="平均延迟" value={dur(t.average_latency_ms)} sub={`零 token ${fmt(t.zero_token_calls)}`} />
        <Card label="缓存" value={fmt((t.cache_read_tokens || 0) + (t.cache_creation_tokens || 0))} sub={`读 ${fmt(t.cache_read_tokens)} / 建 ${fmt(t.cache_creation_tokens)}`} />
      </div>

      {/* Charts row */}
      <div className="split">
        {/* Traffic timeline */}
        {arr(data.traffic_timeline).length > 2 && (
          <div className="chart-box">
            <h3>今日流量</h3>
            <ResponsiveContainer width="100%" height={220}>
              <LineChart data={arr(data.traffic_timeline).map(p => ({ ...p, time: new Date(p.bucket_ms).toLocaleTimeString("zh-CN", { hour: "2-digit", minute: "2-digit" }) }))}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
                <XAxis dataKey="time" tick={{ fontSize: 11, fill: "var(--text-muted)" }} />
                <YAxis tick={{ fontSize: 11, fill: "var(--text-muted)" }} />
                <Tooltip contentStyle={{ background: "var(--bg-card)", border: "1px solid var(--border)", borderRadius: 8, fontSize: 12 }} />
                <Line type="monotone" dataKey="calls" stroke="#2563eb" strokeWidth={2} dot={false} />
                <Line type="monotone" dataKey="failure" stroke="#dc2626" strokeWidth={1.5} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        )}

        {/* Token mix pie */}
        {arr(data.token_mix).length > 0 && (
          <div className="chart-box">
            <h3>Token 构成</h3>
            <ResponsiveContainer width="100%" height={220}>
              <PieChart>
                <Pie data={arr(data.token_mix).filter(s => s.tokens > 0)} dataKey="tokens" nameKey="key" cx="50%" cy="50%" outerRadius={80} innerRadius={40} label={({ key, share }: any) => `${key} ${((share || 0) * 100).toFixed(0)}%`}>
                  {arr(data.token_mix).map((_, i) => <Cell key={i} fill={COLORS[i % COLORS.length]} />)}
                </Pie>
                <Tooltip />
              </PieChart>
            </ResponsiveContainer>
          </div>
        )}
      </div>

      {/* Top models bar chart */}
      {arr(data.top_models_today).length > 0 && (
        <div className="chart-box">
          <h3>今日 Top 模型</h3>
          <ResponsiveContainer width="100%" height={Math.max(200, arr(data.top_models_today).length * 28 + 40)}>
            <BarChart data={arr(data.top_models_today).map(m => ({ model: m.model, calls: m.calls }))} layout="vertical" margin={{ left: 100 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
              <XAxis type="number" tick={{ fontSize: 11, fill: "var(--text-muted)" }} />
              <YAxis type="category" dataKey="model" tick={{ fontSize: 11, fill: "var(--text-muted)" }} width={100} />
              <Tooltip contentStyle={{ background: "var(--bg-card)", border: "1px solid var(--border)", borderRadius: 8, fontSize: 12 }} />
              <Bar dataKey="calls" fill="#2563eb" radius={[0, 4, 4, 0]} />
            </BarChart>
          </ResponsiveContainer>
          <table style={{ marginTop: "0.5rem" }}>
            <thead><tr><th>模型</th><th className="num">调用</th><th className="num">Token</th><th className="num">成功率</th><th className="num">成本</th></tr></thead>
            <tbody>
              {arr(data.top_models_today).map((m, i) => (
                <tr key={i}>
                  <td>{esc(m.model)}</td>
                  <td className="num">{fmt(m.calls)}</td>
                  <td className="num">{fmt(m.tokens)}</td>
                  <td className="num">{pct(m.success_rate)}</td>
                  <td className="num">${cost(m.model, m.tokens, "input").toFixed(4)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Channel health */}
      {arr(data.channel_health).length > 0 && (
        <>
          <h3>渠道健康</h3>
          <table>
            <thead><tr><th>渠道</th><th className="num">调用</th><th className="num">失败</th><th className="num">失败率</th><th className="num">Token</th><th className="num">延迟</th></tr></thead>
            <tbody>
              {arr(data.channel_health).map((ch, i) => (
                <tr key={i}>
                  <td>{esc(ch.auth_index)} {esc(ch.auth_label_snapshot || "")}</td>
                  <td className="num">{fmt(ch.calls)}</td>
                  <td className="num">{fmt(ch.failures)}</td>
                  <td className="num"><span className={ch.failures > 0 ? "tag tag-red" : "tag tag-green"}>{pct(ch.failure_rate)}</span></td>
                  <td className="num">{fmt(ch.tokens)}</td>
                  <td className="num">{dur(ch.average_latency_ms)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </>
      )}

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
  return (
    <div className="card">
      <div className="k">{label}</div>
      <div className="v">{value}</div>
      <div className="sub">{sub}</div>
    </div>
  );
}
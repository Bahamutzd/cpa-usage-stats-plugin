import { useState, useEffect } from "react";
import { api, h24, nowMS, fmt, pct, dur, esc, MonitoringAnalytics } from "../services/api";

export default function Monitoring() {
  const [data, setData] = useState<MonitoringAnalytics | null>(null);
  const [err, setErr] = useState("");

  useEffect(() => {
    api("/v0/management/monitoring/analytics", {
      method: "POST",
      body: JSON.stringify({
        from_ms: h24(),
        to_ms: nowMS(),
        include: {
          summary: true, model_stats: true, timeline: true,
          hourly_distribution: true, channel_share: true,
          failure_sources: true, recent_failures: 20,
        },
        granularity: "hour",
      }),
    })
      .then((d) => setData(d as MonitoringAnalytics))
      .catch((e) => setErr(e.message));
  }, []);

  if (err) return <div className="err">{err}</div>;
  if (!data) return <div className="loading">加载中...</div>;

  return (
    <div>
      {data.summary && (
        <div className="cards">
          <div className="card">
            <div className="k">总请求</div>
            <div className="v">{fmt(data.summary.total_calls)}</div>
            <div className="sub">成功 {fmt(data.summary.success_calls)} / 失败 {fmt(data.summary.failure_calls)}</div>
          </div>
          <div className="card">
            <div className="k">总 Token</div>
            <div className="v">{fmt(data.summary.total_tokens)}</div>
            <div className="sub">入 {fmt(data.summary.input_tokens)} / 出 {fmt(data.summary.output_tokens)}</div>
          </div>
          <div className="card">
            <div className="k">成功率</div>
            <div className="v">{pct(data.summary.success_rate)}</div>
            <div className="sub">30m RPM {fmt(data.summary.rpm_30m)}</div>
          </div>
          <div className="card">
            <div className="k">平均延迟</div>
            <div className="v">{dur(data.summary.average_latency_ms)}</div>
            <div className="sub">零 token 调用 {fmt(data.summary.zero_token_calls)}</div>
          </div>
        </div>
      )}

      {data.model_stats?.length > 0 && (
        <>
          <h3>模型统计</h3>
          <table>
            <thead>
              <tr>
                <th>模型</th>
                <th className="num">调用</th>
                <th className="num">成功</th>
                <th className="num">失败</th>
                <th className="num">成功率</th>
                <th className="num">Token</th>
              </tr>
            </thead>
            <tbody>
              {data.model_stats.map((m, i) => (
                <tr key={i}>
                  <td>{esc(m.model)}</td>
                  <td className="num">{fmt(m.calls)}</td>
                  <td className="num">{fmt(m.success_calls)}</td>
                  <td className="num">{fmt(m.failure_calls)}</td>
                  <td className="num">{pct(m.success_rate)}</td>
                  <td className="num">{fmt(m.total_tokens)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </>
      )}

      <div className="split">
        {data.timeline?.length > 0 && (
          <div>
            <h3>时间线</h3>
            <table>
              <thead>
                <tr>
                  <th>时间</th><th className="num">调用</th><th className="num">成功</th>
                  <th className="num">失败</th>
                </tr>
              </thead>
              <tbody>
                {data.timeline.map((p, i) => (
                  <tr key={i}>
                    <td className="muted">{esc(p.label)}</td>
                    <td className="num">{fmt(p.calls)}</td>
                    <td className="num">{fmt(p.success)}</td>
                    <td className="num">{fmt(p.failure)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {data.channel_share?.length > 0 && (
          <div>
            <h3>渠道</h3>
            <table>
              <thead>
                <tr>
                  <th>渠道</th><th className="num">调用</th><th className="num">失败率</th>
                </tr>
              </thead>
              <tbody>
                {data.channel_share.map((c, i) => (
                  <tr key={i}>
                    <td>{esc(c.auth_index)} {esc(c.auth_label_snapshot || "")}</td>
                    <td className="num">{fmt(c.calls)}</td>
                    <td className="num">
                      <span className={c.failure > 0 ? "tag tag-red" : "tag tag-green"}>
                        {pct(c.failure / Math.max(c.calls, 1))}
                      </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {data.recent_failures?.length > 0 && (
        <>
          <h3>最近失败</h3>
          <table>
            <thead>
              <tr><th>时间</th><th>模型</th><th>来源</th><th className="num">状态码</th><th>摘要</th></tr>
            </thead>
            <tbody>
              {data.recent_failures.map((f, i) => (
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
import { useState, useEffect } from "react";
import { api, dayStartMS, nowMS, fmt, pct, dur, esc, DashboardSummary } from "../services/api";

export default function Dashboard() {
  const [data, setData] = useState<DashboardSummary | null>(null);
  const [err, setErr] = useState("");

  useEffect(() => {
    api(
      `/v0/management/dashboard/summary?today_start_ms=${dayStartMS()}&now_ms=${nowMS()}&top_models=8&recent_failures=10`
    )
      .then((d) => setData(d as DashboardSummary))
      .catch((e) => setErr(e.message));
  }, []);

  if (err) return <div className="err">{err}</div>;
  if (!data) return <div className="loading">加载中...</div>;

  const t = data.today;
  const r = data.rolling_30m;
  const cards = [
    ["今日请求", fmt(t.total_calls), `成功 ${fmt(t.success_calls)} · 失败 ${fmt(t.failure_calls)}`],
    ["成功率", pct(t.success_rate), `失败率 ${pct(1 - t.success_rate)}`],
    ["今日 Token", fmt(t.total_tokens), `入 ${fmt(t.input_tokens)} / 出 ${fmt(t.output_tokens)}`],
    ["30m RPM / TPM", `${fmt(r.rpm)} / ${fmt(r.tpm)}`, `30m 请求 ${fmt(r.total_calls)}`],
    ["平均延迟", dur(t.average_latency_ms), `零 token ${fmt(t.zero_token_calls)}`],
    ["缓存读/建", `${fmt(t.cache_read_tokens)} / ${fmt(t.cache_creation_tokens)}`, `reasoning ${fmt(t.reasoning_tokens)}`],
  ];

  return (
    <div>
      <div className="cards">
        {cards.map((c, i) => (
          <div className="card" key={i}>
            <div className="k">{c[0]}</div>
            <div className="v">{c[1]}</div>
            <div className="sub">{c[2]}</div>
          </div>
        ))}
      </div>

      {data.top_models_today?.length > 0 && (
        <>
          <h3>今日 Top 模型</h3>
          <table>
            <thead>
              <tr>
                <th>模型</th><th className="num">调用</th><th className="num">Token</th>
                <th className="num">成功率</th><th style={{ width: "8rem" }}></th>
              </tr>
            </thead>
            <tbody>
              {data.top_models_today.map((m, i) => {
                const max = Math.max(...data.top_models_today.map((x) => x.calls), 1);
                return (
                  <tr key={i}>
                    <td>{esc(m.model)}</td>
                    <td className="num">{fmt(m.calls)}</td>
                    <td className="num">{fmt(m.tokens)}</td>
                    <td className="num">{pct(m.success_rate)}</td>
                    <td>
                      <div className="bar"><i style={{ width: (m.calls / max * 100).toFixed(0) + "%" }} /></div>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </>
      )}

      {data.token_mix?.length > 0 && (
        <>
          <h3>Token 构成</h3>
          <div className="row">
            {data.token_mix.map((seg, i) => (
              <span className="tag" key={i}>
                {esc(seg.key)} {pct(seg.share)}
              </span>
            ))}
          </div>
        </>
      )}

      {data.channel_health?.length > 0 && (
        <>
          <h3>渠道</h3>
          <table>
            <thead>
              <tr>
                <th>渠道</th>
                <th className="num">调用</th><th className="num">失败</th>
                <th className="num">失败率</th><th className="num">Token</th><th className="num">延迟</th>
              </tr>
            </thead>
            <tbody>
              {data.channel_health.map((ch, i) => (
                <tr key={i}>
                  <td>{esc(ch.auth_index)} {esc(ch.auth_label_snapshot || "")}</td>
                  <td className="num">{fmt(ch.calls)}</td>
                  <td className="num">{fmt(ch.failures)}</td>
                  <td className="num">
                    <span className={ch.failures > 0 ? "tag tag-red" : "tag tag-green"}>
                      {pct(ch.failures / Math.max(ch.calls, 1))}
                    </span>
                  </td>
                  <td className="num">{fmt(ch.tokens)}</td>
                  <td className="num">{dur(ch.average_latency_ms)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </>
      )}

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
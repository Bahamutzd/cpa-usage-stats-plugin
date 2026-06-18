import { useState, useEffect, useCallback } from "react";
import { Routes, Route, NavLink, useNavigate } from "react-router-dom";
import { getConfig, saveConfig } from "./services/api";
import Dashboard from "./pages/Dashboard";
import Monitoring from "./pages/Monitoring";
import Usage from "./pages/Usage";
import ApiKeys from "./pages/ApiKeys";
import Prices from "./pages/Prices";

export default function App() {
  const [base, setBase] = useState(getConfig().base);
  const [key, setKey] = useState(getConfig().key);
  const [status, setStatus] = useState("");
  const navigate = useNavigate();

  useEffect(() => {
    const hash = location.hash.replace(/^#\/?/, "");
    if (!hash) navigate("/dashboard", { replace: true });
  }, [navigate]);

  const handleSave = useCallback(() => {
    saveConfig(base, key);
    setStatus(key ? "已配置" : "未配置 key");
  }, [base, key]);

  useEffect(() => {
    setStatus(key ? "已配置" : "未配置 key");
  }, [key]);

  return (
    <>
      <header className="app-header">
        <h1>CPA Usage Stats</h1>
        <label>
          API base
          <input
            value={base}
            onChange={(e) => setBase(e.target.value)}
            placeholder="留空=同源"
          />
        </label>
        <label>
          Key
          <input
            name="key"
            type="password"
            value={key}
            onChange={(e) => setKey(e.target.value)}
            placeholder="Management Key"
          />
        </label>
        <button onClick={handleSave}>保存配置</button>
        <span className="cfg-status">{status}</span>
      </header>
      <nav className="app-nav">
        <NavLink to="/dashboard">概览</NavLink>
        <NavLink to="/monitoring">监控分析</NavLink>
        <NavLink to="/usage">事件</NavLink>
        <NavLink to="/apikeys">API Key 别名</NavLink>
        <NavLink to="/prices">模型价格</NavLink>
      </nav>
      <main>
        <Routes>
          <Route path="/dashboard" element={<Dashboard />} />
          <Route path="/monitoring" element={<Monitoring />} />
          <Route path="/usage" element={<Usage />} />
          <Route path="/apikeys" element={<ApiKeys />} />
          <Route path="/prices" element={<Prices />} />
          <Route path="*" element={<Dashboard />} />
        </Routes>
      </main>
    </>
  );
}
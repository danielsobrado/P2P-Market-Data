/* eslint-disable */
// Atomic components for the P2P Market Terminal.
// All components are pushed to window at the bottom so other babel scripts
// can use them without ES modules.

const { useState, useEffect, useRef, useMemo } = React;

/* --------------------------------------------------------------------------
   Icon — lucide-style strokes drawn inline so we have no external dep.
   Keep them tiny (14–16 px), 1.6 stroke, currentColor.
   -------------------------------------------------------------------------- */
function Icon({ name, size = 14, strokeWidth = 1.6, style }) {
  const s = size;
  const common = {
    width: s, height: s, viewBox: "0 0 24 24",
    fill: "none", stroke: "currentColor",
    strokeWidth, strokeLinecap: "round", strokeLinejoin: "round",
    style,
  };
  switch (name) {
    case "dashboard":
      return <svg {...common}><rect x="3" y="3" width="7" height="9"/><rect x="14" y="3" width="7" height="5"/><rect x="14" y="12" width="7" height="9"/><rect x="3" y="16" width="7" height="5"/></svg>;
    case "search":
      return <svg {...common}><circle cx="11" cy="11" r="7"/><path d="M21 21l-4.3-4.3"/></svg>;
    case "candles":
      return <svg {...common}><line x1="7" y1="2" x2="7" y2="22"/><rect x="4" y="7" width="6" height="10"/><line x1="17" y1="2" x2="17" y2="22"/><rect x="14" y="4" width="6" height="14"/></svg>;
    case "transfer":
      return <svg {...common}><path d="M3 7h14"/><path d="M14 3l4 4-4 4"/><path d="M21 17H7"/><path d="M10 21l-4-4 4-4"/></svg>;
    case "code":
      return <svg {...common}><path d="M16 18l6-6-6-6"/><path d="M8 6l-6 6 6 6"/></svg>;
    case "users":
      return <svg {...common}><circle cx="9" cy="8" r="3.5"/><path d="M2 21c0-3.5 3-6 7-6s7 2.5 7 6"/><circle cx="17" cy="6" r="2.5"/><path d="M22 19c0-2.4-2-4-4.5-4"/></svg>;
    case "refresh":
      return <svg {...common}><path d="M3 12a9 9 0 0 1 15.5-6.3L21 8"/><path d="M21 3v5h-5"/><path d="M21 12a9 9 0 0 1-15.5 6.3L3 16"/><path d="M3 21v-5h5"/></svg>;
    case "download":
      return <svg {...common}><path d="M12 4v12"/><path d="M7 11l5 5 5-5"/><path d="M4 21h16"/></svg>;
    case "upload":
      return <svg {...common}><path d="M12 20V8"/><path d="M7 13l5-5 5 5"/><path d="M4 21h16"/></svg>;
    case "play":
      return <svg {...common}><path d="M6 4l14 8-14 8z" fill="currentColor"/></svg>;
    case "pause":
      return <svg {...common}><rect x="6" y="4" width="4" height="16" fill="currentColor"/><rect x="14" y="4" width="4" height="16" fill="currentColor"/></svg>;
    case "stop":
      return <svg {...common}><rect x="5" y="5" width="14" height="14" fill="currentColor"/></svg>;
    case "trash":
      return <svg {...common}><path d="M3 6h18"/><path d="M8 6V4h8v2"/><path d="M19 6l-1 14H6L5 6"/></svg>;
    case "check":
      return <svg {...common}><path d="M5 13l4 4L19 7"/></svg>;
    case "x":
      return <svg {...common}><path d="M6 6l12 12M18 6L6 18"/></svg>;
    case "alert":
      return <svg {...common}><path d="M12 2L1 21h22z"/><line x1="12" y1="9" x2="12" y2="14"/><line x1="12" y1="17" x2="12" y2="17.5"/></svg>;
    case "settings":
      return <svg {...common}><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.7 1.7 0 0 0 .34 1.86l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.7 1.7 0 0 0-1.86-.34 1.7 1.7 0 0 0-1.03 1.55V21a2 2 0 0 1-4 0v-.1a1.7 1.7 0 0 0-1.11-1.55 1.7 1.7 0 0 0-1.86.34l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06a1.7 1.7 0 0 0 .34-1.86 1.7 1.7 0 0 0-1.55-1.03H3a2 2 0 0 1 0-4h.1a1.7 1.7 0 0 0 1.55-1.11 1.7 1.7 0 0 0-.34-1.86l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06a1.7 1.7 0 0 0 1.86.34h.04a1.7 1.7 0 0 0 1.03-1.55V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1.03 1.55 1.7 1.7 0 0 0 1.86-.34l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06a1.7 1.7 0 0 0-.34 1.86V9a1.7 1.7 0 0 0 1.55 1.03H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.55 1.03z"/></svg>;
    case "sun":
      return <svg {...common}><circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2M2 12h2M20 12h2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4"/></svg>;
    case "moon":
      return <svg {...common}><path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8z"/></svg>;
    case "arrow-up":
      return <svg {...common}><path d="M12 19V5"/><path d="M5 12l7-7 7 7"/></svg>;
    case "arrow-down":
      return <svg {...common}><path d="M12 5v14"/><path d="M5 12l7 7 7-7"/></svg>;
    case "arrow-right":
      return <svg {...common}><path d="M5 12h14"/><path d="M12 5l7 7-7 7"/></svg>;
    case "filter":
      return <svg {...common}><path d="M3 5h18"/><path d="M6 12h12"/><path d="M10 19h4"/></svg>;
    case "sliders":
      return <svg {...common}><line x1="3" y1="6" x2="14" y2="6"/><line x1="18" y1="6" x2="21" y2="6"/><circle cx="16" cy="6" r="2"/><line x1="3" y1="12" x2="6" y2="12"/><line x1="10" y1="12" x2="21" y2="12"/><circle cx="8" cy="12" r="2"/><line x1="3" y1="18" x2="14" y2="18"/><line x1="18" y1="18" x2="21" y2="18"/><circle cx="16" cy="18" r="2"/></svg>;
    case "more":
      return <svg {...common}><circle cx="5" cy="12" r="1"/><circle cx="12" cy="12" r="1"/><circle cx="19" cy="12" r="1"/></svg>;
    case "thumbs-up":
      return <svg {...common}><path d="M7 10v11h11a2 2 0 0 0 2-1.6l1.2-7A2 2 0 0 0 19.2 10H14V5a2 2 0 0 0-4 0c0 3-3 5-3 5z"/><line x1="3" y1="10" x2="3" y2="21"/></svg>;
    case "thumbs-down":
      return <svg {...common}><path d="M17 14V3H6a2 2 0 0 0-2 1.6l-1.2 7A2 2 0 0 0 4.8 14H10v5a2 2 0 0 0 4 0c0-3 3-5 3-5z"/><line x1="21" y1="3" x2="21" y2="14"/></svg>;
    case "network":
      return <svg {...common}><circle cx="12" cy="4" r="2"/><circle cx="4" cy="20" r="2"/><circle cx="20" cy="20" r="2"/><path d="M12 6v6M12 12L4 20M12 12l8 8"/></svg>;
    case "db":
      return <svg {...common}><ellipse cx="12" cy="5" rx="9" ry="2.5"/><path d="M3 5v6c0 1.4 4 2.5 9 2.5s9-1.1 9-2.5V5"/><path d="M3 11v6c0 1.4 4 2.5 9 2.5s9-1.1 9-2.5v-6"/></svg>;
    case "clock":
      return <svg {...common}><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></svg>;
    case "globe":
      return <svg {...common}><circle cx="12" cy="12" r="9"/><path d="M3 12h18"/><path d="M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18"/></svg>;
    case "command":
      return <svg {...common}><path d="M9 3a3 3 0 1 0 0 6h12v6a3 3 0 1 1-6 0V3"/></svg>;
    case "info":
      return <svg {...common}><circle cx="12" cy="12" r="9"/><line x1="12" y1="16" x2="12" y2="12"/><line x1="12" y1="8" x2="12" y2="8"/></svg>;
    default:
      return <svg {...common}><rect x="4" y="4" width="16" height="16"/></svg>;
  }
}

/* --------------------------------------------------------------------------
   StatusBar
   -------------------------------------------------------------------------- */
function StatusBar({ isConnected, serverStatus, view, lastRefresh, theme, onToggleTheme }) {
  const [cmd, setCmd] = useState("");
  const [now, setNow] = useState(() => new Date());

  useEffect(() => {
    const i = setInterval(() => setNow(new Date()), 1000);
    return () => clearInterval(i);
  }, []);

  const utc = now.toISOString().slice(11, 19) + " UTC";

  return (
    <div className="statusbar" role="banner">
      <div className="sb-brand">
        <div className="mark">P</div>
        <div>
          <span className="name">P2P MARKET</span>
          <span className="sub" style={{ marginLeft: 8 }}>TERMINAL</span>
        </div>
      </div>

      <div className="sb-cell">
        <span className={`led ${isConnected ? "pos pulse" : "neg"}`}></span>
        <span className="label">NET</span>
        <span className="val">{isConnected ? "LIVE" : "OFFLINE"}</span>
      </div>

      <div className="sb-cell">
        <span className={`led ${serverStatus === "ok" ? "pos" : serverStatus === "warn" ? "warn" : "neg"}`}></span>
        <span className="label">DB</span>
        <span className="val">{serverStatus === "ok" ? "READY" : serverStatus === "warn" ? "SYNC" : "DOWN"}</span>
      </div>

      <div className="sb-cell">
        <span className="label">VIEW</span>
        <span className="val">{view}</span>
      </div>

      <div className="sb-cell">
        <Icon name="clock" size={11} />
        <span className="label">LAST</span>
        <span className="val">{lastRefresh}</span>
      </div>

      <div className="sb-spacer" />

      <div className="sb-cell hide-md">
        <span className="label">UTC</span>
        <span className="val">{utc}</span>
      </div>

      <div className="sb-cmd">
        <span className="caret">›</span>
        <input
          value={cmd}
          onChange={(e) => setCmd(e.target.value)}
          placeholder="cmd · aapl eod · go peers · /help"
          spellCheck="false"
        />
      </div>

      <button className="sb-iconbtn" title="Settings"><Icon name="sliders" size={13} /></button>
      <button className="sb-iconbtn" title="Toggle theme" onClick={onToggleTheme}>
        <Icon name={theme === "light" ? "sun" : "moon"} size={13} />
      </button>
    </div>
  );
}

/* --------------------------------------------------------------------------
   TickerStrip
   -------------------------------------------------------------------------- */
function TickerStrip({ items }) {
  const doubled = [...items, ...items];
  return (
    <div className="ticker">
      <div className="ticker-label">
        <span className="led pos pulse" />
        MKT · LIVE
      </div>
      <div className="ticker-track">
        <div className="ticker-marquee">
          {doubled.map((t, i) => {
            const positive = t.chg >= 0;
            return (
              <span key={i} className="tk-item">
                <span className="sym">{t.sym}</span>
                <span className="px">{t.px.toLocaleString(undefined, { maximumFractionDigits: 2 })}</span>
                <span className={`chg ${positive ? "pos" : "neg"}`}>
                  {positive ? "▲" : "▼"} {Math.abs(t.chg).toFixed(2)}%
                </span>
              </span>
            );
          })}
        </div>
      </div>
    </div>
  );
}

/* --------------------------------------------------------------------------
   SideNav
   -------------------------------------------------------------------------- */
const NAV = [
  { id: "dashboard", label: "Dashboard",   icon: "dashboard", key: "F1" },
  { id: "search",    label: "Search",      icon: "search",    key: "F2" },
  { id: "market",    label: "Market Data", icon: "candles",   key: "F3" },
  { id: "transfers", label: "Transfers",   icon: "transfer",  key: "F4" },
  { id: "scripts",   label: "Scripts",     icon: "code",      key: "F5" },
  { id: "peers",     label: "Peers",       icon: "users",     key: "F6" },
];

function SideNav({ active, onSelect, peersCount, transferCount }) {
  return (
    <nav className="sidenav">
      <div className="sn-section">Workspace</div>
      {NAV.map((n) => (
        <div
          key={n.id}
          className={`sn-item ${active === n.id ? "active" : ""}`}
          onClick={() => onSelect(n.id)}
        >
          <Icon name={n.icon} size={14} className="icon" />
          <span>{n.label}</span>
          <span className="key">{n.key}</span>
        </div>
      ))}

      <div className="sn-section" style={{ marginTop: 8 }}>Session</div>
      <div className="sn-footer">
        <div className="row"><span>Node</span><span className="v">7a3f·ec19</span></div>
        <div className="row"><span>Peers</span><span className="v">{peersCount}</span></div>
        <div className="row"><span>Active TX</span><span className="v">{transferCount}</span></div>
        <div className="row"><span>Build</span><span className="v">v0.8.2</span></div>
      </div>
    </nav>
  );
}

/* --------------------------------------------------------------------------
   Panel — wrapper for any workstation panel
   -------------------------------------------------------------------------- */
function Panel({ title, tag, sub, actions, children, flush, style }) {
  return (
    <div className="panel" style={style}>
      <div className="panel-head">
        <span className="panel-title">{title}</span>
        {tag && <span className="panel-tag">{tag}</span>}
        {sub && <span className="panel-sub">{sub}</span>}
        {actions && <div className="panel-actions">{actions}</div>}
      </div>
      <div className={`panel-body ${flush ? "flush" : ""}`}>{children}</div>
    </div>
  );
}

/* --------------------------------------------------------------------------
   StatusBadge
   -------------------------------------------------------------------------- */
function StatusBadge({ kind = "default", children, dot = true, solid = false }) {
  const k = kind === "default" ? "" : kind;
  return (
    <span className={`badge ${k} ${solid ? "solid" : ""}`}>
      {dot && <span className="dot" />}
      {children}
    </span>
  );
}

/* --------------------------------------------------------------------------
   MetricTile
   -------------------------------------------------------------------------- */
function Sparkline({ data, color = "currentColor", width = 88, height = 22 }) {
  if (!data || !data.length) return null;
  const min = Math.min(...data);
  const max = Math.max(...data);
  const range = max - min || 1;
  const pts = data
    .map((v, i) => {
      const x = (i / (data.length - 1)) * (width - 2) + 1;
      const y = height - 1 - ((v - min) / range) * (height - 2);
      return `${x.toFixed(1)},${y.toFixed(1)}`;
    })
    .join(" ");
  return (
    <svg width={width} height={height} className="sparkline">
      <polyline points={pts} fill="none" stroke={color} strokeWidth="1.2" />
    </svg>
  );
}

function MetricTile({ label, value, unit, delta, deltaPct, sub, kind, spark, sparkColor }) {
  const positive = (delta ?? 0) >= 0;
  return (
    <div className={`metric ${kind || ""}`}>
      <div className="m-label">
        <span>{label}</span>
      </div>
      <div className="m-val">
        {value}
        {unit && <span className="unit">{unit}</span>}
      </div>
      <div className="m-sub">
        {delta !== undefined && (
          <span className={`delta ${positive ? "pos" : "neg"}`}>
            {positive ? "▲" : "▼"} {Math.abs(delta)}
            {deltaPct !== undefined ? ` (${positive ? "+" : "-"}${Math.abs(deltaPct)}%)` : ""}
          </span>
        )}
        {sub && <span>{sub}</span>}
      </div>
      {spark && <Sparkline data={spark} color={sparkColor || "var(--accent)"} />}
    </div>
  );
}

/* --------------------------------------------------------------------------
   ReputationBar
   -------------------------------------------------------------------------- */
function ReputationBar({ value }) {
  const pct = Math.round(value * 100);
  const cls = pct >= 75 ? "" : pct >= 50 ? "mid" : "low";
  return (
    <span className="repbar">
      <span className="track"><span className={`fill ${cls}`} style={{ width: `${pct}%` }} /></span>
      <span className="val">{pct}%</span>
    </span>
  );
}

/* --------------------------------------------------------------------------
   ProgressBar
   -------------------------------------------------------------------------- */
function ProgressBar({ value, status, striped, thick }) {
  const cls =
    status === "completed" ? "pos" :
    status === "failed" ? "neg" :
    status === "pending" ? "warn" : "";
  return (
    <div className={`progress ${cls} ${thick ? "thick" : ""} ${striped ? "striped" : ""}`}>
      <div className="fill" style={{ width: `${Math.min(100, Math.max(0, value))}%` }} />
    </div>
  );
}

/* --------------------------------------------------------------------------
   LogStrip — bottom system console
   -------------------------------------------------------------------------- */
function LogStrip({ lines }) {
  const ref = useRef(null);
  const [idx, setIdx] = useState(0);
  useEffect(() => {
    const i = setInterval(() => setIdx((v) => (v + 1) % lines.length), 3200);
    return () => clearInterval(i);
  }, [lines.length]);
  const line = lines[idx];
  return (
    <div className="logstrip" ref={ref}>
      <span className="ls-label">CON</span>
      <span className="ls-line">
        <span className="ts">{line.ts}</span>
        <span className={`lvl ${line.lvl}`}>{line.lvl.toUpperCase()}</span>
        <span>{line.msg}</span>
      </span>
      <span style={{ flex: 1 }} />
      <span style={{ color: "var(--text-mute)", letterSpacing: "0.04em" }}>
        ▎ buffer 7/64 · scroll-lock OFF
      </span>
    </div>
  );
}

/* --------------------------------------------------------------------------
   Tabs (simple controlled)
   -------------------------------------------------------------------------- */
function Tabs({ tabs, value, onChange }) {
  return (
    <div className="tabs">
      {tabs.map((t) => (
        <div
          key={t.id}
          className={`tab ${value === t.id ? "active" : ""}`}
          onClick={() => onChange(t.id)}
        >
          {t.label}
          {t.count !== undefined && <span className="ct">{t.count}</span>}
        </div>
      ))}
    </div>
  );
}

/* --------------------------------------------------------------------------
   Empty state
   -------------------------------------------------------------------------- */
function EmptyState({ icon = "info", title, hint }) {
  return (
    <div className="empty">
      <Icon name={icon} size={28} className="icon" />
      <div>{title}</div>
      {hint && <div className="hint">{hint}</div>}
    </div>
  );
}

Object.assign(window, {
  Icon, StatusBar, TickerStrip, SideNav, NAV,
  Panel, StatusBadge, MetricTile, Sparkline, ReputationBar, ProgressBar,
  LogStrip, Tabs, EmptyState,
});

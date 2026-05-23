/* eslint-disable */
const { useState, useEffect, useMemo } = React;
const M = window.MOCK;

/* ==========================================================================
   DASHBOARD
   ========================================================================== */
function DashboardView() {
  return (
    <div className="dash">
      {/* KPI row */}
      <div className="span-12" style={{ display: "grid", gridTemplateColumns: "repeat(6, 1fr)", gap: 8 }}>
        <MetricTile
          label="CONNECTED PEERS" kind=""
          value={M.KPIS.peers.val}
          delta={M.KPIS.peers.delta}
          sub={M.KPIS.peers.sub}
          spark={M.SPARK.peers}
        />
        <MetricTile
          label="ACTIVE TRANSFERS" kind="info"
          value={M.KPIS.transfers.val}
          delta={M.KPIS.transfers.delta}
          sub={M.KPIS.transfers.sub}
          spark={M.SPARK.transfers}
          sparkColor="var(--info)"
        />
        <MetricTile
          label="LOCAL DATASETS" kind="pos"
          value={M.KPIS.datasets.val.toLocaleString()}
          delta={M.KPIS.datasets.delta}
          sub={M.KPIS.datasets.sub}
          spark={M.SPARK.datasets}
          sparkColor="var(--pos)"
        />
        <MetricTile
          label="SYMBOLS INDEXED" kind=""
          value={M.KPIS.symbols.val.toLocaleString()}
          delta={M.KPIS.symbols.delta}
          sub={M.KPIS.symbols.sub}
          spark={M.SPARK.symbols}
        />
        <MetricTile
          label="SERVER UPTIME" kind="pos"
          value={M.KPIS.uptime.val.toFixed(2)} unit="%"
          delta={M.KPIS.uptime.delta} deltaPct={undefined}
          sub={M.KPIS.uptime.sub}
          spark={M.SPARK.uptime}
          sparkColor="var(--pos)"
        />
        <MetricTile
          label="SEARCH HIT-RATE" kind="warn"
          value={M.KPIS.hitrate.val.toFixed(1)} unit="%"
          delta={M.KPIS.hitrate.delta}
          sub={M.KPIS.hitrate.sub}
          spark={M.SPARK.hitrate}
          sparkColor="var(--warn)"
        />
      </div>

      {/* Market table */}
      <div className="span-8" style={{ display: "flex", minHeight: 360 }}>
        <Panel
          title="Market Watch · EOD"
          tag="LIVE"
          sub="15 SYMBOLS · LAST UPDATE 21:04:11"
          flush
          actions={
            <>
              <div className="btn-group">
                <button className="btn sm active">1D</button>
                <button className="btn sm">1W</button>
                <button className="btn sm">1M</button>
                <button className="btn sm">1Y</button>
              </div>
              <button className="btn sm ghost icon"><Icon name="filter" size={12} /></button>
              <button className="btn sm ghost icon"><Icon name="refresh" size={12} /></button>
            </>
          }
          style={{ flex: 1 }}
        >
          <EODTable rows={M.EOD} compact />
        </Panel>
      </div>

      {/* Network health */}
      <div className="span-4" style={{ display: "flex", flexDirection: "column", gap: 8, minHeight: 360 }}>
        <Panel
          title="Network Health"
          tag="DHT"
          actions={<button className="btn sm ghost icon"><Icon name="refresh" size={12} /></button>}
          style={{ flex: 1 }}
        >
          <NetworkHealth />
        </Panel>
      </div>

      {/* Recent activity / transfers */}
      <div className="span-7" style={{ display: "flex", minHeight: 260 }}>
        <Panel
          title="Active Transfers"
          tag="TX"
          sub="9 IN FLIGHT"
          flush
          actions={
            <>
              <button className="btn sm ghost"><Icon name="filter" size={11} /> Filter</button>
              <button className="btn sm ghost icon"><Icon name="refresh" size={12} /></button>
            </>
          }
          style={{ flex: 1 }}
        >
          <TransfersTable rows={M.TRANSFERS.slice(0, 6)} compact />
        </Panel>
      </div>

      <div className="span-5" style={{ display: "flex", minHeight: 260 }}>
        <Panel title="Top Providers" tag="REP" style={{ flex: 1 }} flush>
          <TopProvidersTable rows={M.PEERS.filter((p) => p.conn).slice(0, 6)} />
        </Panel>
      </div>
    </div>
  );
}

/* ==========================================================================
   NETWORK HEALTH (dashboard right column)
   ========================================================================== */
function NetworkHealth() {
  const rows = [
    { lbl: "Inbound",  v: "4.62 MB/s", b: 62, kind: "info" },
    { lbl: "Outbound", v: "1.84 MB/s", b: 24, kind: "warn" },
    { lbl: "DHT Lookup", v: "18 ms p50", b: 88, kind: "pos" },
    { lbl: "Stream RTT", v: "44 ms p95", b: 71, kind: "pos" },
    { lbl: "Validation Q", v: "3 / 64", b: 5, kind: "info" },
    { lbl: "Replication", v: "2.8× factor", b: 56, kind: "warn" },
  ];
  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 10 }}>
      {rows.map((r, i) => (
        <div key={i}>
          <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 4 }}>
            <span style={{ fontSize: 10.5, letterSpacing: "0.06em", color: "var(--text-dim)" }}>
              {r.lbl.toUpperCase()}
            </span>
            <span className="mono" style={{ fontSize: 11, color: "var(--text-bright)" }}>{r.v}</span>
          </div>
          <ProgressBar value={r.b} status={r.kind === "pos" ? "completed" : r.kind === "warn" ? "pending" : ""} />
        </div>
      ))}

      <div style={{ marginTop: 6, paddingTop: 10, borderTop: "1px solid var(--border)" }}>
        <div style={{ display: "flex", justifyContent: "space-between", marginBottom: 8 }}>
          <span style={{ fontSize: 10.5, letterSpacing: "0.14em", color: "var(--text-dim)" }}>GEO DISTRIBUTION</span>
          <span className="mono" style={{ fontSize: 10.5, color: "var(--text-mute)" }}>47 NODES</span>
        </div>
        <div style={{ display: "grid", gridTemplateColumns: "1fr 1fr", gap: 6 }}>
          {[
            ["NYC", 14], ["FRA", 11], ["TYO", 7], ["AMS", 6], ["SGP", 5], ["SAO", 3], ["LON", 1], ["SYD", 0],
          ].map(([g, n]) => (
            <div key={g} style={{ display: "flex", alignItems: "center", gap: 6, fontFamily: "var(--font-mono)", fontSize: 10.5 }}>
              <span style={{ color: "var(--accent-text)", width: 32 }}>{g}</span>
              <span style={{ flex: 1, height: 3, background: "var(--bg-deep)" }}>
                <span style={{ display: "block", height: "100%", width: `${(n / 14) * 100}%`, background: "var(--accent)" }} />
              </span>
              <span style={{ color: "var(--text-dim)", width: 18, textAlign: "right" }}>{n}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

/* ==========================================================================
   EOD TABLE
   ========================================================================== */
function EODTable({ rows, compact, onError }) {
  return (
    <table className="dense-table">
      <thead>
        <tr>
          <th>Symbol</th>
          <th className="num">Last</th>
          <th className="num">Chg</th>
          <th className="num">%Chg</th>
          <th className="num">Open</th>
          <th className="num">High</th>
          <th className="num">Low</th>
          <th className="num">Prev Close</th>
          <th className="num">Volume</th>
          <th>Source</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((r, i) => {
          const positive = r.chg.startsWith("+");
          return (
            <tr key={r.symbol} className={positive ? "row-pos" : "row-neg"}>
              <td className="sym">{r.symbol}</td>
              <td className="num bright flash">{r.close}</td>
              <td className={`num ${positive ? "pos" : "neg"}`}>{r.chg}</td>
              <td className={`num ${positive ? "pos" : "neg"}`}>{r.pct}%</td>
              <td className="num dim">{r.open}</td>
              <td className="num">{r.high}</td>
              <td className="num">{r.low}</td>
              <td className="num dim">{r.prevClose}</td>
              <td className="num">{r.volume}</td>
              <td className="dim">peer-{["7a3f", "c812", "2f1a", "9d4e", "44b8"][i % 5]}</td>
            </tr>
          );
        })}
      </tbody>
    </table>
  );
}

/* ==========================================================================
   TRANSFERS TABLE (dashboard variant + full)
   ========================================================================== */
function TransfersTable({ rows, compact, onCancel }) {
  return (
    <table className="dense-table">
      <thead>
        <tr>
          <th style={{ width: 28 }}></th>
          <th>Tx ID</th>
          <th>Symbol</th>
          <th>Type</th>
          <th>Source → Dest</th>
          <th style={{ width: 140 }}>Progress</th>
          <th className="num">Speed</th>
          <th className="num">Size</th>
          <th className="num">ETA</th>
          <th>Status</th>
          {!compact && <th></th>}
        </tr>
      </thead>
      <tbody>
        {rows.map((t) => (
          <tr key={t.id}>
            <td>
              <Icon
                name={t.dir === "in" ? "arrow-down" : "arrow-up"}
                size={12}
                style={{ color: t.dir === "in" ? "var(--info)" : "var(--accent)" }}
              />
            </td>
            <td className="dim mono">{t.id}</td>
            <td className="sym">{t.sym}</td>
            <td className="dim">{t.type}</td>
            <td className="mono" style={{ fontSize: 11 }}>
              <span style={{ color: "var(--text)" }}>{t.src}</span>
              <span style={{ color: "var(--text-mute)" }}> → </span>
              <span style={{ color: "var(--text)" }}>{t.dest}</span>
            </td>
            <td>
              <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
                <ProgressBar
                  value={t.progress}
                  status={t.status}
                  striped={t.status === "transferring"}
                />
                <span className="mono" style={{ fontSize: 10.5, color: "var(--text-dim)", width: 32, textAlign: "right" }}>
                  {t.progress}%
                </span>
              </div>
            </td>
            <td className="num dim">
              {t.speed > 0 ? `${(t.speed / 1024 / 1024).toFixed(2)} MB/s` : "—"}
            </td>
            <td className="num dim">{t.size}</td>
            <td className="num dim">{t.eta}</td>
            <td>
              <StatusBadge
                kind={
                  t.status === "completed" ? "pos" :
                  t.status === "failed" ? "neg" :
                  t.status === "transferring" ? "info" :
                  "warn"
                }
              >
                {t.status}
              </StatusBadge>
            </td>
            {!compact && (
              <td style={{ textAlign: "right" }}>
                <button className="btn sm ghost icon" title="Cancel" onClick={() => onCancel?.(t.id)}>
                  <Icon name="x" size={11} />
                </button>
              </td>
            )}
          </tr>
        ))}
      </tbody>
    </table>
  );
}

/* ==========================================================================
   TOP PROVIDERS
   ========================================================================== */
function TopProvidersTable({ rows }) {
  return (
    <table className="dense-table">
      <thead>
        <tr>
          <th>Peer</th>
          <th>Geo</th>
          <th className="num">Reputation</th>
          <th className="num">RTT</th>
          <th>Shared</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((p) => (
          <tr key={p.id}>
            <td>
              <span className="mono" style={{ color: "var(--text-bright)" }}>{p.id.slice(0, 14)}</span>
            </td>
            <td className="dim">{p.geo}</td>
            <td className="num"><ReputationBar value={p.rep} /></td>
            <td className="num dim">{p.rttMs} ms</td>
            <td className="mono dim" style={{ fontSize: 10.5 }}>{p.shared}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

/* ==========================================================================
   SEARCH VIEW
   ========================================================================== */
function SearchView() {
  const [dataType, setDataType] = useState("EOD");
  const [symbol, setSymbol] = useState("AAPL");
  const [start, setStart] = useState("2018-01-02");
  const [end, setEnd] = useState("2026-05-22");
  const [granularity, setGranularity] = useState("DAILY");
  const [hasResults, setHasResults] = useState(true);
  const [selected, setSelected] = useState(null);
  const [downloading, setDownloading] = useState({});

  const onSearch = () => {
    setHasResults(true);
  };
  const onDownload = (peerId) => {
    setDownloading((d) => ({ ...d, [peerId]: true }));
    setTimeout(() => setDownloading((d) => ({ ...d, [peerId]: false })), 1800);
  };
  const onClear = () => {
    setHasResults(false);
    setSelected(null);
  };

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8, padding: 8, flex: 1, minHeight: 0, overflow: "auto" }}>
      {/* Compact toolbar */}
      <div className="toolbar">
        <div className="field" style={{ flex: "0 0 140px" }}>
          <label className="lbl">Data Type</label>
          <div className="select">
            <select value={dataType} onChange={(e) => setDataType(e.target.value)}>
              <option value="EOD">End of Day</option>
              <option value="DIVIDEND">Dividends</option>
              <option value="INSIDER_TRADE">Insider Trading</option>
              <option value="SPLIT">Splits</option>
            </select>
          </div>
        </div>

        <div className="field" style={{ flex: "0 0 160px" }}>
          <label className="lbl">Symbol</label>
          <input
            className="input"
            placeholder="e.g. AAPL"
            value={symbol}
            onChange={(e) => setSymbol(e.target.value.toUpperCase())}
          />
        </div>

        <div className="field" style={{ flex: "0 0 140px" }}>
          <label className="lbl">Start Date</label>
          <input type="date" className="input" value={start} onChange={(e) => setStart(e.target.value)} />
        </div>

        <div className="field" style={{ flex: "0 0 140px" }}>
          <label className="lbl">End Date</label>
          <input type="date" className="input" value={end} onChange={(e) => setEnd(e.target.value)} />
        </div>

        <div className="field" style={{ flex: "0 0 120px" }}>
          <label className="lbl">Granularity</label>
          <div className="select">
            <select value={granularity} onChange={(e) => setGranularity(e.target.value)}>
              <option>DAILY</option>
              <option>WEEKLY</option>
              <option>MONTHLY</option>
              <option>YEARLY</option>
            </select>
          </div>
        </div>

        <div style={{ flex: 1 }} />

        <div className="field">
          <label className="lbl">&nbsp;</label>
          <div style={{ display: "flex", gap: 6 }}>
            <button className="btn ghost" onClick={onClear}><Icon name="x" size={12} /> Clear</button>
            <button className="btn ghost"><Icon name="refresh" size={12} /> Refresh</button>
            <button className="btn primary" onClick={onSearch}>
              <Icon name="search" size={12} /> Search Network
            </button>
          </div>
        </div>
      </div>

      {/* Query summary */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 12,
          padding: "6px 12px",
          background: "var(--bg-panel-hi)",
          border: "1px solid var(--border)",
          fontFamily: "var(--font-mono)",
          fontSize: 11,
          color: "var(--text-dim)",
        }}
      >
        <span style={{ color: "var(--accent-text)", letterSpacing: "0.14em" }}>QUERY ›</span>
        <span style={{ color: "var(--text-bright)" }}>{dataType}</span>
        <span>·</span>
        <span style={{ color: "var(--text-bright)" }}>{symbol}</span>
        <span>·</span>
        <span style={{ color: "var(--text-bright)" }}>{start} → {end}</span>
        <span>·</span>
        <span style={{ color: "var(--text-bright)" }}>{granularity}</span>
        <span style={{ flex: 1 }} />
        <StatusBadge kind="info">{hasResults ? `${M.SEARCH_RESULTS.length} OFFERS` : "0 OFFERS"}</StatusBadge>
        <span style={{ color: "var(--text-mute)" }}>scanned 47 peers · 0.34s</span>
      </div>

      <Panel
        title="Provider Offers"
        tag="P2P"
        sub={hasResults ? "RANKED BY REPUTATION × LATENCY" : "NO ACTIVE QUERY"}
        flush
        actions={
          <>
            <button className="btn sm ghost"><Icon name="filter" size={11} /> rep ≥ 50%</button>
            <button className="btn sm ghost icon"><Icon name="more" size={12} /></button>
          </>
        }
        style={{ flex: 1, minHeight: 320 }}
      >
        {hasResults ? (
          <table className="dense-table">
            <thead>
              <tr>
                <th>Peer ID</th>
                <th>Geo</th>
                <th className="num">Reputation</th>
                <th className="num">Latency</th>
                <th className="num">Throughput</th>
                <th className="num">Rows</th>
                <th>Range Available</th>
                <th>Updated</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {M.SEARCH_RESULTS.map((r) => (
                <tr
                  key={r.peerId}
                  className={selected === r.peerId ? "selected" : ""}
                  onClick={() => setSelected(r.peerId)}
                >
                  <td><span className="mono" style={{ color: "var(--text-bright)" }}>{r.peerId}</span></td>
                  <td className="dim">{r.geo}</td>
                  <td className="num"><ReputationBar value={r.rep} /></td>
                  <td className="num"><span className={r.latency < 50 ? "pos" : r.latency < 100 ? "" : "neg"}>{r.latency} ms</span></td>
                  <td className="num">{r.speed}</td>
                  <td className="num dim">{r.rows.toLocaleString()}</td>
                  <td className="mono" style={{ fontSize: 10.5, color: "var(--text-dim)" }}>{r.range}</td>
                  <td className="dim">{r.updated}</td>
                  <td style={{ textAlign: "right" }}>
                    <button
                      className="btn sm"
                      onClick={(e) => { e.stopPropagation(); onDownload(r.peerId); }}
                      disabled={downloading[r.peerId]}
                    >
                      {downloading[r.peerId] ? (
                        <><Icon name="refresh" size={11} /> Queued</>
                      ) : (
                        <><Icon name="download" size={11} /> Download</>
                      )}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <EmptyState icon="search" title="No active query" hint="Configure the toolbar and press SEARCH NETWORK" />
        )}
      </Panel>
    </div>
  );
}

/* ==========================================================================
   MARKET DATA VIEW
   ========================================================================== */
function MarketDataView() {
  const [tab, setTab] = useState("EOD");
  return (
    <div style={{ display: "flex", flexDirection: "column", flex: 1, minHeight: 0 }}>
      <div style={{ display: "flex", alignItems: "center", borderBottom: "1px solid var(--border)" }}>
        <Tabs
          tabs={[
            { id: "EOD",          label: "End of Day",     count: M.EOD.length },
            { id: "DIVIDEND",     label: "Dividends",      count: M.DIVIDENDS.length },
            { id: "INSIDER_TRADE",label: "Insider Trades", count: M.INSIDER.length },
            { id: "SPLIT",        label: "Splits",         count: M.SPLITS.length },
          ]}
          value={tab}
          onChange={setTab}
        />
        <div style={{ flex: 1 }} />
        <div style={{ display: "flex", alignItems: "center", gap: 6, padding: "0 10px" }}>
          <span className="mono" style={{ fontSize: 10.5, color: "var(--text-mute)", letterSpacing: "0.04em" }}>
            LOCAL · 12.8 GB · {M.EOD.length + M.DIVIDENDS.length + M.INSIDER.length + M.SPLITS.length} rows shown
          </span>
          <div style={{ width: 1, alignSelf: "stretch", background: "var(--border)", margin: "0 6px" }} />
          <input className="input" style={{ width: 180, height: 24 }} placeholder="filter symbol…" />
          <button className="btn sm ghost"><Icon name="filter" size={11} /> Source</button>
          <button className="btn sm ghost"><Icon name="download" size={11} /> Export</button>
          <button className="btn sm"><Icon name="upload" size={11} /> Upload CSV</button>
        </div>
      </div>

      <div style={{ flex: 1, minHeight: 0, padding: 8 }}>
        <Panel
          title={
            tab === "EOD" ? "End of Day · Price History" :
            tab === "DIVIDEND" ? "Dividends · Cash & Stock" :
            tab === "INSIDER_TRADE" ? "Insider Trades · SEC Form 4" :
            "Splits · Historical Ratios"
          }
          tag={tab}
          sub="LIVE · 5s POLL"
          flush
          actions={
            <>
              <div className="btn-group">
                <button className="btn sm">Daily</button>
                <button className="btn sm active">Weekly</button>
                <button className="btn sm">Monthly</button>
              </div>
              <button className="btn sm ghost icon"><Icon name="refresh" size={12} /></button>
            </>
          }
          style={{ height: "100%" }}
        >
          {tab === "EOD" && <EODTable rows={M.EOD} />}
          {tab === "DIVIDEND" && <DividendTable rows={M.DIVIDENDS} />}
          {tab === "INSIDER_TRADE" && <InsiderTable rows={M.INSIDER} />}
          {tab === "SPLIT" && <SplitsTable rows={M.SPLITS} />}
        </Panel>
      </div>
    </div>
  );
}

function DividendTable({ rows }) {
  return (
    <table className="dense-table">
      <thead>
        <tr>
          <th>Symbol</th>
          <th>Ex-Date</th>
          <th className="num">Stock Price</th>
          <th className="num">Dividend</th>
          <th className="num">Yield</th>
          <th>Type</th>
          <th>Currency</th>
          <th>Source</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((d, i) => (
          <tr key={i}>
            <td className="sym">{d.sym}</td>
            <td className="dim">{d.date}</td>
            <td className="num">{d.price.toFixed(2)}</td>
            <td className="num bright">{d.amount.toFixed(2)}</td>
            <td className="num pos">{((d.amount * 4 / d.price) * 100).toFixed(2)}%</td>
            <td><StatusBadge kind={d.type === "Cash" ? "pos" : "default"} dot={false}>{d.type}</StatusBadge></td>
            <td className="dim">{d.currency}</td>
            <td className="mono dim" style={{ fontSize: 10.5 }}>{d.src}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function InsiderTable({ rows }) {
  return (
    <table className="dense-table">
      <thead>
        <tr>
          <th>Symbol</th>
          <th>Date</th>
          <th>Insider</th>
          <th>Position</th>
          <th>Tx</th>
          <th className="num">Shares</th>
          <th className="num">Price</th>
          <th className="num">Value (USD)</th>
          <th>Form</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((r, i) => (
          <tr key={i}>
            <td className="sym">{r.sym}</td>
            <td className="dim">{r.date}</td>
            <td className="bright">{r.name}</td>
            <td className="dim">{r.pos}</td>
            <td>
              <StatusBadge kind={r.ttype === "BUY" ? "pos" : "neg"} dot={false}>{r.ttype}</StatusBadge>
            </td>
            <td className="num">{r.shares.toLocaleString()}</td>
            <td className="num">{r.price.toFixed(2)}</td>
            <td className={`num ${r.ttype === "BUY" ? "pos" : "neg"}`}>
              {r.ttype === "BUY" ? "+" : "-"}${r.value.toLocaleString()}
            </td>
            <td className="dim">{r.form}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

function SplitsTable({ rows }) {
  return (
    <table className="dense-table">
      <thead>
        <tr>
          <th>Symbol</th>
          <th>Date</th>
          <th className="num">Ratio</th>
          <th className="num">Pre-Split Price</th>
          <th className="num">Post-Split Price</th>
          <th>Source</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((s, i) => (
          <tr key={i}>
            <td className="sym">{s.sym}</td>
            <td className="dim">{s.date}</td>
            <td className="num"><StatusBadge kind="accent" dot={false}>{s.ratio}</StatusBadge></td>
            <td className="num dim">{s.prev.toFixed(2)}</td>
            <td className="num bright">{s.post.toFixed(2)}</td>
            <td className="mono dim" style={{ fontSize: 10.5 }}>{s.src}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}

/* ==========================================================================
   TRANSFERS VIEW
   ========================================================================== */
function TransfersView() {
  const [filter, setFilter] = useState("all");
  const filtered = useMemo(() => {
    if (filter === "all") return M.TRANSFERS;
    if (filter === "active") return M.TRANSFERS.filter((t) => t.status === "transferring" || t.status === "pending");
    if (filter === "done") return M.TRANSFERS.filter((t) => t.status === "completed");
    if (filter === "fail") return M.TRANSFERS.filter((t) => t.status === "failed");
    return M.TRANSFERS;
  }, [filter]);

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8, padding: 8, flex: 1, minHeight: 0 }}>
      {/* mini-KPIs */}
      <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 8 }}>
        <MetricTile label="ACTIVE" kind="info" value={M.TRANSFERS.filter(t=>t.status==="transferring").length} sub="2 IN · 2 OUT" />
        <MetricTile label="QUEUED" kind="warn" value={M.TRANSFERS.filter(t=>t.status==="pending").length} sub="awaiting capacity" />
        <MetricTile label="COMPLETED (24H)" kind="pos" value="38" sub="2.1 GB total" />
        <MetricTile label="FAILED (24H)" kind="neg" value="3" sub="1 retry · 2 abandoned" />
      </div>

      <div style={{ display: "flex", alignItems: "center", gap: 6 }}>
        <div className="btn-group">
          <button className={`btn sm ${filter === "all" ? "active" : ""}`} onClick={() => setFilter("all")}>All <span className="kbd" style={{marginLeft:6}}>{M.TRANSFERS.length}</span></button>
          <button className={`btn sm ${filter === "active" ? "active" : ""}`} onClick={() => setFilter("active")}>Active</button>
          <button className={`btn sm ${filter === "done" ? "active" : ""}`} onClick={() => setFilter("done")}>Completed</button>
          <button className={`btn sm ${filter === "fail" ? "active" : ""}`} onClick={() => setFilter("fail")}>Failed</button>
        </div>
        <div style={{ flex: 1 }} />
        <button className="btn sm ghost"><Icon name="refresh" size={11} /> Refresh</button>
        <button className="btn sm danger"><Icon name="x" size={11} /> Cancel All</button>
      </div>

      <Panel title="Transfer Queue" tag="TX" sub={`${filtered.length} ROWS`} flush style={{ flex: 1, minHeight: 0 }}>
        <TransfersTable rows={filtered} />
      </Panel>
    </div>
  );
}

/* ==========================================================================
   SCRIPTS VIEW
   ========================================================================== */
function ScriptsView() {
  const [q, setQ] = useState("");
  const [filter, setFilter] = useState("all");
  const filtered = useMemo(() => {
    return M.SCRIPTS.filter((s) => {
      if (filter === "installed" && !s.isInstalled) return false;
      if (filter === "available" && s.isInstalled) return false;
      if (q && !s.name.toLowerCase().includes(q.toLowerCase())) return false;
      return true;
    });
  }, [q, filter]);

  return (
    <div style={{ display: "flex", flexDirection: "column", gap: 8, padding: 8, flex: 1, minHeight: 0 }}>
      <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
        <div className="select" style={{ width: 160 }}>
          <select value={filter} onChange={(e) => setFilter(e.target.value)}>
            <option value="all">All Scripts</option>
            <option value="installed">Installed</option>
            <option value="available">Available</option>
          </select>
        </div>
        <input className="input" style={{ width: 260 }} placeholder="search scripts…" value={q} onChange={(e) => setQ(e.target.value)} />
        <div style={{ flex: 1 }} />
        <button className="btn ghost"><Icon name="download" size={12} /> Sync Repo</button>
        <button className="btn primary"><Icon name="upload" size={12} /> Upload Script</button>
      </div>

      <Panel title="Data Collection Scripts" tag="EXEC" sub={`${filtered.length} OF ${M.SCRIPTS.length}`} flush style={{ flex: 1, minHeight: 0 }}>
        <table className="dense-table">
          <thead>
            <tr>
              <th>Name</th>
              <th>Status</th>
              <th>Data Type</th>
              <th>Source</th>
              <th className="mono">Schedule</th>
              <th>Last Run</th>
              <th>Next Run</th>
              <th className="num">Votes</th>
              <th>Author</th>
              <th style={{ textAlign: "right" }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {filtered.map((s) => {
              const lastRun = s.lastRun ? new Date(s.lastRun).toISOString().slice(0, 16).replace("T", " ") : "—";
              const nextRun = s.nextRun ? new Date(s.nextRun).toISOString().slice(0, 16).replace("T", " ") : "—";
              return (
                <tr key={s.id}>
                  <td className="bright">
                    <span style={{ display: "inline-flex", alignItems: "center", gap: 8 }}>
                      <Icon name="code" size={12} style={{ color: "var(--accent-text)" }} />
                      {s.name}
                    </span>
                  </td>
                  <td>
                    <StatusBadge
                      kind={
                        s.status === "running" ? "info" :
                        s.status === "scheduled" ? "warn" :
                        s.status === "failed" ? "neg" :
                        s.status === "available" ? "default" :
                        "pos"
                      }
                    >
                      {s.status}
                    </StatusBadge>
                  </td>
                  <td className="dim">{s.dataType}</td>
                  <td className="dim">{s.source}</td>
                  <td className="dim">{s.schedule}</td>
                  <td className="dim">{lastRun}</td>
                  <td className="dim">{nextRun}</td>
                  <td className="num" style={{ color: s.votes >= 0 ? "var(--pos)" : "var(--neg)" }}>
                    {s.votes >= 0 ? "+" : ""}{s.votes}
                  </td>
                  <td className="mono dim" style={{ fontSize: 10.5 }}>{s.author}</td>
                  <td style={{ textAlign: "right" }}>
                    <div style={{ display: "inline-flex", gap: 2 }}>
                      <button className="btn sm ghost icon" title="View code"><Icon name="code" size={11} /></button>
                      <button className="btn sm ghost icon" title="Run"><Icon name={s.status === "running" ? "pause" : "play"} size={11} /></button>
                      <button className="btn sm ghost icon" title={s.isInstalled ? "Uninstall" : "Install"}>
                        <Icon name={s.isInstalled ? "trash" : "download"} size={11} />
                      </button>
                      <button className="btn sm ghost icon" title="More"><Icon name="more" size={11} /></button>
                    </div>
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </Panel>
    </div>
  );
}

/* ==========================================================================
   PEERS VIEW
   ========================================================================== */
function PeersView() {
  const [selected, setSelected] = useState(M.PEERS[0].id);
  const sel = M.PEERS.find((p) => p.id === selected) || M.PEERS[0];

  return (
    <div style={{ display: "flex", flexDirection: "column", flex: 1, minHeight: 0 }}>
      <div style={{ display: "grid", gridTemplateColumns: "repeat(4, 1fr)", gap: 8, padding: 8 }}>
        <MetricTile label="CONNECTED" kind="pos" value={M.PEERS.filter(p=>p.conn).length} sub={`of ${M.PEERS.length} known`} />
        <MetricTile label="VERIFIED PROVIDERS" value={M.PEERS.filter(p=>p.roles.includes("provider")).length} sub="signed offers" />
        <MetricTile label="VALIDATORS" kind="info" value={M.PEERS.filter(p=>p.roles.includes("validator")).length} sub="quorum 2/3" />
        <MetricTile label="AVG REPUTATION" kind="warn" value={(M.PEERS.filter(p=>p.conn).reduce((a,p)=>a+p.rep,0) / M.PEERS.filter(p=>p.conn).length * 100).toFixed(0)} unit="%" sub="weighted by rows shared" />
      </div>

      <div className="split-row" style={{ flex: 1, minHeight: 0, padding: "0 8px 8px" }}>
        <Panel
          title="Peer Directory"
          tag="DHT"
          sub={`${M.PEERS.length} KNOWN · ${M.PEERS.filter(p=>p.conn).length} ONLINE`}
          flush
          style={{ flex: 2, minWidth: 0 }}
          actions={
            <>
              <input className="input" style={{ width: 200, height: 22 }} placeholder="filter peer-id, geo, role…" />
              <button className="btn sm ghost icon"><Icon name="filter" size={11} /></button>
              <button className="btn sm ghost icon"><Icon name="refresh" size={11} /></button>
            </>
          }
        >
          <table className="dense-table">
            <thead>
              <tr>
                <th style={{ width: 16 }}></th>
                <th>Peer ID</th>
                <th>Address</th>
                <th>Geo</th>
                <th className="num">Rep</th>
                <th>Roles</th>
                <th className="num">RTT</th>
                <th>Last Seen</th>
              </tr>
            </thead>
            <tbody>
              {M.PEERS.map((p) => (
                <tr
                  key={p.id}
                  className={selected === p.id ? "selected" : ""}
                  onClick={() => setSelected(p.id)}
                >
                  <td>
                    <span className={`led ${p.conn ? "pos" : "neg"}`} />
                  </td>
                  <td className="mono bright">{p.id}</td>
                  <td className="dim">{p.addr}</td>
                  <td className="dim">{p.geo}</td>
                  <td className="num"><ReputationBar value={p.rep} /></td>
                  <td>
                    <span style={{ display: "inline-flex", gap: 4, flexWrap: "wrap" }}>
                      {p.roles.map((r) => (
                        <StatusBadge
                          key={r}
                          dot={false}
                          kind={r === "validator" ? "info" : r === "provider" ? "accent" : "default"}
                        >
                          {r}
                        </StatusBadge>
                      ))}
                    </span>
                  </td>
                  <td className="num dim">{p.conn ? `${p.rttMs} ms` : "—"}</td>
                  <td className="dim">{formatRelative(p.lastSeen)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </Panel>

        <Panel
          title="Peer Detail"
          tag={sel.geo}
          sub={sel.conn ? "ONLINE" : "OFFLINE"}
          style={{ flex: 1, minWidth: 280, maxWidth: 360 }}
          actions={<button className="btn sm ghost icon"><Icon name="more" size={11} /></button>}
        >
          <PeerDetail peer={sel} />
        </Panel>
      </div>
    </div>
  );
}

function formatRelative(d) {
  const ts = typeof d === "string" ? new Date(d) : d;
  const diff = (Date.now() - ts.getTime()) / 1000;
  if (diff < 60) return `${Math.floor(diff)}s ago`;
  if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
  if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
  return `${Math.floor(diff / 86400)}d ago`;
}

function PeerDetail({ peer }) {
  const rows = [
    ["PEER ID", peer.id],
    ["ADDRESS", peer.addr],
    ["GEO", peer.geo],
    ["RTT", peer.conn ? `${peer.rttMs} ms` : "—"],
    ["UPTIME", peer.uptime],
    ["SHARED", peer.shared],
    ["LAST SEEN", formatRelative(peer.lastSeen)],
  ];
  return (
    <div>
      <div style={{ display: "flex", alignItems: "center", gap: 8, marginBottom: 12 }}>
        <span className={`led ${peer.conn ? "pos pulse" : "neg"}`} style={{ width: 9, height: 9 }} />
        <span className="mono" style={{ color: "var(--text-bright)", fontSize: 12 }}>{peer.id}</span>
      </div>

      <div style={{ display: "flex", flexDirection: "column", gap: 4, marginBottom: 14 }}>
        {rows.map(([k, v]) => (
          <div key={k} style={{ display: "flex", fontFamily: "var(--font-mono)", fontSize: 11 }}>
            <span style={{ color: "var(--text-mute)", letterSpacing: "0.1em", width: 86 }}>{k}</span>
            <span style={{ color: "var(--text)" }}>{v}</span>
          </div>
        ))}
      </div>

      <div style={{ marginBottom: 12 }}>
        <div style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--text-mute)", marginBottom: 6 }}>REPUTATION</div>
        <ReputationBar value={peer.rep} />
      </div>

      <div style={{ marginBottom: 12 }}>
        <div style={{ fontSize: 10, letterSpacing: "0.14em", color: "var(--text-mute)", marginBottom: 6 }}>ROLES</div>
        <div style={{ display: "flex", gap: 4, flexWrap: "wrap" }}>
          {peer.roles.map((r) => (
            <StatusBadge key={r} dot={false} kind={r === "validator" ? "info" : r === "provider" ? "accent" : "default"}>{r}</StatusBadge>
          ))}
        </div>
      </div>

      <div style={{ display: "flex", gap: 6, marginTop: 16 }}>
        <button className="btn sm primary" style={{ flex: 1 }} disabled={!peer.conn}>
          <Icon name="search" size={11} /> Query
        </button>
        <button className="btn sm" style={{ flex: 1 }} disabled={!peer.conn}>
          <Icon name="thumbs-up" size={11} /> Vote
        </button>
        <button className="btn sm danger icon"><Icon name="x" size={11} /></button>
      </div>
    </div>
  );
}

Object.assign(window, {
  DashboardView, SearchView, MarketDataView, TransfersView, ScriptsView, PeersView,
  EODTable, TransfersTable, TopProvidersTable, DividendTable, InsiderTable, SplitsTable,
  NetworkHealth, formatRelative, PeerDetail,
});

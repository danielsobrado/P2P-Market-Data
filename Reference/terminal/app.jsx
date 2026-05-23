/* eslint-disable */
const { useState, useEffect } = React;

const TERMINAL_DEFAULTS = /*EDITMODE-BEGIN*/{
  "accent": "amber",
  "density": "default",
  "showTicker": true,
  "theme": "dark"
}/*EDITMODE-END*/;

const ACCENT_PALETTES = {
  amber:  { accent: "#ff9a1f", soft: "rgba(255,154,31,0.12)",  line: "rgba(255,154,31,0.45)", text: "#ffb35a" },
  orange: { accent: "#ff6f3c", soft: "rgba(255,111,60,0.12)",  line: "rgba(255,111,60,0.45)", text: "#ff8f6a" },
  cyan:   { accent: "#4dd0e1", soft: "rgba(77,208,225,0.12)",  line: "rgba(77,208,225,0.45)", text: "#7ee0ee" },
  lime:   { accent: "#a5e635", soft: "rgba(165,230,53,0.12)",  line: "rgba(165,230,53,0.45)", text: "#c8f06b" },
};

function TerminalApp() {
  const [tw, setTweak] = useTweaks(TERMINAL_DEFAULTS);

  const [view, setView] = useState("dashboard");
  const [isConnected, setIsConnected] = useState(true);
  const [serverStatus, setServerStatus] = useState("ok");
  const [lastRefresh, setLastRefresh] = useState(() => new Date());

  // Mock the existing GetServerStatus poll
  useEffect(() => {
    const i = setInterval(() => {
      setLastRefresh(new Date());
      // tiny chance of glitching to "warn" — keeps the strip alive
      const r = Math.random();
      setServerStatus(r < 0.05 ? "warn" : "ok");
    }, 5000);
    return () => clearInterval(i);
  }, []);

  // Keyboard: F1–F6 nav
  useEffect(() => {
    const onKey = (e) => {
      const map = { F1: "dashboard", F2: "search", F3: "market", F4: "transfers", F5: "scripts", F6: "peers" };
      if (map[e.key]) {
        e.preventDefault();
        setView(map[e.key]);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  // Apply tweakable CSS vars
  useEffect(() => {
    const pal = ACCENT_PALETTES[tw.accent] || ACCENT_PALETTES.amber;
    const root = document.documentElement;
    root.style.setProperty("--accent", pal.accent);
    root.style.setProperty("--accent-soft", pal.soft);
    root.style.setProperty("--accent-line", pal.line);
    root.style.setProperty("--accent-text", pal.text);
    root.setAttribute("data-density", tw.density);
    root.setAttribute("data-theme", tw.theme);
  }, [tw.accent, tw.density, tw.theme]);

  const viewLabel = {
    dashboard: "DASHBOARD",
    search: "SEARCH",
    market: "MARKET DATA",
    transfers: "TRANSFERS",
    scripts: "SCRIPTS",
    peers: "PEERS",
  }[view];

  const refreshAgo = secondsAgo(lastRefresh);

  return (
    <div className="shell" data-screen-label={`01 ${viewLabel}`}>
      <StatusBar
        isConnected={isConnected}
        serverStatus={serverStatus}
        view={viewLabel}
        lastRefresh={`${refreshAgo}s AGO`}
        theme={tw.theme}
        onToggleTheme={() => setTweak("theme", tw.theme === "dark" ? "light" : "dark")}
      />

      {tw.showTicker && <TickerStrip items={window.MOCK.TICKER} />}

      <div className="shell-body" style={tw.showTicker ? null : { gridRow: "2 / span 2" }}>
        <SideNav
          active={view}
          onSelect={setView}
          peersCount={window.MOCK.PEERS.filter((p) => p.conn).length}
          transferCount={window.MOCK.TRANSFERS.filter((t) => t.status === "transferring").length}
        />

        <div className="main-pane">
          {view === "dashboard" && <DashboardView />}
          {view === "search"    && <SearchView />}
          {view === "market"    && <MarketDataView />}
          {view === "transfers" && <TransfersView />}
          {view === "scripts"   && <ScriptsView />}
          {view === "peers"     && <PeersView />}
        </div>
      </div>

      <LogStrip lines={window.MOCK.LOG_LINES} />

      <TerminalTweaks tw={tw} setTweak={setTweak} />
    </div>
  );
}

function secondsAgo(d) {
  return Math.max(1, Math.floor((Date.now() - d.getTime()) / 1000));
}

/* --------------------------------------------------------------------------
   TWEAKS PANEL — wraps the starter component
   -------------------------------------------------------------------------- */
function TerminalTweaks({ tw, setTweak }) {
  return (
    <TweaksPanel title="Tweaks">
      <TweakSection label="Accent" />
      <TweakRadio
        label="Color"
        value={tw.accent}
        onChange={(v) => setTweak("accent", v)}
        options={["amber", "orange", "cyan", "lime"]}
      />

      <TweakSection label="Layout" />
      <TweakRadio
        label="Density"
        value={tw.density}
        onChange={(v) => setTweak("density", v)}
        options={["compact", "default", "comfortable"]}
      />
      <TweakToggle
        label="Live ticker strip"
        value={tw.showTicker}
        onChange={(v) => setTweak("showTicker", v)}
      />

      <TweakSection label="Theme" />
      <TweakRadio
        label="Mode"
        value={tw.theme}
        onChange={(v) => setTweak("theme", v)}
        options={["dark", "light"]}
      />
    </TweaksPanel>
  );
}

ReactDOM.createRoot(document.getElementById("root")).render(<TerminalApp />);

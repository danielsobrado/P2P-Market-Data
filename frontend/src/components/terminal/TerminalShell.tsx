import { useEffect, useState } from 'react'
import { useTerminalData } from '@/hooks/useTerminalData'
import {
  LogStrip,
  SideNav,
  StatusBar,
  TickerStrip,
  type NavId,
} from './TerminalComponents'
import { DashboardView } from './views/DashboardView'
import { SearchView } from './views/SearchView'
import { MarketDataView } from './views/MarketDataView'
import { TransfersView } from './views/TransfersView'
import { ScriptsView } from './views/ScriptsView'
import { PeersView } from './views/PeersView'

const VIEW_LABELS: Record<NavId, string> = {
  dashboard: 'DASHBOARD',
  search: 'SEARCH',
  market: 'MARKET DATA',
  transfers: 'TRANSFERS',
  scripts: 'SCRIPTS',
  peers: 'PEERS',
}

const FKEY_MAP: Record<string, NavId> = {
  F1: 'dashboard',
  F2: 'search',
  F3: 'market',
  F4: 'transfers',
  F5: 'scripts',
  F6: 'peers',
}

export function TerminalShell() {
  const data = useTerminalData()
  const [view, setView] = useState<NavId>('dashboard')

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (FKEY_MAP[e.key]) {
        e.preventDefault()
        setView(FKEY_MAP[e.key])
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  const activeTransfers = data.transfers.filter(
    (t) => t.status === 'transferring' || t.status === 'pending',
  ).length

  return (
    <div className="terminal-app">
      <div className="shell">
        <StatusBar
          isConnected={data.isConnected}
          serverStatus={data.serverStatus}
          view={VIEW_LABELS[view]}
          lastRefresh={data.lastRefresh}
        />

        <TickerStrip items={data.tickerItems} live={data.isConnected} />

        <div className="shell-body">
          <SideNav
            active={view}
            onSelect={setView}
            peersCount={data.kpis.connectedPeers}
            transferCount={activeTransfers}
          />

          <div className="main-pane">
            {view === 'dashboard' && <DashboardView data={data} />}
            {view === 'search' && <SearchView data={data} />}
            {view === 'market' && <MarketDataView data={data} />}
            {view === 'transfers' && <TransfersView data={data} />}
            {view === 'scripts' && <ScriptsView data={data} />}
            {view === 'peers' && <PeersView data={data} />}
          </div>
        </div>

        <LogStrip lines={data.logLines} />
      </div>
    </div>
  )
}

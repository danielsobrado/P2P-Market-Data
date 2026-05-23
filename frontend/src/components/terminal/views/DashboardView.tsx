import { RefreshCw } from 'lucide-react'
import type { TerminalData } from '@/hooks/useTerminalData'
import { averageReputation, topPeersByReputation } from '@/lib/terminal/adapters'
import { formatBytes, formatNumber, formatRelativeTime, formatSpeed, transferStatusKind } from '@/lib/terminal/formatters'
import {
  EmptyState,
  MetricTile,
  Panel,
  ReputationBar,
  StatusBadge,
  TerminalProgress,
} from '../TerminalComponents'
import type { TerminalEODRow, TerminalPeer, TerminalTransfer } from '@/lib/terminal/types'

export function DashboardView({ data }: { data: TerminalData }) {
  const { kpis, eodData, transfers, peers, isConnected, serverStatus } = data
  const activeTransfers = transfers.filter(
    (t) => t.status === 'transferring' || t.status === 'pending',
  )
  const topPeers = topPeersByReputation(peers.filter((p) => p.isConnected))

  return (
    <div className="dash">
      <div
        className="span-12"
        style={{ display: 'grid', gridTemplateColumns: 'repeat(6, 1fr)', gap: 8 }}
      >
        <MetricTile
          label="CONNECTED PEERS"
          value={kpis.connectedPeers}
          sub={`${peers.length} known`}
          kind={kpis.connectedPeers > 0 ? 'pos' : ''}
        />
        <MetricTile
          label="ACTIVE TRANSFERS"
          value={kpis.activeTransfers}
          sub={`${transfers.length} total`}
          kind="info"
        />
        <MetricTile
          label="DATA SOURCES"
          value={kpis.dataSources}
          sub="known providers"
          kind="pos"
        />
        <MetricTile
          label="SYMBOLS INDEXED"
          value={kpis.symbols}
          sub="deduplicated"
        />
        <MetricTile
          label="SERVER"
          value={kpis.serverRunning ? 'RUNNING' : 'STOPPED'}
          sub={kpis.dbConnected ? 'DB connected' : 'DB offline'}
          kind={kpis.serverRunning && kpis.dbConnected ? 'pos' : 'neg'}
        />
        <MetricTile
          label="NETWORK"
          value={isConnected ? 'LIVE' : 'OFFLINE'}
          sub={`P2P ${kpis.p2pRunning ? 'up' : 'down'}`}
          kind={isConnected ? 'pos' : 'warn'}
        />
      </div>

      <div className="span-8" style={{ display: 'flex', minHeight: 360 }}>
        <Panel
          title="Market Watch · EOD"
          tag="LIVE"
          sub={`${eodData.length} rows · ${data.marketQuery.symbol}`}
          flush
          actions={
            <button className="btn sm ghost icon" onClick={() => data.fetchMarketData()} title="Refresh">
              <RefreshCw size={12} />
            </button>
          }
          style={{ flex: 1 }}
        >
          {eodData.length > 0 ? (
            <EODTable rows={eodData.slice(-15).reverse()} />
          ) : (
            <EmptyState title="No EOD data loaded" hint="Market data loads on startup for default symbol" />
          )}
        </Panel>
      </div>

      <div className="span-4" style={{ display: 'flex', flexDirection: 'column', gap: 8, minHeight: 360 }}>
        <Panel title="Network Health" tag="STATUS" style={{ flex: 1 }}>
          <NetworkHealth peers={peers} isConnected={isConnected} serverStatus={serverStatus} />
        </Panel>
      </div>

      <div className="span-7" style={{ display: 'flex', minHeight: 260 }}>
        <Panel
          title="Active Transfers"
          tag="TX"
          sub={`${activeTransfers.length} in flight`}
          flush
          style={{ flex: 1 }}
        >
          {activeTransfers.length > 0 ? (
            <TransfersTable rows={activeTransfers.slice(0, 6)} compact />
          ) : (
            <EmptyState title="No active transfers" hint="Request data from search to queue transfers" />
          )}
        </Panel>
      </div>

      <div className="span-5" style={{ display: 'flex', minHeight: 260 }}>
        <Panel title="Top Providers" tag="REP" flush style={{ flex: 1 }}>
          {topPeers.length > 0 ? (
            <TopProvidersTable rows={topPeers} />
          ) : (
            <EmptyState title="No connected peers" hint="Peers appear when P2P network is active" />
          )}
        </Panel>
      </div>
    </div>
  )
}

function NetworkHealth({
  peers,
  isConnected,
  serverStatus,
}: {
  peers: TerminalPeer[]
  isConnected: boolean
  serverStatus: TerminalData['serverStatus']
}) {
  const avgRep = averageReputation(peers) * 100
  const connected = peers.filter((p) => p.isConnected).length

  const rows = [
    { lbl: 'Connection', v: isConnected ? 'LIVE' : 'OFFLINE', b: isConnected ? 100 : 0, kind: isConnected ? 'pos' : 'neg' },
    { lbl: 'Database', v: serverStatus?.databaseConnected ? 'READY' : 'DOWN', b: serverStatus?.databaseConnected ? 100 : 0, kind: serverStatus?.databaseConnected ? 'pos' : 'neg' },
    { lbl: 'P2P Host', v: serverStatus?.p2pHostRunning ? 'RUNNING' : 'STOPPED', b: serverStatus?.p2pHostRunning ? 100 : 0, kind: serverStatus?.p2pHostRunning ? 'pos' : 'warn' },
    { lbl: 'Script Mgr', v: serverStatus?.scriptMgrRunning ? 'RUNNING' : 'STOPPED', b: serverStatus?.scriptMgrRunning ? 100 : 0, kind: serverStatus?.scriptMgrRunning ? 'pos' : 'warn' },
    { lbl: 'Avg Reputation', v: `${avgRep.toFixed(0)}%`, b: avgRep, kind: avgRep >= 50 ? 'pos' : 'warn' },
    { lbl: 'Connected Peers', v: `${connected} / ${peers.length}`, b: peers.length ? (connected / peers.length) * 100 : 0, kind: 'info' },
  ]

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      {rows.map((r) => (
        <div key={r.lbl}>
          <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 4 }}>
            <span style={{ fontSize: 10.5, letterSpacing: '0.06em', color: 'var(--text-dim)' }}>
              {r.lbl.toUpperCase()}
            </span>
            <span className="mono" style={{ fontSize: 11, color: 'var(--text-bright)' }}>
              {r.v}
            </span>
          </div>
          <TerminalProgress
            value={r.b}
            status={r.kind === 'pos' ? 'completed' : r.kind === 'warn' ? 'pending' : r.kind === 'neg' ? 'failed' : ''}
          />
        </div>
      ))}
    </div>
  )
}

function EODTable({ rows }: { rows: TerminalEODRow[] }) {
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
          <th className="num">Volume</th>
          <th>Source</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((r) => {
          const positive = r.change >= 0
          return (
            <tr key={`${r.symbol}-${r.date}`}>
              <td className="sym">{r.symbol}</td>
              <td className="num bright">{formatNumber(r.close)}</td>
              <td className={`num ${positive ? 'pos' : 'neg'}`}>
                {positive ? '+' : ''}
                {formatNumber(r.change)}
              </td>
              <td className={`num ${positive ? 'pos' : 'neg'}`}>
                {positive ? '+' : ''}
                {formatNumber(r.changePct)}%
              </td>
              <td className="num dim">{formatNumber(r.open)}</td>
              <td className="num">{formatNumber(r.high)}</td>
              <td className="num">{formatNumber(r.low)}</td>
              <td className="num">{r.volume.toLocaleString()}</td>
              <td className="dim">{r.source || '—'}</td>
            </tr>
          )
        })}
      </tbody>
    </table>
  )
}

function TransfersTable({ rows, compact }: { rows: TerminalTransfer[]; compact?: boolean }) {
  return (
    <table className="dense-table">
      <thead>
        <tr>
          <th>Tx ID</th>
          <th>Symbol</th>
          <th>Type</th>
          <th>Source → Dest</th>
          <th style={{ width: 140 }}>Progress</th>
          <th className="num">Speed</th>
          <th className="num">Size</th>
          <th>Status</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((t) => (
          <tr key={t.id}>
            <td className="dim mono">{t.id.slice(0, 12)}</td>
            <td className="sym">{t.symbol}</td>
            <td className="dim">{t.type}</td>
            <td className="mono" style={{ fontSize: 11 }}>
              {t.source} → {t.destination}
            </td>
            <td>
              <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                <TerminalProgress
                  value={t.progress}
                  status={t.status}
                  striped={t.status === 'transferring'}
                />
                <span className="mono" style={{ fontSize: 10.5, color: 'var(--text-dim)', width: 32, textAlign: 'right' }}>
                  {t.progress}%
                </span>
              </div>
            </td>
            <td className="num dim">{formatSpeed(t.speed)}</td>
            <td className="num dim">{formatBytes(t.size)}</td>
            <td>
              <StatusBadge kind={transferStatusKind(t.status)}>{t.status}</StatusBadge>
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

function TopProvidersTable({ rows }: { rows: TerminalPeer[] }) {
  return (
    <table className="dense-table">
      <thead>
        <tr>
          <th>Peer</th>
          <th>Address</th>
          <th className="num">Reputation</th>
          <th>Status</th>
          <th>Roles</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((p) => (
          <tr key={p.id}>
            <td>
              <span className="mono" style={{ color: 'var(--text-bright)' }}>
                {p.id.slice(0, 14)}
              </span>
            </td>
            <td className="dim">{p.address}</td>
            <td className="num">
              <ReputationBar value={p.reputation} />
            </td>
            <td>
              <StatusBadge kind={p.isConnected ? 'pos' : 'neg'}>{p.status}</StatusBadge>
            </td>
            <td className="mono dim" style={{ fontSize: 10.5 }}>
              {p.roles.join(', ') || '—'}
            </td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

export { EODTable, TransfersTable }

import { useState } from 'react'
import { Download, RefreshCw, Search, X } from 'lucide-react'
import type { TerminalData } from '@/hooks/useTerminalData'
import type { MarketDataType, TimeGranularity } from '@/types/marketData'
import { formatRelativeTime } from '@/lib/terminal/formatters'
import { EmptyState, Panel, ReputationBar, StatusBadge } from '../TerminalComponents'

export function SearchView({ data }: { data: TerminalData }) {
  const { searchQuery, searchResults, searchLoading, searchData, requestData, clearSearchResults } = data
  const [selected, setSelected] = useState<string | null>(null)
  const [downloading, setDownloading] = useState<Record<string, boolean>>({})

  const onSearch = () => searchData(searchQuery)

  const onClear = () => {
    clearSearchResults()
    setSelected(null)
  }

  const onDownload = async (peerId: string) => {
    setDownloading((d) => ({ ...d, [peerId]: true }))
    try {
      await requestData(peerId)
    } finally {
      setDownloading((d) => ({ ...d, [peerId]: false }))
    }
  }

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        gap: 8,
        padding: 8,
        flex: 1,
        minHeight: 0,
        overflow: 'auto',
      }}
    >
      <div className="toolbar">
        <div className="field" style={{ flex: '0 0 140px' }}>
          <label className="lbl">Data Type</label>
          <div className="select">
            <select
              value={searchQuery.type}
              onChange={(e) =>
                data.updateSearchQuery({ type: e.target.value as MarketDataType })
              }
            >
              <option value="EOD">End of Day</option>
              <option value="DIVIDEND">Dividends</option>
              <option value="INSIDER_TRADE">Insider Trading</option>
              <option value="SPLIT">Splits</option>
            </select>
          </div>
        </div>

        <div className="field" style={{ flex: '0 0 160px' }}>
          <label className="lbl">Symbol</label>
          <input
            className="input"
            placeholder="e.g. AAPL"
            value={searchQuery.symbol}
            onChange={(e) => data.updateSearchQuery({ symbol: e.target.value.toUpperCase() })}
          />
        </div>

        <div className="field" style={{ flex: '0 0 140px' }}>
          <label className="lbl">Start Date</label>
          <input
            type="date"
            className="input"
            value={searchQuery.startDate}
            onChange={(e) => data.updateSearchQuery({ startDate: e.target.value })}
          />
        </div>

        <div className="field" style={{ flex: '0 0 140px' }}>
          <label className="lbl">End Date</label>
          <input
            type="date"
            className="input"
            value={searchQuery.endDate}
            onChange={(e) => data.updateSearchQuery({ endDate: e.target.value })}
          />
        </div>

        <div className="field" style={{ flex: '0 0 120px' }}>
          <label className="lbl">Granularity</label>
          <div className="select">
            <select
              value={searchQuery.granularity}
              onChange={(e) =>
                data.updateSearchQuery({ granularity: e.target.value as TimeGranularity })
              }
            >
              <option value="DAILY">DAILY</option>
              <option value="WEEKLY">WEEKLY</option>
              <option value="MONTHLY">MONTHLY</option>
              <option value="YEARLY">YEARLY</option>
            </select>
          </div>
        </div>

        <div style={{ flex: 1 }} />

        <div className="field">
          <label className="lbl">&nbsp;</label>
          <div style={{ display: 'flex', gap: 6 }}>
            <button className="btn ghost" onClick={onClear}>
              <X size={12} /> Clear
            </button>
            <button className="btn ghost" onClick={onSearch} disabled={searchLoading}>
              <RefreshCw size={12} /> Refresh
            </button>
            <button className="btn primary" onClick={onSearch} disabled={searchLoading}>
              <Search size={12} /> {searchLoading ? 'Searching…' : 'Search Network'}
            </button>
          </div>
        </div>
      </div>

      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          gap: 12,
          padding: '6px 12px',
          background: 'var(--bg-panel-hi)',
          border: '1px solid var(--border)',
          fontFamily: 'var(--font-mono)',
          fontSize: 11,
          color: 'var(--text-dim)',
        }}
      >
        <span style={{ color: 'var(--accent-text)', letterSpacing: '0.14em' }}>QUERY ›</span>
        <span style={{ color: 'var(--text-bright)' }}>{searchQuery.type}</span>
        <span>·</span>
        <span style={{ color: 'var(--text-bright)' }}>{searchQuery.symbol}</span>
        <span>·</span>
        <span style={{ color: 'var(--text-bright)' }}>
          {searchQuery.startDate} → {searchQuery.endDate}
        </span>
        <span>·</span>
        <span style={{ color: 'var(--text-bright)' }}>{searchQuery.granularity}</span>
        <span style={{ flex: 1 }} />
        <StatusBadge kind="info">{searchResults.length} OFFERS</StatusBadge>
      </div>

      <Panel
        title="Provider Offers"
        tag="P2P"
        sub={searchResults.length ? 'RANKED BY REPUTATION' : 'NO ACTIVE QUERY'}
        flush
        style={{ flex: 1, minHeight: 320 }}
      >
        {searchLoading ? (
          <EmptyState title="Searching network…" hint="Querying peer data sources" />
        ) : searchResults.length > 0 ? (
          <table className="dense-table">
            <thead>
              <tr>
                <th>Peer ID</th>
                <th className="num">Reputation</th>
                <th className="num">Reliability</th>
                <th>Data Types</th>
                <th>Range Available</th>
                <th>Updated</th>
                <th></th>
              </tr>
            </thead>
            <tbody>
              {searchResults.map((r) => (
                <tr
                  key={r.peerId || r.id}
                  className={selected === r.peerId ? 'selected' : ''}
                  onClick={() => setSelected(r.peerId)}
                >
                  <td>
                    <span className="mono" style={{ color: 'var(--text-bright)' }}>
                      {r.peerId.slice(0, 20)}
                    </span>
                  </td>
                  <td className="num">
                    <ReputationBar value={r.reputation} />
                  </td>
                  <td className="num dim">{(r.reliability * 100).toFixed(0)}%</td>
                  <td className="dim">{r.dataTypes.join(', ') || '—'}</td>
                  <td className="mono dim" style={{ fontSize: 10.5 }}>
                    {r.dataRange.start} → {r.dataRange.end}
                  </td>
                  <td className="dim">{formatRelativeTime(r.lastUpdate)}</td>
                  <td style={{ textAlign: 'right' }}>
                    <button
                      className="btn sm"
                      onClick={(e) => {
                        e.stopPropagation()
                        onDownload(r.peerId)
                      }}
                      disabled={downloading[r.peerId]}
                    >
                      {downloading[r.peerId] ? (
                        <>Queued</>
                      ) : (
                        <>
                          <Download size={11} /> Download
                        </>
                      )}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <EmptyState
            icon={<Search size={28} />}
            title="No active query"
            hint="Configure the toolbar and press SEARCH NETWORK"
          />
        )}
      </Panel>
    </div>
  )
}

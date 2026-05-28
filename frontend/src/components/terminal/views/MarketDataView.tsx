import { useState } from 'react'
import { Info, RefreshCw, Upload } from 'lucide-react'
import type { TerminalData } from '@/hooks/useTerminalData'
import { formatNumber } from '@/lib/terminal/formatters'
import { EmptyState, Panel, TerminalTabs } from '../TerminalComponents'

type MarketTab = 'EOD' | 'DIVIDEND' | 'INSIDER_TRADE' | 'SPLIT'

export function MarketDataView({ data }: { data: TerminalData }) {
  const [tab, setTab] = useState<MarketTab>('EOD')
  const { eodData, dividendData, insiderData, splitData, marketQuery, marketLoading, fetchMarketData, updateMarketQuery, uploadMarketData } = data
  const [uploadOpen, setUploadOpen] = useState(false)

  const tabs = [
    { id: 'EOD', label: 'End of Day', count: eodData.length },
    { id: 'DIVIDEND', label: 'Dividends', count: dividendData.length },
    { id: 'INSIDER_TRADE', label: 'Insider Trades', count: insiderData.length },
    { id: 'SPLIT', label: 'Splits', count: splitData.length },
  ]

  const loadData = () => fetchMarketData(marketQuery)

  return (
    <div className="view-fill">
      <div style={{ display: 'flex', alignItems: 'center', borderBottom: '1px solid var(--border)' }}>
        <TerminalTabs tabs={tabs} value={tab} onChange={(id) => setTab(id as MarketTab)} />
        <div style={{ flex: 1 }} />
        <div style={{ display: 'flex', alignItems: 'center', gap: 6, padding: '0 10px' }}>
          <input
            className="input"
            style={{ width: 80, height: 24 }}
            placeholder="SYM"
            value={marketQuery.symbol}
            onChange={(e) => updateMarketQuery({ symbol: e.target.value.toUpperCase() })}
          />
          <input
            type="date"
            className="input"
            style={{ width: 130, height: 24 }}
            value={marketQuery.startDate}
            onChange={(e) => updateMarketQuery({ startDate: e.target.value })}
          />
          <input
            type="date"
            className="input"
            style={{ width: 130, height: 24 }}
            value={marketQuery.endDate}
            onChange={(e) => updateMarketQuery({ endDate: e.target.value })}
          />
          <button className="btn sm" onClick={loadData} disabled={marketLoading}>
            <RefreshCw size={11} /> Load
          </button>
          <button className="btn sm" onClick={() => setUploadOpen(true)}>
            <Upload size={11} /> Upload
          </button>
        </div>
      </div>

      <div style={{ flex: 1, minHeight: 0, padding: 8 }}>
        <Panel
          title={
            tab === 'EOD'
              ? 'End of Day · Price History'
              : tab === 'DIVIDEND'
                ? 'Dividends · Cash & Stock'
                : tab === 'INSIDER_TRADE'
                  ? 'Insider Trades · SEC Form 4'
                  : 'Splits · Historical Ratios'
          }
          tag={tab}
          sub={`${marketQuery.symbol} · ${marketQuery.startDate} → ${marketQuery.endDate}`}
          flush
          style={{ height: '100%' }}
        >
          {marketLoading ? (
            <EmptyState title="Loading market data…" />
          ) : tab === 'EOD' ? (
            eodData.length > 0 ? (
              <EODTable rows={eodData} />
            ) : (
              <EmptyState title="No EOD data" hint="Adjust symbol/dates and press Load" />
            )
          ) : tab === 'DIVIDEND' ? (
            dividendData.length > 0 ? (
              <DividendTable rows={dividendData} />
            ) : (
              <EmptyState title="No dividend data" hint="No records for this query" />
            )
          ) : tab === 'INSIDER_TRADE' ? (
            insiderData.length > 0 ? (
              <InsiderTable rows={insiderData} />
            ) : (
              <EmptyState title="No insider trades" hint="No records for this query" />
            )
          ) : splitData.length > 0 ? (
            <SplitTable rows={splitData} />
          ) : (
            <EmptyState icon={<Info size={28} />} title="No split data" hint="No records for this query" />
          )}
        </Panel>
      </div>

      {uploadOpen && (
        <UploadModal
          onClose={() => setUploadOpen(false)}
          onUpload={async (payload) => {
            await uploadMarketData(payload)
            setUploadOpen(false)
          }}
        />
      )}
    </div>
  )
}

function EODTable({ rows }: { rows: TerminalData['eodData'] }) {
  return (
    <table className="dense-table">
      <thead>
        <tr>
          <th>Symbol</th>
          <th>Date</th>
          <th className="num">Open</th>
          <th className="num">High</th>
          <th className="num">Low</th>
          <th className="num">Close</th>
          <th className="num">Volume</th>
          <th>Source</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((r) => (
          <tr key={`${r.symbol}-${r.date}`}>
            <td className="sym">{r.symbol}</td>
            <td className="dim">{r.date}</td>
            <td className="num">{formatNumber(r.open)}</td>
            <td className="num">{formatNumber(r.high)}</td>
            <td className="num">{formatNumber(r.low)}</td>
            <td className="num bright">{formatNumber(r.close)}</td>
            <td className="num">{r.volume.toLocaleString()}</td>
            <td className="dim">{r.source || '—'}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

function DividendTable({ rows }: { rows: TerminalData['dividendData'] }) {
  return (
    <table className="dense-table">
      <thead>
        <tr>
          <th>Symbol</th>
          <th>Ex-Date</th>
          <th className="num">Amount</th>
          <th>Type</th>
          <th>Currency</th>
          <th>Source</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((d, i) => (
          <tr key={i}>
            <td className="sym">{d.symbol}</td>
            <td className="dim">{d.exDate}</td>
            <td className="num bright">{formatNumber(d.amount)}</td>
            <td className="dim">{d.type || '—'}</td>
            <td className="dim">{d.currency}</td>
            <td className="dim">{d.source || '—'}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

function InsiderTable({ rows }: { rows: TerminalData['insiderData'] }) {
  return (
    <table className="dense-table">
      <thead>
        <tr>
          <th>Symbol</th>
          <th>Date</th>
          <th>Insider</th>
          <th>Position</th>
          <th>Type</th>
          <th className="num">Shares</th>
          <th className="num">Price</th>
          <th className="num">Value</th>
          <th>Source</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((r, i) => (
          <tr key={i}>
            <td className="sym">{r.symbol}</td>
            <td className="dim">{r.date}</td>
            <td className="bright">{r.insiderName}</td>
            <td className="dim">{r.position || '—'}</td>
            <td className="dim">{r.tradeType}</td>
            <td className="num">{r.shares.toLocaleString()}</td>
            <td className="num">{formatNumber(r.price)}</td>
            <td className="num">{formatNumber(r.value)}</td>
            <td className="dim">{r.source || '—'}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

function SplitTable({ rows }: { rows: TerminalData['splitData'] }) {
  return (
    <table className="dense-table">
      <thead>
        <tr>
          <th>Symbol</th>
          <th>Ex-Date</th>
          <th className="num">Ratio</th>
          <th className="num">Old</th>
          <th className="num">New</th>
          <th>Status</th>
          <th>Source</th>
        </tr>
      </thead>
      <tbody>
        {rows.map((r, i) => (
          <tr key={`${r.symbol}-${r.exDate}-${i}`}>
            <td className="sym">{r.symbol}</td>
            <td className="dim">{r.exDate}</td>
            <td className="num bright">{formatNumber(r.ratio)}</td>
            <td className="num">{r.oldShares}</td>
            <td className="num">{r.newShares}</td>
            <td className="dim">{r.status}</td>
            <td className="dim">{r.source || '-'}</td>
          </tr>
        ))}
      </tbody>
    </table>
  )
}

function UploadModal({
  onClose,
  onUpload,
}: {
  onClose: () => void
  onUpload: (payload: Record<string, unknown>) => Promise<void>
}) {
  const [symbol, setSymbol] = useState('AAPL')
  const [price, setPrice] = useState('150')
  const [volume, setVolume] = useState('1000')
  const [type, setType] = useState('EOD')
  const [amount, setAmount] = useState('0.25')
  const [ratio, setRatio] = useState('2')
  const [oldShares, setOldShares] = useState('1')
  const [newShares, setNewShares] = useState('2')
  const [loading, setLoading] = useState(false)

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        background: 'rgba(0,0,0,0.6)',
        display: 'grid',
        placeItems: 'center',
        zIndex: 100,
      }}
      onClick={onClose}
    >
      <div
        className="panel"
        style={{ width: 420, padding: 0 }}
        onClick={(e) => e.stopPropagation()}
      >
        <div className="panel-head">
          <span className="panel-title">Upload Market Data</span>
        </div>
        <div className="panel-body" style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <div className="field">
            <label className="lbl">Symbol</label>
            <input className="input" value={symbol} onChange={(e) => setSymbol(e.target.value.toUpperCase())} />
          </div>
          <div className="field">
            <label className="lbl">Type</label>
            <div className="select">
              <select value={type} onChange={(e) => setType(e.target.value)}>
                <option value="EOD">EOD</option>
                <option value="DIVIDEND">DIVIDEND</option>
                <option value="SPLIT">SPLIT</option>
              </select>
            </div>
          </div>
          {type === 'DIVIDEND' ? (
            <div className="field">
              <label className="lbl">Amount</label>
              <input className="input" value={amount} onChange={(e) => setAmount(e.target.value)} />
            </div>
          ) : type === 'SPLIT' ? (
            <>
              <div className="field">
                <label className="lbl">Ratio</label>
                <input className="input" value={ratio} onChange={(e) => setRatio(e.target.value)} />
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: 8 }}>
                <div className="field">
                  <label className="lbl">Old Shares</label>
                  <input className="input" value={oldShares} onChange={(e) => setOldShares(e.target.value)} />
                </div>
                <div className="field">
                  <label className="lbl">New Shares</label>
                  <input className="input" value={newShares} onChange={(e) => setNewShares(e.target.value)} />
                </div>
              </div>
            </>
          ) : (
            <>
              <div className="field">
                <label className="lbl">Price</label>
                <input className="input" value={price} onChange={(e) => setPrice(e.target.value)} />
              </div>
              <div className="field">
                <label className="lbl">Volume</label>
                <input className="input" value={volume} onChange={(e) => setVolume(e.target.value)} />
              </div>
            </>
          )}
          <div style={{ display: 'flex', gap: 8, justifyContent: 'flex-end', marginTop: 8 }}>
            <button className="btn ghost" onClick={onClose}>
              Cancel
            </button>
            <button
              className="btn primary"
              disabled={loading}
              onClick={async () => {
                setLoading(true)
                try {
                  await onUpload({
                    symbol,
                    type,
                    price: parseFloat(price),
                    volume: parseFloat(volume),
                    amount: parseFloat(amount),
                    ratio: parseFloat(ratio),
                    oldShares: parseFloat(oldShares),
                    newShares: parseFloat(newShares),
                    exDate: new Date().toISOString().slice(0, 10),
                  })
                } finally {
                  setLoading(false)
                }
              }}
            >
              Upload
            </button>
          </div>
        </div>
      </div>
    </div>
  )
}

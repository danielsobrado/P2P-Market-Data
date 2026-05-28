import { ArrowDownToLine } from 'lucide-react'
import type { TerminalData } from '@/hooks/useTerminalData'
import { formatBytes, formatSpeed, transferStatusKind } from '@/lib/terminal/formatters'
import { EmptyState, Panel, StatusBadge, TerminalProgress } from '../TerminalComponents'

export function TransfersView({ data }: { data: TerminalData }) {
  const { transfers, refreshCore } = data
  const active = transfers.filter(
    (t) => t.status === 'transferring' || t.status === 'pending',
  )

  return (
    <div style={{ padding: 8, flex: 1, minHeight: 0, display: 'flex', flexDirection: 'column' }}>
      <Panel
        title="Transfer Queue"
        tag="TX"
        sub={`${transfers.length} total / ${active.length} active`}
        flush
        actions={
          <button className="btn sm ghost" onClick={() => refreshCore()}>
            Refresh
          </button>
        }
        style={{ flex: 1, minHeight: 400 }}
      >
        {transfers.length > 0 ? (
          <table className="dense-table">
            <thead>
              <tr>
                <th>Tx ID</th>
                <th>Symbol</th>
                <th>Type</th>
                <th>Source / Dest</th>
                <th style={{ width: 160 }}>Progress</th>
                <th className="num">Chunks</th>
                <th className="num">Speed</th>
                <th className="num">Size</th>
                <th>Started</th>
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {transfers.map((t) => (
                <tr key={t.id}>
                  <td className="dim mono">{t.id.slice(0, 16)}</td>
                  <td className="sym">{t.symbol}</td>
                  <td className="dim">{t.type}</td>
                  <td className="mono" style={{ fontSize: 11 }}>
                    {t.source} / {t.destination}
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
                    {t.error ? (
                      <div className="dim" style={{ fontSize: 10.5, marginTop: 3, color: 'var(--danger)' }}>
                        {t.error}
                      </div>
                    ) : null}
                  </td>
                  <td className="num dim">
                    {t.totalChunks ? `${t.completedChunks ?? 0}/${t.totalChunks}` : '-'}
                  </td>
                  <td className="num dim">{formatSpeed(t.speed)}</td>
                  <td className="num dim">{formatBytes(t.size)}</td>
                  <td className="dim">{t.startTime?.slice(0, 19) || '-'}</td>
                  <td>
                    <StatusBadge kind={transferStatusKind(t.status)}>{t.status}</StatusBadge>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <EmptyState
            icon={<ArrowDownToLine size={28} />}
            title="No active transfers"
            hint="Transfer history will appear after a local or peer download starts."
          />
        )}
      </Panel>
    </div>
  )
}

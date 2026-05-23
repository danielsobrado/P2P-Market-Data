import { useState } from 'react'
import { Users } from 'lucide-react'
import type { TerminalData } from '@/hooks/useTerminalData'
import { formatRelativeTime, peerStatusKind, truncateId } from '@/lib/terminal/formatters'
import { EmptyState, Panel, ReputationBar, StatusBadge } from '../TerminalComponents'

export function PeersView({ data }: { data: TerminalData }) {
  const { peers, refreshCore } = data
  const [selected, setSelected] = useState<string | null>(null)
  const selectedPeer = peers.find((p) => p.id === selected)

  return (
    <div style={{ padding: 8, flex: 1, minHeight: 0, display: 'flex', gap: 8 }}>
      <Panel
        title="Peer Directory"
        tag="P2P"
        sub={`${peers.length} peers`}
        flush
        actions={
          <button className="btn sm ghost" onClick={() => refreshCore()}>
            Refresh
          </button>
        }
        style={{ flex: 2, minHeight: 400 }}
      >
        {peers.length > 0 ? (
          <table className="dense-table">
            <thead>
              <tr>
                <th>Peer ID</th>
                <th>Address</th>
                <th>Status</th>
                <th className="num">Reputation</th>
                <th>Authority</th>
                <th>Roles</th>
                <th>Last Seen</th>
              </tr>
            </thead>
            <tbody>
              {peers.map((p) => (
                <tr
                  key={p.id}
                  className={selected === p.id ? 'selected' : ''}
                  onClick={() => setSelected(p.id)}
                  style={{ cursor: 'pointer' }}
                >
                  <td>
                    <span className="mono" style={{ color: 'var(--text-bright)' }}>
                      {truncateId(p.id, 20)}
                    </span>
                  </td>
                  <td className="dim">{p.address || '—'}</td>
                  <td>
                    <StatusBadge kind={peerStatusKind(p.status, p.isConnected)}>
                      {p.status || (p.isConnected ? 'connected' : 'offline')}
                    </StatusBadge>
                  </td>
                  <td className="num">
                    <ReputationBar value={p.reputation} />
                  </td>
                  <td>
                    {p.isAuthority ? (
                      <StatusBadge kind="accent">AUTH</StatusBadge>
                    ) : (
                      <span className="dim">—</span>
                    )}
                  </td>
                  <td className="mono dim" style={{ fontSize: 10.5 }}>
                    {p.roles.join(', ') || '—'}
                  </td>
                  <td className="dim">{formatRelativeTime(p.lastSeen)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <EmptyState
            icon={<Users size={28} />}
            title="No peers discovered"
            hint="Peers appear when the P2P host connects to the network"
          />
        )}
      </Panel>

      {selectedPeer && (
        <Panel title="Peer Detail" tag="INFO" style={{ flex: 1, minWidth: 260 }}>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 10, fontFamily: 'var(--font-mono)', fontSize: 11 }}>
            <Row label="ID" value={selectedPeer.id} />
            <Row label="Address" value={selectedPeer.address || '—'} />
            <Row label="Status" value={selectedPeer.status} />
            <Row label="Reputation" value={`${Math.round(selectedPeer.reputation > 1 ? selectedPeer.reputation : selectedPeer.reputation * 100)}%`} />
            <Row label="Authority" value={selectedPeer.isAuthority ? 'Yes' : 'No'} />
            <Row label="Roles" value={selectedPeer.roles.join(', ') || '—'} />
            <Row label="Last Seen" value={selectedPeer.lastSeen} />
          </div>
        </Panel>
      )}
    </div>
  )
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <div style={{ fontSize: 9.5, letterSpacing: '0.14em', color: 'var(--text-mute)', marginBottom: 2 }}>
        {label.toUpperCase()}
      </div>
      <div style={{ color: 'var(--text-bright)', wordBreak: 'break-all' }}>{value}</div>
    </div>
  )
}

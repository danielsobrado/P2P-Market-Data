import { ReactNode, useEffect, useState } from 'react'
import {
  ArrowLeftRight,
  CandlestickChart,
  Code,
  LayoutDashboard,
  Search,
  Users,
} from 'lucide-react'
import type { ServerStatus } from '@/types/global'
import { secondsAgo } from '@/lib/terminal/formatters'

interface StatusBarProps {
  isConnected: boolean
  serverStatus: ServerStatus | null
  view: string
  lastRefresh: Date | null
}

export function StatusBar({ isConnected, serverStatus, view, lastRefresh }: StatusBarProps) {
  const dbOk = serverStatus?.databaseConnected
  const p2pOk = serverStatus?.p2pHostRunning
  const scriptOk = serverStatus?.scriptMgrRunning
  const embedOk = serverStatus?.embeddedDbRunning
  const [now, setNow] = useState(() => new Date())

  useEffect(() => {
    const i = setInterval(() => setNow(new Date()), 1000)
    return () => clearInterval(i)
  }, [])

  return (
    <div className="statusbar" role="banner">
      <div className="sb-brand">
        <div className="mark">P</div>
        <div>
          <span className="name">P2P MARKET</span>
          <span className="sub" style={{ marginLeft: 8 }}>
            TERMINAL
          </span>
        </div>
      </div>

      <div className="sb-cell">
        <span className={`led ${isConnected ? 'pos pulse' : 'neg'}`} />
        <span className="label">NET</span>
        <span className="val">{isConnected ? 'LIVE' : 'OFFLINE'}</span>
      </div>

      <div className="sb-cell">
        <span className={`led ${dbOk ? 'pos' : 'neg'}`} />
        <span className="label">DB</span>
        <span className="val">{dbOk ? 'READY' : 'DOWN'}</span>
      </div>

      <div className="sb-cell hide-md">
        <span className={`led ${p2pOk ? 'pos' : 'warn'}`} />
        <span className="label">P2P</span>
        <span className="val">{p2pOk ? 'UP' : 'DOWN'}</span>
      </div>

      <div className="sb-cell hide-md">
        <span className={`led ${scriptOk ? 'pos' : 'warn'}`} />
        <span className="label">SCR</span>
        <span className="val">{scriptOk ? 'UP' : 'DOWN'}</span>
      </div>

      <div className="sb-cell hide-md">
        <span className={`led ${embedOk ? 'pos' : 'warn'}`} />
        <span className="label">PG</span>
        <span className="val">{embedOk ? 'UP' : 'DOWN'}</span>
      </div>

      <div className="sb-cell">
        <span className="label">VIEW</span>
        <span className="val">{view}</span>
      </div>

      <div className="sb-cell">
        <span className="label">LAST</span>
        <span className="val">{lastRefresh ? `${secondsAgo(lastRefresh)}s AGO` : '—'}</span>
      </div>

      <div className="sb-spacer" />

      <div className="sb-cell hide-md">
        <span className="label">UTC</span>
        <span className="val">{now.toISOString().slice(11, 19)} UTC</span>
      </div>
    </div>
  )
}

interface TickerStripProps {
  items: { symbol: string; price: number; changePct: number }[]
  live?: boolean
}

export function TickerStrip({ items, live = true }: TickerStripProps) {
  const doubled =
    items.length > 0 ? [...items, ...items] : [{ symbol: '—', price: 0, changePct: 0 }]

  return (
    <div className="ticker">
      <div className="ticker-label">
        <span className={`led ${live ? 'pos pulse' : 'warn'}`} />
        MKT · {live ? 'LIVE' : 'IDLE'}
      </div>
      <div className="ticker-track">
        <div className="ticker-marquee">
          {doubled.map((t, i) => {
            const positive = t.changePct >= 0
            return (
              <span key={`${t.symbol}-${i}`} className="tk-item">
                <span className="sym">{t.symbol}</span>
                <span className="px">
                  {t.price > 0
                    ? t.price.toLocaleString(undefined, { maximumFractionDigits: 2 })
                    : '—'}
                </span>
                {t.price > 0 && (
                  <span className={`chg ${positive ? 'pos' : 'neg'}`}>
                    {positive ? '▲' : '▼'} {Math.abs(t.changePct).toFixed(2)}%
                  </span>
                )}
              </span>
            )
          })}
        </div>
      </div>
    </div>
  )
}

export type NavId = 'dashboard' | 'search' | 'market' | 'transfers' | 'scripts' | 'peers'

const NAV: { id: NavId; label: string; icon: ReactNode; key: string }[] = [
  { id: 'dashboard', label: 'Dashboard', icon: <LayoutDashboard size={14} />, key: 'F1' },
  { id: 'search', label: 'Search', icon: <Search size={14} />, key: 'F2' },
  { id: 'market', label: 'Market Data', icon: <CandlestickChart size={14} />, key: 'F3' },
  { id: 'transfers', label: 'Transfers', icon: <ArrowLeftRight size={14} />, key: 'F4' },
  { id: 'scripts', label: 'Scripts', icon: <Code size={14} />, key: 'F5' },
  { id: 'peers', label: 'Peers', icon: <Users size={14} />, key: 'F6' },
]

interface SideNavProps {
  active: NavId
  onSelect: (id: NavId) => void
  peersCount: number
  transferCount: number
}

export function SideNav({ active, onSelect, peersCount, transferCount }: SideNavProps) {
  return (
    <nav className="sidenav">
      <div className="sn-section">Workspace</div>
      {NAV.map((n) => (
        <div
          key={n.id}
          className={`sn-item ${active === n.id ? 'active' : ''}`}
          onClick={() => onSelect(n.id)}
          role="button"
          tabIndex={0}
          onKeyDown={(e) => e.key === 'Enter' && onSelect(n.id)}
        >
          {n.icon}
          <span>{n.label}</span>
          <span className="key">{n.key}</span>
        </div>
      ))}

      <div className="sn-section" style={{ marginTop: 8 }}>
        Session
      </div>
      <div className="sn-footer">
        <div className="row">
          <span>Peers</span>
          <span className="v">{peersCount}</span>
        </div>
        <div className="row">
          <span>Active TX</span>
          <span className="v">{transferCount}</span>
        </div>
      </div>
    </nav>
  )
}

export { NAV }

interface PanelProps {
  title: string
  tag?: string
  sub?: string
  actions?: ReactNode
  children: ReactNode
  flush?: boolean
  style?: React.CSSProperties
}

export function Panel({ title, tag, sub, actions, children, flush, style }: PanelProps) {
  return (
    <div className="panel" style={style}>
      <div className="panel-head">
        <span className="panel-title">{title}</span>
        {tag && <span className="panel-tag">{tag}</span>}
        {sub && <span className="panel-sub">{sub}</span>}
        {actions && <div className="panel-actions">{actions}</div>}
      </div>
      <div className={`panel-body ${flush ? 'flush' : ''}`}>{children}</div>
    </div>
  )
}

type BadgeKind = 'pos' | 'neg' | 'warn' | 'info' | 'accent' | 'default'

interface StatusBadgeProps {
  kind?: BadgeKind
  children: ReactNode
  dot?: boolean
}

export function StatusBadge({ kind = 'default', children, dot = true }: StatusBadgeProps) {
  const k = kind === 'default' ? '' : kind
  return (
    <span className={`badge ${k}`}>
      {dot && <span className="dot" />}
      {children}
    </span>
  )
}

interface MetricTileProps {
  label: string
  value: string | number
  unit?: string
  sub?: string
  kind?: '' | 'pos' | 'neg' | 'warn' | 'info'
}

export function MetricTile({ label, value, unit, sub, kind }: MetricTileProps) {
  return (
    <div className={`metric ${kind || ''}`}>
      <div className="m-label">{label}</div>
      <div className="m-val">
        {value}
        {unit && <span className="unit">{unit}</span>}
      </div>
      {sub && <div className="m-sub">{sub}</div>}
    </div>
  )
}

interface ReputationBarProps {
  value: number
}

export function ReputationBar({ value }: ReputationBarProps) {
  const pct = Math.round(value > 1 ? Math.min(value, 100) : value * 100)
  const cls = pct >= 75 ? '' : pct >= 50 ? 'mid' : 'low'
  return (
    <span className="repbar">
      <span className="track">
        <span className={`fill ${cls}`} style={{ width: `${pct}%` }} />
      </span>
      <span className="val">{pct}%</span>
    </span>
  )
}

interface TerminalProgressProps {
  value: number
  status?: string
  striped?: boolean
}

export function TerminalProgress({ value, status, striped }: TerminalProgressProps) {
  const cls =
    status === 'completed'
      ? 'pos'
      : status === 'failed'
        ? 'neg'
        : status === 'pending'
          ? 'warn'
          : ''
  return (
    <div className={`progress ${cls} ${striped ? 'striped' : ''}`}>
      <div className="fill" style={{ width: `${Math.min(100, Math.max(0, value))}%` }} />
    </div>
  )
}

interface TerminalTabsProps {
  tabs: { id: string; label: string; count?: number }[]
  value: string
  onChange: (id: string) => void
}

export function TerminalTabs({ tabs, value, onChange }: TerminalTabsProps) {
  return (
    <div className="tabs">
      {tabs.map((t) => (
        <div
          key={t.id}
          className={`tab ${value === t.id ? 'active' : ''}`}
          onClick={() => onChange(t.id)}
          role="tab"
          tabIndex={0}
          onKeyDown={(e) => e.key === 'Enter' && onChange(t.id)}
        >
          {t.label}
          {t.count !== undefined && <span className="ct">{t.count}</span>}
        </div>
      ))}
    </div>
  )
}

interface EmptyStateProps {
  icon?: ReactNode
  title: string
  hint?: string
}

export function EmptyState({ icon, title, hint }: EmptyStateProps) {
  return (
    <div className="empty">
      {icon && <div style={{ color: 'var(--text-faint)' }}>{icon}</div>}
      <div>{title}</div>
      {hint && <div className="hint">{hint}</div>}
    </div>
  )
}

interface LogStripProps {
  lines: { ts: string; lvl: string; msg: string }[]
}

export function LogStrip({ lines }: LogStripProps) {
  const line = lines[0]
  if (!line) return null

  return (
    <div className="logstrip">
      <span className="ls-label">CON</span>
      <span className="ls-line">
        <span className="ts">{line.ts}</span>
        <span className={`lvl ${line.lvl}`}>{line.lvl.toUpperCase()}</span>
        <span>{line.msg}</span>
      </span>
      <span style={{ flex: 1 }} />
      <span style={{ color: 'var(--text-mute)', letterSpacing: '0.04em' }}>
        ▎ buffer {Math.min(lines.length, 64)}/64
      </span>
    </div>
  )
}

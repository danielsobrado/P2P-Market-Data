export function toISOString(value: unknown): string {
  if (!value) return ''
  if (typeof value === 'string') return value
  if (value instanceof Date) return value.toISOString()
  if (typeof value === 'object' && value !== null) {
    const v = value as Record<string, unknown>
    if (typeof v.Time === 'string') return v.Time
  }
  return String(value)
}

export function formatDate(value: unknown): string {
  if (!value) return '—'
  if (typeof value === 'string') {
    return value.length >= 10 ? value.slice(0, 10) : value
  }
  if (value instanceof Date) {
    return value.toISOString().slice(0, 10)
  }
  if (typeof value === 'object' && value !== null) {
    const v = value as Record<string, unknown>
    if (typeof v.Time === 'string') return v.Time.slice(0, 10)
  }
  return String(value).slice(0, 10)
}

export function formatDateTime(value: unknown): string {
  if (!value) return '—'
  if (typeof value === 'string') return value.replace('T', ' ').slice(0, 19)
  if (value instanceof Date) return value.toISOString().replace('T', ' ').slice(0, 19)
  return String(value)
}

export function formatRelativeTime(value: unknown): string {
  if (!value) return '—'
  const date = parseDate(value)
  if (!date) return '—'
  const seconds = Math.floor((Date.now() - date.getTime()) / 1000)
  if (seconds < 60) return `${seconds}s ago`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`
  return `${Math.floor(seconds / 86400)}d ago`
}

function parseDate(value: unknown): Date | null {
  if (typeof value === 'string') {
    const d = new Date(value)
    return Number.isNaN(d.getTime()) ? null : d
  }
  if (value instanceof Date) return value
  if (typeof value === 'object' && value !== null) {
    const v = value as Record<string, unknown>
    if (typeof v.Time === 'string') return new Date(v.Time)
  }
  return null
}

export function formatBytes(bytes: number): string {
  if (!bytes || bytes <= 0) return '—'
  const units = ['B', 'KB', 'MB', 'GB']
  let n = bytes
  let i = 0
  while (n >= 1024 && i < units.length - 1) {
    n /= 1024
    i++
  }
  return `${n.toFixed(i === 0 ? 0 : 2)} ${units[i]}`
}

export function formatSpeed(bytesPerSec: number): string {
  if (!bytesPerSec || bytesPerSec <= 0) return '—'
  const mb = bytesPerSec / (1024 * 1024)
  return `${mb.toFixed(2)} MB/s`
}

export function formatReputation(value: number): string {
  const pct = Math.round(normalizeReputation(value) * 100)
  return `${pct}%`
}

export function normalizeReputation(value: number): number {
  if (value > 1) return Math.min(value / 100, 1)
  if (value < 0) return 0
  return value
}

export function reputationClass(value: number): '' | 'mid' | 'low' {
  const pct = normalizeReputation(value) * 100
  if (pct >= 75) return ''
  if (pct >= 50) return 'mid'
  return 'low'
}

export type StatusKind = 'pos' | 'neg' | 'warn' | 'info' | 'accent' | 'default'

export function transferStatusKind(status: string): StatusKind {
  switch (status?.toLowerCase()) {
    case 'completed':
      return 'pos'
    case 'failed':
      return 'neg'
    case 'transferring':
      return 'info'
    case 'pending':
      return 'warn'
    default:
      return 'default'
  }
}

export function peerStatusKind(status: string, isConnected: boolean): StatusKind {
  if (isConnected || status?.toLowerCase() === 'connected' || status?.toLowerCase() === 'online') {
    return 'pos'
  }
  if (status?.toLowerCase() === 'connecting') return 'warn'
  return 'neg'
}

export function scriptStatusKind(status: string): StatusKind {
  switch (status?.toLowerCase()) {
    case 'running':
      return 'info'
    case 'failed':
      return 'neg'
    case 'idle':
      return 'default'
    default:
      return 'default'
  }
}

export function formatNumber(value: number, decimals = 2): string {
  if (value === undefined || value === null || Number.isNaN(value)) return '—'
  return value.toLocaleString(undefined, {
    minimumFractionDigits: decimals,
    maximumFractionDigits: decimals,
  })
}

export function formatChange(value: number, pct: number): { text: string; positive: boolean } {
  const positive = value >= 0
  const sign = positive ? '+' : ''
  return {
    text: `${sign}${formatNumber(value)} (${sign}${formatNumber(pct)}%)`,
    positive,
  }
}

export function secondsAgo(date: Date | null): number {
  if (!date) return 0
  return Math.max(0, Math.floor((Date.now() - date.getTime()) / 1000))
}

export function defaultDateRange(days = 365): { startDate: string; endDate: string } {
  const end = new Date()
  const start = new Date()
  start.setDate(start.getDate() - days)
  return {
    startDate: start.toISOString().slice(0, 10),
    endDate: end.toISOString().slice(0, 10),
  }
}

export function truncateId(id: string, len = 14): string {
  if (!id) return '—'
  return id.length <= len ? id : `${id.slice(0, len)}…`
}

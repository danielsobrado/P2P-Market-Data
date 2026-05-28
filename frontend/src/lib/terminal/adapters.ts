import type { DataRequest } from '@/types/marketData'
import type {
  TerminalDataSource,
  TerminalDividendRow,
  TerminalEODRow,
  TerminalInsiderRow,
  TerminalPeer,
  TerminalScript,
  TerminalSplitRow,
  TerminalTransfer,
  TickerItem,
} from './types'
import { formatDate, normalizeReputation, toISOString } from './formatters'

type RawRecord = Record<string, unknown>

function str(v: unknown, fallback = ''): string {
  return typeof v === 'string' ? v : v != null ? String(v) : fallback
}

function num(v: unknown, fallback = 0): number {
  return typeof v === 'number' && !Number.isNaN(v) ? v : fallback
}

function arr(v: unknown): string[] {
  return Array.isArray(v) ? v.map(String) : []
}

function bool(v: unknown): boolean {
  return v === true
}

export function toWailsDataRequest(request: DataRequest): Record<string, string> {
  return {
    type: request.type,
    symbol: request.symbol,
    start_date: request.startDate,
    end_date: request.endDate,
    granularity: request.granularity,
  }
}

export function adaptDataSource(raw: RawRecord): TerminalDataSource {
  const dr = raw.dataRange as RawRecord | undefined
  const start = raw.data_range_start ?? raw.dataRangeStart ?? dr?.start
  const end = raw.data_range_end ?? raw.dataRangeEnd ?? dr?.end
  const last = raw.last_update ?? raw.lastUpdate

  return {
    id: str(raw.id),
    peerId: str(raw.peer_id ?? raw.peerId),
    reputation: num(raw.reputation),
    dataTypes: arr(raw.data_types ?? raw.dataTypes),
    availableSymbols: arr(raw.available_symbols ?? raw.availableSymbols),
    dataRange: {
      start: formatDate(start),
      end: formatDate(end),
    },
    lastUpdate: toISOString(last),
    reliability: num(raw.reliability),
  }
}

export function adaptDataSources(raw: unknown): TerminalDataSource[] {
  if (!Array.isArray(raw)) return []
  return raw.map((item) => adaptDataSource(item as RawRecord))
}

export function adaptPeer(raw: RawRecord): TerminalPeer {
  const status = str(raw.status, '')
  const lastSeenRaw = raw.last_seen ?? raw.lastSeen
  const lastSeen = toISOString(lastSeenRaw)
  const isConnected = inferPeerConnected(status, lastSeen, bool(raw.isConnected))

  return {
    id: str(raw.id),
    address: str(raw.address),
    reputation: num(raw.reputation),
    status: status || (isConnected ? 'connected' : 'offline'),
    isAuthority: bool(raw.is_authority ?? raw.isAuthority),
    roles: arr(raw.roles),
    lastSeen,
    isConnected,
  }
}

function inferPeerConnected(status: string, lastSeen: string, explicit?: boolean): boolean {
  if (explicit) return true
  const s = status.toLowerCase()
  if (s === 'connected' || s === 'online' || s === 'active') return true
  if (lastSeen) {
    const d = new Date(lastSeen)
    if (!Number.isNaN(d.getTime()) && Date.now() - d.getTime() < 5 * 60 * 1000) return true
  }
  return false
}

export function adaptPeers(raw: unknown): TerminalPeer[] {
  if (!Array.isArray(raw)) return []
  return raw.map((item) => adaptPeer(item as RawRecord))
}

export function adaptTransfer(raw: RawRecord): TerminalTransfer {
  return {
    id: str(raw.id),
    type: str(raw.type),
    symbol: str(raw.symbol),
    source: str(raw.source),
    destination: str(raw.destination),
    progress: num(raw.progress),
    status: str(raw.status, 'pending'),
    startTime: str(raw.startTime ?? raw.start_time),
    endTime: raw.endTime || raw.end_time ? str(raw.endTime ?? raw.end_time) : undefined,
    size: num(raw.size),
    speed: num(raw.speed),
  }
}

export function adaptTransfers(raw: unknown): TerminalTransfer[] {
  if (!Array.isArray(raw)) return []
  return raw.map((item) => adaptTransfer(item as RawRecord))
}

export function adaptScript(raw: RawRecord): TerminalScript {
  return {
    id: str(raw.id),
    name: str(raw.name),
    description: str(raw.description),
    author: str(raw.author),
    version: str(raw.version),
    size: num(raw.size),
    created: str(raw.created),
    updated: str(raw.updated),
    status: str(raw.status, 'idle'),
    isInstalled: raw.isInstalled !== false && raw.is_installed !== false,
  }
}

export function adaptScripts(raw: unknown): TerminalScript[] {
  if (!Array.isArray(raw)) return []
  return raw.map((item) => adaptScript(item as RawRecord))
}

export function adaptEODRows(raw: unknown): TerminalEODRow[] {
  if (!Array.isArray(raw)) return []

  const sorted = [...raw].sort((a, b) => {
    const da = formatDate((a as RawRecord).date)
    const db = formatDate((b as RawRecord).date)
    return da.localeCompare(db)
  })

  return sorted.map((item, index) => {
    const r = item as RawRecord
    const close = num(r.close)
    const prev = index > 0 ? num((sorted[index - 1] as RawRecord).close) : close
    const change = close - prev
    const changePct = prev !== 0 ? (change / prev) * 100 : 0

    return {
      symbol: str(r.symbol),
      date: formatDate(r.date),
      open: num(r.open),
      high: num(r.high),
      low: num(r.low),
      close,
      prevClose: prev,
      change,
      changePct,
      volume: num(r.volume),
      source: str(r.source),
    }
  })
}

export function adaptDividendRows(raw: unknown): TerminalDividendRow[] {
  if (!Array.isArray(raw)) return []
  return raw.map((item) => {
    const r = item as RawRecord
    const meta = r.metadata as RawRecord | undefined
    const stockPrice = num(r.stock_price ?? r.stockPrice ?? meta?.stock_price)
    return {
      symbol: str(r.symbol),
      exDate: formatDate(r.ex_date ?? r.exDate ?? r.date),
      amount: num(r.amount),
      type: str(r.type),
      currency: str(r.currency, 'USD'),
      stockPrice,
      source: str(r.source),
    }
  })
}

export function adaptInsiderRows(raw: unknown): TerminalInsiderRow[] {
  if (!Array.isArray(raw)) return []
  return raw.map((item) => {
    const r = item as RawRecord
    const meta = r.metadata as RawRecord | undefined
    return {
      symbol: str(r.symbol),
      date: formatDate(r.trade_date ?? r.tradeDate ?? r.date),
      insiderName: str(r.insider_name ?? r.insiderName),
      position: str(r.position ?? r.insider_title ?? r.insiderTitle),
      tradeType: str(r.trade_type ?? r.tradeType ?? r.transaction_type ?? r.transactionType),
      shares: num(r.shares),
      price: num(r.price_per_share ?? r.pricePerShare ?? r.price),
      value: num(r.value),
      form: str(meta?.form ?? r.secForm ?? '4'),
      source: str(r.source),
    }
  })
}

export function adaptSplitRows(raw: unknown): TerminalSplitRow[] {
  if (!Array.isArray(raw)) return []
  return raw.map((item) => {
    const r = item as RawRecord
    const meta = r.metadata as RawRecord | undefined
    return {
      symbol: str(r.symbol),
      exDate: formatDate(r.ex_date ?? r.exDate),
      announcementDate: formatDate(r.announcement_date ?? r.announcementDate),
      ratio: num(r.split_ratio ?? r.splitRatio),
      oldShares: num(r.old_shares ?? r.oldShares),
      newShares: num(r.new_shares ?? r.newShares),
      status: str(r.status, 'completed'),
      source: str(r.source ?? meta?.source),
    }
  })
}

export function buildTickerFromEOD(rows: TerminalEODRow[]): TickerItem[] {
  const bySymbol = new Map<string, TerminalEODRow>()
  for (const row of rows) {
    bySymbol.set(row.symbol, row)
  }
  return Array.from(bySymbol.values()).map((row) => ({
    symbol: row.symbol,
    price: row.close,
    changePct: row.changePct,
  }))
}

export function countUniqueSymbols(sources: TerminalDataSource[]): number {
  const symbols = new Set<string>()
  for (const source of sources) {
    for (const sym of source.availableSymbols) {
      symbols.add(sym)
    }
  }
  return symbols.size
}

export function connectedPeerCount(peers: TerminalPeer[]): number {
  return peers.filter((p) => p.isConnected).length
}

export function averageReputation(peers: TerminalPeer[]): number {
  if (peers.length === 0) return 0
  const total = peers.reduce((sum, p) => sum + normalizeReputation(p.reputation), 0)
  return total / peers.length
}

export function topPeersByReputation(peers: TerminalPeer[], limit = 6): TerminalPeer[] {
  return [...peers]
    .sort((a, b) => normalizeReputation(b.reputation) - normalizeReputation(a.reputation))
    .slice(0, limit)
}

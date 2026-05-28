import type { DataRequest, MarketDataType, TimeGranularity } from '@/types/marketData'
import type { ServerStatus } from '@/types/global'

export type TerminalView = 'dashboard' | 'search' | 'market' | 'transfers' | 'scripts' | 'peers'

export interface TerminalDataSource {
  id: string
  peerId: string
  reputation: number
  dataTypes: string[]
  availableSymbols: string[]
  dataRange: { start: string; end: string }
  lastUpdate: string
  reliability: number
}

export interface TerminalPeer {
  id: string
  address: string
  reputation: number
  status: string
  isAuthority: boolean
  roles: string[]
  lastSeen: string
  isConnected: boolean
}

export interface TerminalTransfer {
  id: string
  type: string
  symbol: string
  source: string
  destination: string
  progress: number
  status: string
  startTime: string
  endTime?: string
  size: number
  speed: number
}

export interface TerminalScript {
  id: string
  name: string
  description: string
  author: string
  version: string
  size: number
  created: string
  updated: string
  status: string
  isInstalled: boolean
}

export interface TerminalEODRow {
  symbol: string
  date: string
  open: number
  high: number
  low: number
  close: number
  prevClose: number
  change: number
  changePct: number
  volume: number
  source: string
}

export interface TerminalDividendRow {
  symbol: string
  exDate: string
  amount: number
  type: string
  currency: string
  stockPrice: number
  source: string
}

export interface TerminalInsiderRow {
  symbol: string
  date: string
  insiderName: string
  position: string
  tradeType: string
  shares: number
  price: number
  value: number
  form: string
  source: string
}

export interface TerminalSplitRow {
  symbol: string
  exDate: string
  announcementDate: string
  ratio: number
  oldShares: number
  newShares: number
  status: string
  source: string
}

export interface TickerItem {
  symbol: string
  price: number
  changePct: number
}

export interface LogLine {
  ts: string
  lvl: 'info' | 'ok' | 'warn' | 'err'
  msg: string
}

export interface MarketQuery {
  symbol: string
  startDate: string
  endDate: string
}

export interface SearchQuery extends DataRequest {
  type: MarketDataType
  symbol: string
  startDate: string
  endDate: string
  granularity: TimeGranularity
}

export interface TerminalState {
  serverStatus: ServerStatus | null
  isConnected: boolean
  peers: TerminalPeer[]
  dataSources: TerminalDataSource[]
  transfers: TerminalTransfer[]
  scripts: TerminalScript[]
  searchResults: TerminalDataSource[]
  eodData: TerminalEODRow[]
  dividendData: TerminalDividendRow[]
  insiderData: TerminalInsiderRow[]
  splitData: TerminalSplitRow[]
  marketQuery: MarketQuery
  searchQuery: SearchQuery
  lastRefresh: Date | null
  isLoading: boolean
  error: string | null
  logLines: LogLine[]
}

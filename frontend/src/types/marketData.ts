// Market Data Types
export type MarketDataType = 'EOD' | 'DIVIDEND' | 'INSIDER_TRADE'
export type TimeGranularity = 'DAILY' | 'WEEKLY' | 'MONTHLY' | 'YEARLY'

export interface DataRequest {
  type: string
  symbol: string
  startDate: string
  endDate: string
  granularity: TimeGranularity
}

// First create the base interface
export interface BaseMarketData {
  id: string
  symbol: string
  source: string
  dataType: string
  timestamp: string
  upVotes: number
  downVotes: number
  validationScore: number
  validatedBy: string[]
}

// Update specific data types to extend base
export interface EODData extends BaseMarketData {
  date: string
  open: number
  high: number
  low: number
  close: number
  volume: number
  adjustedClose: number
}

export interface DividendData extends BaseMarketData {
  stockPrice: number
  date: string
  amount: number
  type: string
  currency: string
}

export interface InsiderTrade extends BaseMarketData {
  date: string
  insiderName: string
  position: string
  transactionType: string
  shares: number
  pricePerShare: number
  value: number
  secForm: string
}

export interface DataSource {
  peerId: string
  reputation: number
  dataTypes: MarketDataType[]
  availableSymbols: string[]
  dataRange: {
    start: string
    end: string
  }
  lastUpdate: string
  reliability: number
}

export interface DataTransfer {
  id: string
  type: MarketDataType
  symbol: string
  source: string
  destination: string
  progress: number
  status: 'pending' | 'transferring' | 'completed' | 'failed'
  startTime: string
  endTime?: string
  size: number
  speed: number
}


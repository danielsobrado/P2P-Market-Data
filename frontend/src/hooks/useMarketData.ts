import { useState, useEffect, useCallback, useRef } from 'react'
import { format, subDays, isAfter, isBefore } from 'date-fns'
import type { 
  MarketDataType, 
  EODData, 
  DividendData, 
  InsiderTrade,
  TimeGranularity 
} from '@/types/marketData'

interface CacheItem<T> {
  data: T[]
  timestamp: number
  symbol: string
  startDate: string
  endDate: string
}

interface UseMarketDataOptions {
  cacheTime?: number // milliseconds
  retryCount?: number
  retryDelay?: number
  onError?: (error: Error) => void
}

const DEFAULT_CACHE_TIME = 5 * 60 * 1000 // 5 minutes
const DEFAULT_RETRY_COUNT = 3
const DEFAULT_RETRY_DELAY = 1000 // 1 second

// Generic cache manager
class DataCache<T> {
  private cache: Map<string, CacheItem<T>> = new Map()

  getCacheKey(symbol: string, startDate: string, endDate: string): string {
    return `${symbol}-${startDate}-${endDate}`
  }

  get(key: string): T[] | null {
    const item = this.cache.get(key)
    if (!item) return null
    
    if (Date.now() - item.timestamp > DEFAULT_CACHE_TIME) {
      this.cache.delete(key)
      return null
    }
    
    return item.data
  }

  set(key: string, data: T[], symbol: string, startDate: string, endDate: string) {
    this.cache.set(key, {
      data,
      timestamp: Date.now(),
      symbol,
      startDate,
      endDate
    })
  }

  clear() {
    this.cache.clear()
  }
}

// Specialized market data hook with caching and retry logic
export function useMarketData<T>(
  type: MarketDataType,
  symbol: string,
  startDate: string,
  endDate: string,
  options: UseMarketDataOptions = {}
) {
  const [data, setData] = useState<T[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<Error | null>(null)
  const cache = useRef(new DataCache<T>())

  const {
    retryCount = DEFAULT_RETRY_COUNT,
    retryDelay = DEFAULT_RETRY_DELAY,
    onError
  } = options

  const fetchData = useCallback(async (retries = 0) => {
    if (!symbol || !startDate || !endDate) return

    const cacheKey = cache.current.getCacheKey(symbol, startDate, endDate)
    const cachedData = cache.current.get(cacheKey)

    if (cachedData) {
      setData(cachedData)
      return
    }

    try {
      setIsLoading(true)
      setError(null)

      let response: T[]
      switch (type) {
        case 'EOD':
          response = await window.go.main.App.GetEODData(symbol, startDate, endDate)
          break
        case 'DIVIDEND':
          response = await window.go.main.App.GetDividendData(symbol, startDate, endDate)
          break
        case 'INSIDER_TRADE':
          response = await window.go.main.App.GetInsiderData(symbol, startDate, endDate)
          break
        default:
          throw new Error(`Unsupported data type: ${type}`)
      }

      cache.current.set(cacheKey, response, symbol, startDate, endDate)
      setData(response)
    } catch (err) {
      const error = err as Error
      if (retries < retryCount) {
        setTimeout(() => {
          fetchData(retries + 1)
        }, retryDelay * (retries + 1))
      } else {
        setError(error)
        onError?.(error)
      }
    } finally {
      setIsLoading(false)
    }
  }, [type, symbol, startDate, endDate, retryCount, retryDelay, onError])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  // Utility functions for data analysis
  const getDataInRange = useCallback((start: Date, end: Date) => {
    return data.filter(item => {
      const itemDate = new Date((item as any).date)
      return isAfter(itemDate, start) && isBefore(itemDate, end)
    })
  }, [data])

  const aggregateByInterval = useCallback((granularity: TimeGranularity) => {
    // Implementation depends on data type
    switch (type) {
      case 'EOD':
        return aggregateEODData(data as unknown as EODData[], granularity)
      case 'DIVIDEND':
        return aggregateDividendData(data as unknown as DividendData[], granularity)
      case 'INSIDER_TRADE':
        return aggregateInsiderData(data as unknown as InsiderTrade[], granularity)
      default:
        return data
    }
  }, [data, type])

  return {
    data,
    isLoading,
    error,
    refetch: () => fetchData(0),
    clearCache: () => cache.current.clear(),
    getDataInRange,
    aggregateByInterval
  }
}

// Specialized hooks for each data type
export function useEODAnalysis(symbol: string, days: number = 30) {
  const endDate = format(new Date(), 'yyyy-MM-dd')
  const startDate = format(subDays(new Date(), days), 'yyyy-MM-dd')
  
  const {
    data,
    isLoading,
    error,
    ...rest
  } = useMarketData<EODData>('EOD', symbol, startDate, endDate)

  const analysis = useCallback(() => {
    if (!data.length) return null

    const latestPrice = data[data.length - 1].close
    const earliestPrice = data[0].close
    const priceChange = latestPrice - earliestPrice
    const percentChange = (priceChange / earliestPrice) * 100
    const highestPrice = Math.max(...data.map(d => d.high))
    const lowestPrice = Math.min(...data.map(d => d.low))
    const averageVolume = data.reduce((sum, d) => sum + d.volume, 0) / data.length

    return {
      latestPrice,
      priceChange,
      percentChange,
      highestPrice,
      lowestPrice,
      averageVolume,
      volatility: calculateVolatility(data)
    }
  }, [data])

  return {
    data,
    isLoading,
    error,
    analysis: analysis(),
    ...rest
  }
}

export function useDividendAnalysis(symbol: string) {
  const endDate = format(new Date(), 'yyyy-MM-dd')
  const startDate = format(subDays(new Date(), 365), 'yyyy-MM-dd')

  const {
    data,
    isLoading,
    error,
    ...rest
  } = useMarketData<DividendData>('DIVIDEND', symbol, startDate, endDate)

  const analysis = useCallback(() => {
    if (!data.length) return null

    const totalDividends = data.reduce((sum, d) => sum + d.amount, 0)
    const averageDividend = totalDividends / data.length
    const dividendYield = calculateDividendYield(data)
    const payoutFrequency = calculatePayoutFrequency(data)

    return {
      totalDividends,
      averageDividend,
      dividendYield,
      payoutFrequency,
      lastDividend: data[data.length - 1]
    }
  }, [data])

  return {
    data,
    isLoading,
    error,
    analysis: analysis(),
    ...rest
  }
}

export function useInsiderAnalysis(symbol: string) {
  const endDate = format(new Date(), 'yyyy-MM-dd')
  const startDate = format(subDays(new Date(), 90), 'yyyy-MM-dd')

  const {
    data,
    isLoading,
    error,
    ...rest
  } = useMarketData<InsiderTrade>('INSIDER_TRADE', symbol, startDate, endDate)

  const analysis = useCallback(() => {
    if (!data.length) return null

    const buyTotal = data
      .filter(t => t.transactionType === 'BUY')
      .reduce((sum, t) => sum + t.value, 0)

    const sellTotal = data
      .filter(t => t.transactionType === 'SELL')
      .reduce((sum, t) => sum + t.value, 0)

    const netActivity = buyTotal - sellTotal
    const uniqueInsiders = new Set(data.map(t => t.insiderName)).size

    return {
      buyTotal,
      sellTotal,
      netActivity,
      uniqueInsiders,
      significantTransactions: findSignificantTransactions(data)
    }
  }, [data])

  return {
    data,
    isLoading,
    error,
    analysis: analysis(),
    ...rest
  }
}

// Helper functions for data analysis
function calculateVolatility(data: EODData[]): number {
  // Implementation of volatility calculation
  const returns = data.slice(1).map((d, i) => 
    Math.log(d.close / data[i].close)
  )
  const mean = returns.reduce((sum, r) => sum + r, 0) / returns.length
  const variance = returns.reduce((sum, r) => sum + Math.pow(r - mean, 2), 0) / returns.length
  return Math.sqrt(variance * 252) // Annualized volatility
}

function calculateDividendYield(data: DividendData[]): number {
  if (data.length === 0) return 0;

  const totalDividends = data.reduce((sum, record) => sum + record.amount, 0);
  const averageStockPrice = data.reduce((sum, record) => sum + record.stockPrice, 0) / data.length;

  return (totalDividends / averageStockPrice) * 100; // Dividend yield as a percentage
}

function calculatePayoutFrequency(data: DividendData[]): string {
  if (data.length < 2) return 'UNKNOWN';

  const intervals = data.slice(1).map((record, index) => {
    const prevDate = new Date(data[index].date);
    const currDate = new Date(record.date);
    return (currDate.getTime() - prevDate.getTime()) / (1000 * 60 * 60 * 24); // Difference in days
  });

  const averageInterval = intervals.reduce((sum, interval) => sum + interval, 0) / intervals.length;

  if (averageInterval <= 30) return 'MONTHLY';
  if (averageInterval <= 90) return 'QUARTERLY';
  if (averageInterval <= 180) return 'SEMI-ANNUALLY';
  return 'ANNUALLY';
}

function findSignificantTransactions(data: InsiderTrade[]): InsiderTrade[] {
  const significantThreshold = 100000; // Example threshold for significant trades

  return data.filter(trade => trade.value >= significantThreshold);
}
function aggregateDividendData(data: DividendData[], granularity: TimeGranularity): any {
  const aggregatedData: { [key: string]: DividendData } = {}

  data.forEach(item => {
    let key: string
    const date = new Date(item.date)

    switch (granularity) {
      case 'DAILY':
        key = format(date, 'yyyy-MM-dd')
        break
      case 'WEEKLY':
        key = format(date, 'yyyy-ww')
        break
      case 'MONTHLY':
        key = format(date, 'yyyy-MM')
        break
      case 'YEARLY':
        key = format(date, 'yyyy')
        break
      default:
        throw new Error(`Unsupported granularity: ${granularity}`)
    }

    if (!aggregatedData[key]) {
      aggregatedData[key] = { ...item }
    } else {
      aggregatedData[key].amount += item.amount
    }
  })

  return Object.values(aggregatedData)
}

function aggregateInsiderData(data: InsiderTrade[], granularity: TimeGranularity): any {
  const aggregatedData: { [key: string]: InsiderTrade } = {}

  data.forEach(item => {
    let key: string
    const date = new Date(item.date)

    switch (granularity) {
      case 'DAILY':
        key = format(date, 'yyyy-MM-dd')
        break
      case 'WEEKLY':
        key = format(date, 'yyyy-ww')
        break
      case 'MONTHLY':
        key = format(date, 'yyyy-MM')
        break
      case 'YEARLY':
        key = format(date, 'yyyy')
        break
      default:
        throw new Error(`Unsupported granularity: ${granularity}`)
    }

    if (!aggregatedData[key]) {
      aggregatedData[key] = { ...item }
    } else {
      aggregatedData[key].value += item.value
      aggregatedData[key].shares += item.shares
    }
  })

  return Object.values(aggregatedData)
}

function aggregateEODData(data: EODData[], granularity: TimeGranularity): any {

  const aggregatedData: { [key: string]: EODData } = {}

  data.forEach(item => {
    let key: string
    const date = new Date(item.date)

    switch (granularity) {
      case 'DAILY':
        key = format(date, 'yyyy-MM-dd')
        break
      case 'WEEKLY':
        key = format(date, 'yyyy-ww')
        break
      case 'MONTHLY':
        key = format(date, 'yyyy-MM')
        break
      case 'YEARLY':
        key = format(date, 'yyyy')
        break
      default:
        throw new Error(`Unsupported granularity: ${granularity}`)
    }

    if (!aggregatedData[key]) {
      aggregatedData[key] = { ...item }
    } else {
      aggregatedData[key].close = item.close
      aggregatedData[key].high = Math.max(aggregatedData[key].high, item.high)
      aggregatedData[key].low = Math.min(aggregatedData[key].low, item.low)
      aggregatedData[key].volume += item.volume
    }
  })

  return Object.values(aggregatedData)
}

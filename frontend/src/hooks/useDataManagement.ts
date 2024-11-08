import { useState, useEffect, useCallback, useReducer } from 'react'
import type { 
  DataSource, 
  DataTransfer, 
  DataRequest 
} from '@/types/marketData'

interface DataState {
  isLoading: boolean
  error: Error | null
  sources: DataSource[]
  transfers: DataTransfer[]
  searchResults: DataSource[]
}

type DataAction =
  | { type: 'FETCH_START' }
  | { type: 'FETCH_SUCCESS'; payload: Partial<DataState> }
  | { type: 'FETCH_ERROR'; payload: Error }
  | { type: 'UPDATE_TRANSFERS'; payload: DataTransfer[] }
  | { type: 'UPDATE_SEARCH_RESULTS'; payload: DataSource[] }
  | { type: 'CLEAR_SEARCH_RESULTS' }

const dataReducer = (state: DataState, action: DataAction): DataState => {
  switch (action.type) {
    case 'FETCH_START':
      return { ...state, isLoading: true, error: null }
    case 'FETCH_SUCCESS':
      return { ...state, isLoading: false, ...action.payload }
    case 'FETCH_ERROR':
      return { ...state, isLoading: false, error: action.payload }
    case 'UPDATE_TRANSFERS':
      return { ...state, transfers: action.payload }
    case 'UPDATE_SEARCH_RESULTS':
      return { ...state, searchResults: action.payload }
    case 'CLEAR_SEARCH_RESULTS':
      return { ...state, searchResults: [] }
    default:
      return state
  }
}

const initialState: DataState = {
  isLoading: false,
  error: null,
  sources: [],
  transfers: [],
  searchResults: []
}

export function useDataManagement() {
  const [state, dispatch] = useReducer(dataReducer, initialState)
  const [pollingEnabled, setPollingEnabled] = useState(true)

  const fetchDataSources = useCallback(async () => {
    try {
      dispatch({ type: 'FETCH_START' })
      const sources = await window.go.main.App.GetDataSources()
      dispatch({ type: 'FETCH_SUCCESS', payload: { sources } })
    } catch (error) {
      dispatch({ type: 'FETCH_ERROR', payload: error as Error })
    }
  }, [])

  const fetchActiveTransfers = useCallback(async () => {
    try {
      const transfers = await window.go.main.App.GetActiveTransfers()
      dispatch({ type: 'UPDATE_TRANSFERS', payload: transfers })
    } catch (error) {
      console.error('Error fetching transfers:', error)
    }
  }, [])

  const searchData = useCallback(async (request: DataRequest) => {
    try {
      dispatch({ type: 'FETCH_START' })
      const results = await window.go.main.App.SearchData(request)
      dispatch({ type: 'UPDATE_SEARCH_RESULTS', payload: results })
    } catch (error) {
      dispatch({ type: 'FETCH_ERROR', payload: error as Error })
    }
  }, [])

  const requestData = useCallback(async (peerId: string, request: DataRequest) => {
    try {
      await window.go.main.App.RequestData(peerId, request)
      fetchActiveTransfers()
    } catch (error) {
      throw new Error(`Failed to request data: ${error}`)
    }
  }, [fetchActiveTransfers])

  // Set up polling
  useEffect(() => {
    if (!pollingEnabled) return

    const fetchInterval = setInterval(() => {
      fetchActiveTransfers()
    }, 5000)

    return () => clearInterval(fetchInterval)
  }, [pollingEnabled, fetchActiveTransfers])

  // Initial data fetch
  useEffect(() => {
    fetchDataSources()
  }, [fetchDataSources])

  return {
    ...state,
    fetchDataSources,
    fetchActiveTransfers,
    searchData,
    requestData,
    clearSearchResults: () => dispatch({ type: 'CLEAR_SEARCH_RESULTS' }),
    setPollingEnabled
  }
}

// Additional hooks for specific data types
export function useEODData(symbol: string, startDate: string, endDate: string) {
  const [data, setData] = useState<any[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<Error | null>(null)

  const fetchData = useCallback(async () => {
    if (!symbol || !startDate || !endDate) return

    try {
      setIsLoading(true)
      setError(null)
      const response = await window.go.main.App.GetEODData(symbol, startDate, endDate)
      setData(response)
    } catch (error) {
      setError(error as Error)
    } finally {
      setIsLoading(false)
    }
  }, [symbol, startDate, endDate])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  return { data, isLoading, error, refetch: fetchData }
}

export function useDividendData(symbol: string, startDate: string, endDate: string) {
  const [data, setData] = useState<any[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<Error | null>(null)

  const fetchData = useCallback(async () => {
    if (!symbol || !startDate || !endDate) return

    try {
      setIsLoading(true)
      setError(null)
      const response = await window.go.main.App.GetDividendData(symbol, startDate, endDate)
      setData(response)
    } catch (error) {
      setError(error as Error)
    } finally {
      setIsLoading(false)
    }
  }, [symbol, startDate, endDate])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  return { data, isLoading, error, refetch: fetchData }
}

export function useInsiderData(symbol: string, startDate: string, endDate: string) {
  const [data, setData] = useState<any[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState<Error | null>(null)

  const fetchData = useCallback(async () => {
    if (!symbol || !startDate || !endDate) return

    try {
      setIsLoading(true)
      setError(null)
      const response = await window.go.main.App.GetInsiderData(symbol, startDate, endDate)
      setData(response)
    } catch (error) {
      setError(error as Error)
    } finally {
      setIsLoading(false)
    }
  }, [symbol, startDate, endDate])

  useEffect(() => {
    fetchData()
  }, [fetchData])

  return { data, isLoading, error, refetch: fetchData }
}
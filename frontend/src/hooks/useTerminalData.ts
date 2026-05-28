import { useCallback, useEffect, useMemo, useState } from 'react'
import type { DataRequest } from '@/types/marketData'
import type { ServerStatus } from '@/types/global'
import {
  adaptDataSources,
  adaptDividendRows,
  adaptEODRows,
  adaptInsiderRows,
  adaptPeers,
  adaptScripts,
  adaptSplitRows,
  adaptTransfers,
  buildTickerFromEOD,
  connectedPeerCount,
  countUniqueSymbols,
  toWailsDataRequest,
} from '@/lib/terminal/adapters'
import { defaultDateRange } from '@/lib/terminal/formatters'
import type {
  LogLine,
  MarketQuery,
  SearchQuery,
  TerminalDataSource,
  TerminalDividendRow,
  TerminalEODRow,
  TerminalInsiderRow,
  TerminalPeer,
  TerminalScript,
  TerminalSplitRow,
  TerminalTransfer,
  TickerItem,
} from '@/lib/terminal/types'

const app = () => window.go.main.App

function nowTs(): string {
  return new Date().toISOString().slice(11, 19)
}

export function useTerminalData() {
  const defaults = useMemo(() => defaultDateRange(365), [])
  const [serverStatus, setServerStatus] = useState<ServerStatus | null>(null)
  const [peers, setPeers] = useState<TerminalPeer[]>([])
  const [dataSources, setDataSources] = useState<TerminalDataSource[]>([])
  const [transfers, setTransfers] = useState<TerminalTransfer[]>([])
  const [scripts, setScripts] = useState<TerminalScript[]>([])
  const [searchResults, setSearchResults] = useState<TerminalDataSource[]>([])
  const [eodData, setEodData] = useState<TerminalEODRow[]>([])
  const [dividendData, setDividendData] = useState<TerminalDividendRow[]>([])
  const [insiderData, setInsiderData] = useState<TerminalInsiderRow[]>([])
  const [splitData, setSplitData] = useState<TerminalSplitRow[]>([])
  const [marketQuery, setMarketQuery] = useState<MarketQuery>({
    symbol: 'AAPL',
    startDate: defaults.startDate,
    endDate: defaults.endDate,
  })
  const [searchQuery, setSearchQuery] = useState<SearchQuery>({
    type: 'EOD',
    symbol: 'AAPL',
    startDate: defaults.startDate,
    endDate: defaults.endDate,
    granularity: 'DAILY',
  })
  const [lastRefresh, setLastRefresh] = useState<Date | null>(null)
  const [marketLoading, setMarketLoading] = useState(false)
  const [searchLoading, setSearchLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [logLines, setLogLines] = useState<LogLine[]>([
    { ts: nowTs(), lvl: 'info', msg: 'Terminal initialized — awaiting backend status' },
  ])

  const pushLog = useCallback((lvl: LogLine['lvl'], msg: string) => {
    setLogLines((prev) => {
      const next = [{ ts: nowTs(), lvl, msg }, ...prev]
      return next.slice(0, 32)
    })
  }, [])

  const isConnected = Boolean(serverStatus?.running && serverStatus?.databaseConnected)

  const refreshCore = useCallback(async () => {
    try {
      const [status, peerList, sources, activeTransfers, scriptList] = await Promise.all([
        app().GetServerStatus(),
        app().GetPeers(),
        app().GetDataSources(),
        app().GetActiveTransfers(),
        app().GetScripts(),
      ])
      setServerStatus(status)
      setPeers(adaptPeers(peerList))
      setDataSources(adaptDataSources(sources))
      setTransfers(adaptTransfers(activeTransfers))
      setScripts(adaptScripts(scriptList))
      setLastRefresh(new Date())
      setError(null)
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      setError(message)
      pushLog('err', `Poll failed: ${message}`)
    }
  }, [pushLog])

  const fetchMarketData = useCallback(
    async (query: MarketQuery = marketQuery) => {
      setMarketLoading(true)
      setMarketQuery(query)

      const [eodResult, divResult, insiderResult, splitResult] = await Promise.allSettled([
        app().GetEODData(query.symbol, query.startDate, query.endDate),
        app().GetDividendData(query.symbol, query.startDate, query.endDate),
        app().GetInsiderData(query.symbol, query.startDate, query.endDate),
        app().GetSplitData(query.symbol, query.startDate, query.endDate),
      ])

      let loaded = 0

      if (eodResult.status === 'fulfilled') {
        setEodData(adaptEODRows(eodResult.value))
        loaded++
      } else {
        const message = eodResult.reason instanceof Error ? eodResult.reason.message : String(eodResult.reason)
        pushLog('err', `EOD fetch failed: ${message}`)
      }

      if (divResult.status === 'fulfilled') {
        setDividendData(adaptDividendRows(divResult.value))
        loaded++
      } else {
        const message = divResult.reason instanceof Error ? divResult.reason.message : String(divResult.reason)
        pushLog('err', `Dividend fetch failed: ${message}`)
      }

      if (insiderResult.status === 'fulfilled') {
        setInsiderData(adaptInsiderRows(insiderResult.value))
        loaded++
      } else {
        const message = insiderResult.reason instanceof Error ? insiderResult.reason.message : String(insiderResult.reason)
        pushLog('err', `Insider fetch failed: ${message}`)
      }

      if (splitResult.status === 'fulfilled') {
        setSplitData(adaptSplitRows(splitResult.value))
        loaded++
      } else {
        const message = splitResult.reason instanceof Error ? splitResult.reason.message : String(splitResult.reason)
        pushLog('err', `Split fetch failed: ${message}`)
      }

      if (loaded > 0) {
        pushLog('ok', `Market data loaded for ${query.symbol} (${loaded}/4 datasets)`)
        setError(null)
      } else {
        const message = 'All market data requests failed'
        setError(message)
        pushLog('err', message)
      }

      setMarketLoading(false)
    },
    [marketQuery, pushLog],
  )

  const searchData = useCallback(
    async (query: SearchQuery) => {
      setSearchLoading(true)
      setSearchQuery(query)
      try {
        const results = await app().SearchData(toWailsDataRequest(query) as unknown as DataRequest)
        const adapted = adaptDataSources(results)
        setSearchResults(adapted)
        pushLog('ok', `Search returned ${adapted.length} provider offer(s)`)
        setError(null)
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err)
        setError(message)
        pushLog('err', `Search failed: ${message}`)
      } finally {
        setSearchLoading(false)
      }
    },
    [pushLog],
  )

  const requestData = useCallback(
    async (peerId: string, query: SearchQuery = searchQuery) => {
      try {
        await app().RequestData(peerId, toWailsDataRequest(query) as unknown as DataRequest)
        pushLog('info', `Data request queued from peer ${peerId.slice(0, 8)}…`)
        await refreshCore()
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err)
        pushLog('err', `Request failed: ${message}`)
        throw err
      }
    },
    [searchQuery, refreshCore, pushLog],
  )

  const refreshScripts = useCallback(async () => {
    try {
      const scriptList = await app().GetScripts()
      setScripts(adaptScripts(scriptList))
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err)
      pushLog('err', `Script list refresh failed: ${message}`)
    }
  }, [pushLog])

  const runScript = useCallback(
    async (scriptId: string) => {
      await app().RunScript(scriptId)
      pushLog('info', `Script ${scriptId.slice(0, 8)}… started`)
      await refreshScripts()
    },
    [refreshScripts, pushLog],
  )

  const stopScript = useCallback(
    async (scriptId: string) => {
      await app().StopScript(scriptId)
      pushLog('warn', `Script ${scriptId.slice(0, 8)}… stopped`)
      await refreshScripts()
    },
    [refreshScripts, pushLog],
  )

  const deleteScript = useCallback(
    async (scriptId: string) => {
      await app().DeleteScript(scriptId)
      pushLog('warn', `Script ${scriptId.slice(0, 8)}… deleted`)
      await refreshScripts()
    },
    [refreshScripts, pushLog],
  )

  const installScript = useCallback(
    async (scriptId: string) => {
      await app().InstallScript(scriptId)
      pushLog('ok', `Script ${scriptId.slice(0, 8)}… installed`)
      await refreshScripts()
    },
    [refreshScripts, pushLog],
  )

  const uninstallScript = useCallback(
    async (scriptId: string) => {
      await app().UninstallScript(scriptId)
      pushLog('warn', `Script ${scriptId.slice(0, 8)}… uninstalled`)
      await refreshScripts()
    },
    [refreshScripts, pushLog],
  )

  const getScriptContent = useCallback(async (scriptId: string) => {
    return app().GetScriptContent(scriptId)
  }, [])

  const uploadScript = useCallback(
    async (data: { name: string; content: string; description?: string; author?: string; version?: string }) => {
      await app().UploadScript(data)
      pushLog('ok', `Script "${data.name}" uploaded`)
      await refreshScripts()
    },
    [refreshScripts, pushLog],
  )

  const uploadMarketData = useCallback(
    async (payload: Record<string, unknown>) => {
      await app().UploadMarketData(payload)
      pushLog('ok', `Market data uploaded for ${payload.symbol ?? 'unknown'}`)
      await fetchMarketData()
      await refreshCore()
    },
    [fetchMarketData, refreshCore, pushLog],
  )

  useEffect(() => {
    refreshCore()
      .then(() => fetchMarketData())
      .then(() => pushLog('ok', 'Initial data load complete'))

    const interval = setInterval(refreshCore, 5000)
    return () => clearInterval(interval)
  }, []) // eslint-disable-line react-hooks/exhaustive-deps

  const tickerItems: TickerItem[] = useMemo(() => buildTickerFromEOD(eodData), [eodData])

  const kpis = useMemo(
    () => ({
      connectedPeers: connectedPeerCount(peers),
      activeTransfers: transfers.filter((t) => t.status === 'transferring' || t.status === 'pending').length,
      dataSources: dataSources.length,
      symbols: countUniqueSymbols(dataSources),
      serverRunning: serverStatus?.running ?? false,
      dbConnected: serverStatus?.databaseConnected ?? false,
      p2pRunning: serverStatus?.p2pHostRunning ?? false,
      scriptMgrRunning: serverStatus?.scriptMgrRunning ?? false,
      embeddedDbRunning: serverStatus?.embeddedDbRunning ?? false,
    }),
    [peers, transfers, dataSources, serverStatus],
  )

  const updateSearchQuery = useCallback(
    (patch: Partial<SearchQuery>) => {
      setSearchQuery((prev) => ({ ...prev, ...patch }))
    },
    [],
  )

  const updateMarketQuery = useCallback((patch: Partial<MarketQuery>) => {
    setMarketQuery((prev) => ({ ...prev, ...patch }))
  }, [])

  const clearSearchResults = useCallback(() => {
    setSearchResults([])
  }, [])

  return {
    serverStatus,
    isConnected,
    peers,
    dataSources,
    transfers,
    scripts,
    searchResults,
    eodData,
    dividendData,
    insiderData,
    splitData,
    marketQuery,
    searchQuery,
    lastRefresh,
    marketLoading,
    searchLoading,
    error,
    logLines,
    tickerItems,
    kpis,
    refreshCore,
    fetchMarketData,
    searchData,
    requestData,
    refreshScripts,
    runScript,
    stopScript,
    deleteScript,
    installScript,
    uninstallScript,
    getScriptContent,
    uploadScript,
    uploadMarketData,
    updateSearchQuery,
    updateMarketQuery,
    clearSearchResults,
    setSearchQuery,
    pushLog,
  }
}

export type TerminalData = ReturnType<typeof useTerminalData>

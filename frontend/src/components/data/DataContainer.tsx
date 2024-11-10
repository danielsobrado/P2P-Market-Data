import { SetStateAction, Suspense, useState, useCallback } from 'react'
import DataManagementComponent from './DataManagement'
import { DataErrorBoundary } from '@/components/data/DataErrorBoundary'
import { DataSourceSkeleton, TransferSkeleton } from '@/components/data/skeletons/DataSkeletons'
import { useDataManagement } from '@/hooks/useDataManagement'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { AlertCircle, RefreshCw } from 'lucide-react'
import { DataSource, MarketDataBase, DataTransfer, DataRequest } from './interfaces/MarketDataBase'

export const DataContainer: React.FC = () => {
  
  const [marketData, setMarketData] = useState<MarketDataBase[]>([])
  const [sources, setSources] = useState<DataSource[]>([])
  const [transfers, setTransfers] = useState<DataTransfer[]>([])
  const [searchResults, setSearchResults] = useState<DataSource[]>([])
  const [isLoading, setIsLoading] = useState(false)
  const [isPolling, setIsPolling] = useState(false)

  const updateTransfers = useCallback(async () => {
    try {
      const activeTransfers = await window.go.main.App.GetActiveTransfers()
      setTransfers(activeTransfers.map((transfer: any): DataTransfer => ({
        id: transfer.id,
        type: transfer.type,
        symbol: transfer.symbol,
        source: transfer.source,
        destination: transfer.destination,
        progress: transfer.progress,
        status: transfer.status,
        startTime: transfer.startTime,
        endTime: transfer.endTime,
        size: transfer.size,
        speed: transfer.speed
      })))
    } catch (error) {
      console.error('Failed to fetch transfers:', error)
    }
  }, [])

  const updateMarketData = useCallback((newData: SetStateAction<MarketDataBase[]>) => {
    setMarketData(newData)

    // The selected code with types:
    const resolvedData = typeof newData === 'function' ? newData(marketData) : newData;
    window.go.main.App.UpdateMarketData(resolvedData).catch((error: Error) => {
      console.error('Failed to update market data:', error);
    });
  }, [marketData])

  const {
    error,
    fetchDataSources,
    fetchActiveTransfers,
    searchData,
    requestData,
    clearSearchResults,
  } = useDataManagement()

  if (error) {
    return (
      <Alert variant="destructive">
        <AlertCircle className="h-4 w-4" />
        <AlertTitle>Error loading data</AlertTitle>
        <AlertDescription className="space-y-4">
          <p>{error.message}</p>
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              fetchDataSources()
              fetchActiveTransfers()
            }}
          >
            <RefreshCw className="h-4 w-4 mr-2" />
            Retry
          </Button>
        </AlertDescription>
      </Alert>
    )
  }

  return (
    <DataErrorBoundary
      onReset={() => {
        fetchDataSources()
        fetchActiveTransfers()
      }}
    >
      <Suspense
        fallback={
          <div className="space-y-4">
            <DataSourceSkeleton />
            <TransferSkeleton />
          </div>
        }
      >
        <DataManagementComponent
          data={marketData}
          setData={updateMarketData}
          sources={sources}
          transfers={transfers}
          searchResults={searchResults}
          onSearch={searchData}
          onRequestData={requestData}
          onClearSearch={clearSearchResults}
          isLoading={isLoading}
          setPollingEnabled={setIsPolling}
        />
      </Suspense>
    </DataErrorBoundary>
  )
}

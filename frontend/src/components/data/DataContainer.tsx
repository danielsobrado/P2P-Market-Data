import { Suspense } from 'react'
import DataManagementComponent from './DataManagement'
import { DataErrorBoundary } from '@/components/ui/error-boundary/DataErrorBoundary'
import { DataSourceSkeleton, TransferSkeleton } from '@/components/ui/skeletons/DataSkeletons'
import { useDataManagement } from '@/hooks/useDataManagement'
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert'
import { Button } from '@/components/ui/button'
import { AlertCircle, RefreshCw } from 'lucide-react'

// Integration component that handles loading, errors, and data management
export function DataContainer() {
  const {
    isLoading,
    error,
    sources,
    transfers,
    searchResults,
    fetchDataSources,
    fetchActiveTransfers,
    searchData,
    requestData,
    clearSearchResults,
    setPollingEnabled,
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
          sources={sources}
          transfers={transfers}
          searchResults={searchResults}
          onSearch={searchData}
          onRequestData={requestData}
          onClearSearch={clearSearchResults}
          isLoading={isLoading}
          setPollingEnabled={setPollingEnabled}
        />
      </Suspense>
    </DataErrorBoundary>
  )
}

export interface DataSource {
  id: string;
  name: string;
}

export interface DataTransfer {
  id: string;
  name: string;
}

export interface DataRequest {
  // Add necessary fields based on your requirements
  query: string;
}

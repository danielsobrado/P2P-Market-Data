// DataManagementComponent.tsx
import React, { useState, useEffect } from 'react'
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs'
import { DataManagementProps } from './interfaces/DataManagementProps'
import SearchDataTab from './tabs/SearchDataTab'
import DataSourcesTab from './tabs/DataSourcesTab'
import ActiveTransfersTab from './tabs/ActiveTransfersTab'
import ViewDataTab from './tabs/ViewDataTab'
import ScriptsTab from './tabs/ScriptsTab'
import { MarketData } from './interfaces/MarketData'
import { ScriptInfo } from './interfaces/ScriptInfo'

const DataManagementComponent: React.FC<DataManagementProps> = ({
  sources,
  transfers,
  searchResults,
  onSearch,
  onRequestData,
  setPollingEnabled,
  onError,
}) => {
  const [data, setData] = useState<MarketData[]>([])
  const [scripts, setScripts] = useState<ScriptInfo[]>([])

  useEffect(() => {
    setPollingEnabled(true)
    return () => setPollingEnabled(false)
  }, [setPollingEnabled])

  return (
    <Tabs defaultValue="search" className="space-y-4">
      <TabsList>
        <TabsTrigger value="search">Search Data</TabsTrigger>
        <TabsTrigger value="sources">Data Sources</TabsTrigger>
        <TabsTrigger value="transfers">Active Transfers</TabsTrigger>
        <TabsTrigger value="view">View Data</TabsTrigger>
        {/* <TabsTrigger value="analytics">Analytics</TabsTrigger> */}
        <TabsTrigger value="scripts">Scripts</TabsTrigger>
      </TabsList>

      <TabsContent value="search" className="space-y-4">
        <SearchDataTab
          searchResults={searchResults}
          onSearch={onSearch}
          onRequestData={onRequestData}
          onError={onError}
        />
      </TabsContent>

      <TabsContent value="sources">
        <DataSourcesTab sources={sources} />
      </TabsContent>

      <TabsContent value="transfers">
        <ActiveTransfersTab
          transfers={transfers}
          setPollingEnabled={setPollingEnabled}
        />
      </TabsContent>

      <TabsContent value="view">
        <ViewDataTab
          data={data}
          setData={setData}
          onError={onError}
        />
      </TabsContent>

      <TabsContent value="scripts">
        <ScriptsTab
          scripts={scripts}
          setScripts={setScripts}
          onError={onError}
        />
      </TabsContent>
    </Tabs>
  )
}

export default DataManagementComponent

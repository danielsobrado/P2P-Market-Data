// components/data/DataManagement.tsx
import React from 'react'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import SearchDataTab from './tabs/SearchDataTab'
import MarketDataTab from './tabs/MarketDataTab'
import ActiveTransfersTab from './tabs/ActiveTransfersTab'
import ScriptsTab from './tabs/ScriptsTab'
import type { DataManagementProps } from './interfaces/DataManagementProps'

const DataManagement: React.FC<DataManagementProps> = ({
  data,
  setData,
  sources,
  transfers,
  searchResults,
  onSearch,
  onRequestData,
  onClearSearch,
  isLoading,
  setPollingEnabled: setPollingEnabledProp,
  onError
}) => {
  return (
    <div className="container py-8">
      <Tabs defaultValue="search">
        <TabsList>
          <TabsTrigger value="search">Search Data</TabsTrigger>
          <TabsTrigger value="view">Market Data</TabsTrigger>
          <TabsTrigger value="transfers">Active Transfers</TabsTrigger>
          <TabsTrigger value="scripts">Scripts</TabsTrigger>
        </TabsList>

        <TabsContent value="search">
          <SearchDataTab
            searchResults={searchResults}
            onSearch={onSearch}
            onRequestData={onRequestData}
            onError={onError}
          />
        </TabsContent>

        <TabsContent value="view">
          <MarketDataTab
            data={data}
            setData={setData}
            sources={sources}
            transfers={transfers}
            searchResults={searchResults}
            onSearch={onSearch}
            onRequestData={onRequestData}
            onClearSearch={onClearSearch}
            isLoading={isLoading}
            onError={onError}
            setPollingEnabled={(value) => setPollingEnabledProp(typeof value === 'function' ? value(false) : value)}
          />
        </TabsContent>

        <TabsContent value="transfers">
          <ActiveTransfersTab
            transfers={transfers}
            setPollingEnabled={(value) => setPollingEnabledProp(typeof value === 'function' ? value(false) : value)}
          />
        </TabsContent>

        <TabsContent value="scripts">
          <ScriptsTab
            scripts={[]} // Pass scripts state
            setScripts={() => {}} // Pass setScripts function
            onError={onError}
          />
        </TabsContent>
      </Tabs>
    </div>
  )
}

export default DataManagement

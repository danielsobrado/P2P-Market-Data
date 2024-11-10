// tabs/MarketDataTab.tsx
import React, { useState } from 'react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { MarketDataType } from '@/types/marketData'
import EODDataTab from './EODDataTab'
import DividendDataTab from './DividendDataTab'
import InsiderTradeDataTab from './InsiderTradeDataTab'
import SplitDataTab from './SplitDataTab'
import { MarketDataTabProps } from '../interfaces/MarketDataTabProps'
import { UploadDataModal } from './UploadDataModal'

const MarketDataTab: React.FC<MarketDataTabProps> = ({ 
  data, 
  setData, 
  sources,
  transfers,
  searchResults,
  onSearch,
  onRequestData,
  onClearSearch,
  isLoading,
  setPollingEnabled,
  onError 
}) => {
  const [activeTab, setActiveTab] = useState<MarketDataType>('EOD')

  return (
    <Card>
      <CardHeader>
        <div className="flex justify-between items-center">
          <div>
            <CardTitle>Market Data</CardTitle>
            <CardDescription>View and manage market data entries</CardDescription>
          </div>
          <UploadDataModal
            onUpload={async (file, source, type) => {
              try {
                const formData = new FormData()
                formData.append('file', file)
                formData.append('source', source)
                formData.append('type', type)
                
                await window.go.main.App.UploadMarketData(formData)
              } catch (error) {
                onError?.(error as Error)
              }
            }}
            onError={onError}
          />
        </div>
      </CardHeader>
      <CardContent>
        <Tabs value={activeTab} onValueChange={(value: string) => setActiveTab(value as MarketDataType)}>
          <TabsList>
            <TabsTrigger value="EOD">End of Day</TabsTrigger>
            <TabsTrigger value="DIVIDEND">Dividends</TabsTrigger>
            <TabsTrigger value="INSIDER_TRADE">Insider Trades</TabsTrigger>
            <TabsTrigger value="SPLIT">Splits</TabsTrigger>
          </TabsList>

          <TabsContent value="EOD">
            <EODDataTab onError={onError} />
          </TabsContent>

          <TabsContent value="DIVIDEND">
            <DividendDataTab onError={onError} />
          </TabsContent>

          <TabsContent value="INSIDER_TRADE">
            <InsiderTradeDataTab onError={onError} />
          </TabsContent>

          <TabsContent value="SPLIT">
            <SplitDataTab onError={onError} />
          </TabsContent>
        </Tabs>
      </CardContent>
    </Card>
  )
}

export default MarketDataTab

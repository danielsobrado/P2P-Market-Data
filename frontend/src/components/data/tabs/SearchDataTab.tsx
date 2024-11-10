// tabs/SearchDataTab.tsx
import React, { useState } from 'react'
import { format, isAfter, isBefore, startOfToday } from 'date-fns'
import { Calendar } from '@/components/ui/calendar'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from "@/components/ui/select"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Search, Download } from 'lucide-react'
import type { MarketDataType, TimeGranularity, DataSource, DataRequest } from '@/types/marketData'

interface SearchDataTabProps {
  searchResults: DataSource[];
  onSearch: (request: DataRequest) => Promise<void>;
  onRequestData: (peerId: string, request: DataRequest) => Promise<void>;
  onError?: (error: Error) => void;
}

const SearchDataTab: React.FC<SearchDataTabProps> = ({
  searchResults,
  onSearch,
  onRequestData,
  onError,
}) => {
  const [dataType, setDataType] = useState<MarketDataType>('EOD')
  const [symbol, setSymbol] = useState<string>('')
  const [startDate, setStartDate] = useState<Date>()
  const [endDate, setEndDate] = useState<Date>()
  const [granularity, setGranularity] = useState<TimeGranularity>('DAILY')

  const handleSearch = async (): Promise<void> => {
    if (!symbol || !startDate || !endDate) return

    const request: DataRequest = {
      type: dataType,
      symbol: symbol.toUpperCase(),
      startDate: format(startDate, 'yyyy-MM-dd'),
      endDate: format(endDate, 'yyyy-MM-dd'),
      granularity,
    }

    await onSearch(request)
  }

  const handleDownload = async (source: DataSource): Promise<void> => {
    if (!symbol || !startDate || !endDate) return

    const request: DataRequest = {
      type: dataType,
      symbol: symbol.toUpperCase(),
      startDate: format(startDate, 'yyyy-MM-dd'),
      endDate: format(endDate, 'yyyy-MM-dd'),
      granularity,
    }

    await onRequestData(source.peerId, request)
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Search Market Data</CardTitle>
        <CardDescription>
          Search for available market data across the P2P network
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <label>Data Type</label>
            <Select
              value={dataType}
              onValueChange={(value: MarketDataType) => setDataType(value)}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="EOD">End of Day</SelectItem>
                <SelectItem value="DIVIDEND">Dividends</SelectItem>
                <SelectItem value="INSIDER_TRADE">Insider Trading</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <label>Symbol</label>
            <Input
              placeholder="e.g., AAPL"
              value={symbol}
              onChange={(e) => setSymbol(e.target.value)}
            />
          </div>

          <div className="space-y-2">
            <label>Start Date</label>
            <Calendar
              mode="single"
              selected={startDate}
              onSelect={setStartDate}
              disabled={{ after: new Date() }}
            />
          </div>

          <div className="space-y-2">
            <label>End Date</label>
            <Calendar
              mode="single"
              selected={endDate}
              onSelect={setEndDate}
              disabled={(date) => {
                const today = startOfToday()
                return !!(
                  isAfter(date, today) || // Prevent future dates
                  (startDate && isBefore(date, startDate)) // Ensure after start date
                )
              }}
            />
          </div>

          <div className="space-y-2">
            <label>Granularity</label>
            <Select
              value={granularity}
              onValueChange={(value: TimeGranularity) => setGranularity(value)}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="DAILY">Daily</SelectItem>
                <SelectItem value="WEEKLY">Weekly</SelectItem>
                <SelectItem value="MONTHLY">Monthly</SelectItem>
                <SelectItem value="YEARLY">Yearly</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>

        <Button onClick={handleSearch} className="w-full">
          <Search className="mr-2 h-4 w-4" />
          Search
        </Button>

        {searchResults.length > 0 && (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Peer</TableHead>
                <TableHead>Reputation</TableHead>
                <TableHead>Data Range</TableHead>
                <TableHead>Last Update</TableHead>
                <TableHead>Action</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {searchResults.map((source) => (
                <TableRow key={source.peerId}>
                  <TableCell>{source.peerId}</TableCell>
                  <TableCell>
                    <Badge variant={source.reputation > 0.7 ? 'secondary' : 'destructive'}>
                      {(source.reputation * 100).toFixed(0)}%
                    </Badge>
                  </TableCell>
                  <TableCell>
                    {format(new Date(source.dataRange.start), 'PP')} -{' '}
                    {format(new Date(source.dataRange.end), 'PP')}
                  </TableCell>
                  <TableCell>{format(new Date(source.lastUpdate), 'PPp')}</TableCell>
                  <TableCell>
                    <Button size="sm" onClick={() => handleDownload(source)}>
                      <Download className="mr-2 h-4 w-4" />
                      Download
                    </Button>
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </CardContent>
    </Card>
  )
}

export default SearchDataTab

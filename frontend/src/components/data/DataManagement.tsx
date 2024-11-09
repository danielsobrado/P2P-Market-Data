import { useState, useEffect } from 'react'
import { format } from 'date-fns'
import { Calendar } from '@/components/ui/calendar'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from '@/components/ui/tabs'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Progress } from '@/components/ui/progress'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import {
  Download,
  Search,
  RefreshCw,
  AlertCircle,
  Check,
  X,
} from 'lucide-react'
import type {
  MarketDataType,
  TimeGranularity,
  DataSource,
  DataTransfer,
  DataRequest,
} from '@/types/marketData'
import React from 'react'

interface MarketData {
  id: string
  symbol: string
  price: number
  volume: number
  timestamp: string
  source: string
  dataType: string
  validationScore: number
}

interface DataManagementProps {
  sources: DataSource[];
  transfers: DataTransfer[];
  searchResults: DataSource[];
  onSearch: (request: DataRequest) => Promise<void>;
  onRequestData: (peerId: string, request: DataRequest) => Promise<void>;
  onClearSearch: () => void;
  isLoading: boolean;
  setPollingEnabled: React.Dispatch<React.SetStateAction<boolean>>;
  onError?: (error: Error) => void
}

const DataManagementComponent: React.FC<DataManagementProps> = ({
  sources,
  transfers,
  searchResults,
  onSearch,
  onRequestData,
  setPollingEnabled,
  onError,
}) => {
  const [dataType, setDataType] = useState<MarketDataType>('EOD')
  const [symbol, setSymbol] = useState<string>('')
  const [startDate, setStartDate] = useState<Date>()
  const [endDate, setEndDate] = useState<Date>()
  const [granularity, setGranularity] = useState<TimeGranularity>('DAILY')
  const [data, setData] = useState<MarketData[]>([])
  const [loading, setLoading] = useState(false)
  const [selectedDate, setSelectedDate] = useState<Date>()
  const [searchQuery, setSearchQuery] = useState('')

  useEffect(() => {
    setPollingEnabled(true)
    return () => setPollingEnabled(false)
  }, [setPollingEnabled])

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

  const fetchData = async () => {
    try {
      setLoading(true)
      const response = await fetch('/api/market-data')
      const result = await response.json()
      setData(result)
    } catch (error) {
      onError?.(error as Error)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  return (
    <Tabs defaultValue="search" className="space-y-4">
      <TabsList>
        <TabsTrigger value="search">Search Data</TabsTrigger>
        <TabsTrigger value="sources">Data Sources</TabsTrigger>
        <TabsTrigger value="transfers">Active Transfers</TabsTrigger>
        <TabsTrigger value="view">View Data</TabsTrigger>
        <TabsTrigger value="analytics">Analytics</TabsTrigger>
      </TabsList>

      <TabsContent value="search" className="space-y-4">
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
                  <SelectTrigger />
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
                  disabled={(date: Date) => startDate ? date < startDate : false}
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
      </TabsContent>

      <TabsContent value="sources">
        <Card>
          <CardHeader>
            <CardTitle>Available Data Sources</CardTitle>
            <CardDescription>Connected peers with available market data</CardDescription>
          </CardHeader>
          <CardContent>
            <ScrollArea className="h-[400px]">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Peer</TableHead>
                    <TableHead>Data Types</TableHead>
                    <TableHead>Symbols</TableHead>
                    <TableHead>Reliability</TableHead>
                    <TableHead>Last Update</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {sources.map((source) => (
                    <TableRow key={source.peerId}>
                      <TableCell>{source.peerId}</TableCell>
                      <TableCell>
                        <div className="flex gap-1">
                          {source.dataTypes.map((type) => (
                            <Badge key={type} variant="outline">
                              {type}
                            </Badge>
                          ))}
                        </div>
                      </TableCell>
                      <TableCell>{source.availableSymbols.length} symbols</TableCell>
                      <TableCell>
                        <Badge variant={source.reliability > 0.7 ? 'secondary' : 'destructive'}>
                          {(source.reliability * 100).toFixed(0)}%
                        </Badge>
                      </TableCell>
                      <TableCell>{format(new Date(source.lastUpdate), 'PPp')}</TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </ScrollArea>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="transfers">
        <Card>
          <CardHeader>
            <div className="flex justify-between items-center">
              <div>
                <CardTitle>Active Transfers</CardTitle>
                <CardDescription>Current data transfers</CardDescription>
              </div>
              <Button variant="outline" size="icon" onClick={() => setPollingEnabled(true)}>
                <RefreshCw className="h-4 w-4" />
              </Button>
            </div>
          </CardHeader>
          <CardContent>
            <ScrollArea className="h-[400px]">
              <div className="space-y-4">
                {transfers.map((transfer) => (
                  <Card key={transfer.id}>
                    <CardHeader className="py-2">
                      <div className="flex justify-between items-center">
                        <div className="flex items-center space-x-2">
                          {transfer.status === 'transferring' ? (
                            <Download className="h-4 w-4 animate-pulse" />
                          ) : transfer.status === 'completed' ? (
                            <Check className="h-4 w-4 text-green-500" />
                          ) : transfer.status === 'failed' ? (
                            <X className="h-4 w-4 text-red-500" />
                          ) : (
                            <AlertCircle className="h-4 w-4" />
                          )}
                          <div>
                            <p className="font-medium">
                              {transfer.symbol} - {transfer.type}
                            </p>
                            <p className="text-sm text-muted-foreground">From {transfer.source}</p>
                          </div>
                        </div>
                        <Badge
                          variant={
                            transfer.status === 'completed'
                              ? 'secondary'
                              : transfer.status === 'failed'
                              ? 'destructive'
                              : 'default'
                          }
                        >
                          {transfer.status}
                        </Badge>
                      </div>
                    </CardHeader>
                    <CardContent className="py-2">
                      <div className="space-y-2">
                        <Progress value={transfer.progress} />
                        <div className="flex justify-between text-sm text-muted-foreground">
                          <span>{Math.round(transfer.progress)}%</span>
                          <span>{(transfer.speed / 1024).toFixed(2)} KB/s</span>
                        </div>
                      </div>
                    </CardContent>
                  </Card>
                ))}
              </div>
            </ScrollArea>
          </CardContent>
        </Card>
      </TabsContent>

      <TabsContent value="view">
        <Card>
          <CardHeader>
            <CardTitle>Market Data Management</CardTitle>
            <CardDescription>View and manage market data entries</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex gap-4 mb-4">
              <Input
                placeholder="Search..."
                value={searchQuery}
                onChange={(e) => setSearchQuery(e.target.value)}
                className="max-w-sm"
              />
              <Button onClick={fetchData}>
                <RefreshCw className="w-4 h-4 mr-2" />
                Refresh
              </Button>
            </div>
            <ScrollArea className="h-[400px]">
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Symbol</TableHead>
                    <TableHead>Price</TableHead>
                    <TableHead>Volume</TableHead>
                    <TableHead>Source</TableHead>
                    <TableHead>Validation</TableHead>
                    <TableHead>Timestamp</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {data.map((item) => (
                    <TableRow key={item.id}>
                      <TableCell>{item.symbol}</TableCell>
                      <TableCell>{item.price}</TableCell>
                      <TableCell>{item.volume}</TableCell>
                      <TableCell>{item.source}</TableCell>
                      <TableCell>
                        <Progress value={item.validationScore * 100} />
                      </TableCell>
                      <TableCell>
                        {format(new Date(item.timestamp), 'PPpp')}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </ScrollArea>
          </CardContent>
        </Card>
      </TabsContent>
    </Tabs>
  )
}

export default DataManagementComponent

// tabs/ViewDataTab.tsx
import React, { useState } from 'react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Progress } from '@/components/ui/progress'
import { RefreshCw } from 'lucide-react'
import { format } from 'date-fns'
import { fetchData } from '../utils/fetchData'
import type { MarketData } from '../interfaces/MarketData'

interface ViewDataTabProps {
  data: MarketData[];
  setData: React.Dispatch<React.SetStateAction<MarketData[]>>;
  onError?: (error: Error) => void;
}

const ViewDataTab: React.FC<ViewDataTabProps> = ({ data, setData, onError }) => {
  const [searchQuery, setSearchQuery] = useState('')

  const handleFetchData = async () => {
    try {
      const result = await fetchData()
      setData(result)
    } catch (error) {
      onError?.(error as Error)
    }
  }

  return (
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
          <Button onClick={handleFetchData}>
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
              {data
                .filter((item) =>
                  item.symbol.toLowerCase().includes(searchQuery.toLowerCase())
                )
                .map((item) => (
                  <TableRow key={item.id}>
                    <TableCell>{item.symbol}</TableCell>
                    <TableCell>{item.price}</TableCell>
                    <TableCell>{item.volume}</TableCell>
                    <TableCell>{item.source}</TableCell>
                    <TableCell>
                      <Progress value={item.validationScore * 100} />
                    </TableCell>
                    <TableCell>{format(new Date(item.timestamp), 'PPpp')}</TableCell>
                  </TableRow>
                ))}
            </TableBody>
          </Table>
        </ScrollArea>
      </CardContent>
    </Card>
  )
}

export default ViewDataTab

// tabs/DataSourcesTab.tsx
import React from 'react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Badge } from '@/components/ui/badge'
import { format } from 'date-fns'
import type { DataSource } from '@/types/marketData'

interface DataSourcesTabProps {
  sources: DataSource[];
}

const DataSourcesTab: React.FC<DataSourcesTabProps> = ({ sources }) => {
  return (
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
  )
}

export default DataSourcesTab

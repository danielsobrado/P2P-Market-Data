// tabs/EODDataTab.tsx
import React, { useState, useEffect } from 'react'
import { format } from 'date-fns'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { RefreshCw, Trash, ThumbsUp, ThumbsDown } from 'lucide-react'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { EODData } from '../interfaces/EODData'

interface EODDataTabProps {
  onError?: (error: Error) => void;
}

const EODDataTab: React.FC<EODDataTabProps> = ({ onError }) => {
  const [data, setData] = useState<EODData[]>([])
  const [searchQuery, setSearchQuery] = useState('')

  const fetchData = async () => {
    try {
      const response = await fetch('/api/eod-data') // Adjust API endpoint
      const result = await response.json()
      setData(result)
    } catch (error) {
      onError?.(error as Error)
    }
  }

  useEffect(() => {
    fetchData()
  }, [])

  const handleDelete = async (id: string) => {
    try {
      await fetch(`/api/eod-data/${id}`, { method: 'DELETE' });
      setData((prevData) => prevData.filter((item) => item.id !== id))
    } catch (error) {
      onError?.(error as Error)
    }
  }

  const handleVote = (id: string, voteType: 'up' | 'down') => {
    setData((prevData) =>
      prevData.map((item) =>
        item.id === id
          ? {
              ...item,
              upVotes: voteType === 'up' ? item.upVotes + 1 : item.upVotes,
              downVotes: voteType === 'down' ? item.downVotes + 1 : item.downVotes,
            }
          : item
      )
    )
  }

  return (
    <>
      <div className="flex gap-4 mb-4">
        <Input
          placeholder="Search data..."
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
              <TableHead>ID</TableHead>
              <TableHead>Symbol</TableHead>
              <TableHead>Exchange</TableHead>
              <TableHead>Date</TableHead>
              <TableHead>Open</TableHead>
              <TableHead>High</TableHead>
              <TableHead>Low</TableHead>
              <TableHead>Close</TableHead>
              <TableHead>Adj Close</TableHead>
              <TableHead>Volume</TableHead>
              <TableHead>Source</TableHead>
              <TableHead>Metadata</TableHead>
              <TableHead>Actions</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data
              .filter((item) => item.symbol.toLowerCase().includes(searchQuery.toLowerCase()))
              .map((item) => (
                <TableRow key={item.id}>
                  <TableCell>{item.id}</TableCell>
                  <TableCell>{item.symbol}</TableCell>
                  <TableCell>{item.exchange}</TableCell>
                  <TableCell>{format(new Date(item.timestamp), 'yyyy-MM-dd')}</TableCell>
                  <TableCell>{item.open}</TableCell>
                  <TableCell>{item.high}</TableCell>
                  <TableCell>{item.low}</TableCell>
                  <TableCell>{item.close}</TableCell>
                  <TableCell>{item.adjusted_close}</TableCell>
                  <TableCell>{item.volume}</TableCell>
                  <TableCell>{item.source}</TableCell>
                  <TableCell>
                    {item.metadata && (
                      <Tooltip>
                        <TooltipTrigger>
                          <Button variant="ghost" size="icon">
                            ℹ️
                          </Button>
                        </TooltipTrigger>
                        <TooltipContent>
                          {Object.entries(item.metadata).map(([key, value]) => (
                            <div key={key}>
                              <strong>{key}</strong>: {value}
                            </div>
                          ))}
                        </TooltipContent>
                      </Tooltip>
                    )}
                  </TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <ThumbsUp
                        className="h-4 w-4 cursor-pointer hover:text-primary"
                        onClick={() => handleVote(item.id, 'up')}
                      />
                      <ThumbsDown
                        className="h-4 w-4 cursor-pointer hover:text-primary"
                        onClick={() => handleVote(item.id, 'down')}
                      />
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleDelete(item.id)}
                        className="text-destructive hover:text-destructive"
                      >
                        <Trash className="h-4 w-4" />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </ScrollArea>
    </>
  )
}

export default EODDataTab

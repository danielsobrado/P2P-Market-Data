// tabs/DividendDataTab.tsx
import React, { useState, useEffect } from 'react'
import { format } from 'date-fns'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { RefreshCw, Trash, ThumbsUp, ThumbsDown } from 'lucide-react'
import { Tooltip, TooltipContent, TooltipTrigger } from '@/components/ui/tooltip'
import { DividendData } from '../interfaces/DividendData'

interface DividendDataTabProps {
  onError?: (error: Error) => void;
}

const DividendDataTab: React.FC<DividendDataTabProps> = ({ onError }) => {
  const [data, setData] = useState<DividendData[]>([])
  const [searchQuery, setSearchQuery] = useState('')

  const fetchData = async () => {
    try {
      const response = await fetch('/api/dividend-data') // Adjust API endpoint to your backend
      if (!response.ok) {
        throw new Error(`Error fetching data: ${response.statusText}`);
      }
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
      const response = await fetch(`/api/dividend-data/${id}`, { method: 'DELETE' }) // Adjust API endpoint to delete specific dividend data
      if (!response.ok) {
        throw new Error(`Error deleting data: ${response.statusText}`);
      }
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
              <TableHead>Ex-Date</TableHead>
              <TableHead>Payment Date</TableHead>
              <TableHead>Record Date</TableHead>
              <TableHead>Declared Date</TableHead>
              <TableHead>Amount</TableHead>
              <TableHead>Currency</TableHead>
              <TableHead>Frequency</TableHead>
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
                  <TableCell className="font-mono">{item.id.substring(0, 8)}...</TableCell>
                  <TableCell>{item.symbol}</TableCell>
                  <TableCell>{format(new Date(item.ex_date * 1000), 'yyyy-MM-dd')}</TableCell>
                  <TableCell>{format(new Date(item.payment_date * 1000), 'yyyy-MM-dd')}</TableCell>
                  <TableCell>{format(new Date(item.record_date * 1000), 'yyyy-MM-dd')}</TableCell>
                  <TableCell>{format(new Date(item.declared_date * 1000), 'yyyy-MM-dd')}</TableCell>
                  <TableCell>{item.amount.toFixed(2)}</TableCell>
                  <TableCell>{item.currency || 'N/A'}</TableCell>
                  <TableCell>{item.frequency || 'N/A'}</TableCell>
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

export default DividendDataTab

// tabs/ActiveTransfersTab.tsx
import React from 'react'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Progress } from '@/components/ui/progress'
import { Download, RefreshCw, Check, X, AlertCircle } from 'lucide-react'
import type { DataTransfer } from '@/types/marketData'

interface ActiveTransfersTabProps {
  transfers: DataTransfer[];
  setPollingEnabled: React.Dispatch<React.SetStateAction<boolean>>;
}

const ActiveTransfersTab: React.FC<ActiveTransfersTabProps> = ({
  transfers,
  setPollingEnabled,
}) => {
  return (
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
  )
}

export default ActiveTransfersTab

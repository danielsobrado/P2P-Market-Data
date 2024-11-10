// src/components/peer/PeerManagement.tsx
import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { DataErrorBoundary } from '@/components/data/DataErrorBoundary'
import { RefreshCw, Power, X } from 'lucide-react'
import { useToast } from '@/components/ui/toast/use-toast'
import { cn } from '@/lib/utils'

interface Peer {
  id: string
  address: string
  reputation: number
  isConnected: boolean
  lastSeen: string
  roles: string[]
}

// Runtime check function
const isWailsRuntime = () => {
  return typeof window !== 'undefined' && 'go' in window
}

export function PeerManagement() {
  const [peers, setPeers] = useState<Peer[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<Error | null>(null)
  const [mounted, setMounted] = useState(false)
  const { toast } = useToast()

  const waitForRuntime = (timeout = 10000): Promise<void> => {
    return new Promise((resolve, reject) => {
      if (!isWailsRuntime()) {
        resolve()
        return
      }

      const startTime = Date.now()
      let retryCount = 0
      const maxRetries = 10

      const checkRuntime = () => {
        if (!mounted) return
        
        if (window.go?.main?.App) {
          resolve()
          return
        }

        retryCount++
        if (Date.now() - startTime > timeout || retryCount > maxRetries) {
          reject(new Error('Timeout waiting for Wails runtime'))
          return
        }

        setTimeout(checkRuntime, Math.min(100 * Math.pow(2, retryCount), 1000))
      }

      checkRuntime()
    })
  }

  const fetchPeers = async () => {
    if (!mounted) return
    
    try {
      setLoading(true)
      setError(null)

      await waitForRuntime()
      const response = await window.go.main.App.GetPeers()
      if (mounted) {
        setPeers(response || [])
      }
    } catch (err) {
      console.error('Failed to fetch peers:', err)
      if (mounted) {
        setError(err as Error)
        toast({
          variant: "destructive",
          title: "Error",
          description: "Failed to fetch peers. Retrying...",
        })
        // Retry after error
        setTimeout(fetchPeers, 2000)
      }
    } finally {
      if (mounted) {
        setLoading(false)
      }
    }
  }

  useEffect(() => {
    setMounted(true)
    fetchPeers()

    const interval = setInterval(fetchPeers, 5000)
    
    return () => {
      setMounted(false)
      clearInterval(interval)
    }
  }, [])

  if (!mounted) return null

  return (
    <Card>
      <CardHeader>
        <div className="flex justify-between items-center">
          <div>
            <CardTitle>Peer Management</CardTitle>
            <CardDescription>View and manage connected peers</CardDescription>
          </div>
          <Button
            variant="outline"
            size="icon"
            onClick={fetchPeers}
            disabled={loading}
          >
            <RefreshCw className={cn("h-4 w-4", loading && "animate-spin")} />
          </Button>
        </div>
      </CardHeader>
      <CardContent>
        {loading && peers.length === 0 ? (
          <div className="flex justify-center p-4">
            <RefreshCw className="h-6 w-6 animate-spin" />
          </div>
        ) : error ? (
          <div className="text-destructive text-center p-4">
            {error.message}
          </div>
        ) : peers.length === 0 ? (
          <div className="text-muted-foreground text-center p-4">
            No peers connected
          </div>
        ) : (
          <ScrollArea className="h-[400px]">
            {/* Rest of peer table */}
          </ScrollArea>
        )}
      </CardContent>
    </Card>
  )
}

export default PeerManagement
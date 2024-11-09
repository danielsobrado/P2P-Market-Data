// src/components/peer/PeerManagement.tsx
import { useState, useEffect } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from '@/components/ui/table'
import { Badge } from '@/components/ui/badge'
import { ScrollArea } from '@/components/ui/scroll-area'
import { DataErrorBoundary } from '@/components/data/DataErrorBoundary'
import { RefreshCw, Power, X } from 'lucide-react'

interface Peer {
  id: string
  address: string
  reputation: number
  isConnected: boolean
  lastSeen: string
  roles: string[]
}

export function PeerManagement() {
  const [peers, setPeers] = useState<Peer[]>([])
  const [loading, setLoading] = useState(false)

  const fetchPeers = async () => {
    try {
      setLoading(true)
      const response = await window.go.main.App.GetPeers()
      setPeers(response)
    } catch (error) {
      console.error('Failed to fetch peers:', error)
    } finally {
      setLoading(false)
    }
  }

  const disconnectPeer = async (peerId: string) => {
    try {
      await window.go.main.App.DisconnectPeer(peerId)
      fetchPeers() // Refresh list after disconnection
    } catch (error) {
      console.error('Failed to disconnect peer:', error)
    }
  }

  useEffect(() => {
    fetchPeers()
    const interval = setInterval(fetchPeers, 30000) // Refresh every 30s
    return () => clearInterval(interval)
  }, [])

  return (
    <DataErrorBoundary>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <div>
              <CardTitle>Peer Management</CardTitle>
              <CardDescription>Manage connected P2P network peers</CardDescription>
            </div>
            <Button onClick={fetchPeers} variant="outline" size="icon">
              <RefreshCw className="h-4 w-4" />
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <ScrollArea className="h-[400px]">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>ID</TableHead>
                  <TableHead>Address</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Reputation</TableHead>
                  <TableHead>Roles</TableHead>
                  <TableHead>Last Seen</TableHead>
                  <TableHead>Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {peers.map((peer) => (
                  <TableRow key={peer.id}>
                    <TableCell className="font-mono">{peer.id.substring(0, 12)}...</TableCell>
                    <TableCell>{peer.address}</TableCell>
                    <TableCell>
                      <Badge variant={peer.isConnected ? "default" : "secondary"}>
                        {peer.isConnected ? "Connected" : "Disconnected"}
                      </Badge>
                    </TableCell>
                    <TableCell>{peer.reputation.toFixed(2)}</TableCell>
                    <TableCell>
                      {peer.roles.map((role) => (
                        <Badge key={role} variant="outline" className="mr-1">
                          {role}
                        </Badge>
                      ))}
                    </TableCell>
                    <TableCell>{new Date(peer.lastSeen).toLocaleString()}</TableCell>
                    <TableCell>
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => disconnectPeer(peer.id)}
                        disabled={!peer.isConnected}
                      >
                        {peer.isConnected ? <Power className="h-4 w-4" /> : <X className="h-4 w-4" />}
                      </Button>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </ScrollArea>
        </CardContent>
      </Card>
    </DataErrorBoundary>
  )
}

export default PeerManagement
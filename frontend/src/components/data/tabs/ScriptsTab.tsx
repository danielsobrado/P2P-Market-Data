// tabs/ScriptsTab.tsx
import React, { useState } from 'react'
import { format } from 'date-fns'
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
import { Badge } from '@/components/ui/badge'
import { Select, SelectTrigger, SelectContent, SelectItem, SelectValue } from "@/components/ui/select"
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle, DialogTrigger } from "@/components/ui/dialog"
import { Textarea } from "@/components/ui/textarea"
import {
  Download,
  Upload,
  Code,
  Play,
  Pause,
  Trash,
  ThumbsUp,
  ThumbsDown,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import type { ScriptInfo } from '../interfaces/ScriptInfo'
import { handleScripts } from '../utils/handleScripts'
import type { MarketDataType } from '@/types/marketData'
import { Label } from '@/components/ui/label'

interface ScriptsTabProps {
  scripts: ScriptInfo[]
  setScripts: React.Dispatch<React.SetStateAction<ScriptInfo[]>>
  onError?: (error: Error) => void
}

const ScriptsTab: React.FC<ScriptsTabProps> = ({ scripts, setScripts, onError }) => {
  const [scriptSearch, setScriptSearch] = useState('')
  const [scriptFilter, setScriptFilter] = useState('all')
  const [showUploadModal, setShowUploadModal] = useState(false)
  const [selectedScriptCode, setSelectedScriptCode] = useState("")
  const [newScriptContent, setNewScriptContent] = useState("")
  const [newScriptName, setNewScriptName] = useState("")
  const [newScriptDataType, setNewScriptDataType] = useState<MarketDataType>('EOD')

  const {
    handleViewCode,
    handleUploadScript,
    handleRunScript,
    handleStopScript,
    handleDeleteScript,
    handleInstallScript,
    handleUninstallScript,
  } = handleScripts({ setScripts, onError })

  // Function to handle the upload and reset form
  const handleUpload = async () => {
    await handleUploadScript({
      name: newScriptName,
      dataType: newScriptDataType,
      content: newScriptContent,
    })
    // Reset form fields
    setNewScriptName('')
    setNewScriptDataType('EOD')
    setNewScriptContent('')
    // Close dialog
    setShowUploadModal(false)
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle>Data Collection Scripts</CardTitle>
        <CardDescription>Manage and monitor data collection scripts</CardDescription>
      </CardHeader>
      <CardContent>
        <div className="flex justify-between items-center mb-4">
          <div className="flex gap-4">
            <Input 
              placeholder="Search scripts..."
              value={scriptSearch}
              onChange={(e) => setScriptSearch(e.target.value)}
              className="max-w-sm"
            />
            <Select value={scriptFilter} onValueChange={setScriptFilter}>
              <SelectTrigger className="w-[180px]">
                <SelectValue placeholder="Filter by status" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">All Scripts</SelectItem>
                <SelectItem value="installed">Installed</SelectItem>
                <SelectItem value="available">Available</SelectItem>
              </SelectContent>
            </Select>
          </div>
          
          <Dialog open={showUploadModal} onOpenChange={setShowUploadModal}>
            <DialogTrigger asChild>
              <Button>
                <Upload className="h-4 w-4 mr-2" />
                Upload Script
              </Button>
            </DialogTrigger>
            <DialogContent className="sm:max-w-[625px]">
              <DialogHeader>
                <DialogTitle>Upload New Script</DialogTitle>
                <DialogDescription>
                  Add a new data collection script for market data
                </DialogDescription>
              </DialogHeader>
              <div className="grid gap-4 py-4">
                <div className="space-y-2">
                  <Label>Script Name</Label>
                  <Input
                    placeholder="Enter script name..."
                    value={newScriptName}
                    onChange={(e) => setNewScriptName(e.target.value)}
                  />
                </div>
                <div className="space-y-2">
                  <Label>Data Type</Label>
                  <Select
                    value={newScriptDataType}
                    onValueChange={(value: MarketDataType) => setNewScriptDataType(value)}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="Select data type" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="EOD">End of Day</SelectItem>
                      <SelectItem value="DIVIDEND">Dividends</SelectItem>
                      <SelectItem value="INSIDER_TRADE">Insider Trading</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label>Script Content</Label>
                  <Textarea
                    value={newScriptContent}
                    onChange={(e) => setNewScriptContent(e.target.value)}
                    placeholder="Paste your script here..."
                    className="h-[300px] font-mono"
                  />
                </div>
              </div>
              <DialogFooter>
                <Button 
                  onClick={handleUpload}
                  disabled={!newScriptName || !newScriptDataType || !newScriptContent}
                >
                  Upload
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        </div>
        
        <ScrollArea className="h-[400px]">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Name</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Data Type</TableHead>
                <TableHead>Source</TableHead>
                <TableHead>Schedule</TableHead>
                <TableHead>Last Run</TableHead>
                <TableHead>Next Run</TableHead>
                <TableHead>Votes</TableHead>
                <TableHead>Actions</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {scripts
                .filter(script => 
                  script.name.toLowerCase().includes(scriptSearch.toLowerCase()) &&
                  (scriptFilter === 'all' || (scriptFilter === 'installed' && script.isInstalled) || (scriptFilter === 'available' && !script.isInstalled))
                )
                .map((script) => (
                <TableRow key={script.id}>
                  <TableCell>{script.name}</TableCell>
                  <TableCell>
                    <Badge variant={
                      script.status === 'running' ? 'default' :
                      script.status === 'failed' ? 'destructive' :
                      script.status === 'scheduled' ? 'outline' :
                      'secondary'
                    }>
                      {script.status}
                    </Badge>
                  </TableCell>
                  <TableCell>{script.dataType}</TableCell>
                  <TableCell>{script.source}</TableCell>
                  <TableCell>{script.schedule}</TableCell>
                  <TableCell>{script.lastRun && format(new Date(script.lastRun), 'PPpp')}</TableCell>
                  <TableCell>{script.nextRun && format(new Date(script.nextRun), 'PPpp')}</TableCell>
                  <TableCell>
                    <div className="flex items-center gap-2">
                      <ThumbsUp className={cn(
                        "h-4 w-4",
                        "cursor-pointer hover:text-primary"
                      )} />
                      <span>{script.upVotes - script.downVotes}</span>
                      <ThumbsDown className={cn(
                        "h-4 w-4",
                        "cursor-pointer hover:text-primary"
                      )} />
                    </div>
                  </TableCell>
                  <TableCell>
                    <div className="flex gap-2">
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => script.isInstalled ? 
                          handleUninstallScript(script.id) : 
                          handleInstallScript(script.id)
                        }
                      >
                        {script.isInstalled ? 
                          <Trash className="h-4 w-4" /> : 
                          <Download className="h-4 w-4" />
                        }
                      </Button>
                      
                      <Dialog>
                        <DialogTrigger asChild>
                          <Button
                            variant="ghost"
                            size="icon"
                            onClick={() => {
                              handleViewCode(script.id)
                              setSelectedScriptCode('') // Reset code while fetching
                            }}
                          >
                            <Code className="h-4 w-4" />
                          </Button>
                        </DialogTrigger>
                        <DialogContent className="max-w-4xl">
                          <DialogHeader>
                            <DialogTitle>{script.name}</DialogTitle>
                            <DialogDescription>Script Source Code</DialogDescription>
                          </DialogHeader>
                          <pre className="min-h-[400px]">
                            <code>{selectedScriptCode}</code>
                          </pre>
                        </DialogContent>
                      </Dialog>
                  
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => script.status === 'running' ? 
                          handleStopScript(script.id) : 
                          handleRunScript(script.id)
                        }
                      >
                        {script.status === 'running' ? 
                          <Pause className="h-4 w-4" /> : 
                          <Play className="h-4 w-4" />
                        }
                      </Button>
                  
                      <Button
                        variant="ghost"
                        size="icon"
                        onClick={() => handleDeleteScript(script.id)}
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
      </CardContent>
    </Card>
  )
}

export default ScriptsTab

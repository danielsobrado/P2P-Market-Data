// components/data/UploadDataModal.tsx
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Upload } from "lucide-react"
import { useState } from "react"
import { MarketDataTypes, MarketDataType } from '@/components/data/interfaces/MarketDataBase'

interface UploadDataModalProps {
  onUpload: (file: File, source: string, type: MarketDataType) => Promise<void>
  onError?: (error: Error) => void
}

export function UploadDataModal({ onUpload, onError }: UploadDataModalProps) {
  const [file, setFile] = useState<File>()
  const [source, setSource] = useState("")
  const [type, setType] = useState<MarketDataType>()
  const [open, setOpen] = useState(false)

  const handleUpload = async () => {
    if (!file || !source || !type) return

    try {
      await onUpload(file, source, type)
      setOpen(false)
      setFile(undefined)
      setSource("")
      setType(undefined)
    } catch (error) {
      onError?.(error as Error)
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>
          <Upload className="mr-2 h-4 w-4" />
          Upload Data
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Upload Market Data</DialogTitle>
          <DialogDescription>
            Upload market data from a CSV file. Please specify the data source and type.
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-4">
          <div className="grid gap-2">
            <Label>Data Type</Label>
            <Select
              value={type}
              onValueChange={(value: MarketDataType) => setType(value)}
            >
              <SelectTrigger>
                <SelectValue placeholder="Select data type" />
              </SelectTrigger>
              <SelectContent>
                {Object.values(MarketDataTypes).map((type) => (
                  <SelectItem key={type} value={type}>
                    {type.replace(/_/g, ' ')}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid gap-2">
            <Label>Source</Label>
            <Input
              placeholder="Enter data source"
              value={source}
              onChange={(e) => setSource(e.target.value)}
            />
          </div>
          <div className="grid gap-2">
            <Label>CSV File</Label>
            <Input
              type="file"
              accept=".csv"
              onChange={(e) => setFile(e.target.files?.[0])}
            />
          </div>
        </div>
        <DialogFooter>
          <Button
            onClick={handleUpload}
            disabled={!file || !source || !type}
          >
            Upload
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
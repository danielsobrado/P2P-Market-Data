// interfaces/ScriptInfo.ts
import type { MarketDataType } from '@/types/marketData'

export interface ScriptInfo {
  id: string
  name: string
  dataType: MarketDataType
  source: string
  schedule: string
  status: 'running' | 'failed' | 'idle' | 'scheduled'
  lastRun?: string
  nextRun?: string
  votes: number
  upVotes: number
  downVotes: number
  description: string
  isInstalled: boolean
}

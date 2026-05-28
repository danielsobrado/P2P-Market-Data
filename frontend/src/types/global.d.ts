import type {
  DataRequest,
  DataSource,
  DataTransfer,
  DividendData,
  EODData,
  InsiderTrade,
  SplitData,
  BaseMarketData,
} from "./marketData"

export interface Peer {
  id: string
  address: string
  reputation: number
  isConnected?: boolean
  lastSeen?: string
  last_seen?: string
  is_authority?: boolean
  isAuthority?: boolean
  roles: string[]
  status?: string
}

export interface ScriptInfo {
  id: string
  name: string
  description: string
  author: string
  version: string
  size: number
  created: string
  updated: string
  status: string
  isInstalled: boolean
}

export interface ScriptUploadData {
  name: string
  content: string
  description?: string
  author?: string
  version?: string
}

export interface ServerStatus {
  running: boolean
  databaseConnected: boolean
  p2pHostRunning: boolean
  scriptMgrRunning: boolean
  embeddedDbRunning: boolean
}

export interface P2PMetricsDiagnostics {
  connectedPeers: number
  totalPeers: number
  messagesProcessed: number
  networkLatencyMs: number
  requestsReceived: number
  requestsRejected: number
  authFailures: number
  transfersStarted: number
  transfersComplete: number
  transfersFailed: number
  chunksSent: number
  chunksReceived: number
  rowsSent: number
  rowsReceived: number
  bytesSent: number
  bytesReceived: number
  lastError?: string
  lastRequestAt?: string
  lastTransferAt?: string
  lastUpdated?: string
}

export interface SecurityHealthDiagnostics {
  requestSigningRequired: boolean
  responseSigningRequired: boolean
  pubSubStrictSigning: boolean
  keyFileConfigured: boolean
  keyFileExists: boolean
  authFailures: number
  lastSecurityError?: string
}

export interface TransferSummaryDiagnostics {
  pending: number
  transferring: number
  completed: number
  failed: number
}

export interface AppHealthDiagnostics {
  generatedAt: string
  uptimeSeconds: number
  status: ServerStatus
  databaseUrl: string
  databaseLatencyMs: number
  requiredTables: Record<string, boolean>
  p2pHostId: string
  p2pListenAddresses: string[]
  connectedPeers: string[]
  p2pMetrics: P2PMetricsDiagnostics
  transferSummary: TransferSummaryDiagnostics
  security: SecurityHealthDiagnostics
  scriptManagerRunning: boolean
  pythonRuntime: string
  latestTransferErrors: string[]
  operationalWarnings: string[]
}

export {}

declare global {
  interface Window {
    go: {
      main: {
        App: {
          // Data management
          UploadMarketData: (payload: Record<string, unknown>) => Promise<void>;
          UpdateMarketData: (data: BaseMarketData[]) => Promise<void>;
          GetActiveTransfers: () => Promise<DataTransfer[]>;
          GetDataSources: () => Promise<DataSource[]>;
          SearchData: (request: DataRequest) => Promise<DataSource[]>;
          RequestData: (peerId: string, request: DataRequest) => Promise<void>;
          GetEODData: (symbol: string, startDate: string, endDate: string) => Promise<EODData[]>;
          GetDividendData: (symbol: string, startDate: string, endDate: string) => Promise<DividendData[]>;
          GetInsiderData: (symbol: string, startDate: string, endDate: string) => Promise<InsiderTrade[]>;
          GetSplitData: (symbol: string, startDate: string, endDate: string) => Promise<SplitData[]>;
          
          // Script management
          GetScripts: () => Promise<ScriptInfo[]>;
          GetScriptContent: (scriptId: string) => Promise<string>;
          UploadScript: (scriptData: ScriptUploadData) => Promise<void>;
          RunScript: (scriptId: string) => Promise<void>;
          StopScript: (scriptId: string) => Promise<void>;
          DeleteScript: (scriptId: string) => Promise<void>;
          InstallScript: (scriptId: string) => Promise<void>;
          UninstallScript: (scriptId: string) => Promise<void>;
          
          // Peer management
          GetPeers: () => Promise<Peer[]>;
          
          // Status
          GetServerStatus: () => Promise<ServerStatus>;
          GetHealthDiagnostics: () => Promise<AppHealthDiagnostics>;

          // Error handling
          ResetDataConnection: () => Promise<void>;
          ResetDataProcessing: () => Promise<void>;
          RetryConnection: () => Promise<void>;
        };
      };
    };
  }
}

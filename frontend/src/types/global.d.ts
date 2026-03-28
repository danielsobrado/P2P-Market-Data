import type {
  DataRequest,
  DataSource,
  DataTransfer,
  DividendData,
  EODData,
  InsiderTrade,
  BaseMarketData,
} from "./marketData"

export interface Peer {
  id: string
  address: string
  reputation: number
  isConnected: boolean
  lastSeen: string
  roles: string[]
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
          
          // Script management
          GetScriptContent: (scriptId: string) => Promise<string>;
          UploadScript: (scriptData: ScriptUploadData) => Promise<void>;
          RunScript: (scriptId: string) => Promise<void>;
          StopScript: (scriptId: string) => Promise<void>;
          DeleteScript: (scriptId: string) => Promise<void>;
          InstallScript: (scriptId: string) => Promise<void>;
          UninstallScript: (scriptId: string) => Promise<void>;
          
          // Peer management
          GetPeers: () => Promise<Peer[]>;
          DisconnectPeer: (peerId: string) => Promise<void>;
          
          // Status
          GetServerStatus: () => Promise<ServerStatus>;

          // Error handling
          ResetDataConnection: () => Promise<void>;
          ResetDataProcessing: () => Promise<void>;
          RetryConnection: () => Promise<void>;
        };
      };
    };
  }
}

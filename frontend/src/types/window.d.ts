// src/types/window.d.ts
import type { 
  MarketDataBase, 
  DataTransfer, 
  DataSource, 
  DataRequest,
  EODData,
  DividendData,
  InsiderTrade 
} from './marketData'
import type { ScriptUploadData } from './scripts'
import type { Peer } from './peer'

declare global {
  interface Window {
    go: {
      main: {
        App: {
          // Data management
          UpdateMarketData: (data: MarketDataBase[]) => Promise<void>;
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
          
          // Error handling
          ResetDataConnection: () => Promise<void>;
          ResetDataProcessing: () => Promise<void>;
          RetryConnection: () => Promise<void>;
        };
      };
    };
  }
}

export {};
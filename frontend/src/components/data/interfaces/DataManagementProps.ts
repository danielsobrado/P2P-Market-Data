// interfaces/DataManagementProps.ts
import { DataSource, DataTransfer, DataRequest, MarketDataBase } from "./MarketDataBase";

export interface DataManagementProps {
  data: MarketDataBase[];
  setData: React.Dispatch<React.SetStateAction<MarketDataBase[]>>;
  sources: DataSource[];
  transfers: DataTransfer[];
  searchResults: DataSource[];
  onSearch: (request: DataRequest) => Promise<void>;
  onRequestData: (peerId: string, request: DataRequest) => Promise<void>;
  onClearSearch: () => void;
  isLoading: boolean;
  setPollingEnabled: (enabled: boolean) => void;
  onError?: (error: Error) => void;
}

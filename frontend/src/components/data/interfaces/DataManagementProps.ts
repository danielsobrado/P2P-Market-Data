// interfaces/DataManagementProps.ts
import type { DataSource, DataTransfer, DataRequest } from '@/types/marketData'

export interface DataManagementProps {
  sources: DataSource[];
  transfers: DataTransfer[];
  searchResults: DataSource[];
  onSearch: (request: DataRequest) => Promise<void>;
  onRequestData: (peerId: string, request: DataRequest) => Promise<void>;
  onClearSearch: () => void;
  isLoading: boolean;
  setPollingEnabled: React.Dispatch<React.SetStateAction<boolean>>;
  onError?: (error: Error) => void;
}

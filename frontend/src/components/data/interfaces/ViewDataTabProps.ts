import { Dispatch, SetStateAction } from 'react';
import { DataSource, MarketDataBase, DataTransfer, DataRequest } from './MarketDataBase'

export interface ViewDataTabProps {
  data: MarketDataBase[];
  setData: Dispatch<SetStateAction<MarketDataBase[]>>;
  sources: DataSource[];
  transfers: DataTransfer[];
  searchResults: DataSource[];
  onSearch: (request: DataRequest) => Promise<void>;
  onRequestData: (peerId: string, request: DataRequest) => Promise<void>;
  onClearSearch: () => void;
  isLoading: boolean;
  setPollingEnabled: Dispatch<SetStateAction<boolean>>;
  onError?: (error: Error) => void;
}
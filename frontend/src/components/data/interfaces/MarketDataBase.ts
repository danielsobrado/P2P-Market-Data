// interfaces/MarketDataBase.ts
export interface MarketDataBase {
  id: string;
  symbol: string;
  timestamp: string;
  source: string;
  dataType: string;
  validationScore: number;
  upVotes: number;
  downVotes: number;
  metadata?: { [key: string]: string };
}

export type MarketDataType = 'EOD' | 'DIVIDEND' | 'INSIDER_TRADE';
export type TimeGranularity = 'DAILY' | 'WEEKLY' | 'MONTHLY' | 'YEARLY';

export interface DataSource {
  peerId: string;
  reputation: number;
  dataTypes: MarketDataType[];
  availableSymbols: string[];
  dataRange: {
    start: string;
    end: string;
  };
  lastUpdate: string;
  reliability: number;
}

export interface DataTransfer {
  id: string;
  type: MarketDataType;
  symbol: string;
  source: string;
  destination: string;
  progress: number;
  status: 'pending' | 'transferring' | 'completed' | 'failed';
  startTime: string;
  endTime?: string;
  size: number;
  speed: number;
}

export interface DataRequest {
  type: string;
  symbol: string;
  startDate: string;
  endDate: string;
  granularity: TimeGranularity;
}
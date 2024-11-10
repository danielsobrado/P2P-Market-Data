// interfaces/EODData.ts
import { MarketDataBase } from './MarketDataBase';

export interface EODData extends MarketDataBase {
  exchange: string;
  open: number;
  high: number;
  low: number;
  close: number;
  adjusted_close: number;
  volume: number;
  pre_market_price?: number;
  after_hours_price?: number;
}

// interfaces/SplitData.ts
import { MarketDataBase } from './MarketDataBase';

export interface SplitData extends MarketDataBase {
  split_date: number; // Unix timestamp in seconds
  ratio: number; // e.g., 2.0 for a 2-for-1 split
  split_type?: string; // e.g., "Stock Split", "Reverse Split"
}

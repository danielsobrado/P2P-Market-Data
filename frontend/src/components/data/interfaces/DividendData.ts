// interfaces/DividendData.ts
import { MarketDataBase } from './MarketDataBase';

export interface DividendData extends MarketDataBase {
  ex_date: number; // Unix timestamp in seconds
  payment_date: number;
  record_date: number;
  declared_date: number;
  amount: number;
  currency?: string;
  frequency?: string;
}

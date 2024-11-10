// interfaces/InsiderTransaction.ts
import { MarketDataBase } from './MarketDataBase';

export interface InsiderTransaction extends MarketDataBase {
  insider_name: string;
  insider_title: string;
  transaction_date: number; // Unix timestamp in seconds
  transaction_type: string; // e.g., "Buy" or "Sell"
  shares_traded: number;
  price: number;
  shares_owned: number;
  sec_form_type?: string;
  filing_date?: number;
  transaction_code?: string;
}

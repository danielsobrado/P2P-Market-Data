// interfaces/MarketData.ts
export interface MarketData {
    id: string
    symbol: string
    price: number
    volume: number
    timestamp: string
    source: string
    dataType: string
    validationScore: number
  }
  
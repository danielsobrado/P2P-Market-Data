// utils/fetchData.ts
export const fetchData = async () => {
    const response = await fetch('/api/market-data')
    const result = await response.json()
    return result
  }
  
import { Card, CardContent } from "@/components/ui/card"
import { Skeleton } from "@/components/ui/skeleton"

export function DataSourceSkeleton() {
  return (
    <Card>
      <CardContent className="pt-6">
        <div className="space-y-4">
          <div className="flex justify-between items-start">
            <div className="space-y-2">
              <Skeleton className="h-4 w-[120px]" />
              <Skeleton className="h-3 w-[200px]" />
            </div>
            <Skeleton className="h-6 w-[100px]" />
          </div>
          <div className="space-y-2">
            <Skeleton className="h-3 w-[150px]" />
            <div className="flex flex-wrap gap-2">
              {[1, 2, 3].map((i) => (
                <Skeleton key={i} className="h-6 w-[80px]" />
              ))}
            </div>
          </div>
          <div className="space-y-2">
            <Skeleton className="h-3 w-[100px]" />
            <div className="flex gap-2">
              <Skeleton className="h-6 w-[120px]" />
              <Skeleton className="h-6 w-[120px]" />
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

export function TransferSkeleton() {
  return (
    <Card>
      <CardContent className="pt-6">
        <div className="space-y-4">
          <div className="flex justify-between items-start">
            <div className="space-y-2">
              <Skeleton className="h-4 w-[150px]" />
              <Skeleton className="h-3 w-[100px]" />
            </div>
            <Skeleton className="h-6 w-[80px]" />
          </div>
          <div className="space-y-2">
            <Skeleton className="h-2 w-full bg-secondary" />
            <div className="flex justify-between">
              <Skeleton className="h-3 w-[60px]" />
              <Skeleton className="h-3 w-[80px]" />
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

export function ChartSkeleton() {
  return (
    <div className="h-[400px] w-full">
      <div className="h-full w-full flex flex-col gap-4 animate-pulse">
        <div className="h-8 w-[200px] bg-muted rounded" />
        <div className="flex-1 bg-muted rounded" />
      </div>
    </div>
  )
}

export function TableSkeleton() {
  return (
    <div className="space-y-4">
      <div className="flex justify-between items-center">
        {[1, 2, 3, 4].map((i) => (
          <Skeleton key={i} className="h-8 w-[120px]" />
        ))}
      </div>
      {[1, 2, 3, 4, 5].map((i) => (
        <div key={i} className="flex justify-between items-center">
          {[1, 2, 3, 4].map((j) => (
            <Skeleton key={j} className="h-6 w-[100px]" />
          ))}
        </div>
      ))}
    </div>
  )
}
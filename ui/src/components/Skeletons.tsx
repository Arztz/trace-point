export function ChartSkeleton() {
  return (
    <div className="animate-pulse">
      {/* Stats Summary Skeleton */}
      <div className="grid grid-cols-4 gap-4 mb-4">
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className="bg-gray-100 rounded-lg h-16" />
        ))}
      </div>
      {/* Chart Skeleton */}
      <div className="h-72 bg-gray-100 rounded-lg" />
    </div>
  );
}

export function SpikeListSkeleton() {
  return (
    <div className="space-y-3">
      {[1, 2, 3, 4].map((i) => (
        <div key={i} className="bg-gray-100 rounded-lg p-4">
          <div className="flex items-center gap-3 mb-3">
            <div className="w-8 h-8 bg-gray-200 rounded-md" />
            <div className="h-4 w-24 bg-gray-200 rounded" />
          </div>
          <div className="h-3 w-32 bg-gray-200 rounded mb-2" />
          <div className="flex gap-4">
            <div className="h-6 w-16 bg-gray-200 rounded" />
            <div className="h-6 w-16 bg-gray-200 rounded" />
          </div>
        </div>
      ))}
    </div>
  );
}

export function FilterBarSkeleton() {
  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4">
      <div className="flex items-center gap-2 mb-4">
        <div className="w-6 h-6 bg-gray-100 rounded-md" />
        <div className="h-4 w-16 bg-gray-100 rounded" />
      </div>
      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div className="h-10 bg-gray-100 rounded-md" />
        <div className="h-10 bg-gray-100 rounded-md" />
      </div>
    </div>
  );
}

export function TableSkeleton({ rows = 5, cols = 4 }: { rows?: number; cols?: number }) {
  return (
    <div className="animate-pulse">
      <div className="h-10 bg-gray-100 rounded-t-lg" />
      {Array.from({ length: rows }).map((_, i) => (
        <div 
          key={i} 
          className={`h-12 border-b border-gray-100 flex items-center gap-4 px-4 ${i === rows - 1 ? 'rounded-b-lg' : ''}`}
        >
          {Array.from({ length: cols }).map((_, j) => (
            <div key={j} className="h-4 flex-1 bg-gray-100 rounded" />
          ))}
        </div>
      ))}
    </div>
  );
}

export default {
  ChartSkeleton,
  SpikeListSkeleton,
  FilterBarSkeleton,
  TableSkeleton,
};
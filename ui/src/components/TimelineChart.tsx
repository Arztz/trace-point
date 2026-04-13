import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import type { TimelineMetric, PodInfo } from '../types/timeline';
import { POD_COLORS } from '../types/timeline';

interface PodLineData {
  timestamp: string;
  time: string;
  date: string;
  [key: string]: string | number; // Dynamic pod lines and baselines
}

interface TimelineChartProps {
  metrics?: TimelineMetric[];
  availablePods?: PodInfo[];
  selectedPods: string[];
  highlightedPod: string | null;
  onHighlight?: (podName: string | null) => void;
}

interface CustomTooltipProps {
  active?: boolean;
  payload?: Array<{
    name: string;
    value: number;
    color: string;
    dataKey: string;
    baseline?: number;
  }>;
  label?: string;
}

function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleTimeString('en-US', { 
    hour: '2-digit', 
    minute: '2-digit',
    hour12: false 
  });
}

function formatDate(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
  });
}

const CustomTooltip = ({ active, payload, label }: CustomTooltipProps) => {
  if (active && payload && payload.length) {
    return (
      <div className="bg-white border border-gray-200 rounded-lg shadow-lg p-3 min-w-[220px]">
        <p className="text-xs font-medium text-gray-500 mb-2">{label}</p>
        {payload.map((entry, idx) => {
          const dataKey = entry.dataKey || '';
          const currentValue = entry.value;
          
          // The baseline is passed via payload as a custom value 
          // (we'll use entry.value for current and estimate baseline from historical data)
          // Get baseline from payload if available, otherwise calculate from payload
          const baseline = entry.baseline ?? Math.max(currentValue * 0.5, 1);
          const percentage = baseline > 0 ? (currentValue / baseline) * 100 : 100;
          
          return (
            <div key={idx} className="flex flex-col gap-1 py-1.5 border-b border-gray-100 last:border-0">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-1.5">
                  <span 
                    className="w-2 h-2 rounded-full" 
                    style={{ backgroundColor: entry.color }}
                  />
                  <span className="text-xs text-gray-600 font-medium">
                    {entry.name}
                  </span>
                </div>
                <span className="text-xs font-semibold text-gray-900">
                  {currentValue.toFixed(1)}%
                </span>
              </div>
              <div className="flex items-center justify-between text-xs text-gray-500 pl-3.5">
                <span>vs Baseline:</span>
                <span>{percentage.toFixed(0)}%</span>
              </div>
              <div className="text-xs text-gray-400 pl-3.5">
                {percentage > 150 ? (
                  <span className="text-red-600 font-medium">Spiking</span>
                ) : percentage > 110 ? (
                  <span className="text-amber-600 font-medium">Elevated</span>
                ) : (
                  <span className="text-green-600">Normal</span>
                )}
              </div>
            </div>
          );
        })}
      </div>
    );
  }
  return null;
};

export default function TimelineChart({
  metrics = [],
  availablePods = [],
  selectedPods = [],
  highlightedPod = null,
  onHighlight,
}: TimelineChartProps) {
  // Get unique replicasets from metrics (group by replicaset_name, not individual pod)
  const uniqueReplicasets = Array.from(new Set(metrics.map(m => m.replicaset_name)));
  
  // Determine which replicasets to display
  const displayReplicasets = selectedPods.length === 0
    ? uniqueReplicasets
    : uniqueReplicasets.filter(r => selectedPods.includes(r));

// Transform metrics into chart data format (one row per timestamp)
// Aggregate metrics by replicaset (average across all pods in that replicaset)
const chartData = (() => {
  // Group metrics by timestamp and replicaset
  const dataMap = new Map<string, Map<string, { cpu: number[]; ram: number[] }>>();
  
  // Sort metrics by timestamp
  const sortedMetrics = [...metrics].sort((a, b) => 
    a.timestamp.localeCompare(b.timestamp)
  );

  for (const metric of sortedMetrics) {
    const replicasetName = metric.replicaset_name;
    
    if (!dataMap.has(metric.timestamp)) {
      dataMap.set(metric.timestamp, new Map());
    }
    
    const replicaMap = dataMap.get(metric.timestamp)!;
    
    if (!replicaMap.has(replicasetName)) {
      replicaMap.set(replicasetName, { cpu: [], ram: [] });
    }
    
    const entry = replicaMap.get(replicasetName)!;
    entry.cpu.push(metric.cpu_percent);
    entry.ram.push(metric.ram_percent);
  }

  // Convert to chart data with aggregated values
  const result: PodLineData[] = [];
  
  for (const [timestamp, replicaMap] of dataMap) {
    const dataPoint: PodLineData = {
      timestamp,
      time: formatTimestamp(timestamp),
      date: formatDate(timestamp),
    };
    
    // Aggregate metrics for each replicaset (average across pods)
    for (const [replicasetName, values] of replicaMap) {
      const avgCpu = values.cpu.length > 0 
        ? values.cpu.reduce((a, b) => a + b, 0) / values.cpu.length 
        : 0;
      const avgRam = values.ram.length > 0 
        ? values.ram.reduce((a, b) => a + b, 0) / values.ram.length 
        : 0;
      
      dataPoint[`${replicasetName}_cpu`] = avgCpu;
      dataPoint[`${replicasetName}_ram`] = avgRam;
    }
    
    result.push(dataPoint);
  }
  
  return result;
})();

  // Get replicaset index for color assignment
  const getReplicaIndex = (replicasetName: string): number => {
    const pod = availablePods.find(p => p.name === replicasetName);
    if (pod) {
      return availablePods.indexOf(pod);
    }
    return uniqueReplicasets.indexOf(replicasetName);
  };

  const getPodColor = (replicasetName: string): string => {
    const index = getReplicaIndex(replicasetName);
    return POD_COLORS[index % POD_COLORS.length];
  };

  // Calculate stats for selected replicasets or all replicasets
  const stats = (() => {
    const targetReplicasets = displayReplicasets;
    if (targetReplicasets.length === 0 || chartData.length === 0) {
      return { avgCpu: 0, avgRam: 0, maxCpu: 0, maxRam: 0 };
    }

    let totalCpu = 0;
    let totalRam = 0;
    let maxCpu = 0;
    let maxRam = 0;
    let cpuCount = 0;
    let ramCount = 0;

    for (const data of chartData) {
      for (const replicasetName of targetReplicasets) {
        const cpu = data[`${replicasetName}_cpu`] as number;
        const ram = data[`${replicasetName}_ram`] as number;
        
        if (cpu !== undefined) {
          totalCpu += cpu;
          cpuCount++;
          if (cpu > maxCpu) maxCpu = cpu;
        }
        if (ram !== undefined) {
          totalRam += ram;
          ramCount++;
          if (ram > maxRam) maxRam = ram;
        }
      }
    }

    return {
      avgCpu: cpuCount > 0 ? totalCpu / cpuCount : 0,
      avgRam: ramCount > 0 ? totalRam / ramCount : 0,
      maxCpu,
      maxRam,
    };
  })();

  // Handle line click - toggle highlight behavior - simplified
  const handleLineClick = (replicasetName: string) => {
    // Only handle click if onHighlight is provided
    if (!onHighlight) return;
    
    // Toggle behavior: if already highlighted, unhighlight; otherwise highlight
    if (highlightedPod === replicasetName) {
      onHighlight(null);
    } else {
      onHighlight(replicasetName);
    }
  };

  // Handle chart click - detect which data point was clicked
  const handleChartClick = (event: any, activePayload: any[]) => {
    // Debug: log the click event
    console.log('[TimelineChart] onClick:', { event, activePayload });
    
    // Only process if there are active payload elements clicked
    if (!activePayload || activePayload.length === 0 || !onHighlight) {
      console.log('[TimelineChart] No valid click - missing payload or onHighlight');
      return;
    }
    
    // Get the first clicked element data
    const element = activePayload[0];
    console.log('[TimelineChart] Clicked element:', element);
    
    if (element && element.dataKey) {
      const dataKey = String(element.dataKey);
      console.log('[TimelineChart] dataKey:', dataKey);
      
      // Extract replicaset name from dataKey (e.g., "game-workflow_cpu" -> "game-workflow")
      const replicasetName = dataKey.replace('_cpu', '').replace('_ram', '');
      console.log('[TimelineChart] Extracted replicasetName:', replicasetName);
      
      handleLineClick(replicasetName);
    }
  };

  // Handle individual line click - more reliable than chart-level onClick
  const handleLineClickEvent = (replicasetName: string) => (event: any) => {
    console.log('[TimelineChart] Line clicked:', replicasetName, event);
    handleLineClick(replicasetName);
  };

  // Generate lines for each replicaset (CPU and RAM)
  const renderLines = () => {
    const lines: JSX.Element[] = [];

    for (const replicasetName of displayReplicasets) {
      const isHighlighted = highlightedPod === replicasetName;
      // When a pod is highlighted, hide all other lines completely
      const isVisible = !highlightedPod || isHighlighted;
      const color = getPodColor(replicasetName);
      const strokeWidth = isHighlighted ? 3 : 1;
      const opacity = isVisible ? 1 : 0;

      // Skip rendering hidden lines entirely
      if (!isVisible) {
        continue;
      }

      // CPU line
      lines.push(
        <Line
          key={`${replicasetName}-cpu`}
          type="monotone"
          dataKey={`${replicasetName}_cpu`}
          name={`${replicasetName} (CPU)`}
          stroke={color}
          strokeWidth={strokeWidth}
          dot={isHighlighted}
          activeDot={{ r: isHighlighted ? 6 : 4, stroke: color, strokeWidth: 2, fill: '#fff' }}
          opacity={opacity}
          connectNulls
          onClick={handleLineClickEvent(replicasetName)}
        />
      );

      // RAM line (use same color but dashed)
      lines.push(
        <Line
          key={`${replicasetName}-ram`}
          type="monotone"
          dataKey={`${replicasetName}_ram`}
          name={`${replicasetName} (RAM)`}
          stroke={color}
          strokeWidth={strokeWidth}
          strokeDasharray="5 5"
          dot={isHighlighted}
          activeDot={{ r: isHighlighted ? 6 : 4, stroke: color, strokeWidth: 2, fill: '#fff' }}
          opacity={opacity}
          connectNulls
          onClick={handleLineClickEvent(replicasetName)}
        />
      );
    }

    return lines;
  };

  // If no metrics, show empty state message
  if (metrics.length === 0) {
    return (
      <div className="text-center py-8 text-gray-500">
        No timeline data available
      </div>
    );
  }

  return (
    <div className="space-y-4">
      {/* Stats Summary */}
      <div className="grid grid-cols-4 gap-4">
        <div className="bg-primary-50 rounded-lg p-3">
          <p className="text-xs text-primary-600 font-medium">Avg CPU</p>
          <p className="text-xl font-bold text-primary-700">{stats.avgCpu.toFixed(1)}%</p>
        </div>
        <div className="bg-purple-50 rounded-lg p-3">
          <p className="text-xs text-purple-600 font-medium">Avg RAM</p>
          <p className="text-xl font-bold text-purple-700">{stats.avgRam.toFixed(1)}%</p>
        </div>
        <div className="bg-red-50 rounded-lg p-3">
          <p className="text-xs text-red-600 font-medium">Max CPU</p>
          <p className="text-xl font-bold text-red-700">{stats.maxCpu.toFixed(1)}%</p>
        </div>
        <div className="bg-amber-50 rounded-lg p-3">
          <p className="text-xs text-amber-600 font-medium">Max RAM</p>
          <p className="text-xl font-bold text-amber-700">{stats.maxRam.toFixed(1)}%</p>
        </div>
      </div>

      {/* Legend info */}
      <div className="text-sm text-gray-500 flex items-center gap-4">
        <span>Showing {displayReplicasets.length} replicaset{displayReplicasets.length !== 1 ? 's' : ''}</span>
        {highlightedPod && (
          <span className="text-primary-600 font-medium">
            Highlighted: {highlightedPod}
          </span>
        )}
        <span className="text-xs">
          Solid = CPU | Dashed = RAM
        </span>
      </div>

      {/* Chart - use fixed height for reliability */}
      <div className="h-[600px] min-w-0">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart 
            data={chartData} 
            margin={{ top: 10, right: 30, left: 0, bottom: 0 }}
            onClick={handleChartClick}
          >
            <CartesianGrid strokeDasharray="3 3" stroke="#e5e7eb" vertical={false} />
            <XAxis
              dataKey="time"
              tick={{ fontSize: 11, fill: '#6b7280' }}
              tickLine={{ stroke: '#e5e7eb' }}
              axisLine={{ stroke: '#e5e7eb' }}
              interval="preserveStartEnd"
              minTickGap={60}
            />
            <YAxis
              tick={{ fontSize: 11, fill: '#6b7280' }}
              tickLine={{ stroke: '#e5e7eb' }}
              axisLine={{ stroke: '#e5e7eb' }}
              tickFormatter={(value) => `${value}%`}
              domain={[0, 'auto']}
            />
            <Tooltip content={<CustomTooltip />} />
            {/* Legend hidden per user request - removed to declutter chart */}
            {renderLines()}
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
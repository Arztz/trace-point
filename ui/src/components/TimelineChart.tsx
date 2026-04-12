import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend,
} from 'recharts';
import type { TimelineMetric, PodInfo } from '../types/timeline';
import { POD_COLORS } from '../types/timeline';

interface PodLineData {
  timestamp: string;
  time: string;
  date: string;
  [key: string]: string | number; // Dynamic pod lines
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
      <div className="bg-white border border-gray-200 rounded-lg shadow-lg p-3 min-w-[200px]">
        <p className="text-xs font-medium text-gray-500 mb-2">{label}</p>
        {payload.map((entry, idx) => (
          <div key={idx} className="flex items-center justify-between gap-3">
            <div className="flex items-center gap-1.5">
              <span 
                className="w-2 h-2 rounded-full" 
                style={{ backgroundColor: entry.color }}
              />
              <span className="text-xs text-gray-600">
                {entry.name}
              </span>
            </div>
            <span className="text-sm font-semibold text-gray-900">
              {entry.value.toFixed(1)}%
            </span>
          </div>
        ))}
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
  // Get unique pods from metrics
  const uniquePods = Array.from(new Set(metrics.map(m => m.pod_name)));
  
  // Determine which pods to display
  const displayPods = selectedPods.length === 0
    ? uniquePods
    : uniquePods.filter(p => selectedPods.includes(p));

  // Transform metrics into chart data format (one row per timestamp)
  const chartData = (() => {
    const dataByTimestamp = new Map<string, PodLineData>();
    
    // Sort metrics by timestamp
    const sortedMetrics = [...metrics].sort((a, b) => 
      a.timestamp.localeCompare(b.timestamp)
    );

    for (const metric of sortedMetrics) {
      if (!dataByTimestamp.has(metric.timestamp)) {
        dataByTimestamp.set(metric.timestamp, {
          timestamp: metric.timestamp,
          time: formatTimestamp(metric.timestamp),
          date: formatDate(metric.timestamp),
        });
      }
      
      const dataPoint = dataByTimestamp.get(metric.timestamp)!;
      
      // Store CPU and RAM for this pod
      dataPoint[`${metric.pod_name}_cpu`] = metric.cpu_percent;
      dataPoint[`${metric.pod_name}_ram`] = metric.ram_percent;
    }

    return Array.from(dataByTimestamp.values());
  })();

  // Get pod index for color assignment
  const getPodIndex = (podName: string): number => {
    const pod = availablePods.find(p => p.name === podName);
    if (pod) {
      return availablePods.indexOf(pod);
    }
    return uniquePods.indexOf(podName);
  };

  const getPodColor = (podName: string): string => {
    const index = getPodIndex(podName);
    return POD_COLORS[index % POD_COLORS.length];
  };

  // Calculate stats for selected pods or all pods
  const stats = (() => {
    const targetPods = displayPods;
    if (targetPods.length === 0 || chartData.length === 0) {
      return { avgCpu: 0, avgRam: 0, maxCpu: 0, maxRam: 0 };
    }

    let totalCpu = 0;
    let totalRam = 0;
    let maxCpu = 0;
    let maxRam = 0;
    let cpuCount = 0;
    let ramCount = 0;

    for (const data of chartData) {
      for (const podName of targetPods) {
        const cpu = data[`${podName}_cpu`] as number;
        const ram = data[`${podName}_ram`] as number;
        
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

  // Generate lines for each pod (CPU and RAM)
  const renderLines = () => {
    const lines: JSX.Element[] = [];

    for (const podName of displayPods) {
      const isHighlighted = highlightedPod === podName;
      const color = getPodColor(podName);
      const strokeWidth = isHighlighted ? 3 : 1;
      const opacity = highlightedPod && !isHighlighted ? 0.3 : 1;

      // CPU line
      lines.push(
        <Line
          key={`${podName}-cpu`}
          type="monotone"
          dataKey={`${podName}_cpu`}
          name={`${podName} (CPU)`}
          stroke={color}
          strokeWidth={strokeWidth}
          dot={false}
          activeDot={{ r: isHighlighted ? 6 : 4, stroke: color, strokeWidth: 2, fill: '#fff' }}
          opacity={opacity}
          connectNulls
        />
      );

      // RAM line (use same color but dashed)
      lines.push(
        <Line
          key={`${podName}-ram`}
          type="monotone"
          dataKey={`${podName}_ram`}
          name={`${podName} (RAM)`}
          stroke={color}
          strokeWidth={strokeWidth}
          strokeDasharray="5 5"
          dot={false}
          activeDot={{ r: isHighlighted ? 6 : 4, stroke: color, strokeWidth: 2, fill: '#fff' }}
          opacity={opacity}
          connectNulls
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
        <span>Showing {displayPods.length} pod{displayPods.length !== 1 ? 's' : ''}</span>
        {highlightedPod && (
          <span className="text-primary-600 font-medium">
            Highlighted: {highlightedPod}
          </span>
        )}
        <span className="text-xs">
          Solid = CPU | Dashed = RAM
        </span>
      </div>

      {/* Chart */}
      <div className="h-80">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart 
            data={chartData} 
            margin={{ top: 10, right: 30, left: 0, bottom: 0 }}
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
            <Legend 
              verticalAlign="top" 
              height={36}
              iconType="circle"
              iconSize={8}
              formatter={(value) => (
                <span className="text-sm font-medium text-gray-700">
                  {value}
                </span>
              )}
            />
            {renderLines()}
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
import { useMemo, useCallback, useState } from 'react';
import type { ChangeEvent } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
} from 'recharts';
import type { TimelineMetric, PodInfo, TimelineSummary } from '../types/timeline';
import { getPodColor } from '../types/timeline';

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
  summary?: TimelineSummary[];
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
  highlightedPod?: string | null;
}

interface DeploymentSummary {
  deployment: string;
  namespace: string;
  avgCpu: number;
  maxCpu: number;
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

const CustomTooltip = ({ active, payload, label, highlightedPod }: CustomTooltipProps) => {
  // Don't show placeholder - only show when actually hovering with data
  if (!active || !payload || payload.length === 0) {
    return null;
  }
  
  // Filter payload to only show highlighted pod when one is selected
  // Backend provides clean names (e.g., "portfolio-service"), so no need to strip suffixes
  const filteredPayload = highlightedPod 
    ? payload.filter(entry => {
        const dataKey = String(entry.dataKey || '').replace('_cpu', '');
        const name = String(entry.name || '').replace(' (CPU)', '');
        // Exact match or contains match - both directions
        return name === highlightedPod || dataKey === highlightedPod ||
               name.includes(highlightedPod) || highlightedPod.includes(name) ||
               dataKey.includes(highlightedPod) || highlightedPod.includes(dataKey);
      })
    : payload;
  
  // If highlighted but no matching pod found, show all (don't hide)
  const displayPayload = highlightedPod && filteredPayload.length === 0 ? payload : filteredPayload;
  
  return (
    <div className="bg-white border border-gray-200 rounded-lg shadow-lg p-3 min-w-[220px]" style={{ visibility: 'visible', opacity: 1 }}>
      <p className="text-xs font-medium text-gray-500 mb-2">{label}</p>
      {displayPayload.map((entry, idx) => {
        const dataKey = entry.dataKey || '';
        const currentValue = entry.value;
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
};

export default function TimelineChart({
  metrics = [],
  availablePods = [],
  selectedPods = [],
  highlightedPod = null,
  summary,
  onHighlight,
}: TimelineChartProps) {
  const [closeTo100Threshold, setCloseTo100Threshold] = useState(85);
  const [farBelow100Threshold, setFarBelow100Threshold] = useState(50);
  const [over100Threshold, setOver100Threshold] = useState(100);
  // Get unique pods from metrics (use pod_name, not replicaset_name)
  const uniquePods = Array.from(new Set(metrics.map(m => m.pod_name)));
  
  // Determine which pods to display
  const displayPods = useMemo(() => 
    selectedPods.length === 0
      ? uniquePods
      : uniquePods.filter(r => selectedPods.includes(r)),
    [uniquePods, selectedPods]
  );

  // Transform metrics into chart data format (one row per timestamp)
  // Aggregate metrics by replicaset (average across all pods in that replicaset)
  const chartData = useMemo(() => {
  // Group metrics by timestamp and replicaset
  const dataMap = new Map<string, Map<string, { cpu: number[] }>>();
  
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
      replicaMap.set(replicasetName, { cpu: [] });
    }
    
    const entry = replicaMap.get(replicasetName)!;
    entry.cpu.push(metric.cpu_percent);
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
      
      dataPoint[`${replicasetName}_cpu`] = avgCpu;
    }
    
    result.push(dataPoint);
  }
  
  return result;
}, [metrics]);

  // Get replicaset index for color assignment
  const getReplicaIndex = (replicasetName: string): number => {
    const pod = availablePods.find(p => p.name === replicasetName);
    if (pod) {
      return availablePods.indexOf(pod);
    }
    return uniquePods.indexOf(replicasetName);
  };

  // Use consistent color based on pod name hash (always use hash, ignore index to prevent order-based colors)
  const getPodColorForReplica = (replicasetName: string): string => {
    return getPodColor(replicasetName); // No index - always use hash for consistent color
  };

  const normalizedSummary = useMemo<DeploymentSummary[]>(() => {
    if (summary && summary.length > 0) {
      return summary.map((entry) => ({
        deployment: entry.deployment,
        namespace: entry.namespace,
        avgCpu: entry.avg_cpu_percent,
        maxCpu: entry.max_cpu_percent,
      }));
    }

    if (metrics.length === 0) {
      return [];
    }

    const accumulator = new Map<string, { deployment: string; namespace: string; sum: number; count: number; max: number }>();
    for (const metric of metrics) {
      const deployment = metric.replicaset_name || metric.pod_name;
      const key = `${metric.namespace}/${deployment}`;
      const existing = accumulator.get(key);
      if (!existing) {
        accumulator.set(key, {
          deployment,
          namespace: metric.namespace,
          sum: metric.cpu_percent,
          count: 1,
          max: metric.cpu_percent,
        });
      } else {
        existing.sum += metric.cpu_percent;
        existing.count += 1;
        if (metric.cpu_percent > existing.max) {
          existing.max = metric.cpu_percent;
        }
      }
    }

    return Array.from(accumulator.values()).map((entry) => ({
      deployment: entry.deployment,
      namespace: entry.namespace,
      avgCpu: entry.count > 0 ? entry.sum / entry.count : 0,
      maxCpu: entry.max,
    }));
  }, [summary, metrics]);

   const classifyDeployment = useCallback((avgCpu: number, maxCpu: number) => {
     const above100 = maxCpu > over100Threshold;
     if (avgCpu >= closeTo100Threshold && above100) {
       return 'needs_more_cpu';
     }
     if (avgCpu <= farBelow100Threshold && !above100) {
       return 'overprovisioned';
     }
     return 'balanced';
   }, [closeTo100Threshold, farBelow100Threshold, over100Threshold]);

   const needsMoreCpu = useMemo(() => (
     normalizedSummary
       .filter((entry) => classifyDeployment(entry.avgCpu, entry.maxCpu) === 'needs_more_cpu')
       .sort((a, b) => b.maxCpu - a.maxCpu)
   ), [normalizedSummary, classifyDeployment]);

   const overprovisioned = useMemo(() => (
     normalizedSummary
       .filter((entry) => classifyDeployment(entry.avgCpu, entry.maxCpu) === 'overprovisioned')
       .sort((a, b) => a.avgCpu - b.avgCpu)
   ), [normalizedSummary, classifyDeployment]);

  // Calculate stats for selected replicasets or all replicasets
  const stats = useMemo(() => {
    const targetReplicasets = displayPods;
    if (targetReplicasets.length === 0 || chartData.length === 0) {
      return { avgCpu: 0, maxCpu: 0 };
    }

    let totalCpu = 0;
    let maxCpu = 0;
    let cpuCount = 0;

    for (const data of chartData) {
      for (const replicasetName of targetReplicasets) {
        const cpu = data[`${replicasetName}_cpu`] as number;
        
        if (cpu !== undefined) {
          totalCpu += cpu;
          cpuCount++;
          if (cpu > maxCpu) maxCpu = cpu;
        }
      }
    }

    return {
      avgCpu: cpuCount > 0 ? totalCpu / cpuCount : 0,
      maxCpu,
    };
  }, [chartData, displayPods]);

  // Handle line click - toggle highlight behavior - simplified
  const handleLineClick = useCallback((replicasetName: string) => {
    // Only handle click if onHighlight is provided
    if (!onHighlight) return;
    
    // Toggle behavior: if already highlighted, unhighlight; otherwise highlight
    if (highlightedPod === replicasetName) {
      onHighlight(null);
    } else {
      onHighlight(replicasetName);
    }
  }, [highlightedPod, onHighlight]);

  // Handle chart click - detect which data point was clicked
  const handleChartClick = useCallback((event: unknown, activePayload: unknown[]) => {
    // Only process if there are active payload elements clicked
    if (!activePayload || activePayload.length === 0 || !onHighlight) {
      return;
    }
    
    // Get the first clicked element data
    const element = activePayload[0] as { dataKey?: unknown } | undefined;
    
    if (element && element.dataKey) {
      const dataKey = String(element.dataKey);
      
      // Extract replicaset name from dataKey (e.g., "game-workflow_cpu" -> "game-workflow")
      const replicasetName = dataKey.replace('_cpu', '');
      
      handleLineClick(replicasetName);
    }
  }, [onHighlight, handleLineClick]);

  // Handle individual line click - more reliable than chart-level onClick
  const handleLineClickEvent = useCallback((replicasetName: string) => {
    return () => {
      handleLineClick(replicasetName);
    };
  }, [handleLineClick]);

  // Generate lines for each replicaset (CPU only)
  const renderLines = () => {
    const lines: JSX.Element[] = [];

    for (const replicasetName of displayPods) {
      const isHighlighted = highlightedPod === replicasetName;
      const isFaded = Boolean(highlightedPod && !isHighlighted);
      const color = getPodColorForReplica(replicasetName);
      const strokeWidth = isHighlighted ? 3 : 1;
      const opacity = isFaded ? 0.2 : 1;

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
    }

    return lines;
  };

  const handleThresholdChange = (setter: (value: number) => void) => {
    return (event: ChangeEvent<HTMLInputElement>) => {
      const value = event.currentTarget.valueAsNumber;
      if (Number.isNaN(value)) {
        return;
      }
      setter(value);
    };
  };

  const renderSummaryTable = (
    title: string,
    description: string,
    rows: DeploymentSummary[],
    emptyMessage: string,
    accentClass: string,
  ) => (
    <div className="bg-white rounded-lg border border-gray-200 overflow-hidden">
      <div className="px-4 py-3 border-b border-gray-100 bg-gray-50">
        <div className="flex items-center justify-between">
          <div>
            <h3 className="text-sm font-semibold text-gray-900">{title}</h3>
            <p className="text-xs text-gray-500">{description}</p>
          </div>
          <span className={`text-xs font-semibold px-2 py-1 rounded-full ${accentClass}`}>
            {rows.length}
          </span>
        </div>
      </div>
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="bg-white border-b border-gray-100">
            <tr>
              <th className="px-4 py-2 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">Deployment</th>
              <th className="px-4 py-2 text-left text-xs font-semibold text-gray-500 uppercase tracking-wider">Namespace</th>
              <th className="px-4 py-2 text-right text-xs font-semibold text-gray-500 uppercase tracking-wider">Avg CPU</th>
              <th className="px-4 py-2 text-right text-xs font-semibold text-gray-500 uppercase tracking-wider">Max CPU</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-100">
            {rows.length === 0 ? (
              <tr>
                <td colSpan={4} className="px-4 py-6 text-center text-sm text-gray-500">
                  {emptyMessage}
                </td>
              </tr>
            ) : (
              rows.map((row) => (
                <tr key={`${row.namespace}/${row.deployment}`} className="hover:bg-gray-50 transition-colors">
                  <td className="px-4 py-2 whitespace-nowrap font-medium text-gray-900">
                    {row.deployment}
                  </td>
                  <td className="px-4 py-2 whitespace-nowrap text-gray-600">
                    {row.namespace}
                  </td>
                  <td className="px-4 py-2 whitespace-nowrap text-right text-gray-700">
                    {row.avgCpu.toFixed(1)}%
                  </td>
                  <td className="px-4 py-2 whitespace-nowrap text-right text-gray-700">
                    {row.maxCpu.toFixed(1)}%
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
    </div>
  );

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
      <div className="grid grid-cols-2 gap-4">
        <div className="bg-primary-50 rounded-lg p-3">
          <p className="text-xs text-primary-600 font-medium">Avg CPU</p>
          <p className="text-xl font-bold text-primary-700">{stats.avgCpu.toFixed(1)}%</p>
        </div>
        <div className="bg-red-50 rounded-lg p-3">
          <p className="text-xs text-red-600 font-medium">Max CPU</p>
          <p className="text-xl font-bold text-red-700">{stats.maxCpu.toFixed(1)}%</p>
        </div>
      </div>

      {/* Legend info */}
      <div className="text-sm text-gray-500 flex flex-wrap items-center gap-4">
        <span>Showing {displayPods.length} replicaset{displayPods.length !== 1 ? 's' : ''}</span>
        {highlightedPod && (
          <span
            className="text-primary-600 font-medium break-words whitespace-normal"
            title={highlightedPod}
          >
            Highlighted: {highlightedPod}
          </span>
        )}
        <span className="text-xs">
          CPU utilization (% of requested)
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
            <Tooltip 
              content={<CustomTooltip highlightedPod={highlightedPod} />} 
              isAnimationActive={false}
              wrapperStyle={{ visibility: 'visible', opacity: 1, zIndex: 9999, position: 'relative' }}
              position={{ x: 0, y: 0 }}
            />
            {/* Legend hidden per user request - removed to declutter chart */}
            {renderLines()}
          </LineChart>
        </ResponsiveContainer>
      </div>

      {/* Summary Thresholds */}
      <div className="bg-white rounded-lg border border-gray-200 p-4">
        <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-3">
          <div>
            <h3 className="text-sm font-semibold text-gray-900">CPU Allocation Thresholds</h3>
            <p className="text-xs text-gray-500">Adjust thresholds to refine deployment classification.</p>
          </div>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 w-full sm:w-auto">
            <label className="text-xs text-gray-600 flex items-center justify-between gap-2">
              Close to 100% ≥
              <input
                type="number"
                min={0}
                max={200}
                step={1}
                className="w-20 rounded-md border border-gray-200 px-2 py-1 text-xs text-gray-700"
                value={closeTo100Threshold}
                onChange={handleThresholdChange(setCloseTo100Threshold)}
              />
            </label>
            <label className="text-xs text-gray-600 flex items-center justify-between gap-2">
              Far below 100% ≤
              <input
                type="number"
                min={0}
                max={200}
                step={1}
                className="w-20 rounded-md border border-gray-200 px-2 py-1 text-xs text-gray-700"
                value={farBelow100Threshold}
                onChange={handleThresholdChange(setFarBelow100Threshold)}
              />
            </label>
            <label className="text-xs text-gray-600 flex items-center justify-between gap-2">
              Over 100% &gt;
              <input
                type="number"
                min={0}
                max={200}
                step={1}
                className="w-20 rounded-md border border-gray-200 px-2 py-1 text-xs text-gray-700"
                value={over100Threshold}
                onChange={handleThresholdChange(setOver100Threshold)}
              />
            </label>
          </div>
        </div>
      </div>

      {/* Deployment Summary Tables */}
      <div className="grid grid-cols-1 xl:grid-cols-2 gap-4">
        {renderSummaryTable(
          'Needs More CPU',
          'Avg CPU close to 100% with at least one sample above 100%.',
          needsMoreCpu,
          'No deployments match the current thresholds.',
          'bg-red-50 text-red-600'
        )}
        {renderSummaryTable(
          'Over-provisioned',
          'Avg CPU far below 100% and never above 100%.',
          overprovisioned,
          'No deployments match the current thresholds.',
          'bg-amber-50 text-amber-600'
        )}
      </div>
    </div>
  );
}

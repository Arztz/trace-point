import { useState, useMemo } from 'react';
import type { SpikeEvent } from '../types';
import { FiActivity, FiCpu, FiServer, FiChevronRight, FiClock, FiAlertTriangle, FiArrowDown, FiArrowUp } from 'react-icons/fi';
import { TbCpu } from 'react-icons/tb';

// Sort type definitions
type SortField = 'timestamp' | 'cpu' | 'ram' | 'podName';
type SortOrder = 'asc' | 'desc';

interface SortOption {
  value: SortField;
  label: string;
  direction: SortOrder;
}

interface SpikeListProps {
  spikes: SpikeEvent[];
  selectedId: string | null;
  onSelect: (id: string) => void;
}

// Sort options configuration
const SORT_OPTIONS: SortOption[] = [
  { value: 'timestamp', label: 'Time', direction: 'desc' },
  { value: 'timestamp', label: 'Time', direction: 'asc' },
  { value: 'cpu', label: 'CPU', direction: 'desc' },
  { value: 'cpu', label: 'CPU', direction: 'asc' },
  { value: 'ram', label: 'RAM', direction: 'desc' },
  { value: 'ram', label: 'RAM', direction: 'asc' },
  { value: 'podName', label: 'Pod Name', direction: 'asc' },
  { value: 'podName', label: 'Pod Name', direction: 'desc' },
];

// Transform raw backend data to frontend format
interface TransformedSpike {
  id: string;
  timestamp: string;
  podName: string;
  namespace: string;
  resourceType: 'cpu' | 'memory';
  threshold: number;
  currentValue: number;
  movingAverage: number;
  activeRoutes: string[];
  possibleRootCauses: Array<{ route: string; functionName: string; confidence: number; filePath: string }>;
}

function transformSpike(raw: Record<string, unknown>): TransformedSpike {
  // Handle both formats: frontend format and raw backend database format
  const isFrontendFormat = 'resourceType' in raw && raw.resourceType !== undefined;
  
  if (isFrontendFormat) {
    return {
      id: String(raw.id || ''),
      timestamp: String(raw.timestamp || ''),
      podName: String(raw.podName || ''),
      namespace: String(raw.namespace || ''),
      resourceType: String(raw.resourceType || 'cpu') as 'cpu' | 'memory',
      threshold: Number(raw.threshold) || 0,
      currentValue: Number(raw.currentValue) || 0,
      movingAverage: Number(raw.movingAverage) || 0,
      activeRoutes: Array.isArray(raw.activeRoutes) ? raw.activeRoutes : [],
      possibleRootCauses: Array.isArray(raw.possibleRootCauses) ? raw.possibleRootCauses : [],
    };
  }
  
  // Raw backend format - map to frontend format
  // Database fields: pod_name, namespace, cpu_usage_percent, ram_usage_percent, threshold_percent, moving_average_percent
  const cpuUsage = Number(raw.cpu_usage_percent || raw.cpuUsagePercent || 0);
  const ramUsage = Number(raw.ram_usage_percent || raw.ramUsagePercent || 0);
  const threshold = Number(raw.threshold_percent || raw.thresholdPercent || 50);
  const avg = Number(raw.moving_average_percent || raw.movingAveragePercent || 0);
  
  // Determine resource type based on which has higher usage
  const resourceType: 'cpu' | 'memory' = cpuUsage >= ramUsage ? 'cpu' : 'memory';
  const currentValue = resourceType === 'cpu' ? cpuUsage : ramUsage;
  
  // Get route name if available
  const routeName = raw.route_name || raw.routeName;
  const activeRoutes = routeName ? [String(routeName)] : [];
  
  return {
    id: String(raw.id || ''),
    timestamp: String(raw.timestamp || raw.created_at || ''),
    podName: String(raw.pod_name || raw.podName || ''),
    namespace: String(raw.namespace || ''),
    resourceType,
    threshold,
    currentValue,
    movingAverage: avg,
    activeRoutes,
    possibleRootCauses: [],
  };
}

function formatTime(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function getSpikeSeverity(currentValue: number, movingAverage: number): 'critical' | 'warning' | 'normal' {
  if (movingAverage <= 0) return 'normal';
  const ratio = currentValue / movingAverage;
  if (ratio >= 3) return 'critical';
  if (ratio >= 2) return 'warning';
  return 'normal';
}

function getSeverityColor(severity: string): string {
  switch (severity) {
    case 'critical':
      return 'text-red-600 bg-red-50 border-red-200';
    case 'warning':
      return 'text-amber-600 bg-amber-50 border-amber-200';
    default:
      return 'text-primary-600 bg-primary-50 border-primary-200';
  }
}

function getResourceIcon(resourceType: string) {
  if (resourceType === 'cpu') {
    return <TbCpu className="w-4 h-4" />;
  }
  return <FiServer className="w-4 h-4" />;
}

function getResourceColor(resourceType: string): string {
  if (resourceType === 'cpu') {
    return 'text-primary-600 bg-primary-100';
  }
  return 'text-purple-600 bg-purple-100';
}

export default function SpikeList({ spikes, selectedId, onSelect }: SpikeListProps) {
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [sortBy, setSortBy] = useState<SortOption>(SORT_OPTIONS[0]);

  // Sort spikes based on selected sort option
  const sortedSpikes = useMemo(() => {
    const transformed = spikes.map(s => transformSpike(s as unknown as Record<string, unknown>));
    
    return [...transformed].sort((a, b) => {
      let comparison = 0;
      
      switch (sortBy.value) {
        case 'timestamp':
          comparison = a.timestamp.localeCompare(b.timestamp);
          break;
        case 'cpu':
          comparison = a.currentValue - b.currentValue;
          break;
        case 'ram':
          // For RAM, use movingAverage as proxy since it's the secondary metric
          comparison = a.movingAverage - b.movingAverage;
          break;
        case 'podName':
          comparison = a.podName.localeCompare(b.podName);
          break;
      }
      
      return sortBy.direction === 'asc' ? comparison : -comparison;
    });
  }, [spikes, sortBy]);

  // Transform spikes to handle both formats
  const transformedSpikes = sortedSpikes;

  if (transformedSpikes.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-center">
        <div className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mb-4">
          <FiActivity className="w-8 h-8 text-gray-400" />
        </div>
        <p className="text-gray-500 font-medium">No spikes detected</p>
        <p className="text-gray-400 text-sm mt-1">Spikes will appear here when detected</p>
      </div>
    );
  }

  const handleToggleExpand = (id: string, e: React.MouseEvent) => {
    e.stopPropagation();
    setExpandedId(expandedId === id ? null : id);
  };

  // Get display label for current sort
  const getSortLabel = (option: SortOption): string => {
    if (option.value === 'timestamp') {
      return option.direction === 'desc' ? 'Newest' : 'Oldest';
    }
    if (option.value === 'podName') {
      return option.direction === 'asc' ? 'A-Z' : 'Z-A';
    }
    return `${option.label} ${option.direction === 'desc' ? 'High' : 'Low'}`;
  };

  return (
    <div>
      {/* Sort controls header */}
      <div className="flex items-center justify-between mb-3 px-1">
        <span className="text-xs text-gray-500 font-medium">
          {transformedSpikes.length} spike{transformedSpikes.length !== 1 ? 's' : ''}
        </span>
        <div className="relative">
          <select
            value={`${sortBy.value}-${sortBy.direction}`}
            onChange={(e) => {
              const [value, direction] = e.target.value.split('-') as [SortField, SortOrder];
              setSortBy({ value, label: '', direction });
            }}
            className="text-xs pl-2 pr-6 py-1 appearance-none bg-gray-50 border border-gray-200 rounded-md text-gray-600 cursor-pointer hover:bg-gray-100 focus:outline-none focus:ring-1 focus:ring-primary-500"
          >
            {SORT_OPTIONS.map((option) => (
              <option key={`${option.value}-${option.direction}`} value={`${option.value}-${option.direction}`}>
                {getSortLabel(option)}
              </option>
            ))}
          </select>
          <FiChevronRight className="absolute right-1.5 top-1/2 -translate-y-1/2 w-3 h-3 text-gray-400 pointer-events-none rotate-90" />
        </div>
      </div>

      {/* Spike list */}
      <div className="space-y-3 max-h-[500px] overflow-y-auto custom-scrollbar pr-1">
        {transformedSpikes.map((spike) => {
        const severity = getSpikeSeverity(spike.currentValue, spike.movingAverage);
        const isExpanded = expandedId === spike.id;
        const isSelected = selectedId === spike.id;
        // Calculate % above average - guard against very small moving averages to prevent unrealistically high ratios
        let spikeRatio = '0';
        if (spike.movingAverage > 1) {  // Only calculate if moving average is > 1% (reasonable baseline)
          const ratio = (spike.currentValue / spike.movingAverage) * 100 - 100;
          spikeRatio = Math.min(ratio, 9999).toFixed(0);  // Cap at 9999% to prevent display overflow
        }

        return (
          <div
            key={spike.id}
            onClick={() => onSelect(spike.id)}
            className={`
              relative overflow-hidden rounded-lg border transition-all duration-200 cursor-pointer
              ${isSelected 
                ? 'ring-2 ring-primary-500 border-primary-300 shadow-md' 
                : 'border-gray-200 hover:border-gray-300 hover:shadow-sm'
              }
              ${getSeverityColor(severity)}
            `}
          >
            {/* Severity indicator bar */}
            <div 
              className={`
                absolute left-0 top-0 bottom-0 w-1
                ${severity === 'critical' ? 'bg-red-500' : severity === 'warning' ? 'bg-amber-500' : 'bg-primary-500'}
              `}
            />

            <div className="p-4 pl-5">
              {/* Header */}
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <span className={`p-1.5 rounded-md ${getResourceColor(spike.resourceType)}`}>
                    {getResourceIcon(spike.resourceType)}
                  </span>
                  <span className="font-semibold text-sm text-gray-900 capitalize">
                    {spike.resourceType === 'cpu' ? 'CPU' : 'Memory'} Spike
                  </span>
                  {severity === 'critical' && (
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-xs font-medium bg-red-100 text-red-700">
                      <FiAlertTriangle className="w-3 h-3" />
                      Critical
                    </span>
                  )}
                </div>
                <button
                  onClick={(e) => handleToggleExpand(spike.id, e)}
                  className="p-1 rounded hover:bg-gray-100 text-gray-400 transition-colors"
                >
                  <FiChevronRight 
                    className={`w-4 h-4 transition-transform duration-200 ${isExpanded ? 'rotate-90' : ''}`} 
                  />
                </button>
              </div>

              {/* Pod info */}
              <div className="flex items-center gap-2 mb-3">
                <FiServer className="w-3.5 h-3.5 text-gray-400" />
                <span className="text-sm text-gray-600 truncate font-mono">
                  {spike.podName}
                </span>
              </div>

              {/* Metrics row */}
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-4">
                  <div>
                    <p className="text-xs text-gray-500">Current</p>
                    <p className={`text-lg font-bold ${severity === 'critical' ? 'text-red-600' : severity === 'warning' ? 'text-amber-600' : 'text-gray-900'}`}>
                      {spike.currentValue.toFixed(1)}%
                    </p>
                  </div>
                  <div>
                    <p className="text-xs text-gray-500">Average</p>
                    <p className="text-sm font-medium text-gray-600">
                      {spike.movingAverage.toFixed(1)}%
                    </p>
                  </div>
                  <div>
                    <p className="text-xs text-gray-500">Threshold</p>
                    <p className="text-sm font-medium text-gray-600">
                      {spike.threshold}%
                    </p>
                  </div>
                </div>
                
                <div className="text-right">
                  <div className="flex items-center gap-1 text-xs text-gray-500">
                    <FiClock className="w-3 h-3" />
                    {formatTime(spike.timestamp)}
                  </div>
                  <p className="text-xs text-amber-600 font-medium mt-1">
                    +{spikeRatio}% above avg
                  </p>
                </div>
              </div>

              {/* Expanded content - Root causes */}
              {isExpanded && spike.possibleRootCauses && spike.possibleRootCauses.length > 0 && (
                <div className="mt-4 pt-4 border-t border-gray-200 animate-fade-in">
                  <p className="text-xs font-semibold text-gray-500 uppercase tracking-wide mb-3">
                    Possible Root Causes
                  </p>
                  <div className="space-y-2">
                    {spike.possibleRootCauses.slice(0, 3).map((cause, idx) => (
                      <div 
                        key={idx} 
                        className="bg-gray-50 rounded-md p-2.5 text-xs"
                      >
                        <div className="flex items-center justify-between mb-1">
                          <span className="font-mono text-primary-600">{cause.route}</span>
                          <span className={`
                            px-1.5 py-0.5 rounded text-xs font-medium
                            ${cause.confidence >= 0.8 ? 'bg-green-100 text-green-700' : 
                              cause.confidence >= 0.5 ? 'bg-amber-100 text-amber-700' : 
                              'bg-gray-100 text-gray-600'}
                          `}>
                            {(cause.confidence * 100).toFixed(0)}%
                          </span>
                        </div>
                        <p className="text-gray-600">
                          <span className="font-medium">Function:</span> {cause.functionName}
                        </p>
                        {cause.filePath && (
                          <p className="text-gray-500 truncate mt-1 font-mono text-xs">
                            {cause.filePath}
                          </p>
                        )}
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          </div>
        );
      })}
      </div>
    </div>
  );
}
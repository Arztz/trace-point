import { useState, useMemo } from 'react';
import { FiChevronDown, FiChevronUp, FiServer, FiCpu, FiActivity } from 'react-icons/fi';
import { TbCpu } from 'react-icons/tb';
import type { HistoricalSpike } from '../types';

interface SpikeAnalysisTableProps {
  spikes: HistoricalSpike[];
  isLoading?: boolean;
}

// Sort field types
type SortField = 'timestamp' | 'replicaset_name' | 'pod_name' | 'type' | 'deviation_percent' | 'severity';
type SortOrder = 'asc' | 'desc';

// Sort configurations
const SORT_OPTIONS: { value: SortField; label: string }[] = [
  { value: 'timestamp', label: 'Time' },
  { value: 'replicaset_name', label: 'Replicaset' },
  { value: 'pod_name', label: 'Pod' },
  { value: 'type', label: 'Type' },
  { value: 'deviation_percent', label: 'Deviation' },
  { value: 'severity', label: 'Severity' },
];

// Format timestamp
function formatTimestamp(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

// Extract replicaset name (remove -xxxx suffix)
function getReplicasetName(podName: string): string {
  // Match pattern: name-number -> name
  const match = podName.match(/^(.+)-\d+$/);
  return match ? match[1] : podName;
}

// Get severity badge styles
function getSeverityStyles(severity: string): { bg: string; text: string; label: string } {
  switch (severity) {
    case 'critical':
      return { bg: 'bg-red-100', text: 'text-red-700', label: 'Critical' };
    case 'medium':
      return { bg: 'bg-orange-100', text: 'text-orange-700', label: 'Medium' };
    case 'low':
      return { bg: 'bg-yellow-100', text: 'text-yellow-700', label: 'Low' };
    default:
      return { bg: 'bg-green-100', text: 'text-green-700', label: 'Normal' };
  }
}

// Get type badge styles
function getTypeStyles(type: string): { bg: string; text: string; icon: JSX.Element } {
  switch (type) {
    case 'cpu':
      return {
        bg: 'bg-primary-100',
        text: 'text-primary-700',
        icon: <TbCpu className="w-3 h-3" />,
      };
    case 'ram':
      return {
        bg: 'bg-purple-100',
        text: 'text-purple-700',
        icon: <FiActivity className="w-3 h-3" />,
      };
    case 'both':
      return {
        bg: 'bg-amber-100',
        text: 'text-amber-700',
        icon: <FiCpu className="w-3 h-3" />,
      };
    default:
      return {
        bg: 'bg-gray-100',
        text: 'text-gray-700',
        icon: <FiServer className="w-3 h-3" />,
      };
  }
}

export default function SpikeAnalysisTable({ spikes, isLoading }: SpikeAnalysisTableProps) {
  const [sortField, setSortField] = useState<SortField>('timestamp');
  const [sortOrder, setSortOrder] = useState<SortOrder>('desc');
  
  // Sort spikes
  const sortedSpikes = useMemo(() => {
    return [...spikes].sort((a, b) => {
      let comparison = 0;
      
      switch (sortField) {
        case 'timestamp':
          comparison = new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime();
          break;
        case 'replicaset_name':
          comparison = a.replicaset_name.localeCompare(b.replicaset_name);
          break;
        case 'pod_name':
          comparison = a.pod_name.localeCompare(b.pod_name);
          break;
        case 'type':
          comparison = a.type.localeCompare(b.type);
          break;
        case 'deviation_percent':
          comparison = a.deviation_percent - b.deviation_percent;
          break;
        case 'severity':
          // Order: critical > medium > low > normal
          const severityOrder: Record<string, number> = { critical: 4, medium: 3, low: 2, normal: 1 };
          comparison = (severityOrder[a.severity] || 0) - (severityOrder[b.severity] || 0);
          break;
      }
      
      return sortOrder === 'asc' ? comparison : -comparison;
    });
  }, [spikes, sortField, sortOrder]);
  
  // Handle sort click
  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc');
    } else {
      setSortField(field);
      setSortOrder('desc'); // Default to descending for new field
    }
  };
  
  // Get sort icon
  const getSortIcon = (field: SortField) => {
    if (sortField !== field) return null;
    return sortOrder === 'asc' 
      ? <FiChevronUp className="w-4 h-4" />
      : <FiChevronDown className="w-4 h-4" />;
  };
  
  if (isLoading) {
    return (
      <div className="card">
        <div className="card-header">
          <h3 className="text-lg font-semibold text-gray-900">Analysis Results</h3>
        </div>
        <div className="card-body">
          <div className="animate-pulse">
            <div className="h-10 bg-gray-200 rounded mb-4"></div>
            <div className="space-y-3">
              {[...Array(5)].map((_, i) => (
                <div key={i} className="h-12 bg-gray-200 rounded"></div>
              ))}
            </div>
          </div>
        </div>
      </div>
    );
  }
  
  if (spikes.length === 0) {
    return (
      <div className="card">
        <div className="card-header">
          <h3 className="text-lg font-semibold text-gray-900">Analysis Results</h3>
        </div>
        <div className="card-body">
          <div className="flex flex-col items-center justify-center py-12 text-center">
            <div className="w-16 h-16 bg-gray-100 rounded-full flex items-center justify-center mb-4">
              <FiActivity className="w-8 h-8 text-gray-400" />
            </div>
            <p className="text-gray-500 font-medium">No spikes found</p>
            <p className="text-gray-400 text-sm mt-1">
              Try adjusting the time range or filters
            </p>
          </div>
        </div>
      </div>
    );
  }
  
  return (
    <div className="card">
      <div className="card-header flex items-center justify-between">
        <h3 className="text-lg font-semibold text-gray-900">Analysis Results</h3>
        <span className="text-sm text-gray-500">
          {sortedSpikes.length} spike{sortedSpikes.length !== 1 ? 's' : ''} found
        </span>
      </div>
      <div className="card-body p-0">
        <div className="overflow-x-auto">
          <table className="w-full">
            <thead>
              <tr className="bg-gray-50 border-b border-gray-200">
                {SORT_OPTIONS.map((option) => (
                  <th
                    key={option.value}
                    onClick={() => handleSort(option.value)}
                    className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider cursor-pointer hover:bg-gray-100 select-none"
                  >
                    <div className="flex items-center gap-1">
                      {option.label}
                      {getSortIcon(option.value)}
                    </div>
                  </th>
                ))}
                <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">
                  CPU %
                </th>
                <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">
                  RAM %
                </th>
                <th className="px-4 py-3 text-left text-xs font-semibold text-gray-600 uppercase tracking-wider">
                  Avg %
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-100">
              {sortedSpikes.map((spike) => {
                const severityStyles = getSeverityStyles(spike.severity);
                const typeStyles = getTypeStyles(spike.type);
                
                return (
                  <tr key={spike.id} className="hover:bg-gray-50 transition-colors">
                    {/* Timestamp */}
                    <td className="px-4 py-3 text-sm text-gray-900 whitespace-nowrap">
                      {formatTimestamp(spike.timestamp)}
                    </td>
                    
                    {/* Replicaset */}
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-2">
                        <FiServer className="w-4 h-4 text-gray-400" />
                        <span className="text-sm font-medium text-gray-900">
                          {getReplicasetName(spike.pod_name)}
                        </span>
                      </div>
                    </td>
                    
                    {/* Pod */}
                    <td className="px-4 py-3">
                      <span className="text-sm text-gray-600 font-mono">
                        {spike.pod_name}
                      </span>
                    </td>
                    
                    {/* Type */}
                    <td className="px-4 py-3">
                      <span className={`inline-flex items-center gap-1 px-2 py-1 rounded-md text-xs font-medium ${typeStyles.bg} ${typeStyles.text}`}>
                        {typeStyles.icon}
                        {spike.type.toUpperCase()}
                      </span>
                    </td>
                    
                    {/* Deviation */}
                    <td className="px-4 py-3">
                      <span className={`text-sm font-semibold ${
                        spike.deviation_percent >= 200 ? 'text-red-600' :
                        spike.deviation_percent >= 100 ? 'text-orange-600' :
                        spike.deviation_percent >= 50 ? 'text-yellow-600' :
                        'text-green-600'
                      }`}>
                        +{spike.deviation_percent.toFixed(1)}%
                      </span>
                    </td>
                    
                    {/* Severity */}
                    <td className="px-4 py-3">
                      <span className={`inline-flex items-center px-2 py-1 rounded-full text-xs font-medium ${severityStyles.bg} ${severityStyles.text}`}>
                        {severityStyles.label}
                      </span>
                    </td>
                    
                    {/* CPU */}
                    <td className="px-4 py-3">
                      <span className="text-sm text-gray-900">
                        {spike.cpu_percent.toFixed(1)}%
                      </span>
                    </td>
                    
                    {/* RAM */}
                    <td className="px-4 py-3">
                      <span className="text-sm text-gray-900">
                        {spike.ram_percent.toFixed(1)}%
                      </span>
                    </td>
                    
                    {/* Moving Average */}
                    <td className="px-4 py-3">
                      <span className="text-sm text-gray-500">
                        {spike.type === 'cpu' || spike.type === 'both'
                          ? spike.moving_average_cpu.toFixed(1)
                          : spike.moving_average_ram.toFixed(1)}%
                      </span>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
}
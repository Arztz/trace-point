import { useState, useMemo } from 'react';
import { FiAlertTriangle, FiTrendingUp, FiArrowUp, FiArrowDown } from 'react-icons/fi';
import type { PodInfo } from '../types/timeline';
import { getPodColor } from '../types/timeline';

interface PodLegendProps {
  pods: PodInfo[];
  selectedPods: string[];
  highlightedPod: string | null;
  onHighlight: (podName: string | null) => void;
  showAllPods?: boolean;
}

type SortField = 'name' | 'cpu' | 'ram';
type SortOrder = 'asc' | 'desc';

export default function PodLegend({
  pods,
  selectedPods,
  highlightedPod,
  onHighlight,
  showAllPods = true,
}: PodLegendProps) {
  const [sortBy, setSortBy] = useState<{ field: SortField; order: SortOrder }>({ field: 'name', order: 'asc' });

  const displayPods = useMemo(() => {
    const filtered = selectedPods.length === 0 || selectedPods.length === pods.length
      ? pods
      : pods.filter((p) => selectedPods.includes(p.name));

    // Sort pods
    return [...filtered].sort((a, b) => {
      let comparison = 0;
      switch (sortBy.field) {
        case 'name':
          comparison = a.name.localeCompare(b.name);
          break;
        case 'cpu':
          comparison = (a.cpu_percent || 0) - (b.cpu_percent || 0);
          break;
        case 'ram':
          comparison = (a.ram_percent || 0) - (b.ram_percent || 0);
          break;
      }
      return sortBy.order === 'asc' ? comparison : -comparison;
    });
  }, [pods, selectedPods, sortBy]);

  const handlePodClick = (podName: string) => {
    if (highlightedPod === podName) {
      onHighlight(null);
    } else {
      onHighlight(podName);
    }
  };

  if (displayPods.length === 0) {
    return null;
  }

  return (
    <div className="space-y-2">
      <div className="flex items-center justify-between">
        <h4 className="text-xs font-semibold text-gray-500 uppercase tracking-wider">
          Replicasets ({displayPods.length})
        </h4>
        <div className="flex gap-1">
          {(['name', 'cpu', 'ram'] as const).map((field) => (
            <button
              key={field}
              type="button"
              onClick={() => setSortBy((prev) => ({
                field,
                order: prev.field === field && prev.order === 'asc' ? 'desc' : 'asc',
              }))}
              className={`p-1 rounded transition-colors ${
                sortBy.field === field
                  ? 'text-primary-600 bg-primary-50'
                  : 'text-gray-400 hover:text-gray-600 hover:bg-gray-100'
              }`}
              title={`Sort by ${field}`}
            >
              {field === 'name' ? (
                <span className="text-xs font-medium">A-Z</span>
              ) : (
                <span className="text-xs font-medium">{field.toUpperCase()}</span>
              )}
            </button>
          ))}
        </div>
      </div>
      <div className="space-y-1 max-h-64 overflow-y-auto">
        {displayPods.map((pod) => {
          const isHighlighted = highlightedPod === pod.name;
          const color = getPodColor(pod.name); // No index - always use hash for consistent color

          return (
            <button
              key={pod.name}
              type="button"
              onClick={() => handlePodClick(pod.name)}
              className={`w-full flex items-center gap-2 px-2 py-1.5 rounded-md transition-all text-left ${
                isHighlighted
                  ? 'bg-primary-100 border border-primary-300'
                  : 'hover:bg-gray-50 border border-transparent'
              }`}
            >
              {/* Color indicator */}
              <div
                className={`w-3 h-3 rounded-full flex-shrink-0 ${
                  isHighlighted ? 'ring-2 ring-primary-500' : ''
                }`}
                style={{ backgroundColor: color }}
              />

              {/* Pod name */}
              <div className="flex-1 min-w-0">
                <span
                  className={`text-sm truncate block ${
                    isHighlighted ? 'font-semibold text-gray-900' : 'text-gray-700'
                  }`}
                  title={pod.name}
                >
                  {pod.name}
                </span>
              </div>

              {/* Metrics */}
              <div className="flex items-center gap-2 text-xs text-gray-500">
                <span
                  className="flex items-center gap-0.5"
                  title="CPU"
                >
                  <FiTrendingUp className="w-3 h-3" />
                  {typeof pod.cpu_percent === 'number' ? pod.cpu_percent.toFixed(1) : '0'}%
                </span>
                <span
                  className="flex items-center gap-0.5"
                  title="RAM"
                >
                  {typeof pod.ram_percent === 'number' ? pod.ram_percent.toFixed(1) : '0'}%
                </span>
              </div>

              {/* Spike indicator */}
              {pod.hasSpike && (
                <div className="flex-shrink-0" title="Has active spike">
                  <FiAlertTriangle className="w-3.5 h-3.5 text-red-500" />
                </div>
              )}

              {/* Highlight indicator */}
              {isHighlighted && (
                <span className="text-xs text-primary-600 font-medium">
                  (highlighted)
                </span>
              )}
            </button>
          );
        })}
      </div>

      {/* Clear highlight button */}
      {highlightedPod && (
        <button
          type="button"
          onClick={() => onHighlight(null)}
          className="w-full text-xs text-gray-500 hover:text-gray-700 py-1"
        >
          Clear highlight
        </button>
      )}
    </div>
  );
}
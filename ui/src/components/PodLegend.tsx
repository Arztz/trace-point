import { FiAlertTriangle, FiTrendingUp } from 'react-icons/fi';
import type { PodInfo } from '../types/timeline';
import { POD_COLORS } from '../types/timeline';

interface PodLegendProps {
  pods: PodInfo[];
  selectedPods: string[];
  highlightedPod: string | null;
  onHighlight: (podName: string | null) => void;
  showAllPods?: boolean;
}

export default function PodLegend({
  pods,
  selectedPods,
  highlightedPod,
  onHighlight,
  showAllPods = true,
}: PodLegendProps) {
  const displayPods = selectedPods.length === 0 || selectedPods.length === pods.length
    ? pods
    : pods.filter((p) => selectedPods.includes(p.name));

  const getPodColor = (index: number): string => {
    return POD_COLORS[index % POD_COLORS.length];
  };

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
      <h4 className="text-xs font-semibold text-gray-500 uppercase tracking-wider">
        Pods ({displayPods.length})
      </h4>
      <div className="space-y-1 max-h-64 overflow-y-auto">
        {displayPods.map((pod, index) => {
          const isHighlighted = highlightedPod === pod.name;
          const color = getPodColor(index);

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
                  {pod.cpu_percent.toFixed(1)}%
                </span>
                <span
                  className="flex items-center gap-0.5"
                  title="RAM"
                >
                  {pod.ram_percent.toFixed(1)}%
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
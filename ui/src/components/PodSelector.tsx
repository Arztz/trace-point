import { useState, useRef, useEffect, useMemo } from 'react';
import { FiChevronDown, FiCheck, FiX, FiSearch, FiArrowUp, FiArrowDown } from 'react-icons/fi';
import type { PodInfo } from '../types/timeline';
import { getPodColor } from '../types/timeline';

interface PodSelectorProps {
  pods: PodInfo[];
  selectedPods: string[];
  onSelectedPodsChange: (pods: string[]) => void;
}

type SortField = 'name' | 'cpu' | 'ram';
type SortOrder = 'asc' | 'desc';

export default function PodSelector({
  pods,
  selectedPods,
  onSelectedPodsChange,
}: PodSelectorProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [search, setSearch] = useState('');
  const [sortBy, setSortBy] = useState<{ field: SortField; order: SortOrder }>({ field: 'name', order: 'asc' });
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    }

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  // Filter and sort pods
  const filteredPods = useMemo(() => {
    let result = search
      ? pods.filter((pod) =>
          pod.name.toLowerCase().includes(search.toLowerCase()) ||
          pod.namespace.toLowerCase().includes(search.toLowerCase())
        )
      : pods;

    // Sort pods
    result = [...result].sort((a, b) => {
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

    return result;
  }, [pods, search, sortBy]);

  const handleTogglePod = (podName: string) => {
    if (selectedPods.includes(podName)) {
      onSelectedPodsChange(selectedPods.filter((p) => p !== podName));
    } else {
      onSelectedPodsChange([...selectedPods, podName]);
    }
  };

  const handleSelectAll = () => {
    onSelectedPodsChange(pods.map((p) => p.name));
  };

  const handleDeselectAll = () => {
    onSelectedPodsChange([]);
  };

  const displayText = selectedPods.length === 0
    ? `All pods (${pods.length})`
    : selectedPods.length === pods.length
    ? `All ${pods.length} pods selected`
    : `${selectedPods.length} pod${selectedPods.length === 1 ? '' : 's'} selected`;

  return (
    <div ref={dropdownRef} className="relative">
      <label className="block text-xs font-medium text-gray-600 mb-1.5">
        Pods
      </label>
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="w-full input flex items-center justify-between cursor-pointer"
      >
        <span className="truncate">{displayText}</span>
        <FiChevronDown
          className={`w-4 h-4 text-gray-400 transition-transform ${
            isOpen ? 'rotate-180' : ''
          }`}
        />
      </button>

      {/* Badge showing selected count */}
      {selectedPods.length > 0 && selectedPods.length < pods.length && (
        <span className="absolute -top-1 -right-1 w-5 h-5 bg-primary-600 text-white text-xs font-medium rounded-full flex items-center justify-center">
          {selectedPods.length}
        </span>
      )}

      {/* Dropdown Menu */}
      {isOpen && (
        <div className="absolute z-50 w-full mt-1 bg-white border border-gray-200 rounded-lg shadow-lg max-h-80 flex flex-col">
          {/* Search Input */}
          <div className="p-2 border-b border-gray-100">
            <div className="relative">
              <FiSearch className="absolute left-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400" />
              <input
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search pods..."
                className="input pl-9 py-1.5 text-sm"
                onClick={(e) => e.stopPropagation()}
              />
            </div>
          </div>

          {/* Sort Controls */}
          <div className="px-2 py-1.5 border-b border-gray-100 flex items-center gap-2">
            <span className="text-xs text-gray-400">Sort:</span>
            <div className="flex gap-1">
              {(['name', 'cpu', 'ram'] as const).map((field) => (
                <button
                  key={field}
                  type="button"
                  onClick={() => setSortBy((prev) => ({
                    field,
                    order: prev.field === field && prev.order === 'asc' ? 'desc' : 'asc',
                  }))}
                  className={`px-2 py-0.5 text-xs rounded transition-colors ${
                    sortBy.field === field
                      ? 'bg-primary-100 text-primary-700 font-medium'
                      : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                  }`}
                >
                  {field === 'name' ? 'Name' : field.toUpperCase()}
                  {sortBy.field === field && (
                    sortBy.order === 'asc' ? <FiArrowUp className="inline w-2 h-2 ml-0.5" /> : <FiArrowDown className="inline w-2 h-2 ml-0.5" />
                  )}
                </button>
              ))}
            </div>
          </div>

          {/* Select All / Deselect All */}
          <div className="p-2 border-b border-gray-100 flex gap-2">
            <button
              type="button"
              onClick={handleSelectAll}
              className="text-xs text-primary-600 hover:text-primary-700 font-medium"
            >
              Select all
            </button>
            <span className="text-gray-300">|</span>
            <button
              type="button"
              onClick={handleDeselectAll}
              className="text-xs text-gray-500 hover:text-gray-700"
            >
              Clear
            </button>
          </div>

          {/* Pod List */}
          <div className="flex-1 overflow-y-auto max-h-48">
            {filteredPods.map((pod) => {
              const isSelected = selectedPods.includes(pod.name);
              const color = getPodColor(pod.name); // No index - always use hash for consistent color

              return (
                <button
                  key={pod.name}
                  type="button"
                  onClick={() => handleTogglePod(pod.name)}
                  className={`w-full px-3 py-2 flex items-center gap-2 text-left hover:bg-gray-50 transition-colors ${
                    isSelected ? 'bg-primary-50' : ''
                  }`}
                >
                  {/* Checkbox */}
                  <div
                    className={`w-4 h-4 rounded border flex items-center justify-center flex-shrink-0 ${
                      isSelected
                        ? 'bg-primary-600 border-primary-600'
                        : 'border-gray-300'
                    }`}
                  >
                    {isSelected && <FiCheck className="w-3 h-3 text-white" />}
                  </div>

                  {/* Color indicator */}
                  <div
                    className="w-2.5 h-2.5 rounded-full flex-shrink-0"
                    style={{ backgroundColor: color }}
                  />

                  {/* Pod name and metrics */}
                  <div className="flex-1 min-w-0">
                    <div className="text-sm font-medium text-gray-900 truncate">
                      {pod.name}
                    </div>
                    <div className="text-xs text-gray-500 flex items-center gap-2">
                      <span>CPU: {pod.cpu_percent.toFixed(1)}%</span>
                      <span>RAM: {pod.ram_percent.toFixed(1)}%</span>
                    </div>
                  </div>

                  {/* Spike indicator */}
                  {pod.hasSpike && (
                    <div className="flex-shrink-0">
                      <span className="w-2 h-2 bg-red-500 rounded-full animate-pulse" />
                    </div>
                  )}
                </button>
              );
            })}
          </div>

          {/* Footer with count */}
          <div className="p-2 border-t border-gray-100 text-xs text-gray-500 text-center">
            {filteredPods.length} of {pods.length} pods
          </div>
        </div>
      )}
    </div>
  );
}
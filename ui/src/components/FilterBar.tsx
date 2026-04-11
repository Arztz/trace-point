import { FiFilter, FiX, FiSearch } from 'react-icons/fi';

interface FilterBarProps {
  namespace: string;
  podName: string;
  onNamespaceChange: (value: string) => void;
  onPodNameChange: (value: string) => void;
  onReset?: () => void;
}

export default function FilterBar({
  namespace,
  podName,
  onNamespaceChange,
  onPodNameChange,
  onReset,
}: FilterBarProps) {
  const hasFilters = namespace || podName;

  const handleReset = () => {
    onNamespaceChange('');
    onPodNameChange('');
    onReset?.();
  };

  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4 shadow-sm">
      <div className="flex items-center gap-2 mb-4">
        <div className="p-1.5 bg-primary-50 rounded-md">
          <FiFilter className="w-4 h-4 text-primary-600" />
        </div>
        <h3 className="text-sm font-semibold text-gray-900">Filters</h3>
        {hasFilters && (
          <span className="ml-auto text-xs bg-primary-100 text-primary-700 px-2 py-0.5 rounded-full font-medium">
            Active
          </span>
        )}
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
        {/* Namespace Input */}
        <div>
          <label 
            htmlFor="namespace" 
            className="block text-xs font-medium text-gray-600 mb-1.5"
          >
            Namespace
          </label>
          <div className="relative">
            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
              <FiSearch className="h-4 w-4 text-gray-400" />
            </div>
            <input
              type="text"
              id="namespace"
              value={namespace}
              onChange={(e) => onNamespaceChange(e.target.value)}
              placeholder="e.g., default"
              className="input pl-10"
            />
            {namespace && (
              <button
                onClick={() => onNamespaceChange('')}
                className="absolute inset-y-0 right-0 pr-3 flex items-center text-gray-400 hover:text-gray-600"
              >
                <FiX className="h-4 w-4" />
              </button>
            )}
          </div>
        </div>

        {/* Pod Name Input */}
        <div>
          <label 
            htmlFor="podName" 
            className="block text-xs font-medium text-gray-600 mb-1.5"
          >
            Pod Name
          </label>
          <div className="relative">
            <div className="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
              <FiSearch className="h-4 w-4 text-gray-400" />
            </div>
            <input
              type="text"
              id="podName"
              value={podName}
              onChange={(e) => onPodNameChange(e.target.value)}
              placeholder="e.g., my-service-abc123"
              className="input pl-10"
            />
            {podName && (
              <button
                onClick={() => onPodNameChange('')}
                className="absolute inset-y-0 right-0 pr-3 flex items-center text-gray-400 hover:text-gray-600"
              >
                <FiX className="h-4 w-4" />
              </button>
            )}
          </div>
        </div>
      </div>

      {/* Reset Button */}
      {hasFilters && (
        <div className="mt-4 pt-4 border-t border-gray-100 flex justify-end">
          <button
            onClick={handleReset}
            className="text-sm text-gray-500 hover:text-gray-700 flex items-center gap-1.5 transition-colors"
          >
            <FiX className="w-3.5 h-3.5" />
            Clear all filters
          </button>
        </div>
      )}
    </div>
  );
}
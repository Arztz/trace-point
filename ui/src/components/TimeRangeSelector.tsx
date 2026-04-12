import { FiClock } from 'react-icons/fi';
import type { TimeRange } from '../types';

interface TimeRangeSelectorProps {
  value: string;
  onChange: (value: string) => void;
  options: TimeRange[];
}

export default function TimeRangeSelector({ value, onChange, options }: TimeRangeSelectorProps) {
  return (
    <div className="bg-white rounded-lg border border-gray-200 p-3 shadow-sm">
      <div className="flex items-center gap-2 mb-2">
        <FiClock className="w-4 h-4 text-gray-500" />
        <span className="text-xs font-medium text-gray-600">Time Range</span>
      </div>
      <div className="flex rounded-md shadow-sm">
        {options.map((option) => (
          <button
            key={option.value}
            onClick={() => onChange(option.value)}
            className={`
              flex-1 px-3 py-2 text-sm font-medium border transition-all duration-200
              ${value === option.value
                ? 'bg-primary-600 text-white border-primary-600 shadow-sm'
                : 'bg-white text-gray-600 border-gray-200 hover:bg-gray-50 hover:border-gray-300'
              }
              ${option.value === options[0].value ? 'rounded-l-md' : ''}
              ${option.value === options[options.length - 1].value ? 'rounded-r-md' : ''}
              ${option.value !== options[0].value && option.value !== options[options.length - 1].value ? 'border-l-0' : ''}
            `}
          >
            {option.label}
          </button>
        ))}
      </div>
    </div>
  );
}
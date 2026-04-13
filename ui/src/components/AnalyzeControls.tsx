import { useState, useEffect } from 'react';
import { FiCalendar, FiClock, FiFilter, FiSearch } from 'react-icons/fi';

interface AnalyzeControlsProps {
  onAnalyze: (params: {
    start: string;
    end: string;
    window: string;
    namespace: string;
  }) => void;
  isLoading?: boolean;
}

type TimePreset = '24h' | '7d' | '30d' | 'custom';

interface DatePreset {
  value: TimePreset;
  label: string;
}

const DATE_PRESETS: DatePreset[] = [
  { value: '24h', label: 'Last 24h' },
  { value: '7d', label: 'Last 7d' },
  { value: '30d', label: 'Last 30d' },
  { value: 'custom', label: 'Custom' },
];

interface WindowOption {
  value: string;
  label: string;
}

const WINDOW_OPTIONS: WindowOption[] = [
  { value: '5m', label: '5 minutes' },
  { value: '15m', label: '15 minutes' },
  { value: '30m', label: '30 minutes' },
  { value: '1h', label: '1 hour' },
];

export default function AnalyzeControls({ onAnalyze, isLoading }: AnalyzeControlsProps) {
  // Time preset selection
  const [preset, setPreset] = useState<TimePreset>('24h');
  
  // Custom date/time values
  const [startDate, setStartDate] = useState('');
  const [startTime, setStartTime] = useState('');
  const [endDate, setEndDate] = useState('');
  const [endTime, setEndTime] = useState('');
  
  // Window size
  const [windowSize, setWindowSize] = useState('30m');
  
  // Namespace
  const [namespace, setNamespace] = useState('');
  
  // Compute default dates for custom mode
  useEffect(() => {
    const now = new Date();
    if (!endDate) {
      // End date = now
      setEndDate(now.toISOString().split('T')[0]);
      setEndTime(now.toTimeString().slice(0, 5)); // HH:MM
    }
    if (!startDate) {
      // Start date = 24 hours ago
      const yesterday = new Date(now.getTime() - 24 * 60 * 60 * 1000);
      setStartDate(yesterday.toISOString().split('T')[0]);
      setStartTime(yesterday.toTimeString().slice(0, 5));
    }
  }, []);
  
  // Handle preset change
  const handlePresetChange = (newPreset: TimePreset) => {
    setPreset(newPreset);
    
    if (newPreset !== 'custom') {
      const now = new Date();
      let startValue = '';
      
      switch (newPreset) {
        case '24h':
          startValue = '24h';
          break;
        case '7d':
          startValue = '7d';
          break;
        case '30d':
          startValue = '30d';
          break;
      }
      
      // Emit analyze immediately for preset modes
      onAnalyze({
        start: startValue,
        end: 'now',
        window: windowSize,
        namespace,
      });
    }
  };
  
  // Handle analyze button click
  const handleAnalyze = () => {
    let startValue: string;
    let endValue: string;
    
    if (preset === 'custom') {
      // Format custom dates as RFC3339
      const startDateTime = startDate && startTime 
        ? `${startDate}T${startTime}:00`
        : startDate 
          ? `${startDate}T00:00:00`
          : '';
      const endDateTime = endDate && endTime 
        ? `${endDate}T${endTime}:00`
        : endDate 
          ? `${endDate}T23:59:59`
          : '';
      
      startValue = startDateTime;
      endValue = endDateTime || 'now';
    } else {
      // Use relative time
      startValue = preset;
      endValue = 'now';
    }
    
    onAnalyze({
      start: startValue,
      end: endValue,
      window: windowSize,
      namespace,
    });
  };
  
  // Initial analysis on mount
  useEffect(() => {
    if (!isLoading) {
      onAnalyze({
        start: '24h',
        end: 'now',
        window: windowSize,
        namespace,
      });
    }
  }, []);
  
  return (
    <div className="bg-white rounded-lg border border-gray-200 p-4 shadow-sm">
      {/* Quick Presets */}
      <div className="flex flex-wrap items-center gap-3 mb-4">
        <span className="text-sm font-medium text-gray-600">Time Range:</span>
        <div className="flex gap-1">
          {DATE_PRESETS.map((p) => (
            <button
              key={p.value}
              onClick={() => handlePresetChange(p.value)}
              className={`
                px-3 py-1.5 text-sm font-medium rounded-md transition-colors
                ${preset === p.value
                  ? 'bg-primary-600 text-white shadow-sm'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200'
                }
              `}
            >
              {p.label}
            </button>
          ))}
        </div>
      </div>
      
      {/* Custom Date/Time Inputs (shown when Custom is selected) */}
      {preset === 'custom' && (
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-4 mb-4 p-4 bg-gray-50 rounded-lg">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Start Date
            </label>
            <div className="flex items-center gap-2">
              <FiCalendar className="w-4 h-4 text-gray-400" />
              <input
                type="date"
                value={startDate}
                onChange={(e) => setStartDate(e.target.value)}
                className="flex-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Start Time
            </label>
            <div className="flex items-center gap-2">
              <FiClock className="w-4 h-4 text-gray-400" />
              <input
                type="time"
                value={startTime}
                onChange={(e) => setStartTime(e.target.value)}
                className="flex-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              End Date
            </label>
            <div className="flex items-center gap-2">
              <FiCalendar className="w-4 h-4 text-gray-400" />
              <input
                type="date"
                value={endDate}
                onChange={(e) => setEndDate(e.target.value)}
                className="flex-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              />
            </div>
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              End Time
            </label>
            <div className="flex items-center gap-2">
              <FiClock className="w-4 h-4 text-gray-400" />
              <input
                type="time"
                value={endTime}
                onChange={(e) => setEndTime(e.target.value)}
                className="flex-1 px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500"
              />
            </div>
          </div>
        </div>
      )}
      
      {/* Filters Row */}
      <div className="flex flex-wrap items-end gap-4">
        {/* Window Size Selector */}
        <div className="min-w-[160px]">
          <label className="block text-sm font-medium text-gray-700 mb-1">
            <FiClock className="w-4 h-4 inline-block mr-1" />
            Window Size
          </label>
          <select
            value={windowSize}
            onChange={(e) => setWindowSize(e.target.value)}
            className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 bg-white"
          >
            {WINDOW_OPTIONS.map((opt) => (
              <option key={opt.value} value={opt.value}>
                {opt.label}
              </option>
            ))}
          </select>
        </div>
        
        {/* Namespace Selector */}
        <div className="min-w-[180px]">
          <label className="block text-sm font-medium text-gray-700 mb-1">
            <FiFilter className="w-4 h-4 inline-block mr-1" />
            Namespace
          </label>
          <div className="relative">
            <select
              value={namespace}
              onChange={(e) => setNamespace(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-primary-500 bg-white appearance-none"
            >
              <option value="">All Namespaces</option>
              <option value="fundii">fundii</option>
              <option value="default">default</option>
            </select>
            <FiFilter className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
          </div>
        </div>
        
        {/* Analyze Button */}
        <button
          onClick={handleAnalyze}
          disabled={isLoading}
          className={`
            px-6 py-2 text-sm font-medium rounded-md transition-colors flex items-center gap-2
            ${isLoading
              ? 'bg-gray-400 cursor-not-allowed text-white'
              : 'bg-primary-600 hover:bg-primary-700 text-white shadow-sm'
            }
          `}
        >
          {isLoading ? (
            <>
              <div className="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin" />
              Analyzing...
            </>
          ) : (
            <>
              <FiSearch className="w-4 h-4" />
              Analyze
            </>
          )}
        </button>
      </div>
    </div>
  );
}
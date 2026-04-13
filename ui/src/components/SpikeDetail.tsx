import { useQuery } from '@tanstack/react-query';
import { FiX, FiClock, FiServer, FiCpu, FiActivity, FiAlertTriangle, FiTarget, FiTrendingUp, FiCode, FiZap } from 'react-icons/fi';
import { api } from '../services/api';
import { ErrorState, LoadingState } from './States';
import type { SpikeDetailsResponse, FunctionProfile } from '../types';

interface SpikeDetailProps {
  spikeId: string;
  onClose: () => void;
}

function formatDateTime(timestamp: string): string {
  const date = new Date(timestamp);
  return date.toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  });
}

function getConfidenceColor(confidence: number): string {
  if (confidence >= 0.8) return 'text-green-600 bg-green-50 border-green-200';
  if (confidence >= 0.5) return 'text-amber-600 bg-amber-50 border-amber-200';
  return 'text-gray-600 bg-gray-50 border-gray-200';
}

function getConfidenceLabel(confidence: number): string {
  if (confidence >= 0.8) return 'High';
  if (confidence >= 0.5) return 'Medium';
  return 'Low';
}

function ProfilerSection({ profilerData, culpritFunction }: { profilerData?: { top_functions: FunctionProfile[] }; culpritFunction?: FunctionProfile }) {
  if (!profilerData && !culpritFunction) {
    return (
      <div className="border-t border-gray-200 pt-6">
        <h3 className="text-sm font-semibold text-gray-900 mb-4">Profiler Data</h3>
        <p className="text-sm text-gray-500">No profiler data available for this spike.</p>
      </div>
    );
  }

  const functions = profilerData?.top_functions || [];

  return (
    <div className="border-t border-gray-200 pt-6">
      <div className="flex items-center gap-2 mb-4">
        <FiZap className="w-4 h-4 text-amber-500" />
        <h3 className="text-sm font-semibold text-gray-900">CPU Profile (Flamegraph)</h3>
      </div>

      {/* Culprit Function Highlight */}
      {culpritFunction && (
        <div className="bg-gradient-to-r from-amber-50 to-orange-50 border border-amber-200 rounded-lg p-4 mb-4">
          <div className="flex items-center gap-2 mb-2">
            <FiAlertTriangle className="w-4 h-4 text-amber-600" />
            <span className="text-sm font-semibold text-amber-800">Culprit Function</span>
          </div>
          <div className="flex items-center gap-3">
            <span className="text-lg font-mono text-gray-900">{culpritFunction.function_name}</span>
            <span className="bg-amber-100 text-amber-800 px-2 py-1 rounded text-xs font-medium">
              {culpritFunction.cpu_percent.toFixed(1)}% CPU
            </span>
          </div>
          {culpritFunction.file_path && (
            <div className="flex items-center gap-2 mt-2 text-xs text-gray-600">
              <FiCode className="w-3 h-3" />
              <span className="font-mono">{culpritFunction.file_path}:{culpritFunction.line_number}</span>
            </div>
          )}
        </div>
      )}

      {/* Top Functions Table */}
      {functions.length > 0 && (
        <div className="space-y-2">
          <p className="text-xs text-gray-500 mb-2">Top CPU-consuming functions:</p>
          {functions.slice(0, 5).map((func, idx) => (
            <div 
              key={idx}
              className={`flex items-center justify-between p-2 rounded text-sm ${
                culpritFunction?.function_name === func.function_name 
                  ? 'bg-amber-50 border border-amber-200' 
                  : 'bg-gray-50'
              }`}
            >
              <div className="flex items-center gap-2">
                <span className="text-xs text-gray-400 w-4">{idx + 1}</span>
                <span className="font-mono text-gray-700">{func.function_name}</span>
              </div>
              <div className="flex items-center gap-2">
                {func.file_path && (
                  <span className="text-xs text-gray-500 font-mono">{func.file_path}:{func.line_number}</span>
                )}
                <span className="text-xs font-medium text-gray-600">{func.cpu_percent.toFixed(1)}%</span>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

export default function SpikeDetail({ spikeId, onClose }: SpikeDetailProps) {
  // Use the new details endpoint with profiler data
  const { data: details, isLoading, error } = useQuery<SpikeDetailsResponse>({
    queryKey: ['spikeDetails', spikeId],
    queryFn: () => api.spikes.getDetails(spikeId),
    enabled: !!spikeId,
  });

  if (isLoading) {
    return (
      <div className="bg-white rounded-lg shadow-lg border border-gray-200 p-6">
        <LoadingState text="Loading spike details..." />
      </div>
    );
  }

  if (error || !details) {
    return (
      <div className="bg-white rounded-lg shadow-lg border border-gray-200 p-6">
        <ErrorState 
          title="Failed to load spike details" 
          message="The spike details could not be retrieved."
          onRetry={() => window.location.reload()}
        />
      </div>
    );
  }

  const spike = details.spike;
  const profilerData = details.profiler_data;
  const culpritFunction = details.culprit_function;
  const activeRoutes = details.active_routes || [];

  // Calculate % above average - guard against very small moving averages
  const spikeRatio = spike.movingAverage > 1 
    ? Math.min((spike.currentValue / spike.movingAverage) * 100 - 100, 9999).toFixed(1)
    : '0.0';

  return (
    <div className="bg-white rounded-lg shadow-lg border border-gray-200 overflow-hidden animate-slide-up">
      {/* Header */}
      <div className="bg-gradient-to-r from-primary-600 to-primary-700 px-6 py-4 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-white/20 rounded-lg">
            <FiActivity className="w-5 h-5 text-white" />
          </div>
          <div>
            <h2 className="text-lg font-semibold text-white">Spike Details</h2>
            <p className="text-sm text-primary-100">ID: {spike.id.slice(0, 12)}...</p>
          </div>
        </div>
        <button
          onClick={onClose}
          className="p-2 hover:bg-white/20 rounded-lg transition-colors text-white"
        >
          <FiX className="w-5 h-5" />
        </button>
      </div>

      {/* Content */}
      <div className="p-6 space-y-6">
        {/* Basic Info */}
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="flex items-center gap-2 text-gray-500 mb-2">
              <FiServer className="w-4 h-4" />
              <span className="text-xs font-medium">Pod</span>
            </div>
            <p className="text-sm font-mono text-gray-900 truncate">{spike.podName}</p>
          </div>
          
          <div className="bg-gray-50 rounded-lg p-4">
            <div className="flex items-center gap-2 text-gray-500 mb-2">
              <FiClock className="w-4 h-4" />
              <span className="text-xs font-medium">Timestamp</span>
            </div>
            <p className="text-sm text-gray-900">{formatDateTime(spike.timestamp)}</p>
          </div>

          <div className="bg-gray-50 rounded-lg p-4">
            <div className="flex items-center gap-2 text-gray-500 mb-2">
              <FiCpu className="w-4 h-4" />
              <span className="text-xs font-medium">Resource</span>
            </div>
            <p className="text-sm font-medium text-gray-900 capitalize">{spike.resourceType}</p>
          </div>

          <div className="bg-gray-50 rounded-lg p-4">
            <div className="flex items-center gap-2 text-gray-500 mb-2">
              <FiTarget className="w-4 h-4" />
              <span className="text-xs font-medium">Threshold</span>
            </div>
            <p className="text-sm font-medium text-gray-900">{spike.threshold}%</p>
          </div>
        </div>

        {/* Metrics */}
        <div className="border-t border-gray-200 pt-6">
          <h3 className="text-sm font-semibold text-gray-900 mb-4">Metrics</h3>
          <div className="grid grid-cols-3 gap-4">
            <div className="bg-red-50 border border-red-100 rounded-lg p-4 text-center">
              <p className="text-xs text-red-600 font-medium mb-1">Current Value</p>
              <p className="text-2xl font-bold text-red-700">{spike.currentValue.toFixed(1)}%</p>
            </div>
            <div className="bg-gray-50 border border-gray-100 rounded-lg p-4 text-center">
              <p className="text-xs text-gray-500 font-medium mb-1">Moving Average</p>
              <p className="text-2xl font-bold text-gray-700">{spike.movingAverage.toFixed(1)}%</p>
            </div>
            <div className="bg-amber-50 border border-amber-100 rounded-lg p-4 text-center">
              <p className="text-xs text-amber-600 font-medium mb-1">Spike Above Avg</p>
              <p className="text-2xl font-bold text-amber-700">+{spikeRatio}%</p>
            </div>
          </div>
        </div>

        {/* Profiler Section */}
        <ProfilerSection 
          profilerData={profilerData} 
          culpritFunction={culpritFunction} 
        />

        {/* Active Routes */}
        {activeRoutes.length > 0 && (
          <div className="border-t border-gray-200 pt-6">
            <h3 className="text-sm font-semibold text-gray-900 mb-4">Active Routes at Spike Time</h3>
            <div className="flex flex-wrap gap-2">
              {activeRoutes.map((route, idx) => (
                <span 
                  key={idx}
                  className="bg-gray-100 text-gray-700 px-3 py-1.5 rounded-full text-sm font-mono"
                >
                  {route}
                </span>
              ))}
            </div>
          </div>
        )}

        {/* Legacy: Possible Root Causes */}
        {spike.possibleRootCauses && spike.possibleRootCauses.length > 0 && (
          <div className="border-t border-gray-200 pt-6">
            <div className="flex items-center gap-2 mb-4">
              <FiAlertTriangle className="w-4 h-4 text-amber-500" />
              <h3 className="text-sm font-semibold text-gray-900">Possible Root Causes</h3>
            </div>
            <div className="space-y-3">
              {spike.possibleRootCauses.map((cause, idx) => (
                <div 
                  key={idx}
                  className="bg-gray-50 rounded-lg p-4 border border-gray-100"
                >
                  <div className="flex items-center justify-between mb-3">
                    <div className="flex items-center gap-3">
                      <span className="bg-primary-100 text-primary-700 px-2 py-1 rounded text-xs font-mono">
                        {cause.route}
                      </span>
                      <span className="text-sm text-gray-600">
                        {cause.functionName}
                      </span>
                    </div>
                    <div className={`flex items-center gap-2 px-3 py-1 rounded-full border ${getConfidenceColor(cause.confidence)}`}>
                      <FiTrendingUp className="w-3 h-3" />
                      <span className="text-sm font-medium">{getConfidenceLabel(cause.confidence)}</span>
                      <span className="text-xs">({(cause.confidence * 100).toFixed(0)}%)</span>
                    </div>
                  </div>
                  {cause.filePath && (
                    <div className="flex items-center gap-2 text-xs text-gray-500">
                      <FiCode className="w-3 h-3" />
                      <span className="font-mono">{cause.filePath}</span>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
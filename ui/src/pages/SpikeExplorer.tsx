import { useState, useEffect, useCallback } from 'react';
import { useQuery } from '@tanstack/react-query';
import { FiActivity, FiServer, FiAlertTriangle, FiTrendingUp } from 'react-icons/fi';
import { api } from '../services/api';
import type { SpikeAnalysisResponse, AnalysisSummary } from '../types';
import AnalyzeControls from '../components/AnalyzeControls';
import SpikeAnalysisTable from '../components/SpikeAnalysisTable';
import { EmptyState, ErrorState } from '../components/States';

interface SummaryCardsProps {
  summary: AnalysisSummary | undefined;
}

function SummaryCards({ summary }: SummaryCardsProps) {
  if (!summary) {
    return (
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        {[...Array(3)].map((_, i) => (
          <div key={i} className="bg-white rounded-lg border border-gray-200 p-4 shadow-sm animate-pulse">
            <div className="h-4 bg-gray-200 rounded w-24 mb-2"></div>
            <div className="h-8 bg-gray-200 rounded w-16"></div>
          </div>
        ))}
      </div>
    );
  }
  
  // Get top offender
  const topOffender = summary.top_replicasets[0];
  
  return (
    <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
      {/* Total Spikes */}
      <div className="bg-white rounded-lg border border-gray-200 p-4 shadow-sm">
        <div className="flex items-center gap-2 mb-2">
          <div className="p-2 bg-red-50 rounded-lg">
            <FiAlertTriangle className="w-4 h-4 text-red-600" />
          </div>
          <span className="text-sm font-medium text-gray-600">Total Spikes</span>
        </div>
        <p className="text-3xl font-bold text-gray-900">{summary.total_spikes}</p>
      </div>
      
      {/* Analyzed Replicasets */}
      <div className="bg-white rounded-lg border border-gray-200 p-4 shadow-sm">
        <div className="flex items-center gap-2 mb-2">
          <div className="p-2 bg-primary-50 rounded-lg">
            <FiServer className="w-4 h-4 text-primary-600" />
          </div>
          <span className="text-sm font-medium text-gray-600">Analyzed RS</span>
        </div>
        <p className="text-3xl font-bold text-gray-900">{summary.analyzed_replicasets}</p>
      </div>
      
      {/* Top Offender */}
      <div className="bg-white rounded-lg border border-gray-200 p-4 shadow-sm">
        <div className="flex items-center gap-2 mb-2">
          <div className="p-2 bg-amber-50 rounded-lg">
            <FiTrendingUp className="w-4 h-4 text-amber-600" />
          </div>
          <span className="text-sm font-medium text-gray-600">Top Offender</span>
        </div>
        {topOffender ? (
          <p className="text-xl font-bold text-gray-900 truncate">
            {topOffender.name}
            <span className="text-sm font-normal text-gray-500 ml-2">
              ({topOffender.spike_count})
            </span>
          </p>
        ) : (
          <p className="text-lg font-medium text-gray-400">N/A</p>
        )}
      </div>
    </div>
  );
}

interface SpikeExplorerProps {
  // Add any props if needed
}

export default function SpikeExplorer({}: SpikeExplorerProps) {
  // Analysis params state
  const [analysisParams, setAnalysisParams] = useState<{
    start: string;
    end: string;
    window: string;
    namespace: string;
  }>({
    start: '24h',
    end: 'now',
    window: '30m',
    namespace: '',
  });
  
  // Fetch analysis data
  const { data, isLoading, isError, error, refetch } = useQuery<SpikeAnalysisResponse>({
    queryKey: ['spikeAnalysis', analysisParams.start, analysisParams.end, analysisParams.window, analysisParams.namespace],
    queryFn: async () => {
      const response = await api.analyze.spikes({
        start: analysisParams.start,
        end: analysisParams.end || undefined,
        window: analysisParams.window || undefined,
        namespace: analysisParams.namespace || undefined,
      });
      return response;
    },
    enabled: true, // Auto-fetch when params change
  });
  
  // Handle analyze button
  const handleAnalyze = useCallback((params: {
    start: string;
    end: string;
    window: string;
    namespace: string;
  }) => {
    setAnalysisParams(params);
  }, []);
  
  // Fetch when params change (for presets that auto-analyze)
  useEffect(() => {
    if (analysisParams.start) {
      refetch();
    }
  }, [analysisParams]);
  
  // Extract data
  const summary = data?.summary;
  const spikes = data?.spikes || [];
  
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div className="flex items-center gap-3">
          <div className="p-2 bg-primary-50 rounded-lg">
            <FiActivity className="w-5 h-5 text-primary-600" />
          </div>
          <div>
            <h1 className="text-2xl font-bold text-gray-900">Spike Explorer</h1>
            <p className="text-sm text-gray-500">
              Analyze historical spike events over custom time ranges
            </p>
          </div>
        </div>
      </div>
      
      {/* Controls */}
      <AnalyzeControls
        onAnalyze={handleAnalyze}
        isLoading={isLoading}
      />
      
      {/* Summary Cards */}
      <SummaryCards summary={summary} />
      
      {/* Results Table */}
      {isError ? (
        <ErrorState
          message={error instanceof Error ? error.message : 'Failed to analyze spikes'}
          onRetry={() => refetch()}
        />
      ) : (
        <SpikeAnalysisTable
          spikes={spikes}
          isLoading={isLoading}
        />
      )}
    </div>
  );
}
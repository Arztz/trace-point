import { useState } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { FiRefreshCw, FiDownload, FiActivity, FiAlertTriangle, FiTrendingUp, FiServer, FiCalendar } from 'react-icons/fi';
import { TIME_RANGES, TimelineData, SpikeListResponse } from '../types';
import { api } from '../services/api';
import TimelineChart from '../components/TimelineChart';
import SpikeList from '../components/SpikeList';
import FilterBar from '../components/FilterBar';
import TimeRangeSelector from '../components/TimeRangeSelector';
import SpikeDetail from '../components/SpikeDetail';
import GravityScoreTable from '../components/GravityScoreTable';
import { ChartSkeleton, SpikeListSkeleton, FilterBarSkeleton } from '../components/Skeletons';
import { EmptyState, ErrorState } from '../components/States';

export default function Dashboard() {
  const [timeRange, setTimeRange] = useState('24h');
  const [namespace, setNamespace] = useState('');
  const [podName, setPodName] = useState('');
  const [selectedSpikeId, setSelectedSpikeId] = useState<string | null>(null);
  const [showSpikeDetail, setShowSpikeDetail] = useState(false);
  const [activeTab, setActiveTab] = useState<'timeline' | 'gravity'>('timeline');

  const queryClient = useQueryClient();

  const handleRefresh = () => {
    queryClient.invalidateQueries({ queryKey: ['timeline'] });
    queryClient.invalidateQueries({ queryKey: ['spikes'] });
    queryClient.invalidateQueries({ queryKey: ['gravityScores'] });
  };

  const handleExport = async () => {
    try {
      const data = await api.export.json({ namespace, startTime: undefined, endTime: undefined });
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = `trace-point-export-${new Date().toISOString().split('T')[0]}.json`;
      document.body.appendChild(a);
      a.click();
      document.body.removeChild(a);
      URL.revokeObjectURL(url);
    } catch (error) {
      console.error('Export failed:', error);
    }
  };

  const { data: timelineData, isLoading: timelineLoading, isError: timelineError } = useQuery<TimelineData>({
    queryKey: ['timeline', timeRange, namespace, podName],
    queryFn: () => api.timeline.get({ timeRange, namespace, podName }),
    refetchInterval: 30000,
  });

  const { data: spikesData, isLoading: spikesLoading, isError: spikesError } = useQuery<SpikeListResponse>({
    queryKey: ['spikes', namespace, podName],
    queryFn: () => api.spikes.list({ namespace, podName }),
    refetchInterval: 30000,
  });

  const handleSpikeSelect = (id: string) => {
    setSelectedSpikeId(id);
    setShowSpikeDetail(true);
  };

  const handleCloseDetail = () => {
    setShowSpikeDetail(false);
    setSelectedSpikeId(null);
  };

  return (
    <div className="space-y-6">
      {/* Header Actions */}
      <div className="flex flex-col sm:flex-row justify-between items-start sm:items-center gap-4">
        <div className="flex items-center gap-4">
          {/* Tab Switcher */}
          <div className="bg-white rounded-lg border border-gray-200 p-1 shadow-sm">
            <div className="flex">
              <button
                onClick={() => setActiveTab('timeline')}
                className={`px-4 py-2 text-sm font-medium rounded-md transition-colors ${
                  activeTab === 'timeline'
                    ? 'bg-primary-600 text-white shadow-sm'
                    : 'text-gray-600 hover:text-gray-900 hover:bg-gray-50'
                }`}
              >
                <FiActivity className="w-4 h-4 inline-block mr-2" />
                Timeline
              </button>
              <button
                onClick={() => setActiveTab('gravity')}
                className={`px-4 py-2 text-sm font-medium rounded-md transition-colors ${
                  activeTab === 'gravity'
                    ? 'bg-primary-600 text-white shadow-sm'
                    : 'text-gray-600 hover:text-gray-900 hover:bg-gray-50'
                }`}
              >
                <FiTrendingUp className="w-4 h-4 inline-block mr-2" />
                Gravity Scores
              </button>
            </div>
          </div>
        </div>

        {/* Action Buttons */}
        <div className="flex items-center gap-2">
          <button
            onClick={handleRefresh}
            className="btn btn-secondary btn-sm"
            title="Refresh data"
          >
            <FiRefreshCw className="w-4 h-4 mr-1.5" />
            Refresh
          </button>
          <button
            onClick={handleExport}
            className="btn btn-outline btn-sm"
            title="Export data"
          >
            <FiDownload className="w-4 h-4 mr-1.5" />
            Export
          </button>
        </div>
      </div>

      {/* Filters */}
      <div className="grid grid-cols-1 lg:grid-cols-4 gap-4">
        <div className="lg:col-span-3">
          {timelineLoading || spikesLoading ? (
            <FilterBarSkeleton />
          ) : (
            <FilterBar
              namespace={namespace}
              podName={podName}
              onNamespaceChange={setNamespace}
              onPodNameChange={setPodName}
            />
          )}
        </div>
        <div>
          <TimeRangeSelector
            value={timeRange}
            onChange={setTimeRange}
            options={TIME_RANGES}
          />
        </div>
      </div>

      {/* Main Content */}
      {activeTab === 'timeline' ? (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          {/* Timeline Chart */}
          <div className="lg:col-span-2">
            <div className="card">
              <div className="card-header flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-primary-50 rounded-lg">
                    <FiActivity className="w-5 h-5 text-primary-600" />
                  </div>
                  <div>
                    <h2 className="text-lg font-semibold text-gray-900">Resource Timeline</h2>
                    <p className="text-sm text-gray-500">
                      {timeRange === '1h' ? 'Last hour' : 
                       timeRange === '6h' ? 'Last 6 hours' : 
                       timeRange === '24h' ? 'Last 24 hours' : 'Last 7 days'} overview
                    </p>
                  </div>
                </div>
                <div className="flex items-center gap-2 text-sm text-gray-500">
                  <FiCalendar className="w-4 h-4" />
                  <span>{new Date().toLocaleTimeString()}</span>
                </div>
              </div>
              <div className="card-body">
                {timelineLoading ? (
                  <ChartSkeleton />
                ) : timelineError ? (
                  <ErrorState 
                    message="Failed to load timeline data. Please try again."
                    onRetry={handleRefresh}
                  />
                ) : timelineData && timelineData.dataPoints.length > 0 ? (
                  <TimelineChart data={timelineData} />
                ) : (
                  <EmptyState
                    icon="chart"
                    title="No timeline data"
                    description="No resource data available for the selected time range. Try adjusting the filters or time range."
                  />
                )}
              </div>
            </div>
          </div>

          {/* Spike List */}
          <div className="lg:col-span-1">
            <div className="card h-full">
              <div className="card-header flex items-center justify-between">
                <div className="flex items-center gap-3">
                  <div className="p-2 bg-amber-50 rounded-lg">
                    <FiAlertTriangle className="w-5 h-5 text-amber-600" />
                  </div>
                  <div>
                    <h2 className="text-lg font-semibold text-gray-900">Recent Spikes</h2>
                    <p className="text-sm text-gray-500">
                      {spikesData?.total ?? 0} detected
                    </p>
                  </div>
                </div>
              </div>
              <div className="card-body custom-scrollbar" style={{ maxHeight: '600px', overflowY: 'auto' }}>
                {spikesLoading ? (
                  <SpikeListSkeleton />
                ) : spikesError ? (
                  <ErrorState 
                    message="Failed to load spike data."
                    onRetry={handleRefresh}
                  />
                ) : spikesData && spikesData.spikes.length > 0 ? (
                  <SpikeList
                    spikes={spikesData.spikes}
                    selectedId={selectedSpikeId}
                    onSelect={handleSpikeSelect}
                  />
                ) : (
                  <EmptyState
                    icon="inbox"
                    title="No spikes detected"
                    description="Resource spikes will appear here when they exceed their threshold."
                  />
                )}
              </div>
            </div>
          </div>
        </div>
      ) : (
        /* Gravity Scores Tab */
        <div>
          <GravityScoreTable 
            namespaceFilter={namespace} 
            minScoreFilter={undefined}
          />
        </div>
      )}

      {/* Spike Detail Modal */}
      {showSpikeDetail && selectedSpikeId && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm">
          <div className="w-full max-w-2xl max-h-[90vh] overflow-y-auto">
            <SpikeDetail
              spikeId={selectedSpikeId}
              onClose={handleCloseDetail}
            />
          </div>
        </div>
      )}
    </div>
  );
}
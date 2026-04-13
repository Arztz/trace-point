import { useState, useEffect, useCallback } from 'react';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { FiRefreshCw, FiDownload, FiActivity, FiAlertTriangle, FiTrendingUp, FiServer, FiCalendar } from 'react-icons/fi';
import { TIME_RANGES, TimelineData, TimelineMetric, SpikeListResponse, PodInfo, AvailablePod } from '../types';
import { api } from '../services/api';
import TimelineChart from '../components/TimelineChart';
import SpikeList from '../components/SpikeList';
import FilterBar from '../components/FilterBar';
import TimeRangeSelector from '../components/TimeRangeSelector';
import SpikeDetail from '../components/SpikeDetail';
import GravityScoreTable from '../components/GravityScoreTable';
import PodLegend from '../components/PodLegend';
import { ChartSkeleton, SpikeListSkeleton, FilterBarSkeleton } from '../components/Skeletons';
import { EmptyState, ErrorState } from '../components/States';

// Transform raw API response to TimelineData format
// Handles both new format (prometheus metrics + spike_markers) and legacy format (data_points/events)
function transformTimelineData(raw: unknown): TimelineData & { metrics?: TimelineMetric[]; availablePods?: PodInfo[] } {
  // If it's already in the correct format, return it
  if (raw && typeof raw === 'object' && 'dataPoints' in raw && Array.isArray((raw as Record<string, unknown>).dataPoints)) {
    return raw as TimelineData & { metrics?: TimelineMetric[]; availablePods?: PodInfo[] };
  }
  
  const data = raw as Record<string, unknown>;
  
  // Handle new format: metrics array from Prometheus (preferred)
  if (data.metrics && Array.isArray(data.metrics)) {
    const metrics = data.metrics as TimelineMetric[];
    
    // Use availablePods from API (replicaset-level aggregated data) if present
    let availablePods: PodInfo[] = [];
    if (data.availablePods && Array.isArray(data.availablePods)) {
      availablePods = (data.availablePods as AvailablePod[]).map(pod => ({
        name: pod.name,           // Replicaset name
        namespace: pod.namespace,
        cpu_percent: pod.current_cpu || pod.currentCpu || 0,
        ram_percent: pod.current_ram || pod.currentRam || 0,
      }));
    } else {
      // Fallback: Extract unique pods from metrics (legacy behavior)
      // Group by replicaset_name since that's the new grouping key
      const podMap = new Map<string, PodInfo>();
      for (const metric of metrics) {
        const key = metric.replicaset_name || metric.pod_name;
        const existing = podMap.get(key);
        if (!existing || metric.timestamp > (existing as unknown as TimelineMetric & { timestamp: string }).timestamp) {
          podMap.set(key, {
            name: key,
            namespace: metric.namespace,
            cpu_percent: metric.cpu_percent,
            ram_percent: metric.ram_percent,
          });
        }
      }
      availablePods = Array.from(podMap.values());
    }
    
    // Group by pod_name to support filtering, or aggregate all for unified view
    // For now, aggregate all metrics into unified data points by timestamp
    const aggregatedByTimestamp = new Map<string, { cpu: number; ram: number; count: number }>();
    
    for (const metric of metrics) {
      const timestamp = metric.timestamp;
      const existing = aggregatedByTimestamp.get(timestamp);
      
      if (existing) {
        existing.cpu += metric.cpu_percent;
        existing.ram += metric.ram_percent;
        existing.count += 1;
      } else {
        aggregatedByTimestamp.set(timestamp, {
          cpu: metric.cpu_percent,
          ram: metric.ram_percent,
          count: 1,
        });
      }
    }
    
    // Convert to data points array
    const dataPoints = Array.from(aggregatedByTimestamp.entries())
      .map(([timestamp, values]) => ({
        timestamp,
        cpu: values.count > 0 ? values.cpu / values.count : 0,
        ram: values.count > 0 ? values.ram / values.count : 0,
      }))
      .sort((a, b) => a.timestamp.localeCompare(b.timestamp));
    
    return {
      dataPoints,
      routes: [],
      startTime: String(data.start_date || data.startDate || ''),
      endTime: String(data.end_date || data.endDate || ''),
      metrics,
      availablePods,
    };
  }
  
  // Legacy format - transform spike events to timeline data points
  const rawDataPoints = Array.isArray(data.events) ? data.events : 
                       Array.isArray(data.data_points) ? data.data_points : 
                       Array.isArray(data.dataPoint) ? data.dataPoint : [];
  
  const dataPoints = rawDataPoints.map((event: Record<string, unknown>) => ({
    timestamp: String(event.timestamp || event.created_at || ''),
    cpu: Number(event.cpu_usage_percent || event.cpuUsagePercent || 0),
    ram: Number(event.ram_usage_percent || event.ramUsagePercent || 0),
  }));
  
  return {
    dataPoints,
    routes: [],
    startTime: String(data.start_date || data.startDate || ''),
    endTime: String(data.end_date || data.endDate || ''),
  };
}

// Transform raw API response to SpikeListResponse format
function transformSpikesData(raw: unknown): SpikeListResponse {
  // If it's already in the correct format, return it
  if (raw && typeof raw === 'object' && 'spikes' in raw && Array.isArray((raw as Record<string, unknown>).spikes)) {
    return raw as SpikeListResponse;
  }
  
  const data = raw as Record<string, unknown>;
  
  // Raw format from backend - transform to frontend format
  const rawSpikes = Array.isArray(data) ? data : 
                   Array.isArray(data.events) ? data.events : 
                   Array.isArray(data.spikes) ? data.spikes : [];
  
  const spikes = rawSpikes.map((event: Record<string, unknown>, idx: number) => {
    const cpuUsage = Number(event.cpu_usage_percent || event.cpuUsagePercent || 0);
    const ramUsage = Number(event.ram_usage_percent || event.ramUsagePercent || 0);
    const threshold = Number(event.threshold_percent || event.thresholdPercent || 50);
    const avg = Number(event.moving_average_percent || event.movingAveragePercent || 0);
    
    return {
      id: String(event.id || `spike-${idx}`),
      timestamp: String(event.timestamp || event.created_at || ''),
      podName: String(event.pod_name || event.podName || ''),
      namespace: String(event.namespace || ''),
      resourceType: cpuUsage >= ramUsage ? 'cpu' as const : 'memory' as const,
      threshold,
      currentValue: cpuUsage >= ramUsage ? cpuUsage : ramUsage,
      movingAverage: avg,
      activeRoutes: event.route_name ? [String(event.route_name)] : [],
      possibleRootCauses: [],
    };
  });
  
  return {
    spikes,
    total: spikes.length,
    page: 1,
    pageSize: 100,
  };
}

export default function Dashboard() {
  const [timeRange, setTimeRange] = useState('24h');
  const [namespace, setNamespace] = useState('');
  const [podName, setPodName] = useState('');
  const [selectedPods, setSelectedPods] = useState<string[]>([]);
  const [highlightedPod, setHighlightedPod] = useState<string | null>(null);
  const [selectedSpikeId, setSelectedSpikeId] = useState<string | null>(null);
  const [showSpikeDetail, setShowSpikeDetail] = useState(false);
  const [activeTab, setActiveTab] = useState<'timeline' | 'gravity'>('timeline');

  // Store deferred values for API calls (updated after debounce)
  const [debouncedNamespace, setDebouncedNamespace] = useState('');
  const [debouncedPodName, setDebouncedPodName] = useState('');
  const [debouncedSelectedPods, setDebouncedSelectedPods] = useState<string[]>([]);

  const queryClient = useQueryClient();

  // Debounce filter changes - wait for user to stop typing before API call
  useEffect(() => {
    const timer = setTimeout(() => {
      setDebouncedNamespace(namespace);
      setDebouncedPodName(podName);
    }, 500); // Wait 500ms after user stops typing

    return () => clearTimeout(timer);
  }, [namespace, podName]);

  // Debounce selected pods changes (no delay needed for dropdown selection)
  useEffect(() => {
    setDebouncedSelectedPods(selectedPods);
  }, [selectedPods]);

  // Export error state
  const [exportError, setExportError] = useState<string | null>(null);

  const handleRefresh = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ['timeline'] });
    queryClient.invalidateQueries({ queryKey: ['spikes'] });
    queryClient.invalidateQueries({ queryKey: ['gravityScores'] });
  }, [queryClient]);

  const handleExport = useCallback(async () => {
    setExportError(null);
    try {
      const data = await api.export.json({ 
        namespace: debouncedNamespace, 
        startTime: undefined, 
        endTime: undefined 
      });
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
      const message = error instanceof Error ? error.message : 'Export failed';
      setExportError(message);
    }
  }, [debouncedNamespace]);

  const { data: timelineData, isLoading: timelineLoading, isError: timelineError } = useQuery({
    queryKey: ['timeline', timeRange, debouncedNamespace, debouncedPodName, debouncedSelectedPods.join(',')],
    queryFn: async () => {
      const podNameParam = debouncedSelectedPods.length > 0 
        ? debouncedSelectedPods.join(',') 
        : debouncedPodName;
      const raw = await api.timeline.get({ 
        timeRange: timeRange,  // Pass the actual timeRange value to trigger new API calls
        namespace: debouncedNamespace, 
        podName: podNameParam 
      });
      return transformTimelineData(raw);
    },
    staleTime: 10000, // Keep data fresh for 10s - no loading spinner on background refetches
    refetchInterval: 30000,
  });

  // Extract available pods from timeline data
  const availablePods: PodInfo[] = timelineData?.availablePods || [];
  const metrics: TimelineMetric[] = timelineData?.metrics || [];

  const { data: spikesData, isLoading: spikesLoading, isError: spikesError } = useQuery<SpikeListResponse>({
    queryKey: ['spikes', debouncedNamespace, debouncedPodName],
    queryFn: async () => {
      const raw = await api.spikes.list({ namespace: debouncedNamespace, podName: debouncedPodName });
      return transformSpikesData(raw);
    },
    staleTime: 10000, // Keep data fresh for 10s - no loading spinner on background refetches
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
          {exportError && (
            <div className="text-sm text-red-600">
              Export failed: {exportError}
            </div>
          )}
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
              selectedPods={selectedPods}
              availablePods={availablePods}
              onNamespaceChange={setNamespace}
              onPodNameChange={setPodName}
              onSelectedPodsChange={setSelectedPods}
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

              {/* Pod Legend - above chart */}
              {availablePods.length > 0 && (
                <div className="px-4 pt-3 border-b border-gray-100">
                  <PodLegend
                    pods={availablePods}
                    selectedPods={selectedPods}
                    highlightedPod={highlightedPod}
                    onHighlight={setHighlightedPod}
                  />
                </div>
              )}

              <div className="card-body">
                {timelineLoading ? (
                  <ChartSkeleton />
                ) : timelineError ? (
                  <ErrorState 
                    message="Failed to load timeline data. Please try again."
                    onRetry={handleRefresh}
                  />
                ) : timelineData && timelineData.dataPoints && timelineData.dataPoints.length > 0 ? (
                  <TimelineChart 
                    metrics={metrics}
                    availablePods={availablePods}
                    selectedPods={selectedPods}
                    highlightedPod={highlightedPod}
                    onHighlight={setHighlightedPod}
                  />
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
                      {(spikesData?.total ?? spikesData?.spikes?.length ?? 0)} detected
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
                ) : spikesData && spikesData.spikes && spikesData.spikes.length > 0 ? (
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
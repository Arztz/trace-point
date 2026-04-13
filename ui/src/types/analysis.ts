// Analysis types for Spike Explorer (v1.0.3)
export interface SpikeAnalysisRequest {
  start: string;
  end?: string;
  window?: string;
  namespace?: string;
  replicaset?: string;
  threshold?: number;
  limit?: number;
  offset?: number;
}

export interface AnalysisSummary {
  total_spikes: number;
  time_range_hours: number;
  analyzed_replicasets: number;
  spikes_by_type: {
    cpu: number;
    ram: number;
    both: number;
  };
  top_replicasets: Array<{
    name: string;
    spike_count: number;
  }>;
}

export interface HistoricalSpike {
  id: string;
  timestamp: string;
  replicaset_name: string;
  pod_name: string;
  namespace: string;
  container_name: string;
  type: 'cpu' | 'ram' | 'both';
  cpu_percent: number;
  ram_percent: number;
  moving_average_cpu: number;
  moving_average_ram: number;
  threshold_percent: number;
  deviation_percent: number;
  severity: 'low' | 'medium' | 'critical' | 'normal';
}

export interface SpikeAnalysisResponse {
  request: {
    start: string;
    end: string;
    window: string;
    namespace: string;
    threshold: number;
  };
  summary: AnalysisSummary;
  spikes: HistoricalSpike[];
  pagination: {
    limit: number;
    offset: number;
    has_more: boolean;
  };
}

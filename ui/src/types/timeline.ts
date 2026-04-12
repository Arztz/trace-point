export interface TimelineDataPoint {
  timestamp: string;
  cpu: number;
  ram: number;
}

export interface TimelineRoute {
  timestamp: string;
  route: string;
  duration: number;
}

export interface TimelineData {
  dataPoints: TimelineDataPoint[];
  routes: TimelineRoute[];
  startTime: string;
  endTime: string;
}

// New types for Prometheus-based timeline response (v1.0.1)
export interface TimelineMetric {
  timestamp: string;
  replicaset_name: string; // Grouped replicaset name (e.g., "payment-service")
  pod_name: string;        // Original pod name (e.g., "payment-service-7d9f8b5c6-abcde")
  namespace: string;
  cpu_percent: number;
  ram_percent: number;
}

export interface SpikeMarker {
  timestamp: string;
  pod_name: string;
  namespace: string;
  cpu_spike: boolean;
  ram_spike: boolean;
  route_name?: string;
  trace_id?: string;
}

export interface TimelineResponse {
  generated_at: string;
  start_date: string;
  end_date: string;
  metrics?: TimelineMetric[];
  spike_markers?: SpikeMarker[];
  availablePods?: AvailablePod[]; // Replicaset-level aggregated data
  // Legacy format support
  data_points?: TimelineDataPoint[];
  events?: TimelineDataPoint[];
}

export interface TimeRange {
  value: string;
  label: string;
  hours: number;
}

export const TIME_RANGES: TimeRange[] = [
  { value: '1h', label: '1 hour', hours: 1 },
  { value: '6h', label: '6 hours', hours: 6 },
  { value: '24h', label: '24 hours', hours: 24 },
  { value: '7d', label: '7 days', hours: 168 },
];

// Pod information for the pod selector and legend
export interface PodInfo {
  name: string;
  namespace: string;
  cpu_percent: number;
  ram_percent: number;
  hasSpike?: boolean;
}

// AvailablePod represents a replicaset with aggregated metrics (from API)
export interface AvailablePod {
  name: string;       // Replicaset name (e.g., "payment-service")
  namespace: string;  // Namespace
  pod_count: number; // Number of pods in this replicaset
  current_cpu: number;  // snake_case from API
  current_ram: number;  // snake_case from API
  currentCpu?: number;   // camelCase fallback
  currentRam?: number;  // camelCase fallback
}

// Color palette for different pods (up to 20 unique colors)
export const POD_COLORS: string[] = [
  '#2563eb', // blue
  '#dc2626', // red
  '#16a34a', // green
  '#d97706', // amber
  '#7c3aed', // violet
  '#0891b2', // cyan
  '#db2777', // pink
  '#4f46e5', // indigo
  '#ca8a04', // yellow
  '#059669', // emerald
  '#ea580c', // orange
  '#0d9488', // teal
  '#e11d48', // rose
  '#6366f1', // violet
  '#84cc16', // lime
  '#f59e0b', // amber
  '#14b8a6', // teal
  '#f43f5e', // rose
  '#8b5cf6', // purple
  '#64748b', // slate
];

// Generate consistent color for a pod based on its name hash
export function getPodColor(podName: string, index?: number): string {
  if (index !== undefined && index < POD_COLORS.length) {
    return POD_COLORS[index];
  }
  // Hash the pod name to get a consistent color
  let hash = 0;
  for (let i = 0; i < podName.length; i++) {
    hash = podName.charCodeAt(i) + ((hash << 5) - hash);
  }
  return POD_COLORS[Math.abs(hash) % POD_COLORS.length];
}
export interface SpikeEvent {
  id: string;
  timestamp: string;
  podName: string;
  namespace: string;
  resourceType: 'cpu' | 'memory';
  threshold: number;
  currentValue: number;
  movingAverage: number;
  activeRoutes: string[];
  possibleRootCauses: RootCause[];
}

export interface RootCause {
  route: string;
  functionName: string;
  confidence: number;
  filePath: string;
}

export interface SpikeListResponse {
  spikes: SpikeEvent[];
  total: number;
  page: number;
  pageSize: number;
}

// New: Spike details with profiler data
export interface FunctionProfile {
  function_name: string;
  file_path: string;
  line_number: number;
  cpu_percent: number;
}

export interface ProfilerData {
  top_functions: FunctionProfile[];
}

export interface SpikeDetailsResponse {
  spike: SpikeEvent;
  profiler_data?: ProfilerData;
  trace_data?: unknown;
  active_routes: string[];
  culprit_function?: FunctionProfile;
}
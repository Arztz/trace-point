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
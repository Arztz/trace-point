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
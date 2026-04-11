export interface GravityScore {
  serviceName: string;
  namespace: string;
  cpuScore: number;
  memoryScore: number;
  totalScore: number;
  suggestion: string;
}

export interface GravityScoresResponse {
  scores: GravityScore[];
}
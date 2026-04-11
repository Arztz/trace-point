export interface Config {
  spikeThreshold: number;
  movingAverageWindow: number;
  cooldownMinutes: number;
  prometheusUrl: string;
  signozUrl: string;
  profilerUrl: string;
}

export interface ConfigResponse {
  config: Config;
}
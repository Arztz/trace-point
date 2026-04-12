import axios from 'axios';
import type { SpikeListResponse, TimelineData, ConfigResponse, GravityScoresResponse, SpikeEvent } from '../types';

const apiClient = axios.create({
  baseURL: '/api/v1',
  headers: {
    'Content-Type': 'application/json',
  },
});

apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    console.error('API Error:', error.message);
    return Promise.reject(error);
  }
);

export const api = {
  spikes: {
    list: async (params?: {
      page?: number;
      pageSize?: number;
      namespace?: string;
      podName?: string;
      resourceType?: string;
    }): Promise<SpikeListResponse> => {
      const response = await apiClient.get<SpikeListResponse>('/spikes', { params });
      return response.data;
    },
    
    getById: async (id: string): Promise<SpikeEvent> => {
      const response = await apiClient.get<SpikeEvent>(`/spikes/${id}`);
      return response.data;
    },
  },
  
  timeline: {
    get: async (params: {
      timeRange: string;
      namespace?: string;
      podName?: string;
    }): Promise<TimelineData> => {
      const response = await apiClient.get<TimelineData>('/timeline', { params });
      return response.data;
    },
  },
  
  export: {
    json: async (params?: {
      namespace?: string;
      startTime?: string;
      endTime?: string;
    }): Promise<unknown> => {
      const response = await apiClient.get('/export', { params });
      return response.data;
    },
  },
  
  config: {
    get: async (): Promise<ConfigResponse> => {
      const response = await apiClient.get<ConfigResponse>('/config');
      return response.data;
    },
  },
  
  gravityScores: {
    list: async (params?: {
      namespace?: string;
      minScore?: number;
    }): Promise<GravityScoresResponse> => {
      const response = await apiClient.get<GravityScoresResponse>('/gravity-scores', { params });
      return response.data;
    },
  },
};

export default api;
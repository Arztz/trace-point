// k6 Load Test Configuration for Heavy Load
// Target: 100 pods, 1000 spikes/day simulation

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Rate, Trend } from 'k6/metrics';

export const options = {
  scenarios: {
    // Spike detection simulation - high frequency requests
    spike_detection: {
      executor: 'constant-vus',
      vus: 50,
      duration: '5m',
      tags: { type: 'spike-detection' },
    },
    
    // Dashboard user simulation
    dashboard_users: {
      executor: 'ramping-vus',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 20 },
        { duration: '2m', target: 20 },
        { duration: '30s', target: 50 },
        { duration: '2m', target: 50 },
        { duration: '30s', target: 0 },
      ],
      tags: { type: 'dashboard' },
    },
    
    // API stress test
    api_stress: {
      executor: 'per-vu-iterations',
      vus: 30,
      iterations: 100,
      maxDuration: '3m',
      tags: { type: 'stress' },
    },
  },
  thresholds: {
    http_req_duration: ['p(95)<1000', 'p(99)<2000'],
    http_req_failed: ['rate<0.02'],
    spike_detection_duration: ['p(95)<500'],
  },
};

// Custom metrics
const spikeDetectionDuration = new Trend('spike_detection_duration');
const apiErrors = new Rate('api_errors');
const timelineRequests = new Counter('timeline_requests');

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8081';

// Simulate spike detection polling
export function spikeDetectionScenario() {
  const startTime = new Date();
  
  // Poll timeline for 100 pods
  for (let pod = 0; pod < 100; pod++) {
    const res = http.get(`${BASE_URL}/api/v1/timeline?range=1h&pod=pod-${pod}`);
    
    check(res, {
      'timeline responds': (r) => r.status === 200,
      'response has data': (r) => r.json('data') !== undefined,
    });
    
    timelineRequests.add(1);
  }
  
  const duration = new Date().getTime() - startTime.getTime();
  spikeDetectionDuration.add(duration);
  
  sleep(1);
}

// Simulate dashboard user behavior
export function dashboardUserScenario() {
  // Homepage
  const home = http.get(`${BASE_URL}/`);
  check(home, { 'homepage loads': (r) => r.status === 200 });
  
  // Get timeline
  const ranges = ['1h', '6h', '24h', '7d'];
  const range = ranges[Math.floor(Math.random() * ranges.length)];
  const timeline = http.get(`${BASE_URL}/api/v1/timeline?range=${range}`);
  check(timeline, { 'timeline loads': (r) => r.status === 200 });
  
  // Get spikes
  const spikes = http.get(`${BASE_URL}/api/v1/spikes?limit=20`);
  check(spikes, { 'spikes load': (r) => r.status === 200 });
  
  // Get gravity scores
  const gravity = http.get(`${BASE_URL}/api/v1/gravity-scores`);
  check(gravity, { 'gravity scores load': (r) => r.status === 200 });
  
  sleep(2);
}

// API stress test
export function apiStressScenario() {
  const endpoints = [
    '/health',
    '/api/v1/spikes?limit=50',
    '/api/v1/timeline?range=1h',
    '/api/v1/timeline?range=6h',
    '/api/v1/timeline?range=24h',
    '/api/v1/timeline?range=7d',
    '/api/v1/gravity-scores',
    '/api/v1/export/spikes',
    '/api/v1/export/refactoring',
    '/api/v1/config',
  ];
  
  // Burst requests
  for (const endpoint of endpoints) {
    const res = http.get(`${BASE_URL}${endpoint}`);
    
    const success = check(res, {
      [`${endpoint} works`]: (r) => r.status === 200,
      [`${endpoint} fast`]: (r) => r.timings.duration < 1000,
    });
    
    apiErrors.add(!success);
  }
  
  sleep(0.5);
}

// Default scenario (runs if no specific scenario selected)
export default function () {
  spikeDetectionScenario();
  dashboardUserScenario();
  apiStressScenario();
}

// Specific scenario exports for targeted testing
export { spikeDetectionScenario as spikeDetection };
export { dashboardUserScenario as dashboardUsers };
export { apiStressScenario as apiStress };
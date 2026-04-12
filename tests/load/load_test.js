import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate } from 'k6/metrics';

export const options = {
  stages: [
    { duration: '30s', target: 10 },  // Ramp up to 10 users
    { duration: '1m', target: 10 }, // Stay at 10 users
    { duration: '30s', target: 50 }, // Ramp up to 50 users
    { duration: '1m', target: 50 },  // Stay at 50 users
    { duration: '30s', target: 100 }, // Ramp up to 100 users
    { duration: '1m', target: 100 }, // Stay at 100 users
    { duration: '30s', target: 0 },  // Ramp down
  ],
  thresholds: {
    http_req_duration: ['p(95)<500'], // 95% of requests should be under 500ms
    http_req_failed: ['rate<0.05'],   // Less than 5% failure rate
  },
};

// Custom error rate metric
const errorRate = new Rate('errors');

// Base URL - configure via environment variable
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8081';

export default function () {
  // Test health endpoint
  testHealthEndpoint();
  
  // Test spikes endpoint  
  testSpikesEndpoint();
  
  // Test timeline endpoint
  testTimelineEndpoint();
  
  // Test gravity scores endpoint
  testGravityScoresEndpoint();
  
  // Think time between requests
  sleep(1);
}

function testHealthEndpoint() {
  const res = http.get(`${BASE_URL}/health`);
  
  const success = check(res, {
    'health status is 200': (r) => r.status === 200,
    'health response time < 200ms': (r) => r.timings.duration < 200,
    'health has status field': (r) => r.json('status') !== undefined,
  });
  
  errorRate.add(!success);
}

function testSpikesEndpoint() {
  const res = http.get(`${BASE_URL}/api/v1/spikes?limit=10`);
  
  const success = check(res, {
    'spikes status is 200': (r) => r.status === 200,
    'spikes response time < 500ms': (r) => r.timings.duration < 500,
    'spikes returns array': (r) => Array.isArray(r.json('spikes')),
  });
  
  errorRate.add(!success);
}

function testTimelineEndpoint() {
  const timeRanges = ['1h', '6h', '24h'];
  const range = timeRanges[Math.floor(Math.random() * timeRanges.length)];
  
  const res = http.get(`${BASE_URL}/api/v1/timeline?range=${range}`);
  
  const success = check(res, {
    'timeline status is 200': (r) => r.status === 200,
    'timeline response time < 500ms': (r) => r.timings.duration < 500,
  });
  
  errorRate.add(!success);
}

function testGravityScoresEndpoint() {
  const res = http.get(`${BASE_URL}/api/v1/gravity-scores`);
  
  const success = check(res, {
    'gravity-scores status is 200': (r) => r.status === 200,
    'gravity-scores response time < 500ms': (r) => r.timings.duration < 500,
  });
  
  errorRate.add(!success);
}

// Test spike detection under load
export function spikeTest() {
  // Simulate rapid timeline requests
  for (let i = 0; i < 10; i++) {
    http.get(`${BASE_URL}/api/v1/timeline?range=1h`);
  }
}

// Test export endpoints
export function testExport() {
  const exports = [
    '/api/v1/export/spikes',
    '/api/v1/export/refactoring',
  ];
  
  for (const endpoint of exports) {
    const res = http.get(`${BASE_URL}${endpoint}`);
    
    check(res, {
      [`${endpoint} returns 200`]: (r) => r.status === 200,
      [`${endpoint} response time < 1s`]: (r) => r.timings.duration < 1000,
    });
  }
}

// Smoke test for CI
export function smokeTest() {
  const BASE = __ENV.BASE_URL || 'http://localhost:8081';
  
  // Health check
  const health = http.get(`${BASE}/health`);
  check(health, { 'health OK': (r) => r.status === 200 });
  
  // Basic API checks
  const spikes = http.get(`${BASE}/api/v1/spikes`);
  check(spikes, { 'spikes OK': (r) => r.status === 200 });
  
  const timeline = http.get(`${BASE}/api/v1/timeline?range=1h`);
  check(timeline, { 'timeline OK': (r) => r.status === 200 });
}
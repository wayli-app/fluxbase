/**
 * K6 Load Testing Suite - REST API
 *
 * Tests REST API performance under various load scenarios
 *
 * Run with:
 *   k6 run test/load/k6-rest-api.js
 *   k6 run --vus 100 --duration 30s test/load/k6-rest-api.js
 */

import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const apiDuration = new Trend('api_duration');
const successfulRequests = new Counter('successful_requests');
const failedRequests = new Counter('failed_requests');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const API_KEY = __ENV.API_KEY || '';

// Test options
export const options = {
  stages: [
    // Ramp-up
    { duration: '30s', target: 10 },   // Ramp up to 10 users
    { duration: '1m', target: 50 },    // Ramp up to 50 users
    { duration: '2m', target: 100 },   // Ramp up to 100 users

    // Sustained load
    { duration: '3m', target: 100 },   // Stay at 100 users

    // Peak load
    { duration: '1m', target: 200 },   // Spike to 200 users
    { duration: '2m', target: 200 },   // Hold at peak

    // Ramp-down
    { duration: '1m', target: 50 },    // Scale down
    { duration: '30s', target: 0 },    // Ramp down to 0
  ],

  thresholds: {
    // 95% of requests should complete within 500ms
    'http_req_duration': ['p(95)<500'],

    // 99% of requests should complete within 1000ms
    'http_req_duration{name:query}': ['p(99)<1000'],

    // Error rate should be less than 1%
    'errors': ['rate<0.01'],

    // 95% of requests should succeed
    'http_req_failed': ['rate<0.05'],
  },
};

// Test data
let testUserId = '';
let authToken = '';
let testProductIds = [];

// Setup function - runs once
export function setup() {
  // Sign up a test user
  const signupRes = http.post(`${BASE_URL}/api/v1/auth/signup`, JSON.stringify({
    email: `loadtest-${Date.now()}@test.com`,
    password: 'LoadTest123!'
  }), {
    headers: { 'Content-Type': 'application/json' }
  });

  if (signupRes.status === 201) {
    const data = JSON.parse(signupRes.body);
    authToken = data.access_token;
    testUserId = data.user.id;
  }

  // Create test products for querying
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${authToken}`
  };

  const productIds = [];
  for (let i = 0; i < 100; i++) {
    const product = {
      name: `Load Test Product ${i}`,
      description: `Product for load testing ${i}`,
      price: Math.random() * 100,
      stock: Math.floor(Math.random() * 1000),
      category: ['electronics', 'books', 'clothing'][i % 3]
    };

    const res = http.post(`${BASE_URL}/api/v1/rest/products`, JSON.stringify(product), { headers });
    if (res.status === 201) {
      const data = JSON.parse(res.body);
      if (data.id) {
        productIds.push(data.id);
      }
    }
  }

  return { authToken, testUserId, productIds };
}

// Main test function
export default function (data) {
  const headers = {
    'Content-Type': 'application/json',
    'Authorization': `Bearer ${data.authToken}`
  };

  // Scenario 1: Read Operations (70% of load)
  if (Math.random() < 0.7) {
    group('Read Operations', () => {
      // List all products
      const listRes = http.get(`${BASE_URL}/api/v1/rest/products`, { headers, tags: { name: 'query' } });

      check(listRes, {
        'list products status 200': (r) => r.status === 200,
        'list has results': (r) => {
          const body = JSON.parse(r.body || '[]');
          return Array.isArray(body) && body.length > 0;
        }
      });

      errorRate.add(listRes.status !== 200);
      apiDuration.add(listRes.timings.duration);

      if (listRes.status === 200) {
        successfulRequests.add(1);
      } else {
        failedRequests.add(1);
      }

      // Query with filters
      const filterRes = http.get(
        `${BASE_URL}/api/v1/rest/products?price=gte.20&price=lte.80&select=id,name,price`,
        { headers, tags: { name: 'query' } }
      );

      check(filterRes, {
        'filtered query status 200': (r) => r.status === 200,
      });

      errorRate.add(filterRes.status !== 200);

      // Get single product
      if (data.productIds && data.productIds.length > 0) {
        const randomId = data.productIds[Math.floor(Math.random() * data.productIds.length)];
        const getRes = http.get(`${BASE_URL}/api/v1/rest/products/${randomId}`, { headers, tags: { name: 'get' } });

        check(getRes, {
          'get product status 200': (r) => r.status === 200,
        });

        errorRate.add(getRes.status !== 200);
      }
    });
  }

  // Scenario 2: Write Operations (20% of load)
  else if (Math.random() < 0.9) {
    group('Write Operations', () => {
      // Insert new product
      const product = {
        name: `Product ${Date.now()}`,
        description: 'Test product',
        price: Math.random() * 100,
        stock: Math.floor(Math.random() * 100)
      };

      const insertRes = http.post(
        `${BASE_URL}/api/v1/rest/products`,
        JSON.stringify(product),
        { headers, tags: { name: 'insert' } }
      );

      check(insertRes, {
        'insert status 201': (r) => r.status === 201,
      });

      errorRate.add(insertRes.status !== 201);

      if (insertRes.status === 201) {
        successfulRequests.add(1);

        // Update the product
        const body = JSON.parse(insertRes.body);
        if (body.id) {
          const updateRes = http.patch(
            `${BASE_URL}/api/v1/rest/products/${body.id}`,
            JSON.stringify({ stock: Math.floor(Math.random() * 100) }),
            { headers, tags: { name: 'update' } }
          );

          check(updateRes, {
            'update status 200': (r) => r.status === 200,
          });

          errorRate.add(updateRes.status !== 200);
        }
      } else {
        failedRequests.add(1);
      }
    });
  }

  // Scenario 3: Complex Operations (10% of load)
  else {
    group('Complex Operations', () => {
      // Batch insert
      const products = Array.from({ length: 5 }, (_, i) => ({
        name: `Batch Product ${i}`,
        price: Math.random() * 50,
        stock: Math.floor(Math.random() * 50)
      }));

      const batchRes = http.post(
        `${BASE_URL}/api/v1/rest/products`,
        JSON.stringify(products),
        { headers, tags: { name: 'batch' } }
      );

      check(batchRes, {
        'batch insert status 201': (r) => r.status === 201,
      });

      errorRate.add(batchRes.status !== 201);

      // Aggregation query
      const aggRes = http.get(
        `${BASE_URL}/api/v1/rest/products?select=category,count,avg(price),sum(stock)&group_by=category`,
        { headers, tags: { name: 'aggregation' } }
      );

      check(aggRes, {
        'aggregation status 200': (r) => r.status === 200,
      });

      errorRate.add(aggRes.status !== 200);
    });
  }

  // Random sleep between 0.5 and 2 seconds to simulate real user behavior
  sleep(Math.random() * 1.5 + 0.5);
}

// Teardown function - runs once after all iterations
export function teardown(data) {
  // Clean up test data
  const headers = {
    'Authorization': `Bearer ${data.authToken}`
  };

  // Delete test products
  if (data.productIds && data.productIds.length > 0) {
    data.productIds.forEach(id => {
      http.del(`${BASE_URL}/api/v1/rest/products/${id}`, null, { headers });
    });
  }

  console.log('Load test completed - cleanup done');
}

// Summary handler
export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    '/workspace/test/load/results-rest-api.json': JSON.stringify(data),
  };
}

// Helper function for text summary
function textSummary(data, options = {}) {
  const indent = options.indent || '';
  const enableColors = options.enableColors || false;

  let summary = '\n';
  summary += `${indent}✓ Test Duration: ${Math.round(data.state.testRunDurationMs / 1000)}s\n`;
  summary += `${indent}✓ Checks: ${data.metrics.checks.values.passes} passed, ${data.metrics.checks.values.fails} failed\n`;

  if (data.metrics.http_reqs) {
    summary += `${indent}✓ Total Requests: ${data.metrics.http_reqs.values.count}\n`;
    summary += `${indent}✓ Request Rate: ${data.metrics.http_reqs.values.rate.toFixed(2)} req/s\n`;
  }

  if (data.metrics.http_req_duration) {
    summary += `${indent}✓ Response Time:\n`;
    summary += `${indent}  - avg: ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms\n`;
    summary += `${indent}  - p50: ${data.metrics.http_req_duration.values['p(50)'].toFixed(2)}ms\n`;
    summary += `${indent}  - p95: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms\n`;
    summary += `${indent}  - p99: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms\n`;
    summary += `${indent}  - max: ${data.metrics.http_req_duration.values.max.toFixed(2)}ms\n`;
  }

  if (data.metrics.http_req_failed) {
    const failRate = (data.metrics.http_req_failed.values.rate * 100).toFixed(2);
    summary += `${indent}✓ Failed Requests: ${failRate}%\n`;
  }

  if (data.metrics.successful_requests) {
    summary += `${indent}✓ Successful: ${data.metrics.successful_requests.values.count}\n`;
  }

  if (data.metrics.failed_requests) {
    summary += `${indent}✓ Failed: ${data.metrics.failed_requests.values.count}\n`;
  }

  return summary;
}

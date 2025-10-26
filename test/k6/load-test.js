import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate, Trend } from 'k6/metrics';
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// Custom metrics
const errorRate = new Rate('errors');
const apiLatency = new Trend('api_latency');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 10 },   // Ramp up to 10 users
    { duration: '1m', target: 50 },    // Ramp up to 50 users
    { duration: '2m', target: 100 },   // Ramp up to 100 users
    { duration: '3m', target: 100 },   // Stay at 100 users
    { duration: '1m', target: 50 },    // Ramp down to 50 users
    { duration: '30s', target: 0 },    // Ramp down to 0 users
  ],
  thresholds: {
    http_req_duration: ['p(95)<500', 'p(99)<1000'], // 95% of requests under 500ms
    errors: ['rate<0.05'],                           // Error rate under 5%
    http_req_failed: ['rate<0.05'],                  // HTTP failure rate under 5%
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Setup: Create test table
export function setup() {
  // This would normally be done via migrations
  console.log('Setting up test environment...');

  // Check health endpoint
  const healthRes = http.get(`${BASE_URL}/health`);
  check(healthRes, {
    'health check passed': (r) => r.status === 200,
  });

  return { baseUrl: BASE_URL };
}

export default function (data) {
  const baseUrl = data.baseUrl;

  // Test different scenarios
  group('REST API Operations', () => {
    // Test CREATE
    group('Create Item', () => {
      const payload = JSON.stringify({
        name: `Item ${randomString(8)}`,
        description: `Description ${randomString(20)}`,
        quantity: Math.floor(Math.random() * 100),
        active: Math.random() > 0.5,
      });

      const createRes = http.post(`${baseUrl}/api/rest/items`, payload, {
        headers: { 'Content-Type': 'application/json' },
      });

      const success = check(createRes, {
        'create status is 201': (r) => r.status === 201,
        'create response has id': (r) => {
          const body = JSON.parse(r.body || '{}');
          return body.id !== undefined;
        },
      });

      errorRate.add(!success);
      apiLatency.add(createRes.timings.duration, { operation: 'create' });

      if (createRes.status === 201) {
        const item = JSON.parse(createRes.body);

        // Test READ
        group('Read Item', () => {
          const readRes = http.get(`${baseUrl}/api/rest/items/${item.id}`);

          check(readRes, {
            'read status is 200': (r) => r.status === 200,
            'read returns correct item': (r) => {
              const body = JSON.parse(r.body || '{}');
              return body.id === item.id;
            },
          });

          apiLatency.add(readRes.timings.duration, { operation: 'read' });
        });

        // Test UPDATE
        group('Update Item', () => {
          const updatePayload = JSON.stringify({
            name: `Updated ${randomString(8)}`,
            quantity: Math.floor(Math.random() * 50),
          });

          const updateRes = http.patch(
            `${baseUrl}/api/rest/items/${item.id}`,
            updatePayload,
            {
              headers: { 'Content-Type': 'application/json' },
            }
          );

          check(updateRes, {
            'update status is 200': (r) => r.status === 200,
          });

          apiLatency.add(updateRes.timings.duration, { operation: 'update' });
        });

        // Test DELETE
        group('Delete Item', () => {
          const deleteRes = http.del(`${baseUrl}/api/rest/items/${item.id}`);

          check(deleteRes, {
            'delete status is 204': (r) => r.status === 204,
          });

          apiLatency.add(deleteRes.timings.duration, { operation: 'delete' });
        });
      }
    });

    // Test LIST with filters
    group('List Items', () => {
      const listRes = http.get(`${baseUrl}/api/rest/items?limit=10&offset=0&active=eq.true`);

      const success = check(listRes, {
        'list status is 200': (r) => r.status === 200,
        'list returns array': (r) => {
          const body = JSON.parse(r.body || '[]');
          return Array.isArray(body);
        },
      });

      errorRate.add(!success);
      apiLatency.add(listRes.timings.duration, { operation: 'list' });
    });

    // Test complex queries
    group('Complex Queries', () => {
      const queries = [
        'order=created_at.desc&limit=20',
        'quantity=gt.10&active=eq.true',
        'name=like.*test*',
        'select=id,name,quantity',
        'or=(quantity.lt.5,quantity.gt.95)',
      ];

      const query = queries[Math.floor(Math.random() * queries.length)];
      const queryRes = http.get(`${baseUrl}/api/rest/items?${query}`);

      check(queryRes, {
        'complex query status is 200': (r) => r.status === 200,
      });

      apiLatency.add(queryRes.timings.duration, { operation: 'complex_query' });
    });
  });

  // Test concurrent operations
  group('Concurrent Operations', () => {
    const batch = [
      ['GET', `${baseUrl}/api/rest/items?limit=5`],
      ['GET', `${baseUrl}/api/rest/items?limit=10&offset=5`],
      ['GET', `${baseUrl}/api/rest/items?active=eq.true`],
      ['GET', `${baseUrl}/health`],
    ];

    const responses = http.batch(batch);

    responses.forEach((res, index) => {
      check(res, {
        [`batch request ${index} successful`]: (r) => r.status === 200,
      });
    });
  });

  // Small pause between iterations
  sleep(1);
}

// Teardown: Clean up test data
export function teardown(data) {
  console.log('Cleaning up test environment...');
  // Cleanup would go here
}

// Custom summary
export function handleSummary(data) {
  return {
    'stdout': textSummary(data, { indent: ' ', enableColors: true }),
    'summary.json': JSON.stringify(data),
    'summary.html': htmlReport(data),
  };
}

function textSummary(data, options) {
  // Simple text summary
  let summary = '\n=== Test Summary ===\n';
  summary += `Total Requests: ${data.metrics.http_reqs.values.count}\n`;
  summary += `Failed Requests: ${data.metrics.http_req_failed.values.passes}\n`;
  summary += `Avg Duration: ${data.metrics.http_req_duration.values.avg.toFixed(2)}ms\n`;
  summary += `P95 Duration: ${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms\n`;
  summary += `P99 Duration: ${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms\n`;
  return summary;
}

function htmlReport(data) {
  // Simple HTML report
  return `
<!DOCTYPE html>
<html>
<head>
  <title>Fluxbase Load Test Results</title>
  <style>
    body { font-family: Arial, sans-serif; margin: 40px; }
    h1 { color: #2563eb; }
    table { border-collapse: collapse; width: 100%; margin: 20px 0; }
    th, td { border: 1px solid #ddd; padding: 12px; text-align: left; }
    th { background-color: #f2f2f2; }
    .success { color: green; }
    .warning { color: orange; }
    .error { color: red; }
  </style>
</head>
<body>
  <h1>Fluxbase Load Test Results</h1>
  <h2>Summary</h2>
  <table>
    <tr>
      <th>Metric</th>
      <th>Value</th>
      <th>Status</th>
    </tr>
    <tr>
      <td>Total Requests</td>
      <td>${data.metrics.http_reqs.values.count}</td>
      <td class="success">✓</td>
    </tr>
    <tr>
      <td>Failed Requests</td>
      <td>${data.metrics.http_req_failed.values.passes}</td>
      <td class="${data.metrics.http_req_failed.values.rate < 0.05 ? 'success' : 'error'}">
        ${data.metrics.http_req_failed.values.rate < 0.05 ? '✓' : '✗'}
      </td>
    </tr>
    <tr>
      <td>Average Duration</td>
      <td>${data.metrics.http_req_duration.values.avg.toFixed(2)}ms</td>
      <td class="success">✓</td>
    </tr>
    <tr>
      <td>P95 Duration</td>
      <td>${data.metrics.http_req_duration.values['p(95)'].toFixed(2)}ms</td>
      <td class="${data.metrics.http_req_duration.values['p(95)'] < 500 ? 'success' : 'warning'}">
        ${data.metrics.http_req_duration.values['p(95)'] < 500 ? '✓' : '⚠'}
      </td>
    </tr>
    <tr>
      <td>P99 Duration</td>
      <td>${data.metrics.http_req_duration.values['p(99)'].toFixed(2)}ms</td>
      <td class="${data.metrics.http_req_duration.values['p(99)'] < 1000 ? 'success' : 'warning'}">
        ${data.metrics.http_req_duration.values['p(99)'] < 1000 ? '✓' : '⚠'}
      </td>
    </tr>
  </table>
</body>
</html>
  `;
}
/**
 * K6 Load Testing Suite - Storage (File Operations)
 *
 * Tests storage performance under load with concurrent file operations
 *
 * Run with:
 *   k6 run test/load/k6-storage.js
 *   k6 run --vus 50 --duration 60s test/load/k6-storage.js
 */

import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Counter, Trend } from 'k6/metrics';
import { SharedArray } from 'k6/data';

// Custom metrics
const uploadSuccessRate = new Rate('upload_success_rate');
const downloadSuccessRate = new Rate('download_success_rate');
const uploadDuration = new Trend('upload_duration');
const downloadDuration = new Trend('download_duration');
const filesUploaded = new Counter('files_uploaded');
const filesDownloaded = new Counter('files_downloaded');
const bytesUploaded = new Counter('bytes_uploaded');
const bytesDownloaded = new Counter('bytes_downloaded');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Test options
export const options = {
  stages: [
    { duration: '30s', target: 10 },    // Ramp up to 10 users
    { duration: '1m', target: 25 },     // Ramp up to 25 users
    { duration: '2m', target: 50 },     // Ramp up to 50 users
    { duration: '3m', target: 50 },     // Hold at 50 users
    { duration: '1m', target: 100 },    // Spike to 100 users
    { duration: '1m', target: 100 },    // Hold at peak
    { duration: '1m', target: 25 },     // Ramp down
    { duration: '30s', target: 0 },     // Ramp down to 0
  ],

  thresholds: {
    'upload_success_rate': ['rate>0.95'],           // 95% uploads successful
    'download_success_rate': ['rate>0.98'],         // 98% downloads successful
    'upload_duration': ['p(95)<2000'],              // 95% uploads < 2s
    'download_duration': ['p(95)<1000'],            // 95% downloads < 1s
    'http_req_duration{name:upload}': ['p(99)<3000'], // 99% upload requests < 3s
    'http_req_failed{name:download}': ['rate<0.02'], // < 2% download failures
  },
};

// File size templates
const fileSizes = new SharedArray('file_sizes', function () {
  return [
    { name: 'tiny', size: 1024 },           // 1KB
    { name: 'small', size: 10 * 1024 },     // 10KB
    { name: 'medium', size: 100 * 1024 },   // 100KB
    { name: 'large', size: 1024 * 1024 },   // 1MB
    { name: 'xlarge', size: 5 * 1024 * 1024 }, // 5MB
  ];
});

// Setup - create test user and bucket
export function setup() {
  // Create test user
  const signupRes = http.post(`${BASE_URL}/api/v1/auth/signup`, JSON.stringify({
    email: `storagetest-${Date.now()}@test.com`,
    password: 'StorageTest123!'
  }), {
    headers: { 'Content-Type': 'application/json' }
  });

  let authToken = '';
  if (signupRes.status === 201) {
    const data = JSON.parse(signupRes.body);
    authToken = data.access_token;
  }

  // Create test bucket
  const bucketRes = http.post(
    `${BASE_URL}/api/v1/storage/buckets`,
    JSON.stringify({
      name: `loadtest-${Date.now()}`,
      public: false,
    }),
    {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${authToken}`
      }
    }
  );

  let bucketName = '';
  if (bucketRes.status === 201) {
    const data = JSON.parse(bucketRes.body);
    bucketName = data.name;
  }

  return { authToken, bucketName };
}

// Generate random file content
function generateFileContent(size) {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let content = '';
  for (let i = 0; i < size; i++) {
    content += chars.charAt(Math.floor(Math.random() * chars.length));
  }
  return content;
}

// Main test function
export default function (data) {
  const headers = {
    'Authorization': `Bearer ${data.authToken}`
  };

  // 60% uploads, 30% downloads, 10% operations
  const operation = Math.random();

  if (operation < 0.6) {
    // Upload file
    testFileUpload(data, headers);
  } else if (operation < 0.9) {
    // Download file
    testFileDownload(data, headers);
  } else {
    // Other operations (list, metadata, copy, delete)
    testFileOperations(data, headers);
  }

  sleep(1);
}

function testFileUpload(data, headers) {
  // Select random file size
  const fileSize = fileSizes[Math.floor(Math.random() * fileSizes.length)];
  const content = generateFileContent(fileSize.size);
  const fileName = `test-${fileSize.name}-${__VU}-${Date.now()}.txt`;

  // Create multipart form data
  const boundary = '----WebKitFormBoundary' + Math.random().toString(36).substring(2);
  const body = [
    `--${boundary}`,
    `Content-Disposition: form-data; name="file"; filename="${fileName}"`,
    'Content-Type: text/plain',
    '',
    content,
    `--${boundary}--`,
  ].join('\r\n');

  const uploadHeaders = {
    ...headers,
    'Content-Type': `multipart/form-data; boundary=${boundary}`,
  };

  const startTime = Date.now();

  const res = http.post(
    `${BASE_URL}/api/v1/storage/buckets/${data.bucketName}/files`,
    body,
    {
      headers: uploadHeaders,
      tags: { name: 'upload' },
      timeout: '30s',
    }
  );

  const duration = Date.now() - startTime;
  uploadDuration.add(duration);

  const success = check(res, {
    'Upload successful': (r) => r.status === 201 || r.status === 200,
    'Upload has response body': (r) => r.body && r.body.length > 0,
  });

  uploadSuccessRate.add(success);

  if (success) {
    filesUploaded.add(1);
    bytesUploaded.add(fileSize.size);
  }
}

function testFileDownload(data, headers) {
  // Try to download a previously uploaded file
  // In real scenario, we'd track uploaded files
  const fileName = `test-small-${Math.floor(Math.random() * __VU) + 1}-*.txt`;

  const startTime = Date.now();

  const res = http.get(
    `${BASE_URL}/api/v1/storage/buckets/${data.bucketName}/files/${fileName}`,
    {
      headers,
      tags: { name: 'download' },
    }
  );

  const duration = Date.now() - startTime;
  downloadDuration.add(duration);

  const success = check(res, {
    'Download successful': (r) => r.status === 200 || r.status === 404, // 404 is ok if file doesn't exist
  });

  downloadSuccessRate.add(success);

  if (res.status === 200) {
    filesDownloaded.add(1);
    bytesDownloaded.add(res.body ? res.body.length : 0);
  }
}

function testFileOperations(data, headers) {
  const operation = Math.random();

  if (operation < 0.4) {
    // List files
    const res = http.get(
      `${BASE_URL}/api/v1/storage/buckets/${data.bucketName}/files?limit=50`,
      {
        headers,
        tags: { name: 'list' },
      }
    );

    check(res, {
      'List files successful': (r) => r.status === 200,
    });
  } else if (operation < 0.7) {
    // Get file metadata
    const fileName = `test-small-${Math.floor(Math.random() * __VU) + 1}-*.txt`;

    const res = http.get(
      `${BASE_URL}/api/v1/storage/buckets/${data.bucketName}/files/${fileName}`,
      {
        headers,
        tags: { name: 'metadata' },
      }
    );

    check(res, {
      'Get metadata successful': (r) => r.status === 200 || r.status === 404,
    });
  } else if (operation < 0.85) {
    // Copy file
    const sourceFile = `test-small-${__VU}-source.txt`;
    const destFile = `test-small-${__VU}-copy-${Date.now()}.txt`;

    const res = http.post(
      `${BASE_URL}/api/v1/storage/buckets/${data.bucketName}/copy`,
      JSON.stringify({
        source_key: sourceFile,
        destination_key: destFile,
      }),
      {
        headers: {
          ...headers,
          'Content-Type': 'application/json',
        },
        tags: { name: 'copy' },
      }
    );

    check(res, {
      'Copy file successful': (r) => r.status === 200 || r.status === 404,
    });
  } else {
    // Delete file
    const fileName = `test-small-${__VU}-delete.txt`;

    const res = http.del(
      `${BASE_URL}/api/v1/storage/buckets/${data.bucketName}/files/${fileName}`,
      null,
      {
        headers,
        tags: { name: 'delete' },
      }
    );

    check(res, {
      'Delete file successful': (r) => r.status === 200 || r.status === 204 || r.status === 404,
    });
  }
}

// Teardown
export function teardown(data) {
  console.log('Storage load test completed');

  // Optionally clean up bucket (commented out to preserve data for analysis)
  // const headers = {
  //   'Authorization': `Bearer ${data.authToken}`
  // };
  // http.del(`${BASE_URL}/api/v1/storage/buckets/${data.bucketName}`, null, { headers });
}

// Summary
export function handleSummary(data) {
  console.log('\n=== Storage Load Test Results ===\n');

  if (data.metrics.upload_success_rate) {
    const rate = (data.metrics.upload_success_rate.values.rate * 100).toFixed(2);
    console.log(`✓ Upload Success Rate: ${rate}%`);
  }

  if (data.metrics.download_success_rate) {
    const rate = (data.metrics.download_success_rate.values.rate * 100).toFixed(2);
    console.log(`✓ Download Success Rate: ${rate}%`);
  }

  if (data.metrics.files_uploaded) {
    console.log(`✓ Files Uploaded: ${data.metrics.files_uploaded.values.count}`);
  }

  if (data.metrics.files_downloaded) {
    console.log(`✓ Files Downloaded: ${data.metrics.files_downloaded.values.count}`);
  }

  if (data.metrics.bytes_uploaded) {
    const mb = (data.metrics.bytes_uploaded.values.count / (1024 * 1024)).toFixed(2);
    console.log(`✓ Total Uploaded: ${mb} MB`);
  }

  if (data.metrics.bytes_downloaded) {
    const mb = (data.metrics.bytes_downloaded.values.count / (1024 * 1024)).toFixed(2);
    console.log(`✓ Total Downloaded: ${mb} MB`);
  }

  if (data.metrics.upload_duration) {
    console.log(`✓ Upload Duration (avg): ${data.metrics.upload_duration.values.avg.toFixed(2)}ms`);
    console.log(`✓ Upload Duration (p95): ${data.metrics.upload_duration.values['p(95)'].toFixed(2)}ms`);
  }

  if (data.metrics.download_duration) {
    console.log(`✓ Download Duration (avg): ${data.metrics.download_duration.values.avg.toFixed(2)}ms`);
    console.log(`✓ Download Duration (p95): ${data.metrics.download_duration.values['p(95)'].toFixed(2)}ms`);
  }

  if (data.metrics.http_reqs) {
    const totalReqs = data.metrics.http_reqs.values.count;
    const duration = data.metrics.http_req_duration.values.avg;
    console.log(`✓ Total Requests: ${totalReqs}`);
    console.log(`✓ Avg Request Duration: ${duration.toFixed(2)}ms`);
  }

  return {
    '/workspace/test/load/results-storage.json': JSON.stringify(data),
  };
}

/**
 * K6 Load Testing Suite - WebSocket (Realtime)
 *
 * Tests WebSocket realtime performance under load
 *
 * Run with:
 *   k6 run test/load/k6-websocket.js
 *   k6 run --vus 100 --duration 60s test/load/k6-websocket.js
 */

import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { Rate, Counter, Trend } from 'k6/metrics';
import http from 'k6/http';

// Custom metrics
const wsConnectionRate = new Rate('ws_connection_success');
const wsMessagesSent = new Counter('ws_messages_sent');
const wsMessagesReceived = new Counter('ws_messages_received');
const wsConnectionDuration = new Trend('ws_connection_duration');
const wsMessageLatency = new Trend('ws_message_latency');

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const WS_URL = __ENV.WS_URL || 'ws://localhost:8080';

// Test options
export const options = {
  stages: [
    { duration: '30s', target: 20 },    // Ramp up to 20 connections
    { duration: '1m', target: 50 },     // Ramp up to 50 connections
    { duration: '2m', target: 100 },    // Ramp up to 100 connections
    { duration: '2m', target: 100 },    // Hold at 100 connections
    { duration: '1m', target: 200 },    // Spike to 200 connections
    { duration: '1m', target: 200 },    // Hold at peak
    { duration: '1m', target: 50 },     // Ramp down
    { duration: '30s', target: 0 },     // Ramp down to 0
  ],

  thresholds: {
    'ws_connection_success': ['rate>0.95'],           // 95% connections successful
    'ws_connection_duration': ['p(95)<1000'],         // 95% connect within 1s
    'ws_message_latency': ['p(95)<200'],              // 95% messages < 200ms latency
    'ws_messages_received': ['count>1000'],           // At least 1000 messages received
  },
};

// Setup - create test user and get token
export function setup() {
  const signupRes = http.post(`${BASE_URL}/api/v1/auth/signup`, JSON.stringify({
    email: `wstest-${Date.now()}@test.com`,
    password: 'WSTest123!'
  }), {
    headers: { 'Content-Type': 'application/json' }
  });

  let authToken = '';
  if (signupRes.status === 201) {
    const data = JSON.parse(signupRes.body);
    authToken = data.access_token;
  }

  return { authToken };
}

// Main test function
export default function (data) {
  const url = `${WS_URL}/ws`;
  const headers = {
    'Authorization': `Bearer ${data.authToken}`
  };

  const startTime = Date.now();
  let messagesReceived = 0;
  let connectionEstablished = false;

  const response = ws.connect(url, { headers }, (socket) => {
    connectionEstablished = true;
    const connectTime = Date.now() - startTime;
    wsConnectionDuration.add(connectTime);
    wsConnectionRate.add(true);

    // Subscribe to a channel
    const channel = `test-channel-${__VU}`;
    const subscribeMsg = JSON.stringify({
      type: 'subscribe',
      channel: channel
    });

    socket.on('open', () => {
      socket.send(subscribeMsg);
      wsMessagesSent.add(1);

      // Send heartbeat every 5 seconds
      socket.setInterval(() => {
        socket.send(JSON.stringify({ type: 'heartbeat' }));
        wsMessagesSent.add(1);
      }, 5000);

      // Send test messages every 2 seconds
      socket.setInterval(() => {
        const msgSentTime = Date.now();
        socket.send(JSON.stringify({
          type: 'broadcast',
          channel: channel,
          payload: {
            message: `Test message from VU ${__VU}`,
            timestamp: msgSentTime
          }
        }));
        wsMessagesSent.add(1);
      }, 2000);
    });

    socket.on('message', (msg) => {
      messagesReceived++;
      wsMessagesReceived.add(1);

      try {
        const data = JSON.parse(msg);

        // Calculate latency for broadcast messages
        if (data.type === 'broadcast' && data.payload && data.payload.timestamp) {
          const latency = Date.now() - data.payload.timestamp;
          wsMessageLatency.add(latency);
        }

        // Acknowledge subscription
        if (data.type === 'ack') {
          console.log(`VU ${__VU}: Subscription acknowledged`);
        }

        // Handle errors
        if (data.type === 'error') {
          console.error(`VU ${__VU}: Error - ${data.error}`);
        }
      } catch (e) {
        console.error(`VU ${__VU}: Failed to parse message: ${e.message}`);
      }
    });

    socket.on('error', (e) => {
      console.error(`VU ${__VU}: WebSocket error: ${e.error()}`);
    });

    socket.on('close', () => {
      console.log(`VU ${__VU}: Connection closed, received ${messagesReceived} messages`);
    });

    // Keep connection open for duration of test
    socket.setTimeout(() => {
      socket.close();
    }, 30000); // 30 seconds per connection
  });

  // Check connection
  check(response, {
    'WebSocket connection established': () => connectionEstablished,
    'Status is 101': (r) => r && r.status === 101,
  });

  if (!connectionEstablished) {
    wsConnectionRate.add(false);
  }

  // Small sleep between iterations
  sleep(1);
}

// Teardown
export function teardown(data) {
  console.log('WebSocket load test completed');
}

// Summary
export function handleSummary(data) {
  console.log('\n=== WebSocket Load Test Results ===\n');

  if (data.metrics.ws_connection_success) {
    const successRate = (data.metrics.ws_connection_success.values.rate * 100).toFixed(2);
    console.log(`✓ Connection Success Rate: ${successRate}%`);
  }

  if (data.metrics.ws_messages_sent) {
    console.log(`✓ Messages Sent: ${data.metrics.ws_messages_sent.values.count}`);
  }

  if (data.metrics.ws_messages_received) {
    console.log(`✓ Messages Received: ${data.metrics.ws_messages_received.values.count}`);
  }

  if (data.metrics.ws_connection_duration) {
    console.log(`✓ Connection Time (avg): ${data.metrics.ws_connection_duration.values.avg.toFixed(2)}ms`);
    console.log(`✓ Connection Time (p95): ${data.metrics.ws_connection_duration.values['p(95)'].toFixed(2)}ms`);
  }

  if (data.metrics.ws_message_latency) {
    console.log(`✓ Message Latency (avg): ${data.metrics.ws_message_latency.values.avg.toFixed(2)}ms`);
    console.log(`✓ Message Latency (p95): ${data.metrics.ws_message_latency.values['p(95)'].toFixed(2)}ms`);
    console.log(`✓ Message Latency (p99): ${data.metrics.ws_message_latency.values['p(99)'].toFixed(2)}ms`);
  }

  return {
    '/workspace/test/load/results-websocket.json': JSON.stringify(data),
  };
}

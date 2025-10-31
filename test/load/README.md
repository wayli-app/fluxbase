# Load Testing Suite - K6

**Purpose**: Establish performance baselines and test system capacity under load

**Tools**: k6 (Grafana's modern load testing tool)

## 📊 Overview

This directory contains comprehensive load testing scenarios for all Fluxbase services:

| Test File | Service | Focus Area | Target Load |
|-----------|---------|------------|-------------|
| [k6-rest-api.js](k6-rest-api.js) | REST API | CRUD, Queries, RLS | 5000+ req/s |
| [k6-websocket.js](k6-websocket.js) | Realtime | WebSocket connections | 10K+ connections |
| [k6-storage.js](k6-storage.js) | Storage | File uploads/downloads | 50+ concurrent users |

## 🚀 Quick Start

### Install k6

**macOS**:
```bash
brew install k6
```

**Linux**:
```bash
sudo gpg -k
sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69
echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list
sudo apt-get update
sudo apt-get install k6
```

**Windows**:
```powershell
choco install k6
```

**Docker**:
```bash
docker pull grafana/k6
```

### Run Tests

**REST API Load Test**:
```bash
# Default test
k6 run test/load/k6-rest-api.js

# Custom configuration
k6 run --vus 100 --duration 60s test/load/k6-rest-api.js

# With custom base URL
BASE_URL=https://api.fluxbase.com k6 run test/load/k6-rest-api.js

# Output to InfluxDB for visualization
k6 run --out influxdb=http://localhost:8086/k6 test/load/k6-rest-api.js
```

**WebSocket Load Test**:
```bash
# Default test
k6 run test/load/k6-websocket.js

# Stress test (more connections)
k6 run --vus 200 --duration 120s test/load/k6-websocket.js

# With custom WebSocket URL
WS_URL=wss://api.fluxbase.com k6 run test/load/k6-websocket.js
```

**Storage Load Test**:
```bash
# Default test
k6 run test/load/k6-storage.js

# High concurrency
k6 run --vus 100 --duration 90s test/load/k6-storage.js

# Focus on large files
k6 run test/load/k6-storage.js
```

## 📋 Test Scenarios

### 1. REST API Load Test

**File**: [k6-rest-api.js](k6-rest-api.js)

**Load Profile**:
```
10 users  → 30s  → 50 users  → 1m → 100 users → 2m → 100 users (hold) → 3m
         → 200 users (spike) → 1m → 200 users (hold) → 2m → 50 users → 1m → 0
```

**Operation Mix**:
- 70% Read operations (SELECT queries)
- 20% Write operations (INSERT, UPDATE, DELETE)
- 10% Complex operations (aggregations, joins, full-text search)

**Performance Targets**:
- ✅ 95% of requests < 500ms
- ✅ 99% of query requests < 1000ms
- ✅ Error rate < 1%
- ✅ Request failure rate < 5%
- ✅ Throughput > 5000 req/s at peak

**Key Metrics**:
- `http_req_duration` - Request latency (p50, p95, p99)
- `http_reqs` - Total requests per second
- `http_req_failed` - Failed request rate
- `errors` - Application error rate

**What's Tested**:
- CRUD operations on multiple tables
- Complex queries with filters (eq, gt, lt, like, in)
- Aggregations (count, sum, avg)
- Full-text search
- Upsert operations
- Batch operations
- RLS enforcement under load
- Database connection pooling

### 2. WebSocket (Realtime) Load Test

**File**: [k6-websocket.js](k6-websocket.js)

**Load Profile**:
```
20 connections → 30s → 50 connections → 1m → 100 connections → 2m
              → 100 (hold) → 2m → 200 (spike) → 1m → 200 (hold) → 1m
              → 50 → 1m → 0
```

**Operation Mix**:
- Connection lifecycle (open, maintain, close)
- Channel subscriptions (subscribe, unsubscribe)
- Message broadcasts (send, receive)
- Heartbeat mechanism

**Performance Targets**:
- ✅ 95% connection success rate
- ✅ 95% connections established < 1s
- ✅ 95% message latency < 200ms
- ✅ At least 1000 messages received
- ✅ Support 10K+ concurrent connections

**Key Metrics**:
- `ws_connection_success` - Connection success rate
- `ws_connection_duration` - Time to establish connection
- `ws_message_latency` - Message round-trip time
- `ws_messages_sent` - Total messages sent
- `ws_messages_received` - Total messages received

**What's Tested**:
- Concurrent WebSocket connections
- Subscription management
- Broadcast message routing
- Connection recovery
- Authentication via JWT
- Heartbeat keepalive
- Memory leak detection

### 3. Storage Load Test

**File**: [k6-storage.js](k6-storage.js)

**Load Profile**:
```
10 users → 30s → 25 users → 1m → 50 users → 2m → 50 (hold) → 3m
        → 100 (spike) → 1m → 100 (hold) → 1m → 25 → 1m → 0
```

**Operation Mix**:
- 60% File uploads (various sizes: 1KB - 5MB)
- 30% File downloads
- 10% Other operations (list, metadata, copy, delete)

**File Sizes**:
- Tiny: 1KB
- Small: 10KB
- Medium: 100KB
- Large: 1MB
- XLarge: 5MB

**Performance Targets**:
- ✅ 95% upload success rate
- ✅ 98% download success rate
- ✅ 95% uploads < 2s
- ✅ 95% downloads < 1s
- ✅ 99% upload requests < 3s
- ✅ Download failure rate < 2%

**Key Metrics**:
- `upload_success_rate` - Upload success percentage
- `download_success_rate` - Download success percentage
- `upload_duration` - Time to upload file
- `download_duration` - Time to download file
- `files_uploaded` - Total files uploaded
- `bytes_uploaded` - Total bandwidth used (uploads)
- `bytes_downloaded` - Total bandwidth used (downloads)

**What's Tested**:
- Concurrent file uploads
- Multipart form data handling
- File size limits
- Download streaming
- File metadata operations
- Copy/move operations
- Bucket management under load
- Storage quota enforcement

## 🎯 Performance Baselines

### Expected Results (Single Node)

**Hardware Assumptions**:
- CPU: 4 cores
- RAM: 8GB
- Storage: SSD
- Network: 1Gbps

| Service | Metric | Target | Production |
|---------|--------|--------|------------|
| **REST API** | Throughput | 5000 req/s | 10K+ req/s |
| | P95 Latency | < 500ms | < 200ms |
| | P99 Latency | < 1000ms | < 500ms |
| | Error Rate | < 1% | < 0.1% |
| **WebSocket** | Connections | 10K | 50K+ |
| | P95 Connect Time | < 1s | < 500ms |
| | P95 Message Latency | < 200ms | < 100ms |
| | Success Rate | > 95% | > 99% |
| **Storage** | Upload Rate | 50 files/s | 100+ files/s |
| | P95 Upload Time (1MB) | < 2s | < 1s |
| | P95 Download Time | < 1s | < 500ms |
| | Success Rate | > 95% | > 99% |

## 📈 Interpreting Results

### Good Performance Indicators

✅ **Stable Response Times**: P95 and P99 latencies remain consistent throughout test
✅ **Low Error Rates**: < 1% errors across all operations
✅ **Linear Scaling**: Throughput increases proportionally with VUs (up to saturation point)
✅ **Quick Recovery**: System recovers quickly after spike loads
✅ **Efficient Resource Usage**: CPU/memory usage proportional to load

### Warning Signs

⚠️ **Increasing Latency**: P95/P99 grows over time (potential memory leak)
⚠️ **High Error Rates**: > 5% errors (system overloaded or misconfigured)
⚠️ **Timeouts**: Requests timing out (insufficient resources or deadlocks)
⚠️ **Connection Failures**: WebSocket connections dropping (connection limit reached)
⚠️ **Uneven Distribution**: Some operations fast, others very slow (resource contention)

### Critical Issues

🚨 **Complete Failures**: > 50% error rate (system down or critical bug)
🚨 **Memory Growth**: Continuous memory increase (memory leak)
🚨 **CPU Saturation**: 100% CPU with low throughput (inefficient code)
🚨 **Database Deadlocks**: Queries timing out or failing (lock contention)
🚨 **Cascading Failures**: Errors in one service affecting others

## 🔧 Troubleshooting

### Test Fails to Start

**Issue**: "connection refused" errors

**Solutions**:
```bash
# Check if Fluxbase is running
curl http://localhost:8080/health

# Check if database is accessible
psql -h localhost -U postgres -d fluxbase_dev -c "SELECT 1"

# Verify environment variables
echo $BASE_URL
echo $WS_URL
```

### Low Throughput

**Issue**: Tests complete but throughput is lower than expected

**Solutions**:
1. **Database Connection Pool**: Increase `max_connections` in PostgreSQL
2. **Rate Limiting**: Check if rate limits are being hit
3. **Resource Limits**: Increase system limits (ulimit, file descriptors)
4. **Network Bandwidth**: Verify network is not saturated

### High Error Rates

**Issue**: > 5% error rate during tests

**Solutions**:
1. **Check Logs**: Review Fluxbase server logs for errors
2. **Database Capacity**: Monitor PostgreSQL connections and query performance
3. **Memory**: Ensure sufficient memory available
4. **Timeouts**: Increase timeout values if requests are slow but successful

### WebSocket Connection Issues

**Issue**: Connections fail or drop frequently

**Solutions**:
1. **Connection Limits**: Check OS file descriptor limits (`ulimit -n`)
2. **Proxy Issues**: Ensure proxy/load balancer supports WebSocket upgrades
3. **Timeout Configuration**: Increase WebSocket timeout values
4. **Heartbeat**: Verify heartbeat mechanism is working

## 📊 Visualization

### InfluxDB + Grafana

Export metrics to InfluxDB for real-time visualization:

```bash
# Start InfluxDB and Grafana
docker-compose up -d influxdb grafana

# Run tests with InfluxDB output
k6 run --out influxdb=http://localhost:8086/k6 test/load/k6-rest-api.js
```

### k6 Cloud

For hosted visualization and storage:

```bash
# Sign up at k6.io/cloud
k6 login cloud

# Run tests with cloud output
k6 run --out cloud test/load/k6-rest-api.js
```

### Custom Dashboard

Results are saved as JSON files for custom analysis:

- `results-rest-api.json` - REST API test results
- `results-websocket.json` - WebSocket test results
- `results-storage.json` - Storage test results

Parse these with your own tools or import into observability platforms.

## 🔄 CI/CD Integration

### GitHub Actions Example

```yaml
name: Performance Tests

on:
  push:
    branches: [main]
  schedule:
    - cron: '0 2 * * *'  # Run daily at 2 AM

jobs:
  load-test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Start Fluxbase
        run: docker-compose up -d

      - name: Install k6
        run: |
          curl https://github.com/grafana/k6/releases/download/v0.47.0/k6-v0.47.0-linux-amd64.tar.gz -L | tar xvz
          sudo cp k6-v0.47.0-linux-amd64/k6 /usr/bin

      - name: Run REST API tests
        run: k6 run test/load/k6-rest-api.js

      - name: Run WebSocket tests
        run: k6 run test/load/k6-websocket.js

      - name: Run Storage tests
        run: k6 run test/load/k6-storage.js

      - name: Upload results
        uses: actions/upload-artifact@v3
        with:
          name: load-test-results
          path: test/load/results-*.json
```

## 📚 Best Practices

### Running Load Tests

1. **Baseline First**: Run tests on known-good system to establish baseline
2. **Incremental Load**: Start with low load and increase gradually
3. **Realistic Scenarios**: Model actual user behavior patterns
4. **Multiple Iterations**: Run multiple times to account for variance
5. **Monitor Resources**: Watch CPU, memory, disk I/O during tests

### Test Environment

1. **Isolated Environment**: Use dedicated test environment
2. **Production Parity**: Match production hardware/configuration
3. **Clean State**: Reset database between test runs
4. **No Other Load**: Ensure no other processes consuming resources
5. **Network Conditions**: Test with realistic network latency

### Interpreting Results

1. **Focus on Percentiles**: P95/P99 more important than average
2. **Look for Patterns**: Identify when performance degrades
3. **Compare Over Time**: Track metrics across test runs
4. **Set SLOs**: Define service level objectives and measure against them
5. **Document Findings**: Record baseline metrics and any issues found

## 🎯 Capacity Planning

### Scaling Guidelines

Based on load test results, you can estimate capacity needs:

**Single Node Capacity** (approximate):
- 5,000 REST API requests/second
- 10,000 concurrent WebSocket connections
- 50 concurrent file uploads/downloads

**Scaling Horizontally**:
- Add more application servers behind load balancer
- Scale PostgreSQL with read replicas
- Use Redis for session state sharing
- CDN for static assets and public files

**Scaling Vertically**:
- Increase CPU cores (helps with concurrency)
- Increase RAM (helps with connection pooling and caching)
- Faster storage (helps with database performance)

### When to Scale

Monitor these metrics in production:

- **CPU Usage** > 70% sustained
- **Memory Usage** > 80%
- **Response Time** P95 > 500ms
- **Error Rate** > 1%
- **Database Connections** > 80% of max

## 📖 Additional Resources

- [k6 Documentation](https://k6.io/docs/)
- [k6 Examples](https://k6.io/docs/examples/)
- [PostgreSQL Performance Tuning](https://wiki.postgresql.org/wiki/Performance_Optimization)
- [WebSocket Scalability](https://www.nginx.com/blog/websocket-nginx/)

## 🤝 Contributing

When adding new load tests:

1. Follow existing file naming convention (`k6-<service>.js`)
2. Include detailed comments explaining test scenarios
3. Define clear performance thresholds
4. Use custom metrics for service-specific measurements
5. Update this README with new test documentation

---

**Last Updated**: 2025-10-30
**k6 Version**: 0.47.0+
**Status**: Phase 3 Complete ✅

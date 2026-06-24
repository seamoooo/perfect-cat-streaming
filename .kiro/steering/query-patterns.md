# Query Patterns

## Golden Signals

### 1. Latency (Response Time)

**Percentile Analysis:**
```nrql
SELECT percentile(duration, 50, 95, 99) 
FROM Transaction 
WHERE appName = 'checkout-api' 
SINCE 1 hour ago
```

**Latency by Endpoint:**
```nrql
SELECT average(duration) as 'Avg', percentile(duration, 95) as 'P95'
FROM Transaction 
WHERE appName = 'checkout-api'
FACET name 
SINCE 1 hour ago 
LIMIT 20
```

**Latency Trend:**
```nrql
SELECT percentile(duration, 95) 
FROM Transaction 
WHERE appName = 'checkout-api'
TIMESERIES AUTO 
SINCE 4 hours ago
```

**Database Query Latency:**
```nrql
SELECT average(duration.ms) as 'Avg Duration (ms)', count(*) as 'Count'
FROM Span 
WHERE category = 'datastore' 
AND \`service.name\` = 'checkout-api'
FACET db.statement 
SINCE 1 hour ago 
LIMIT 20
```

---

### 2. Traffic (Throughput)

**Request Rate:**
```nrql
SELECT rate(count(*), 1 minute) as 'Requests/min'
FROM Transaction 
WHERE appName = 'checkout-api'
TIMESERIES AUTO 
SINCE 1 hour ago
```

**Traffic by Endpoint:**
```nrql
SELECT count(*) as 'Total Requests'
FROM Transaction 
WHERE appName = 'checkout-api'
FACET name 
SINCE 1 hour ago 
LIMIT 20
```

**Traffic by HTTP Method:**
```nrql
SELECT count(*) 
FROM Transaction 
WHERE appName = 'checkout-api'
FACET request.method 
SINCE 1 hour ago
```

---

### 3. Errors

**Error Rate:**
```nrql
SELECT percentage(count(*), WHERE error IS true) as 'Error Rate %'
FROM Transaction 
WHERE appName = 'checkout-api'
SINCE 1 hour ago
```

**Error Count Trend:**
```nrql
SELECT count(*) 
FROM Transaction 
WHERE appName = 'checkout-api' AND error IS true
TIMESERIES AUTO 
SINCE 4 hours ago
```

**Errors by Type:**
```nrql
SELECT count(*) 
FROM TransactionError 
WHERE appName = 'checkout-api'
FACET error.class 
SINCE 1 hour ago 
LIMIT 20
```

**Errors by Endpoint:**
```nrql
SELECT count(*) 
FROM Transaction 
WHERE appName = 'checkout-api' AND error IS true
FACET name 
SINCE 1 hour ago 
LIMIT 20
```

---

### 4. Saturation (Resource Utilization)

**CPU Usage:**
```nrql
SELECT average(cpuPercent) as 'Avg CPU %', max(cpuPercent) as 'Max CPU %'
FROM SystemSample 
WHERE hostname LIKE 'prod-%'
FACET hostname 
SINCE 1 hour ago
```

**Memory Usage:**
```nrql
SELECT average(memoryUsedPercent) as 'Avg Memory %', max(memoryUsedPercent) as 'Max Memory %'
FROM SystemSample 
WHERE hostname LIKE 'prod-%'
FACET hostname 
SINCE 1 hour ago
```

**Resource Trends:**
```nrql
SELECT average(cpuPercent), average(memoryUsedPercent)
FROM SystemSample 
WHERE hostname = 'prod-web-01'
TIMESERIES AUTO 
SINCE 4 hours ago
```

**JVM Heap Usage:**
```nrql
SELECT average(newrelic.timeslice.value) * 100 as 'Heap Used %'
FROM Metric 
WHERE metricTimesliceName = 'Memory/Heap/Used' 
AND appName = 'MyJavaApp'
TIMESERIES AUTO 
SINCE 1 hour ago
```

---

## Log Analysis

**Error Logs:**
```nrql
SELECT * 
FROM Log 
WHERE level = 'ERROR' 
AND \`service.name\` = 'checkout-api'
SINCE 1 hour ago 
LIMIT 100
```

**Log Patterns:**
```nrql
SELECT count(*) 
FROM Log 
WHERE level = 'ERROR'
FACET message 
SINCE 1 hour ago 
LIMIT 50
```

**Logs with Context:**
```nrql
SELECT timestamp, message, level, \`service.name\`, hostname
FROM Log 
WHERE message LIKE '%timeout%'
SINCE 1 hour ago 
LIMIT 100
```

**Log Volume by Level:**
```nrql
SELECT count(*) 
FROM Log 
FACET level 
TIMESERIES AUTO 
SINCE 4 hours ago
```

---

## Distributed Tracing

**Find Slow Traces:**
```nrql
SELECT * 
FROM Span 
WHERE duration.ms > 1000 
AND \`service.name\` = 'checkout-api'
SINCE 1 hour ago 
LIMIT 50
```

**Trace Error Analysis:**
```nrql
SELECT count(*) 
FROM Span 
WHERE error IS true 
FACET \`service.name\`, error.message 
SINCE 1 hour ago 
LIMIT 20
```

**Service-to-Service Calls:**
```nrql
SELECT average(duration.ms) as 'Avg Duration', count(*) as 'Count'
FROM Span 
WHERE \`service.name\` = 'frontend' 
AND category = 'http'
FACET \`span.kind\`, \`peer.service\` 
SINCE 1 hour ago
```

**Database Operations:**
```nrql
SELECT average(duration.ms), count(*), max(duration.ms)
FROM Span 
WHERE category = 'datastore' 
FACET db.operation, db.statement 
SINCE 1 hour ago 
LIMIT 20
```

---

## Transaction Analysis

**Slow Transactions:**
```nrql
SELECT average(duration), percentile(duration, 95, 99), count(*)
FROM Transaction 
WHERE appName = 'MyApp' 
AND duration > 1
FACET name 
SINCE 1 hour ago 
LIMIT 20
```

**Transaction by HTTP Status:**
```nrql
SELECT count(*) 
FROM Transaction 
WHERE appName = 'MyApp'
FACET httpResponseCode 
SINCE 1 hour ago
```

**External Service Calls:**
```nrql
SELECT average(externalDuration), count(*)
FROM Transaction 
WHERE appName = 'MyApp' 
AND externalDuration IS NOT NULL
FACET externalHost 
SINCE 1 hour ago
```

**Transaction Errors with Details:**
```nrql
SELECT error.message, error.class, transactionName, count(*)
FROM TransactionError 
WHERE appName = 'MyApp'
FACET error.class, transactionName 
SINCE 1 hour ago 
LIMIT 20
```

---

## Infrastructure Monitoring

**Host Overview:**
```nrql
SELECT average(cpuPercent), average(memoryUsedPercent), average(diskUsedPercent)
FROM SystemSample 
WHERE hostname LIKE 'prod-%'
FACET hostname 
SINCE 1 hour ago
```

**Process CPU Usage:**
```nrql
SELECT average(cpuPercent) as 'Avg CPU', max(cpuPercent) as 'Max CPU'
FROM ProcessSample 
WHERE commandName LIKE '%java%'
FACET hostname, processDisplayName 
SINCE 1 hour ago
```

**Network Traffic:**
```nrql
SELECT average(receiveBytesPerSecond/1024/1024) as 'RX MB/s', 
       average(transmitBytesPerSecond/1024/1024) as 'TX MB/s'
FROM NetworkSample 
FACET hostname 
SINCE 1 hour ago
```

**Disk I/O:**
```nrql
SELECT average(diskReadsPerSecond), average(diskWritesPerSecond)
FROM SystemSample 
WHERE hostname LIKE 'prod-%'
FACET hostname 
TIMESERIES AUTO 
SINCE 1 hour ago
```

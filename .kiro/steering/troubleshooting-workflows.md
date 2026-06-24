# Troubleshooting Workflows

Step-by-step NRQL workflows for the most common production scenarios.

---

## 1. Investigating Production Errors

**Step 1: Check Error Rate**
```nrql
SELECT percentage(count(*), WHERE error IS true) as 'Error Rate %',
       count(*) as 'Total Requests',
       filter(count(*), WHERE error IS true) as 'Errors'
FROM Transaction 
WHERE appName = 'checkout-api'
SINCE 1 hour ago
```

**Step 2: Find Error Types**
```nrql
SELECT count(*) 
FROM TransactionError 
WHERE appName = 'checkout-api'
FACET error.class, error.message 
SINCE 1 hour ago 
LIMIT 20
```

**Step 3: Find Error Logs**
```nrql
SELECT timestamp, message, level, error.class, error.message
FROM Log 
WHERE level = 'ERROR' 
AND \`service.name\` = 'checkout-api'
SINCE 1 hour ago 
LIMIT 100
```

**Step 4: Find Related Traces**
```nrql
SELECT traceId, duration.ms, \`service.name\`, error.message
FROM Span 
WHERE error IS true 
AND \`service.name\` = 'checkout-api'
SINCE 1 hour ago 
LIMIT 20
```

**Step 5: Check for Recent Deployments**
```nrql
SELECT timestamp, revision, user, changelog
FROM Deployment 
WHERE appName = 'checkout-api'
SINCE 24 hours ago
```

---

## 2. Performance Degradation Analysis

**Step 1: Compare Current vs Historical Latency**
```nrql
SELECT percentile(duration, 50, 95, 99) 
FROM Transaction 
WHERE appName = 'api-service'
SINCE 1 hour ago 
COMPARE WITH 1 day ago
```

**Step 2: Find Slow Endpoints**
```nrql
SELECT average(duration) as 'Avg', percentile(duration, 95) as 'P95', count(*)
FROM Transaction 
WHERE appName = 'api-service'
FACET name 
SINCE 1 hour ago 
LIMIT 20
```

**Step 3: Check Database Performance**
```nrql
SELECT average(duration.ms) as 'Avg DB Time', count(*) as 'Query Count'
FROM Span 
WHERE category = 'datastore' 
AND \`service.name\` = 'api-service'
FACET db.operation, db.collection 
SINCE 1 hour ago 
LIMIT 20
```

**Step 4: Check Infrastructure**
```nrql
SELECT average(cpuPercent), average(memoryUsedPercent)
FROM SystemSample 
WHERE hostname LIKE '%api%'
FACET hostname 
TIMESERIES AUTO 
SINCE 4 hours ago
```

**Step 5: Identify External Service Issues**
```nrql
SELECT average(duration.ms), count(*)
FROM Span 
WHERE category = 'http' 
AND \`service.name\` = 'api-service'
FACET \`http.url\`, \`http.status_code\` 
SINCE 1 hour ago 
LIMIT 20
```

---

## 3. Service Dependency Analysis

**Step 1: List Service Calls**
```nrql
SELECT count(*), average(duration.ms)
FROM Span 
WHERE \`service.name\` = 'frontend'
FACET \`peer.service\`, category 
SINCE 1 hour ago
```

**Step 2: Trace Errors Across Services**
```nrql
SELECT count(*) 
FROM Span 
WHERE error IS true
FACET \`service.name\`, \`peer.service\` 
SINCE 1 hour ago
```

**Step 3: External Dependencies**
```nrql
SELECT average(duration.ms), count(*)
FROM Span 
WHERE category = 'http' 
AND \`http.url\` LIKE 'https://api.external.com%'
FACET \`http.url\`, \`http.status_code\` 
SINCE 1 hour ago
```

---

## 4. Alert Investigation

**Step 1: Find Related Metrics**
```nrql
SELECT average(duration), percentile(duration, 95, 99), count(*)
FROM Transaction 
WHERE appName = 'MyApp'
TIMESERIES AUTO 
SINCE 2 hours ago
```

**Step 2: Correlate with Errors**
```nrql
SELECT count(*) 
FROM Transaction 
WHERE appName = 'MyApp' AND error IS true
TIMESERIES AUTO 
SINCE 2 hours ago
```

**Step 3: Check Infrastructure Impact**
```nrql
SELECT average(cpuPercent), average(memoryUsedPercent)
FROM SystemSample 
WHERE hostname IN (SELECT uniques(host) FROM Transaction WHERE appName = 'MyApp')
TIMESERIES AUTO 
SINCE 2 hours ago
```

**Step 4: Find Related Logs**
```nrql
SELECT timestamp, message, level
FROM Log 
WHERE entity.name = 'MyApp'
SINCE 2 hours ago 
LIMIT 200
```

---

## Summary Checklist

Before making New Relic queries, ensure you:
- ✅ Start with narrow time ranges (30 minutes - 1 hour)
- ✅ Always use LIMIT to control result size
- ✅ Filter by appName or service.name to scope queries
- ✅ Query golden signals first for health overview
- ✅ Use FACET to group and identify patterns
- ✅ Include TIMESERIES for trend visualization
- ✅ Use appropriate aggregation functions
- ✅ Check for recent deployments during investigations
- ✅ Leverage entity GUIDs for precise targeting
- ✅ Use WHERE clauses early in query execution

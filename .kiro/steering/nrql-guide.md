# Comprehensive NRQL Guide

## Core Principles

### 1. Start Narrow, Expand if Needed
Always begin with narrow time ranges and specific queries:
- Start: `SINCE 1 hour ago` or `SINCE 30 minutes ago`
- Expand to: `SINCE 4 hours ago` or `SINCE 24 hours ago` if needed
- Narrower ranges = faster queries and more relevant results

### 2. Use Entity-Based Filtering
Always filter by application or service when available:
- `WHERE appName = 'MyApp'`
- `WHERE \`service.name\` = 'checkout-api'`
- `WHERE entity.guid = 'MTIzNDU2...'`

### 3. Always Use LIMIT
Control result size to avoid overwhelming output:
- Default to `LIMIT 100` for most queries
- Use `LIMIT 20-50` for detailed analysis
- Use `LIMIT 500-1000` for comprehensive data gathering

### 4. Query Golden Signals First
Start with the four golden signals for quick health assessment:
- **Latency**: Response time percentiles
- **Traffic**: Request rate
- **Errors**: Error rate and count
- **Saturation**: Resource utilization

---

## Basic Structure

```nrql
SELECT <attributes/functions>
FROM <event_type>
WHERE <conditions>
FACET <grouping>
SINCE <time_range>
LIMIT <count>
```

---

## Event Types Reference

| Event Type | Description | Key Attributes |
|------------|-------------|----------------|
| **Transaction** | APM transactions | appName, name, duration, error |
| **TransactionError** | Application errors | error.class, error.message, transactionName |
| **Span** | Distributed trace spans | traceId, duration.ms, service.name, category |
| **Log** | Log events | message, level, entity.guid, service.name |
| **Metric** | Dimensional metrics | metricName, entity.guid |
| **SystemSample** | Host metrics | cpuPercent, memoryUsedPercent, hostname |
| **ProcessSample** | Process metrics | cpuPercent, memoryResidentSizeBytes |
| **NetworkSample** | Network metrics | receiveBytesPerSecond, transmitBytesPerSecond |
| **Deployment** | Deployment events | appName, revision, user |

---

## SELECT Clause Functions

**Aggregation:**
```nrql
count(*)                              # Count events
average(duration)                     # Average value
sum(duration)                         # Sum values
min(duration), max(duration)          # Min/max values
percentile(duration, 50, 95, 99)     # Percentiles
rate(count(*), 1 minute)             # Events per time unit
uniqueCount(userId)                   # Count distinct values
latest(attribute)                     # Most recent value
```

**String:**
```nrql
concat(firstName, ' ', lastName)      # Concatenate strings
substring(message, 0, 100)           # Extract substring
```

**Math:**
```nrql
abs(value)                           # Absolute value
round(value, 2)                      # Round to decimals
ceil(value), floor(value)            # Round up/down
```

---

## WHERE Clause Patterns

**Equality & Comparison:**
```nrql
WHERE attribute = 'value'             # Exact match
WHERE attribute != 'value'            # Not equal
WHERE duration > 1000                 # Greater than
WHERE duration BETWEEN 100 AND 1000   # Range
```

**String Matching:**
```nrql
WHERE message LIKE '%timeout%'        # Contains
WHERE message LIKE 'Error:%'          # Starts with
WHERE message NOT LIKE '%health%'     # Doesn't contain
```

**Multiple Values:**
```nrql
WHERE status IN ('error', 'warning')  # One of several values
WHERE status NOT IN ('info', 'debug') # Not in list
```

**Boolean & Null:**
```nrql
WHERE error IS true
WHERE attribute IS NULL
WHERE attribute IS NOT NULL
```

**Logical Operators:**
```nrql
WHERE error IS true AND duration > 1000
WHERE status = 'error' OR status = 'warning'
WHERE (status = 'error' OR status = 'warning') AND appName = 'MyApp'
```

---

## FACET Clause

```nrql
FACET appName                         # Group by one attribute
FACET appName, host                   # Group by multiple
FACET appName LIMIT 20                # Limit facets returned

FACET CASES (
  WHERE duration < 100 AS 'Fast',
  WHERE duration < 1000 AS 'Normal',
  WHERE duration >= 1000 AS 'Slow'
)

FACET buckets(duration, 100, 20)     # Create 20 buckets of 100ms each
```

---

## Time Ranges

```nrql
SINCE 30 minutes ago                  # Last 30 minutes
SINCE 1 hour ago                      # Last hour
SINCE 4 hours ago                     # Last 4 hours
SINCE 1 day ago                       # Last 24 hours
SINCE 2 hours ago UNTIL 1 hour ago   # Specific window
SINCE today                           # Since midnight today
SINCE '2024-03-01 00:00:00'          # Absolute timestamp
```

---

## TIMESERIES Clause

```nrql
SELECT count(*) FROM Transaction TIMESERIES AUTO SINCE 1 hour ago
# Auto interval

SELECT average(duration) FROM Transaction TIMESERIES 1 minute SINCE 1 hour ago
# 1-minute buckets

SELECT percentile(duration, 95) FROM Transaction TIMESERIES MAX SINCE 24 hours ago
# Maximum available buckets (366 max)
```

---

## Advanced Patterns

### Comparing Time Periods
```nrql
SELECT average(duration), percentile(duration, 95) 
FROM Transaction 
WHERE appName = 'MyApp'
SINCE 1 hour ago 
COMPARE WITH 1 day ago
```

### Filtering with Subqueries
```nrql
SELECT count(*) 
FROM Transaction 
WHERE appName IN (
  SELECT uniques(appName) FROM Transaction WHERE error IS true
)
SINCE 1 hour ago
```

### Rate of Change
```nrql
SELECT derivative(average(duration), 1 minute) 
FROM Transaction 
WHERE appName = 'MyApp'
TIMESERIES AUTO 
SINCE 1 hour ago
```

### Conditional Aggregation
```nrql
SELECT 
  filter(count(*), WHERE error IS true) as 'Errors',
  filter(count(*), WHERE error IS false) as 'Success',
  percentage(count(*), WHERE error IS true) as 'Error Rate %'
FROM Transaction 
WHERE appName = 'MyApp'
SINCE 1 hour ago
```

### Histogram Buckets
```nrql
SELECT histogram(duration, 100, 20) 
FROM Transaction 
WHERE appName = 'MyApp'
SINCE 1 hour ago
```

---

## Anti-Patterns

### ❌ Query Without Time Range
```nrql
# Bad
SELECT * FROM Transaction WHERE appName = 'MyApp'
# Good
SELECT * FROM Transaction WHERE appName = 'MyApp' SINCE 1 hour ago
```

### ❌ SELECT * Without LIMIT
```nrql
# Bad
SELECT * FROM Log WHERE level = 'ERROR' SINCE 1 day ago
# Good
SELECT * FROM Log WHERE level = 'ERROR' SINCE 1 hour ago LIMIT 100
```

### ❌ No App/Service Filter
```nrql
# Bad
SELECT count(*) FROM Transaction SINCE 1 hour ago
# Good
SELECT count(*) FROM Transaction WHERE appName = 'MyApp' SINCE 1 hour ago
```

### ❌ Vague LIKE Patterns
```nrql
# Bad
SELECT * FROM Log WHERE message LIKE '%error%' SINCE 1 day ago
# Good
SELECT * FROM Log WHERE level = 'ERROR' AND message LIKE '%timeout%' SINCE 1 hour ago LIMIT 100
```

### ❌ Raw Data Over Long Time Ranges
```nrql
# Bad
SELECT * FROM Transaction WHERE appName = 'MyApp' SINCE 30 days ago
# Good
SELECT count(*), average(duration) 
FROM Transaction WHERE appName = 'MyApp' 
FACET dateOf(timestamp) SINCE 30 days ago
```

### ❌ High-Cardinality FACET
```nrql
# Bad
SELECT count(*) FROM Transaction FACET traceId SINCE 1 day ago
# Good
SELECT count(*) FROM Transaction FACET name SINCE 1 hour ago LIMIT 20
```

---

## Performance Tips

- **Filter early**: Use WHERE before FACET
- **Narrow time ranges**: Start with 1 hour, expand only if needed
- **Use LIMIT**: Always limit results
- **Filter by entity**: Use `appName`, `service.name`, or `entity.guid`
- **Avoid SELECT \***: Specify only the attributes you need
- **FACET on low-cardinality attributes**: Avoid high-cardinality fields like trace IDs
- **Use TIMESERIES**: For trend visualization instead of raw data
- **Use COMPARE WITH**: For time period comparisons instead of two separate queries

### Time Range Progression
```
SINCE 30 minutes ago  → Quick check
SINCE 1 hour ago      → Standard investigation
SINCE 4 hours ago     → Pattern analysis
SINCE 24 hours ago    → Historical trends
```

# Metrics
- [loadtest.client.http.result](#loadtestclienthttpresult) : metric recording each client response status


## loadtest.client.http.result

metric recording each client response status



| Prometheus name | Unit | Metric Type | ValueType |
| --------------- |  ---- | ------------ | --------- |
| loadtest_client_http_result_total |  | Counter | int64|

### Attributes

| Name | Prometheus label | Description | Type | Required |
|------| ---------------- |-------------|------| ------- |
| http.error.type | http_error_type | the type of error encountered during an HTTP request | string | ❌ |
| http.status.code | http_status_code | the HTTP response status code | int64 | ✅ |


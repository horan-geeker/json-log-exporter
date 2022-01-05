# json-log-exporter
A Nginx log parser exporter for prometheus metrics.

## Installation

1. go mod tidy
2. go build
3. run binary

## Configuration

config/config.yml

```
- name: app
  format: $remote_addr - $remote_user [$time_local] "$method $request $protocol" $request_time-$upstream_response_time $status $body_bytes_sent "$http_referer" "$http_user_agent" "$http_x_forwarded_for" $request_id
  source_files:
    - ./test/nginx.log
  external_labels:
    region: zone1
  relabel_config:
    source_labels:
      - request
      - method
      - status
    replacement:
      request:
        trim: "?"
        replace:
          - target: /v1.0/example/\d+
            value: /v1.0/example/:id
  histogram_buckets: [0.1, 0.3, 0.5, 1, 2]
  exemplar_config:
    match:
      request_time: ">= 0.3"
    labels:
      - request_id
      - remote_addr
- name: gin
  format: $clientip - [$time_local] "$method $request $protocol $status $upstream_response_time "$http_user_agent" $err"
  source_files:
    - ./test/gin.log
  external_labels:
    region: zone1
  relabel_config:
    source_labels:
      - request
      - method
      - status
    replacement:
      request:
        trim: "?"
  histogram_buckets: [0.1, 0.3, 0.5, 1, 2]
```

- name: service name, metric will be `{name}_http_response_count_total`, `{name}_http_response_count_total`, `{name}_http_response_size_bytes`, `{name}_http_upstream_time_seconds`, `{name}_http_response_time_seconds`
- source_files: service nginx log, support multiple files.
- external_labels: all metrics will add this labelsets.
- relabel_config:
  * source_labels: what's labels should be use.
  * replacement: source labelvalue format rule, it supports regrex, eg `/v1.0/example/123?id=q=xxx` will relace to `/v1.0/example/:id`, it's very powerful. 
- histogram_buckets: configure histogram metrics buckets.
- exemplar_config: configure exemplars, it used for histogram metrics.
 
## Example

`./test/demo.log`

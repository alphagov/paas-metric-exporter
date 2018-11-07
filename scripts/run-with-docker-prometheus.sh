#!/usr/bin/env bash

config_file=$(pwd)/temp-prometheus-config.yml

cat <<EOF > "$config_file"
global:
  scrape_interval: 10s
  scrape_timeout: 10s
  evaluation_interval: 20s

scrape_configs:
- job_name: paas-metric-exporter
  static_configs:
  - targets:
    - 'metric-exporter.leeporte.dev.cloudpipelineapps.digital:443'

EOF
docker run -d -p 9090:9090 -v "$config_file":/etc/prometheus/prometheus.yml prom/prometheus


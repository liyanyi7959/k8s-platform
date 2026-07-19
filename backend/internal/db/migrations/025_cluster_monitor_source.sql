-- 为集群表增加监控数据源相关字段，支持 Prometheus / metrics-server 双数据源适配。
ALTER TABLE clusters
  ADD COLUMN monitor_source VARCHAR(32) NOT NULL DEFAULT 'auto' COMMENT '监控数据源：auto/prometheus/metrics_server',
  ADD COLUMN prometheus_url VARCHAR(500) NOT NULL DEFAULT '' COMMENT '检测到的 Prometheus 访问地址',
  ADD COLUMN prometheus_status VARCHAR(32) NOT NULL DEFAULT 'unknown' COMMENT 'Prometheus 健康状态：unknown/healthy/unhealthy',
  ADD COLUMN prometheus_detected_at DATETIME DEFAULT NULL COMMENT '最近一次 Prometheus 检测时间';

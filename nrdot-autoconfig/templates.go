package autoconfig

import (
	"fmt"
	"strings"

	"github.com/newrelic/nrdot-host/nrdot-discovery"
	"go.uber.org/zap"
)

// TemplateEngine handles template rendering for configurations
type TemplateEngine struct {
	logger *zap.Logger
}

func NewTemplateEngine(logger *zap.Logger) *TemplateEngine {
	return &TemplateEngine{logger: logger}
}

// RenderHostMetrics renders host metrics receiver configuration
func (te *TemplateEngine) RenderHostMetrics() map[string]interface{} {
	return map[string]interface{}{
		"collection_interval": "60s",
		"scrapers": map[string]interface{}{
			"cpu": map[string]interface{}{
				"metrics": map[string]interface{}{
					"system.cpu.utilization": map[string]bool{"enabled": true},
					"system.cpu.load_average.1m": map[string]bool{"enabled": true},
					"system.cpu.load_average.5m": map[string]bool{"enabled": true},
					"system.cpu.load_average.15m": map[string]bool{"enabled": true},
				},
			},
			"memory": map[string]interface{}{
				"metrics": map[string]interface{}{
					"system.memory.utilization": map[string]bool{"enabled": true},
					"system.memory.usage": map[string]interface{}{
						"enabled": true,
						"attributes": map[string]interface{}{
							"state": []string{"used", "free", "cached", "buffered"},
						},
					},
				},
			},
			"disk": map[string]interface{}{
				"metrics": map[string]interface{}{
					"system.disk.operations": map[string]bool{"enabled": true},
					"system.disk.io": map[string]bool{"enabled": true},
					"system.disk.merged": map[string]bool{"enabled": true},
					"system.disk.time": map[string]bool{"enabled": true},
				},
			},
			"filesystem": map[string]interface{}{
				"metrics": map[string]interface{}{
					"system.filesystem.utilization": map[string]bool{"enabled": true},
					"system.filesystem.usage": map[string]bool{"enabled": true},
				},
				"match_type": "strict",
				"mount_points": []string{"/", "/var", "/tmp"},
			},
			"network": map[string]interface{}{
				"metrics": map[string]interface{}{
					"system.network.packets": map[string]bool{"enabled": true},
					"system.network.errors": map[string]bool{"enabled": true},
					"system.network.io": map[string]bool{"enabled": true},
					"system.network.connections": map[string]bool{"enabled": true},
				},
			},
			"load": map[string]interface{}{
				"metrics": map[string]interface{}{
					"system.cpu.load_average.1m": map[string]bool{"enabled": true},
				},
			},
			"processes": map[string]interface{}{
				"metrics": map[string]interface{}{
					"system.processes.running": map[string]bool{"enabled": true},
					"system.processes.blocked": map[string]bool{"enabled": true},
					"system.processes.count": map[string]bool{"enabled": true},
				},
			},
		},
	}
}

// RenderServiceReceiver renders service-specific receiver configuration
func (te *TemplateEngine) RenderServiceReceiver(service discovery.ServiceInfo) (map[string]interface{}, error) {
	switch service.Type {
	case "mysql":
		return te.renderMySQLReceiver(service), nil
	case "postgresql":
		return te.renderPostgreSQLReceiver(service), nil
	case "redis":
		return te.renderRedisReceiver(service), nil
	case "nginx":
		return te.renderNginxReceiver(service), nil
	case "apache":
		return te.renderApacheReceiver(service), nil
	case "mongodb":
		return te.renderMongoDBReceiver(service), nil
	case "elasticsearch":
		return te.renderElasticsearchReceiver(service), nil
	case "rabbitmq":
		return te.renderRabbitMQReceiver(service), nil
	case "memcached":
		return te.renderMemcachedReceiver(service), nil
	case "kafka":
		return te.renderKafkaReceiver(service), nil
	default:
		return nil, fmt.Errorf("unsupported service type: %s", service.Type)
	}
}

// MySQL receiver configuration
func (te *TemplateEngine) renderMySQLReceiver(service discovery.ServiceInfo) map[string]interface{} {
	endpoint := "localhost:3306"
	if len(service.Endpoints) > 0 {
		endpoint = fmt.Sprintf("%s:%d", service.Endpoints[0].Address, service.Endpoints[0].Port)
	}

	return map[string]interface{}{
		"endpoint":             endpoint,
		"collection_interval": "30s",
		"username":            "${MYSQL_MONITOR_USER}",
		"password":            "${MYSQL_MONITOR_PASS}",
		"metrics": map[string]interface{}{
			"mysql.buffer_pool_pages": map[string]bool{"enabled": true},
			"mysql.buffer_pool_data_pages": map[string]bool{"enabled": true},
			"mysql.buffer_pool_page_changes": map[string]bool{"enabled": true},
			"mysql.buffer_pool_limit": map[string]bool{"enabled": true},
			"mysql.buffer_pool_operations": map[string]bool{"enabled": true},
			"mysql.connection.count": map[string]bool{"enabled": true},
			"mysql.connection.errors": map[string]bool{"enabled": true},
			"mysql.statement.latency.count": map[string]bool{"enabled": true},
			"mysql.statement.latency.time": map[string]bool{"enabled": true},
			"mysql.slow_queries": map[string]bool{"enabled": true},
			"mysql.questions": map[string]bool{"enabled": true},
			"mysql.innodb_buffer_pool_pages": map[string]bool{"enabled": true},
			"mysql.innodb_buffer_pool_bytes_data": map[string]bool{"enabled": true},
			"mysql.innodb_buffer_pool_bytes_dirty": map[string]bool{"enabled": true},
			"mysql.innodb_data_reads": map[string]bool{"enabled": true},
			"mysql.innodb_data_writes": map[string]bool{"enabled": true},
			"mysql.replica.lag": map[string]bool{"enabled": true},
			"mysql.replica.sql_delay": map[string]bool{"enabled": true},
		},
	}
}

// PostgreSQL receiver configuration
func (te *TemplateEngine) renderPostgreSQLReceiver(service discovery.ServiceInfo) map[string]interface{} {
	endpoint := "localhost:5432"
	if len(service.Endpoints) > 0 {
		endpoint = fmt.Sprintf("%s:%d", service.Endpoints[0].Address, service.Endpoints[0].Port)
	}

	return map[string]interface{}{
		"endpoint":             endpoint,
		"collection_interval": "30s",
		"username":            "${POSTGRES_MONITOR_USER}",
		"password":            "${POSTGRES_MONITOR_PASS}",
		"databases":           []string{"${POSTGRES_MONITOR_DB:postgres}"},
		"metrics": map[string]interface{}{
			"postgresql.database.count": map[string]bool{"enabled": true},
			"postgresql.db_size": map[string]bool{"enabled": true},
			"postgresql.backends": map[string]bool{"enabled": true},
			"postgresql.connection.max": map[string]bool{"enabled": true},
			"postgresql.table.count": map[string]bool{"enabled": true},
			"postgresql.table.size": map[string]bool{"enabled": true},
			"postgresql.table.vacuum.count": map[string]bool{"enabled": true},
			"postgresql.operations": map[string]bool{"enabled": true},
			"postgresql.blocks_read": map[string]bool{"enabled": true},
			"postgresql.blocks_hit": map[string]bool{"enabled": true},
			"postgresql.temp_files": map[string]bool{"enabled": true},
			"postgresql.replication.lag": map[string]bool{"enabled": true},
			"postgresql.wal.delay": map[string]bool{"enabled": true},
		},
	}
}

// Redis receiver configuration
func (te *TemplateEngine) renderRedisReceiver(service discovery.ServiceInfo) map[string]interface{} {
	endpoint := "localhost:6379"
	if len(service.Endpoints) > 0 {
		endpoint = fmt.Sprintf("%s:%d", service.Endpoints[0].Address, service.Endpoints[0].Port)
	}

	config := map[string]interface{}{
		"endpoint":             endpoint,
		"collection_interval": "30s",
		"metrics": map[string]interface{}{
			"redis.clients.connected": map[string]bool{"enabled": true},
			"redis.clients.blocked": map[string]bool{"enabled": true},
			"redis.commands.processed": map[string]bool{"enabled": true},
			"redis.memory.used": map[string]bool{"enabled": true},
			"redis.memory.peak": map[string]bool{"enabled": true},
			"redis.memory.rss": map[string]bool{"enabled": true},
			"redis.memory.fragmentation_ratio": map[string]bool{"enabled": true},
			"redis.keys.evicted": map[string]bool{"enabled": true},
			"redis.keys.expired": map[string]bool{"enabled": true},
			"redis.connections.received": map[string]bool{"enabled": true},
			"redis.connections.rejected": map[string]bool{"enabled": true},
			"redis.replication.lag": map[string]bool{"enabled": true},
			"redis.rdb.changes_since_last_save": map[string]bool{"enabled": true},
		},
	}

	// Add password if needed
	if requiresAuth := false; requiresAuth { // Simplified check
		config["password"] = "${REDIS_PASSWORD}"
	}

	return config
}

// Nginx receiver configuration
func (te *TemplateEngine) renderNginxReceiver(service discovery.ServiceInfo) map[string]interface{} {
	return map[string]interface{}{
		"endpoint":             "http://localhost:80/nginx_status",
		"collection_interval": "30s",
		"metrics": map[string]interface{}{
			"nginx.connections_accepted": map[string]bool{"enabled": true},
			"nginx.connections_handled": map[string]bool{"enabled": true},
			"nginx.connections_current": map[string]bool{"enabled": true},
			"nginx.connections_reading": map[string]bool{"enabled": true},
			"nginx.connections_writing": map[string]bool{"enabled": true},
			"nginx.connections_waiting": map[string]bool{"enabled": true},
			"nginx.requests": map[string]bool{"enabled": true},
		},
	}
}

// Apache receiver configuration
func (te *TemplateEngine) renderApacheReceiver(service discovery.ServiceInfo) map[string]interface{} {
	return map[string]interface{}{
		"endpoint":             "http://localhost:80/server-status?auto",
		"collection_interval": "30s",
		"metrics": map[string]interface{}{
			"apache.scoreboard": map[string]bool{"enabled": true},
			"apache.connections": map[string]bool{"enabled": true},
			"apache.requests": map[string]bool{"enabled": true},
			"apache.traffic": map[string]bool{"enabled": true},
			"apache.uptime": map[string]bool{"enabled": true},
			"apache.workers": map[string]bool{"enabled": true},
		},
	}
}

// MongoDB receiver configuration
func (te *TemplateEngine) renderMongoDBReceiver(service discovery.ServiceInfo) map[string]interface{} {
	endpoint := "localhost:27017"
	if len(service.Endpoints) > 0 {
		endpoint = fmt.Sprintf("%s:%d", service.Endpoints[0].Address, service.Endpoints[0].Port)
	}

	return map[string]interface{}{
		"hosts": []map[string]interface{}{
			{
				"endpoint": endpoint,
				"username": "${MONGODB_MONITOR_USER}",
				"password": "${MONGODB_MONITOR_PASS}",
			},
		},
		"collection_interval": "30s",
		"metrics": map[string]interface{}{
			"mongodb.database.count": map[string]bool{"enabled": true},
			"mongodb.collection.count": map[string]bool{"enabled": true},
			"mongodb.memory.usage": map[string]bool{"enabled": true},
			"mongodb.connection.count": map[string]bool{"enabled": true},
			"mongodb.operation.count": map[string]bool{"enabled": true},
			"mongodb.operation.latency.time": map[string]bool{"enabled": true},
			"mongodb.document.count": map[string]bool{"enabled": true},
			"mongodb.index.count": map[string]bool{"enabled": true},
		},
	}
}

// Elasticsearch receiver configuration
func (te *TemplateEngine) renderElasticsearchReceiver(service discovery.ServiceInfo) map[string]interface{} {
	endpoint := "http://localhost:9200"
	if len(service.Endpoints) > 0 {
		endpoint = fmt.Sprintf("http://%s:%d", service.Endpoints[0].Address, service.Endpoints[0].Port)
	}

	return map[string]interface{}{
		"endpoints":            []string{endpoint},
		"collection_interval": "30s",
		"username":            "${ELASTICSEARCH_USER}",
		"password":            "${ELASTICSEARCH_PASS}",
		"metrics": map[string]interface{}{
			"elasticsearch.cluster.health": map[string]bool{"enabled": true},
			"elasticsearch.cluster.nodes": map[string]bool{"enabled": true},
			"elasticsearch.cluster.shards": map[string]bool{"enabled": true},
			"elasticsearch.index.documents": map[string]bool{"enabled": true},
			"elasticsearch.index.operations.completed": map[string]bool{"enabled": true},
			"elasticsearch.node.operations.completed": map[string]bool{"enabled": true},
			"elasticsearch.node.cache.memory.usage": map[string]bool{"enabled": true},
			"elasticsearch.node.fs.disk.available": map[string]bool{"enabled": true},
			"elasticsearch.node.jvm.memory.heap.used": map[string]bool{"enabled": true},
		},
	}
}

// RabbitMQ receiver configuration
func (te *TemplateEngine) renderRabbitMQReceiver(service discovery.ServiceInfo) map[string]interface{} {
	return map[string]interface{}{
		"endpoint":             "http://localhost:15672",
		"collection_interval": "30s",
		"username":            "${RABBITMQ_USER}",
		"password":            "${RABBITMQ_PASS}",
		"metrics": map[string]interface{}{
			"rabbitmq.consumer.count": map[string]bool{"enabled": true},
			"rabbitmq.connection.count": map[string]bool{"enabled": true},
			"rabbitmq.message.count": map[string]bool{"enabled": true},
			"rabbitmq.message.delivered": map[string]bool{"enabled": true},
			"rabbitmq.message.published": map[string]bool{"enabled": true},
			"rabbitmq.queue.count": map[string]bool{"enabled": true},
		},
	}
}

// Memcached receiver configuration
func (te *TemplateEngine) renderMemcachedReceiver(service discovery.ServiceInfo) map[string]interface{} {
	endpoint := "localhost:11211"
	if len(service.Endpoints) > 0 {
		endpoint = fmt.Sprintf("%s:%d", service.Endpoints[0].Address, service.Endpoints[0].Port)
	}

	return map[string]interface{}{
		"endpoint":             endpoint,
		"collection_interval": "30s",
		"metrics": map[string]interface{}{
			"memcached.bytes": map[string]bool{"enabled": true},
			"memcached.connections.current": map[string]bool{"enabled": true},
			"memcached.items.current": map[string]bool{"enabled": true},
			"memcached.operations": map[string]bool{"enabled": true},
			"memcached.operation.hit_ratio": map[string]bool{"enabled": true},
			"memcached.evictions": map[string]bool{"enabled": true},
		},
	}
}

// Kafka receiver configuration (using JMX)
func (te *TemplateEngine) renderKafkaReceiver(service discovery.ServiceInfo) map[string]interface{} {
	return map[string]interface{}{
		"protocol_version": "2.0.0",
		"scrapers":         []string{"kafka"},
		"brokers":          []string{"localhost:9092"},
		"collection_interval": "30s",
		"metrics": map[string]interface{}{
			"kafka.brokers": map[string]bool{"enabled": true},
			"kafka.topic.partitions": map[string]bool{"enabled": true},
			"kafka.consumer_group.lag": map[string]bool{"enabled": true},
			"kafka.consumer_group.members": map[string]bool{"enabled": true},
		},
	}
}

// RenderLogReceivers renders log receiver configurations for a service
func (te *TemplateEngine) RenderLogReceivers(service discovery.ServiceInfo) map[string]interface{} {
	configs := make(map[string]interface{})

	switch service.Type {
	case "mysql":
		configs["filelog/mysql_error"] = te.renderMySQLErrorLog()
		configs["filelog/mysql_slow"] = te.renderMySQLSlowLog()
	case "postgresql":
		configs["filelog/postgresql"] = te.renderPostgreSQLLog()
	case "nginx":
		configs["filelog/nginx_access"] = te.renderNginxAccessLog()
		configs["filelog/nginx_error"] = te.renderNginxErrorLog()
	case "apache":
		configs["filelog/apache_access"] = te.renderApacheAccessLog()
		configs["filelog/apache_error"] = te.renderApacheErrorLog()
	case "redis":
		configs["filelog/redis"] = te.renderRedisLog()
	case "mongodb":
		configs["filelog/mongodb"] = te.renderMongoDBLog()
	}

	return configs
}

// MySQL log configurations
func (te *TemplateEngine) renderMySQLErrorLog() map[string]interface{} {
	return map[string]interface{}{
		"include":           []string{"/var/log/mysql/error.log"},
		"start_at":          "end",
		"include_file_path": true,
		"operators": []map[string]interface{}{
			{
				"type":  "regex_parser",
				"regex": `^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z\s+(?P<severity>\w+)\s+(?P<message>.*)`,
			},
			{
				"type":       "severity_parser",
				"parse_from": "attributes.severity",
			},
		},
		"resource": map[string]interface{}{
			"service.name": "mysql",
			"log.type":     "error",
		},
	}
}

func (te *TemplateEngine) renderMySQLSlowLog() map[string]interface{} {
	return map[string]interface{}{
		"include":  []string{"/var/log/mysql/slow.log"},
		"start_at": "end",
		"multiline": map[string]interface{}{
			"line_start_pattern": "^# Time:",
		},
		"operators": []map[string]interface{}{
			{
				"type":  "regex_parser",
				"regex": `^# Query_time: (?P<query_time>[\d.]+)\s+Lock_time: (?P<lock_time>[\d.]+)`,
			},
		},
		"resource": map[string]interface{}{
			"service.name": "mysql",
			"log.type":     "slow_query",
		},
	}
}

// PostgreSQL log configuration
func (te *TemplateEngine) renderPostgreSQLLog() map[string]interface{} {
	return map[string]interface{}{
		"include":  []string{"/var/log/postgresql/postgresql-*.log"},
		"start_at": "end",
		"operators": []map[string]interface{}{
			{
				"type":  "regex_parser",
				"regex": `^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}.*?\[(?P<level>\w+)\]`,
			},
			{
				"type":       "severity_parser",
				"parse_from": "attributes.level",
				"mapping": map[string]interface{}{
					"error": []string{"ERROR", "FATAL", "PANIC"},
					"warn":  []string{"WARNING"},
					"info":  []string{"INFO", "NOTICE"},
					"debug": []string{"DEBUG"},
				},
			},
		},
		"resource": map[string]interface{}{
			"service.name": "postgresql",
		},
	}
}

// Nginx log configurations
func (te *TemplateEngine) renderNginxAccessLog() map[string]interface{} {
	return map[string]interface{}{
		"include":  []string{"/var/log/nginx/access.log"},
		"start_at": "end",
		"operators": []map[string]interface{}{
			{
				"type":  "regex_parser",
				"regex": `^(?P<remote_addr>\S+) - (?P<remote_user>\S+) \[(?P<time_local>[^\]]+)\] "(?P<request>[^"]+)" (?P<status>\d+) (?P<bytes_sent>\d+)`,
			},
			{
				"type":       "time_parser",
				"parse_from": "attributes.time_local",
				"layout":     "%d/%b/%Y:%H:%M:%S %z",
			},
		},
		"resource": map[string]interface{}{
			"service.name": "nginx",
			"log.type":     "access",
		},
	}
}

func (te *TemplateEngine) renderNginxErrorLog() map[string]interface{} {
	return map[string]interface{}{
		"include":  []string{"/var/log/nginx/error.log"},
		"start_at": "end",
		"operators": []map[string]interface{}{
			{
				"type":  "regex_parser",
				"regex": `^\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2} \[(?P<level>\w+)\]`,
			},
			{
				"type":       "severity_parser",
				"parse_from": "attributes.level",
			},
		},
		"resource": map[string]interface{}{
			"service.name": "nginx",
			"log.type":     "error",
		},
	}
}

// Apache log configurations
func (te *TemplateEngine) renderApacheAccessLog() map[string]interface{} {
	return map[string]interface{}{
		"include":  []string{"/var/log/apache2/access.log", "/var/log/httpd/access_log"},
		"start_at": "end",
		"operators": []map[string]interface{}{
			{
				"type":  "regex_parser",
				"regex": `^(?P<remote_addr>\S+) - (?P<remote_user>\S+) \[(?P<time_local>[^\]]+)\] "(?P<request>[^"]+)" (?P<status>\d+) (?P<bytes_sent>\d+)`,
			},
		},
		"resource": map[string]interface{}{
			"service.name": "apache",
			"log.type":     "access",
		},
	}
}

func (te *TemplateEngine) renderApacheErrorLog() map[string]interface{} {
	return map[string]interface{}{
		"include":  []string{"/var/log/apache2/error.log", "/var/log/httpd/error_log"},
		"start_at": "end",
		"operators": []map[string]interface{}{
			{
				"type":  "regex_parser",
				"regex": `^\[(?P<time>[^\]]+)\] \[(?P<level>\w+)\]`,
			},
			{
				"type":       "severity_parser",
				"parse_from": "attributes.level",
			},
		},
		"resource": map[string]interface{}{
			"service.name": "apache",
			"log.type":     "error",
		},
	}
}

// Redis log configuration
func (te *TemplateEngine) renderRedisLog() map[string]interface{} {
	return map[string]interface{}{
		"include":  []string{"/var/log/redis/redis-server.log", "/var/log/redis.log"},
		"start_at": "end",
		"operators": []map[string]interface{}{
			{
				"type":  "regex_parser",
				"regex": `^\d+:\w \d{2} \w{3} \d{4} \d{2}:\d{2}:\d{2}\.\d{3} (?P<level>\W) (?P<message>.*)`,
			},
		},
		"resource": map[string]interface{}{
			"service.name": "redis",
		},
	}
}

// MongoDB log configuration
func (te *TemplateEngine) renderMongoDBLog() map[string]interface{} {
	return map[string]interface{}{
		"include":  []string{"/var/log/mongodb/mongod.log"},
		"start_at": "end",
		"operators": []map[string]interface{}{
			{
				"type":   "json_parser",
				"parse_from": "body",
			},
		},
		"resource": map[string]interface{}{
			"service.name": "mongodb",
		},
	}
}

// RenderSystemLogs renders system log receiver configuration
func (te *TemplateEngine) RenderSystemLogs() map[string]interface{} {
	return map[string]interface{}{
		"include": []string{"/var/log/syslog", "/var/log/messages"},
		"exclude": []string{"/var/log/syslog.*.gz"},
		"start_at": "end",
		"operators": []map[string]interface{}{
			{
				"type":     "syslog_parser",
				"protocol": "rfc3164",
			},
		},
		"resource": map[string]interface{}{
			"log.type": "system",
		},
	}
}

// RenderResourceProcessor renders resource processor configuration
func (te *TemplateEngine) RenderResourceProcessor() map[string]interface{} {
	return map[string]interface{}{
		"attributes": []map[string]interface{}{
			{
				"key":    "service.name",
				"value":  "${HOSTNAME}",
				"action": "insert",
			},
			{
				"key":    "service.environment",
				"value":  "${ENVIRONMENT:production}",
				"action": "insert",
			},
			{
				"key":    "service.version",
				"value":  "2.0.0",
				"action": "insert",
			},
			{
				"key":    "telemetry.sdk.language",
				"value":  "go",
				"action": "insert",
			},
			{
				"key":    "telemetry.sdk.name",
				"value":  "opentelemetry",
				"action": "insert",
			},
		},
	}
}
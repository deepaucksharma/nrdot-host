package process

import (
	"regexp"
	"strings"
)

// ServicePattern defines patterns to identify services by process name
type ServicePattern struct {
	Service     string
	Patterns    []string
	Regex       []*regexp.Regexp
	Confidence  string // HIGH, MEDIUM, LOW
}

// ServiceDetector detects services based on process patterns
type ServiceDetector struct {
	patterns []ServicePattern
}

// NewServiceDetector creates a new service detector with default patterns
func NewServiceDetector() *ServiceDetector {
	patterns := []ServicePattern{
		{
			Service:    "mysql",
			Patterns:   []string{"mysqld", "mariadbd"},
			Confidence: "HIGH",
		},
		{
			Service:    "postgresql",
			Patterns:   []string{"postgres", "postmaster"},
			Confidence: "HIGH",
		},
		{
			Service:    "redis",
			Patterns:   []string{"redis-server"},
			Confidence: "HIGH",
		},
		{
			Service:    "nginx",
			Patterns:   []string{"nginx"},
			Confidence: "HIGH",
		},
		{
			Service:    "apache",
			Patterns:   []string{"httpd", "apache2"},
			Confidence: "HIGH",
		},
		{
			Service:    "mongodb",
			Patterns:   []string{"mongod"},
			Confidence: "HIGH",
		},
		{
			Service:    "elasticsearch",
			Patterns:   []string{"java.*elasticsearch", "elasticsearch"},
			Confidence: "MEDIUM",
		},
		{
			Service:    "rabbitmq",
			Patterns:   []string{"beam.smp.*rabbitmq", "rabbitmq-server"},
			Confidence: "MEDIUM",
		},
		{
			Service:    "memcached",
			Patterns:   []string{"memcached"},
			Confidence: "HIGH",
		},
		{
			Service:    "kafka",
			Patterns:   []string{"java.*kafka", "kafka"},
			Confidence: "MEDIUM",
		},
		{
			Service:    "zookeeper",
			Patterns:   []string{"java.*zookeeper", "zookeeper"},
			Confidence: "MEDIUM",
		},
		{
			Service:    "cassandra",
			Patterns:   []string{"java.*cassandra", "cassandra"},
			Confidence: "MEDIUM",
		},
		{
			Service:    "docker",
			Patterns:   []string{"dockerd", "docker-containerd"},
			Confidence: "HIGH",
		},
		{
			Service:    "kubernetes",
			Patterns:   []string{"kubelet", "kube-apiserver", "kube-controller", "kube-scheduler"},
			Confidence: "HIGH",
		},
	}

	// Compile regex patterns
	for i := range patterns {
		for _, pattern := range patterns[i].Patterns {
			if strings.Contains(pattern, "*") || strings.Contains(pattern, ".") {
				// Convert simple wildcard to regex
				regexPattern := strings.ReplaceAll(pattern, ".", "\\.")
				regexPattern = strings.ReplaceAll(regexPattern, "*", ".*")
				regexPattern = "^" + regexPattern + "$"
				
				if regex, err := regexp.Compile(regexPattern); err == nil {
					patterns[i].Regex = append(patterns[i].Regex, regex)
				}
			}
		}
	}

	return &ServiceDetector{patterns: patterns}
}

// DetectService attempts to identify a service from process info
func (d *ServiceDetector) DetectService(proc *ProcessInfo) (service string, confidence string) {
	// Check process name first
	for _, pattern := range d.patterns {
		// Exact match
		for _, p := range pattern.Patterns {
			if !strings.Contains(p, "*") && !strings.Contains(p, ".") {
				if proc.Name == p {
					return pattern.Service, pattern.Confidence
				}
			}
		}

		// Regex match
		for _, regex := range pattern.Regex {
			if regex.MatchString(proc.Name) {
				return pattern.Service, pattern.Confidence
			}
		}
	}

	// Check command line for better detection
	if proc.Cmdline != "" {
		cmdLower := strings.ToLower(proc.Cmdline)
		
		// Special cases for Java-based services
		if strings.Contains(proc.Name, "java") {
			if strings.Contains(cmdLower, "elasticsearch") {
				return "elasticsearch", "HIGH"
			}
			if strings.Contains(cmdLower, "kafka") && !strings.Contains(cmdLower, "zookeeper") {
				return "kafka", "HIGH"
			}
			if strings.Contains(cmdLower, "zookeeper") {
				return "zookeeper", "HIGH"
			}
			if strings.Contains(cmdLower, "cassandra") {
				return "cassandra", "HIGH"
			}
		}

		// Beam/Erlang services
		if strings.Contains(proc.Name, "beam.smp") && strings.Contains(cmdLower, "rabbitmq") {
			return "rabbitmq", "HIGH"
		}
	}

	return "", ""
}

// GetServiceMetadata returns additional metadata about a detected service
func (d *ServiceDetector) GetServiceMetadata(service string) map[string]interface{} {
	metadata := make(map[string]interface{})

	switch service {
	case "mysql", "mariadb":
		metadata["default_port"] = 3306
		metadata["metrics_endpoint"] = "performance_schema"
		metadata["log_paths"] = []string{"/var/log/mysql/", "/var/log/mariadb/"}
		metadata["config_paths"] = []string{"/etc/mysql/", "/etc/my.cnf"}
		
	case "postgresql":
		metadata["default_port"] = 5432
		metadata["metrics_endpoint"] = "pg_stat_database"
		metadata["log_paths"] = []string{"/var/log/postgresql/"}
		metadata["config_paths"] = []string{"/etc/postgresql/", "/var/lib/pgsql/data/postgresql.conf"}
		
	case "redis":
		metadata["default_port"] = 6379
		metadata["metrics_endpoint"] = "INFO"
		metadata["log_paths"] = []string{"/var/log/redis/"}
		metadata["config_paths"] = []string{"/etc/redis/", "/etc/redis.conf"}
		
	case "nginx":
		metadata["default_ports"] = []int{80, 443}
		metadata["metrics_endpoint"] = "/nginx_status"
		metadata["log_paths"] = []string{"/var/log/nginx/"}
		metadata["config_paths"] = []string{"/etc/nginx/"}
		
	case "apache":
		metadata["default_ports"] = []int{80, 443}
		metadata["metrics_endpoint"] = "/server-status"
		metadata["log_paths"] = []string{"/var/log/apache2/", "/var/log/httpd/"}
		metadata["config_paths"] = []string{"/etc/apache2/", "/etc/httpd/"}
		
	case "mongodb":
		metadata["default_port"] = 27017
		metadata["metrics_endpoint"] = "serverStatus"
		metadata["log_paths"] = []string{"/var/log/mongodb/"}
		metadata["config_paths"] = []string{"/etc/mongod.conf"}
		
	case "elasticsearch":
		metadata["default_port"] = 9200
		metadata["metrics_endpoint"] = "_cluster/stats"
		metadata["log_paths"] = []string{"/var/log/elasticsearch/"}
		metadata["config_paths"] = []string{"/etc/elasticsearch/"}
		
	case "rabbitmq":
		metadata["default_port"] = 5672
		metadata["management_port"] = 15672
		metadata["metrics_endpoint"] = "/api/overview"
		metadata["log_paths"] = []string{"/var/log/rabbitmq/"}
		metadata["config_paths"] = []string{"/etc/rabbitmq/"}
		
	case "memcached":
		metadata["default_port"] = 11211
		metadata["metrics_endpoint"] = "stats"
		metadata["log_paths"] = []string{"/var/log/memcached.log"}
		metadata["config_paths"] = []string{"/etc/memcached.conf"}
		
	case "kafka":
		metadata["default_port"] = 9092
		metadata["jmx_port"] = 9999
		metadata["log_paths"] = []string{"/var/log/kafka/"}
		metadata["config_paths"] = []string{"/etc/kafka/"}
	}

	return metadata
}

// ProcessRelationship represents parent-child process relationships
type ProcessRelationship struct {
	Parent   *ProcessInfo
	Children []*ProcessInfo
}

// BuildProcessTree builds a tree of process relationships
func BuildProcessTree(processes []*ProcessInfo) map[int32]*ProcessRelationship {
	tree := make(map[int32]*ProcessRelationship)
	
	// Create entries for all processes
	for _, proc := range processes {
		if _, exists := tree[proc.PID]; !exists {
			tree[proc.PID] = &ProcessRelationship{Parent: proc}
		} else {
			tree[proc.PID].Parent = proc
		}
	}

	// Build parent-child relationships
	for _, proc := range processes {
		if proc.PPID > 0 {
			if parent, exists := tree[proc.PPID]; exists {
				parent.Children = append(parent.Children, proc)
			}
		}
	}

	return tree
}

// GetServiceProcessGroup finds all related processes for a service
func GetServiceProcessGroup(tree map[int32]*ProcessRelationship, rootPID int32) []*ProcessInfo {
	var group []*ProcessInfo
	visited := make(map[int32]bool)

	var collect func(pid int32)
	collect = func(pid int32) {
		if visited[pid] {
			return
		}
		visited[pid] = true

		if rel, exists := tree[pid]; exists {
			if rel.Parent != nil {
				group = append(group, rel.Parent)
			}
			for _, child := range rel.Children {
				collect(child.PID)
			}
		}
	}

	collect(rootPID)
	return group
}
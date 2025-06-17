package utils

import (
	"os"
	"regexp"
	"strings"
)

var envVarRegex = regexp.MustCompile(`\$\{([A-Z_][A-Z0-9_]*)\}`)

// ExpandEnvVars expands environment variable placeholders in a string
func ExpandEnvVars(s string) string {
	return envVarRegex.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name from ${VAR_NAME}
		varName := strings.TrimSuffix(strings.TrimPrefix(match, "${"), "}")
		
		// Look up the environment variable
		if value, exists := os.LookupEnv(varName); exists {
			return value
		}
		
		// Return original if not found
		return match
	})
}

// ExpandEnvVarsInMap recursively expands environment variables in a map
func ExpandEnvVarsInMap(m map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	for k, v := range m {
		switch val := v.(type) {
		case string:
			result[k] = ExpandEnvVars(val)
		case map[string]interface{}:
			result[k] = ExpandEnvVarsInMap(val)
		case []interface{}:
			result[k] = expandEnvVarsInSlice(val)
		default:
			result[k] = v
		}
	}
	
	return result
}

// expandEnvVarsInSlice expands environment variables in a slice
func expandEnvVarsInSlice(s []interface{}) []interface{} {
	result := make([]interface{}, len(s))
	
	for i, v := range s {
		switch val := v.(type) {
		case string:
			result[i] = ExpandEnvVars(val)
		case map[string]interface{}:
			result[i] = ExpandEnvVarsInMap(val)
		case []interface{}:
			result[i] = expandEnvVarsInSlice(val)
		default:
			result[i] = v
		}
	}
	
	return result
}

// MergeConfigs merges two configuration maps, with the second map taking precedence
func MergeConfigs(base, override map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Copy base configuration
	for k, v := range base {
		result[k] = v
	}
	
	// Apply overrides
	for k, v := range override {
		if baseVal, exists := result[k]; exists {
			// If both are maps, merge recursively
			if baseMap, ok := baseVal.(map[string]interface{}); ok {
				if overrideMap, ok := v.(map[string]interface{}); ok {
					result[k] = MergeConfigs(baseMap, overrideMap)
					continue
				}
			}
		}
		// Otherwise, override takes precedence
		result[k] = v
	}
	
	return result
}

// ValidateEndpoint checks if an endpoint URL is valid
func ValidateEndpoint(endpoint string) bool {
	// Basic validation - should start with http:// or https://
	return strings.HasPrefix(endpoint, "http://") || strings.HasPrefix(endpoint, "https://")
}

// NormalizeEndpoint ensures the endpoint has the correct format
func NormalizeEndpoint(endpoint string) string {
	endpoint = strings.TrimSpace(endpoint)
	endpoint = strings.TrimSuffix(endpoint, "/")
	
	// Add default scheme if missing
	if !strings.Contains(endpoint, "://") {
		endpoint = "https://" + endpoint
	}
	
	return endpoint
}
package utils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpandEnvVars(t *testing.T) {
	// Set test environment variables
	os.Setenv("TEST_VAR", "test-value")
	os.Setenv("ANOTHER_VAR", "another-value")
	defer os.Unsetenv("TEST_VAR")
	defer os.Unsetenv("ANOTHER_VAR")

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "single variable",
			input:    "Value is ${TEST_VAR}",
			expected: "Value is test-value",
		},
		{
			name:     "multiple variables",
			input:    "${TEST_VAR} and ${ANOTHER_VAR}",
			expected: "test-value and another-value",
		},
		{
			name:     "undefined variable",
			input:    "Value is ${UNDEFINED_VAR}",
			expected: "Value is ${UNDEFINED_VAR}",
		},
		{
			name:     "no variables",
			input:    "Just a plain string",
			expected: "Just a plain string",
		},
		{
			name:     "mixed defined and undefined",
			input:    "${TEST_VAR} and ${UNDEFINED}",
			expected: "test-value and ${UNDEFINED}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandEnvVars(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExpandEnvVarsInMap(t *testing.T) {
	os.Setenv("API_KEY", "secret-key")
	os.Setenv("ENDPOINT", "https://example.com")
	defer os.Unsetenv("API_KEY")
	defer os.Unsetenv("ENDPOINT")

	input := map[string]interface{}{
		"api_key":  "${API_KEY}",
		"endpoint": "${ENDPOINT}",
		"nested": map[string]interface{}{
			"value": "${API_KEY}",
			"static": "no-change",
		},
		"array": []interface{}{
			"${API_KEY}",
			"static",
			map[string]interface{}{
				"nested_in_array": "${ENDPOINT}",
			},
		},
		"number": 42,
		"bool":   true,
	}

	result := ExpandEnvVarsInMap(input)

	// Check top-level expansions
	assert.Equal(t, "secret-key", result["api_key"])
	assert.Equal(t, "https://example.com", result["endpoint"])

	// Check nested map
	nested := result["nested"].(map[string]interface{})
	assert.Equal(t, "secret-key", nested["value"])
	assert.Equal(t, "no-change", nested["static"])

	// Check array
	array := result["array"].([]interface{})
	assert.Equal(t, "secret-key", array[0])
	assert.Equal(t, "static", array[1])
	nestedInArray := array[2].(map[string]interface{})
	assert.Equal(t, "https://example.com", nestedInArray["nested_in_array"])

	// Check non-string values remain unchanged
	assert.Equal(t, 42, result["number"])
	assert.Equal(t, true, result["bool"])
}

func TestMergeConfigs(t *testing.T) {
	base := map[string]interface{}{
		"keep": "base-value",
		"override": "base-value",
		"nested": map[string]interface{}{
			"keep": "base-nested",
			"override": "base-nested",
		},
		"array": []string{"base1", "base2"},
	}

	override := map[string]interface{}{
		"override": "override-value",
		"new": "new-value",
		"nested": map[string]interface{}{
			"override": "override-nested",
			"new": "new-nested",
		},
		"array": []string{"override1"},
	}

	result := MergeConfigs(base, override)

	// Check merged values
	assert.Equal(t, "base-value", result["keep"])
	assert.Equal(t, "override-value", result["override"])
	assert.Equal(t, "new-value", result["new"])

	// Check nested merge
	nested := result["nested"].(map[string]interface{})
	assert.Equal(t, "base-nested", nested["keep"])
	assert.Equal(t, "override-nested", nested["override"])
	assert.Equal(t, "new-nested", nested["new"])

	// Arrays are replaced, not merged
	array := result["array"].([]string)
	assert.Equal(t, []string{"override1"}, array)
}

func TestValidateEndpoint(t *testing.T) {
	tests := []struct {
		endpoint string
		valid    bool
	}{
		{"https://example.com", true},
		{"http://localhost:8080", true},
		{"https://otlp.nr-data.net", true},
		{"example.com", false},
		{"ftp://example.com", false},
		{"", false},
		{"https://", true}, // Technically valid format
	}

	for _, tt := range tests {
		t.Run(tt.endpoint, func(t *testing.T) {
			assert.Equal(t, tt.valid, ValidateEndpoint(tt.endpoint))
		})
	}
}

func TestNormalizeEndpoint(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"https://example.com", "https://example.com"},
		{"https://example.com/", "https://example.com"},
		{"example.com", "https://example.com"},
		{"  https://example.com  ", "https://example.com"},
		{"http://localhost:8080/", "http://localhost:8080"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizeEndpoint(tt.input))
		})
	}
}
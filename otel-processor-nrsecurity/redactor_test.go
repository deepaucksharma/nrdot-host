package nrsecurity

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestNewRedactor(t *testing.T) {
	cfg := &Config{
		Enabled:         true,
		ReplacementText: "[REDACTED]",
		Keywords:        []string{"password", "secret"},
		AllowList:       []string{"service.name"},
		DenyList:        []string{"db.connection_string"},
		Patterns: []PatternConfig{
			{Name: "test", Regex: "TEST-[0-9]+"},
		},
	}

	r, err := NewRedactor(cfg)
	require.NoError(t, err)
	assert.NotNil(t, r)
	assert.Len(t, r.keywordSet, 2)
	assert.Len(t, r.allowSet, 1)
	assert.Len(t, r.denySet, 1)
}

func TestRedactorInvalidPattern(t *testing.T) {
	cfg := &Config{
		Enabled:         true,
		ReplacementText: "[REDACTED]",
		Patterns: []PatternConfig{
			{Name: "invalid", Regex: "["},
		},
	}

	_, err := NewRedactor(cfg)
	assert.Error(t, err)
}

func TestRedactAttributes(t *testing.T) {
	cfg := &Config{
		Enabled:         true,
		ReplacementText: "[REDACTED]",
		Keywords:        []string{"password", "secret"},
		AllowList:       []string{"service.name"},
		DenyList:        []string{"db.connection_string"},
	}

	r, err := NewRedactor(cfg)
	require.NoError(t, err)

	attrs := pcommon.NewMap()
	attrs.PutStr("service.name", "test-service")
	attrs.PutStr("password", "secret123")
	attrs.PutStr("db.connection_string", "postgres://user:pass@localhost")
	attrs.PutStr("api_key", "AKIA1234567890ABCDEF")
	attrs.PutStr("normal_field", "normal value")

	r.RedactAttributes(attrs)

	// Check results
	val, _ := attrs.Get("service.name")
	assert.Equal(t, "test-service", val.Str()) // Allow list

	val, _ = attrs.Get("password")
	assert.Equal(t, "[REDACTED]", val.Str()) // Keyword match

	val, _ = attrs.Get("db.connection_string")
	assert.Equal(t, "[REDACTED]", val.Str()) // Deny list

	val, _ = attrs.Get("api_key")
	assert.Equal(t, "[REDACTED]", val.Str()) // Pattern match

	val, _ = attrs.Get("normal_field")
	assert.Equal(t, "normal value", val.Str()) // No redaction
}

func TestRedactNestedAttributes(t *testing.T) {
	cfg := &Config{
		Enabled:         true,
		ReplacementText: "[REDACTED]",
		Keywords:        []string{"password"},
	}

	r, err := NewRedactor(cfg)
	require.NoError(t, err)

	attrs := pcommon.NewMap()
	userMap := attrs.PutEmptyMap("user")
	userMap.PutStr("name", "John")
	userMap.PutStr("password", "secret123")
	
	configMap := attrs.PutEmptyMap("config")
	dbMap := configMap.PutEmptyMap("database")
	dbMap.PutStr("host", "localhost")
	dbMap.PutStr("password", "dbpass123")

	r.RedactAttributes(attrs)

	// Check nested redaction
	user, _ := attrs.Get("user")
	userAttrs := user.Map()
	
	val, _ := userAttrs.Get("name")
	assert.Equal(t, "John", val.Str())
	
	val, _ = userAttrs.Get("password")
	assert.Equal(t, "[REDACTED]", val.Str())

	config, _ := attrs.Get("config")
	db, _ := config.Map().Get("database")
	dbAttrs := db.Map()
	
	val, _ = dbAttrs.Get("host")
	assert.Equal(t, "localhost", val.Str())
	
	val, _ = dbAttrs.Get("password")
	assert.Equal(t, "[REDACTED]", val.Str())
}

func TestRedactSliceAttributes(t *testing.T) {
	cfg := &Config{
		Enabled:         true,
		ReplacementText: "[REDACTED]",
		Keywords:        []string{"token"},
	}

	r, err := NewRedactor(cfg)
	require.NoError(t, err)

	attrs := pcommon.NewMap()
	slice := attrs.PutEmptySlice("tokens")
	slice.AppendEmpty().SetStr("normal-value")
	slice.AppendEmpty().SetStr("Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U")

	r.RedactAttributes(attrs)

	// Check slice redaction
	val, _ := attrs.Get("tokens")
	sliceVal := val.Slice()
	
	// All values in a "tokens" slice should be redacted due to keyword match
	assert.Equal(t, "[REDACTED]", sliceVal.At(0).Str())
	assert.Equal(t, "[REDACTED]", sliceVal.At(1).Str())
}

func TestContainsKeyword(t *testing.T) {
	cfg := &Config{
		Enabled:         true,
		ReplacementText: "[REDACTED]",
		Keywords:        []string{"password", "secret", "token"},
	}

	r, err := NewRedactor(cfg)
	require.NoError(t, err)

	tests := []struct {
		key      string
		expected bool
	}{
		{"password", true},
		{"user_password", true},
		{"PASSWORD", true},
		{"db.password", true},
		{"secret_key", true},
		{"api_token", true},
		{"username", false},
		{"service.name", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			result := r.containsKeyword(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldRedactString(t *testing.T) {
	cfg := &Config{
		Enabled:         true,
		ReplacementText: "[REDACTED]",
	}

	r, err := NewRedactor(cfg)
	require.NoError(t, err)

	tests := []struct {
		input    string
		expected bool
	}{
		{"", false},
		{"normal text", false},
		{"AKIA1234567890ABCDEF", true}, // AWS key
		{"4111111111111111", true}, // Credit card
		{"123-45-6789", true}, // SSN
		{"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U", true}, // JWT
		{"https://user:password@example.com", true}, // Password in URL
		{"password=secret123", true}, // Password assignment
	}

	for _, tt := range tests {
		t.Run(tt.input[:min(20, len(tt.input))], func(t *testing.T) {
			result := r.shouldRedactString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactString(t *testing.T) {
	cfg := &Config{
		Enabled:         true,
		ReplacementText: "[REDACTED]",
	}

	r, err := NewRedactor(cfg)
	require.NoError(t, err)

	tests := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"normal text", "normal text"},
		{"API Key: AKIA1234567890ABCDEF", "API Key: [REDACTED]"},
		{"Credit card: 4111111111111111", "Credit card: [REDACTED]"},
		{"SSN: 123-45-6789", "SSN: [REDACTED]"},
		{"postgres://user:pass123@localhost:5432/db", "[REDACTED]"},
		{"Multiple secrets: password=secret123 and token=abc123", "Multiple secrets: [REDACTED] and token=abc123"},
	}

	for _, tt := range tests {
		t.Run(tt.input[:min(20, len(tt.input))], func(t *testing.T) {
			result := r.RedactString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactBytesAttribute(t *testing.T) {
	cfg := &Config{
		Enabled:         true,
		ReplacementText: "[REDACTED]",
		DenyList:        []string{"binary_secret"},
	}

	r, err := NewRedactor(cfg)
	require.NoError(t, err)

	attrs := pcommon.NewMap()
	attrs.PutEmptyBytes("binary_secret").FromRaw([]byte("secret binary data"))

	r.RedactAttributes(attrs)

	val, _ := attrs.Get("binary_secret")
	assert.Equal(t, []byte("[REDACTED]"), val.Bytes().AsRaw())
}

func TestRedactWithCustomReplacement(t *testing.T) {
	cfg := &Config{
		Enabled:         true,
		ReplacementText: "***HIDDEN***",
		Keywords:        []string{"secret"},
	}

	r, err := NewRedactor(cfg)
	require.NoError(t, err)

	attrs := pcommon.NewMap()
	attrs.PutStr("secret_key", "my-secret-value")

	r.RedactAttributes(attrs)

	val, _ := attrs.Get("secret_key")
	assert.Equal(t, "***HIDDEN***", val.Str())
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
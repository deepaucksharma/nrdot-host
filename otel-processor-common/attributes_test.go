package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
)

func TestAttributeProcessor(t *testing.T) {
	t.Run("redact patterns", func(t *testing.T) {
		ap := NewAttributeProcessor()
		
		// Add common sensitive patterns
		for _, pattern := range []string{
			`(?i)(password[\s]*[:=][\s]*)\S+`,
			`(?i)(api[_-]?key[\s]*[:=][\s]*)\S+`,
		} {
			err := ap.AddRedactPattern(pattern)
			require.NoError(t, err)
		}
		
		// Test redaction
		attrs := pcommon.NewMap()
		attrs.PutStr("config", "password: secret123")
		attrs.PutStr("headers", "API-Key: abc123xyz")
		attrs.PutStr("safe", "this is safe data")
		
		ap.ProcessAttributes(attrs)
		
		configVal, _ := attrs.Get("config")
		assert.Equal(t, "password: [REDACTED]", configVal.Str())
		
		headersVal, _ := attrs.Get("headers")
		assert.Equal(t, "API-Key: [REDACTED]", headersVal.Str())
		
		safeVal, _ := attrs.Get("safe")
		assert.Equal(t, "this is safe data", safeVal.Str())
	})
	
	t.Run("allowed keys filtering", func(t *testing.T) {
		ap := NewAttributeProcessor()
		ap.SetAllowedKeys([]string{"keep1", "keep2"})
		
		attrs := pcommon.NewMap()
		attrs.PutStr("keep1", "value1")
		attrs.PutStr("keep2", "value2")
		attrs.PutStr("remove1", "value3")
		attrs.PutStr("remove2", "value4")
		
		ap.ProcessAttributes(attrs)
		
		assert.Equal(t, 2, attrs.Len())
		_, exists := attrs.Get("keep1")
		assert.True(t, exists)
		_, exists = attrs.Get("keep2")
		assert.True(t, exists)
		_, exists = attrs.Get("remove1")
		assert.False(t, exists)
	})
	
	t.Run("blocked keys filtering", func(t *testing.T) {
		ap := NewAttributeProcessor()
		ap.SetBlockedKeys([]string{"block1", "block2"})
		
		attrs := pcommon.NewMap()
		attrs.PutStr("keep1", "value1")
		attrs.PutStr("keep2", "value2")
		attrs.PutStr("block1", "value3")
		attrs.PutStr("block2", "value4")
		
		ap.ProcessAttributes(attrs)
		
		assert.Equal(t, 2, attrs.Len())
		_, exists := attrs.Get("keep1")
		assert.True(t, exists)
		_, exists = attrs.Get("keep2")
		assert.True(t, exists)
		_, exists = attrs.Get("block1")
		assert.False(t, exists)
	})
}

func TestAttributeEnricher(t *testing.T) {
	t.Run("static attributes", func(t *testing.T) {
		ae := NewAttributeEnricher()
		ae.AddStaticAttribute("host", "server1")
		ae.AddStaticAttribute("environment", "production")
		ae.AddStaticAttribute("version", 123)
		
		attrs := pcommon.NewMap()
		attrs.PutStr("existing", "value")
		
		ae.EnrichAttributes(attrs)
		
		assert.Equal(t, 4, attrs.Len())
		
		hostVal, _ := attrs.Get("host")
		assert.Equal(t, "server1", hostVal.Str())
		
		envVal, _ := attrs.Get("environment")
		assert.Equal(t, "production", envVal.Str())
		
		versionVal, _ := attrs.Get("version")
		assert.Equal(t, int64(123), versionVal.Int())
		
		existingVal, _ := attrs.Get("existing")
		assert.Equal(t, "value", existingVal.Str())
	})
	
	t.Run("conditional rules", func(t *testing.T) {
		ae := NewAttributeEnricher()
		
		// Add rule: if service.name exists, add service.tier
		tierValue := pcommon.NewValueStr("backend")
		ae.AddConditionalRule(ConditionalRule{
			Condition: func(attrs pcommon.Map) bool {
				_, exists := attrs.Get("service.name")
				return exists
			},
			Key:   "service.tier",
			Value: tierValue,
		})
		
		// Test with matching condition
		attrs1 := pcommon.NewMap()
		attrs1.PutStr("service.name", "api-service")
		ae.EnrichAttributes(attrs1)
		
		tierVal, exists := attrs1.Get("service.tier")
		assert.True(t, exists)
		assert.Equal(t, "backend", tierVal.Str())
		
		// Test without matching condition
		attrs2 := pcommon.NewMap()
		attrs2.PutStr("other", "value")
		ae.EnrichAttributes(attrs2)
		
		_, exists = attrs2.Get("service.tier")
		assert.False(t, exists)
	})
}

func TestNormalizeAttributeKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Host Name", "host.name"},
		{"SERVICE_NAME", "service.name"},
		{"app-version", "app.version"},
		{"metric__name__", "metric.name"},
		{"..dots..", "dots"},
		{"UPPER_CASE_KEY", "upper.case.key"},
		{"mixed-Style_Key", "mixed.style.key"},
	}
	
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizeAttributeKey(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCopyAndMergeAttributes(t *testing.T) {
	t.Run("copy attributes", func(t *testing.T) {
		src := pcommon.NewMap()
		src.PutStr("key1", "value1")
		src.PutInt("key2", 42)
		
		dst := pcommon.NewMap()
		dst.PutStr("existing", "value")
		
		CopyAttributes(src, dst)
		
		assert.Equal(t, 3, dst.Len())
		val1, _ := dst.Get("key1")
		assert.Equal(t, "value1", val1.Str())
		val2, _ := dst.Get("key2")
		assert.Equal(t, int64(42), val2.Int())
	})
	
	t.Run("merge attributes", func(t *testing.T) {
		src := pcommon.NewMap()
		src.PutStr("key1", "new_value")
		src.PutStr("key2", "value2")
		
		dst := pcommon.NewMap()
		dst.PutStr("key1", "existing_value")
		dst.PutStr("key3", "value3")
		
		MergeAttributes(src, dst)
		
		assert.Equal(t, 3, dst.Len())
		
		// Existing key should not be overwritten
		val1, _ := dst.Get("key1")
		assert.Equal(t, "existing_value", val1.Str())
		
		// New key should be added
		val2, _ := dst.Get("key2")
		assert.Equal(t, "value2", val2.Str())
		
		// Original dst key should remain
		val3, _ := dst.Get("key3")
		assert.Equal(t, "value3", val3.Str())
	})
}
package common

import (
	"regexp"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// AttributeProcessor provides utilities for attribute manipulation
type AttributeProcessor struct {
	redactPatterns []*regexp.Regexp
	allowedKeys    map[string]bool
	blockedKeys    map[string]bool
}

// NewAttributeProcessor creates a new attribute processor
func NewAttributeProcessor() *AttributeProcessor {
	return &AttributeProcessor{
		redactPatterns: make([]*regexp.Regexp, 0),
		allowedKeys:    make(map[string]bool),
		blockedKeys:    make(map[string]bool),
	}
}

// AddRedactPattern adds a pattern for value redaction
func (ap *AttributeProcessor) AddRedactPattern(pattern string) error {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}
	ap.redactPatterns = append(ap.redactPatterns, re)
	return nil
}

// SetAllowedKeys sets the list of allowed attribute keys
func (ap *AttributeProcessor) SetAllowedKeys(keys []string) {
	ap.allowedKeys = make(map[string]bool)
	for _, key := range keys {
		ap.allowedKeys[key] = true
	}
}

// SetBlockedKeys sets the list of blocked attribute keys
func (ap *AttributeProcessor) SetBlockedKeys(keys []string) {
	ap.blockedKeys = make(map[string]bool)
	for _, key := range keys {
		ap.blockedKeys[key] = true
	}
}

// ProcessAttributes applies all attribute processing rules
func (ap *AttributeProcessor) ProcessAttributes(attrs pcommon.Map) {
	// First, handle blocked/allowed keys
	attrs.RemoveIf(func(key string, _ pcommon.Value) bool {
		// If we have an allowlist, only keep allowed keys
		if len(ap.allowedKeys) > 0 {
			return !ap.allowedKeys[key]
		}
		
		// Otherwise, remove blocked keys
		return ap.blockedKeys[key]
	})
	
	// Then, apply redaction patterns
	attrs.Range(func(key string, value pcommon.Value) bool {
		if value.Type() == pcommon.ValueTypeStr {
			redacted := ap.redactValue(value.Str())
			if redacted != value.Str() {
				value.SetStr(redacted)
			}
		}
		return true
	})
}

// redactValue applies redaction patterns to a string value
func (ap *AttributeProcessor) redactValue(value string) string {
	for _, pattern := range ap.redactPatterns {
		value = pattern.ReplaceAllString(value, "${1}[REDACTED]")
	}
	return value
}

// AttributeEnricher adds attributes to data
type AttributeEnricher struct {
	staticAttributes map[string]pcommon.Value
	conditionalRules []ConditionalRule
}

// ConditionalRule defines a rule for conditional attribute addition
type ConditionalRule struct {
	Condition func(pcommon.Map) bool
	Key       string
	Value     pcommon.Value
}

// NewAttributeEnricher creates a new attribute enricher
func NewAttributeEnricher() *AttributeEnricher {
	return &AttributeEnricher{
		staticAttributes: make(map[string]pcommon.Value),
		conditionalRules: make([]ConditionalRule, 0),
	}
}

// AddStaticAttribute adds a static attribute
func (ae *AttributeEnricher) AddStaticAttribute(key string, value interface{}) {
	val := pcommon.NewValueEmpty()
	switch v := value.(type) {
	case string:
		val.SetStr(v)
	case int:
		val.SetInt(int64(v))
	case int64:
		val.SetInt(v)
	case float64:
		val.SetDouble(v)
	case bool:
		val.SetBool(v)
	}
	ae.staticAttributes[key] = val
}

// AddConditionalRule adds a conditional attribute rule
func (ae *AttributeEnricher) AddConditionalRule(rule ConditionalRule) {
	ae.conditionalRules = append(ae.conditionalRules, rule)
}

// EnrichAttributes adds configured attributes
func (ae *AttributeEnricher) EnrichAttributes(attrs pcommon.Map) {
	// Add static attributes
	for key, value := range ae.staticAttributes {
		value.CopyTo(attrs.PutEmpty(key))
	}
	
	// Apply conditional rules
	for _, rule := range ae.conditionalRules {
		if rule.Condition(attrs) {
			rule.Value.CopyTo(attrs.PutEmpty(rule.Key))
		}
	}
}

// CommonAttributeKeys defines standard attribute keys used across NRDOT
const (
	AttributeKeyHost            = "host.name"
	AttributeKeyHostID          = "host.id"
	AttributeKeyService         = "service.name"
	AttributeKeyEnvironment     = "deployment.environment"
	AttributeKeyRegion          = "cloud.region"
	AttributeKeyAccountID       = "cloud.account.id"
	AttributeKeyNREntityGUID    = "newrelic.entity.guid"
	AttributeKeyNRAccountID     = "newrelic.account.id"
	AttributeKeyProcessorName   = "nrdot.processor.name"
	AttributeKeyProcessorAction = "nrdot.processor.action"
)

// SensitivePatterns defines common patterns for sensitive data
var SensitivePatterns = []string{
	`(?i)(password[\s]*[:=][\s]*)\S+`,
	`(?i)(api[_-]?key[\s]*[:=][\s]*)\S+`,
	`(?i)(token[\s]*[:=][\s]*)\S+`,
	`(?i)(secret[\s]*[:=][\s]*)\S+`,
	`(?i)(bearer\s+)\S+`,
	`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`, // Email
	`\b(?:\d{4}[-\s]?){3}\d{4}\b`,                          // Credit card
	`\b\d{3}-\d{2}-\d{4}\b`,                                 // SSN
}

// NormalizeAttributeKey normalizes attribute keys to a standard format
func NormalizeAttributeKey(key string) string {
	// Convert to lowercase
	key = strings.ToLower(key)
	
	// Replace spaces and special characters with dots
	key = strings.ReplaceAll(key, " ", ".")
	key = strings.ReplaceAll(key, "-", ".")
	key = strings.ReplaceAll(key, "_", ".")
	
	// Remove duplicate dots
	for strings.Contains(key, "..") {
		key = strings.ReplaceAll(key, "..", ".")
	}
	
	// Trim dots from ends
	key = strings.Trim(key, ".")
	
	return key
}

// CopyAttributes copies attributes from source to destination
func CopyAttributes(src, dst pcommon.Map) {
	src.Range(func(k string, v pcommon.Value) bool {
		v.CopyTo(dst.PutEmpty(k))
		return true
	})
}

// MergeAttributes merges source attributes into destination (source takes precedence)
func MergeAttributes(src, dst pcommon.Map) {
	src.Range(func(k string, v pcommon.Value) bool {
		// Only copy if not already present in destination
		if _, ok := dst.Get(k); !ok {
			v.CopyTo(dst.PutEmpty(k))
		}
		return true
	})
}
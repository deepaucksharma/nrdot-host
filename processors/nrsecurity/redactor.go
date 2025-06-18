package nrsecurity

import (
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
)

// Redactor handles the redaction of sensitive data
type Redactor struct {
	config         *Config
	patternManager *PatternManager
	keywordSet     map[string]bool
	allowSet       map[string]bool
	denySet        map[string]bool
}

// NewRedactor creates a new redactor instance
func NewRedactor(cfg *Config) (*Redactor, error) {
	r := &Redactor{
		config:         cfg,
		patternManager: NewPatternManager(),
		keywordSet:     make(map[string]bool),
		allowSet:       make(map[string]bool),
		denySet:        make(map[string]bool),
	}

	// Initialize sets
	for _, keyword := range cfg.Keywords {
		r.keywordSet[strings.ToLower(keyword)] = true
	}

	for _, allow := range cfg.AllowList {
		r.allowSet[allow] = true
	}

	for _, deny := range cfg.DenyList {
		r.denySet[deny] = true
	}

	// Add custom patterns
	for _, pattern := range cfg.Patterns {
		if err := r.patternManager.AddPattern(pattern.Name, pattern.Regex); err != nil {
			return nil, err
		}
	}

	// Add optional patterns
	if cfg.RedactEmails {
		r.patternManager.AddEmailPattern()
	}

	if cfg.RedactIPs {
		r.patternManager.AddIPPattern()
	}

	return r, nil
}

// RedactAttributes processes attributes and redacts sensitive values
func (r *Redactor) RedactAttributes(attrs pcommon.Map) {
	attrs.Range(func(k string, v pcommon.Value) bool {
		r.redactValue(k, v)
		return true
	})
}

// redactValue handles redaction of a single value
func (r *Redactor) redactValue(key string, value pcommon.Value) {
	// Check allow list first
	if r.allowSet[key] {
		return
	}

	// Check deny list - always redact
	if r.denySet[key] {
		r.redactValueContent(value)
		return
	}

	// Check if key contains sensitive keywords
	if r.containsKeyword(key) {
		r.redactValueContent(value)
		return
	}

	// For non-denied attributes, check the value content
	switch value.Type() {
	case pcommon.ValueTypeStr:
		if r.shouldRedactString(value.Str()) {
			value.SetStr(r.config.ReplacementText)
		}
	case pcommon.ValueTypeMap:
		// Recursively process nested attributes
		r.RedactAttributes(value.Map())
	case pcommon.ValueTypeSlice:
		// Process each element in the slice
		slice := value.Slice()
		for i := 0; i < slice.Len(); i++ {
			elem := slice.At(i)
			r.redactValue(key, elem)
		}
	}
}

// redactValueContent unconditionally redacts the value content
func (r *Redactor) redactValueContent(value pcommon.Value) {
	switch value.Type() {
	case pcommon.ValueTypeStr:
		value.SetStr(r.config.ReplacementText)
	case pcommon.ValueTypeMap:
		// For maps, redact all string values
		value.Map().Range(func(k string, v pcommon.Value) bool {
			r.redactValueContent(v)
			return true
		})
	case pcommon.ValueTypeSlice:
		// For slices, redact all elements
		slice := value.Slice()
		for i := 0; i < slice.Len(); i++ {
			r.redactValueContent(slice.At(i))
		}
	case pcommon.ValueTypeBytes:
		// Redact byte arrays
		value.SetEmptyBytes().FromRaw([]byte(r.config.ReplacementText))
	}
	// Numbers and booleans are left as-is
}

// containsKeyword checks if the key contains any sensitive keywords
func (r *Redactor) containsKeyword(key string) bool {
	lowerKey := strings.ToLower(key)
	for keyword := range r.keywordSet {
		if strings.Contains(lowerKey, keyword) {
			return true
		}
	}
	return false
}

// shouldRedactString checks if a string value should be redacted based on patterns
func (r *Redactor) shouldRedactString(value string) bool {
	if value == "" {
		return false
	}
	return r.patternManager.MatchesAny(value)
}

// RedactString applies pattern-based redaction to a string
func (r *Redactor) RedactString(value string) string {
	if value == "" {
		return value
	}
	return r.patternManager.RedactAll(value, r.config.ReplacementText)
}
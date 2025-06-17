package nrsecurity

import (
	"regexp"
	"sync"
)

// RedactionPattern represents a compiled regex pattern for redaction
type RedactionPattern struct {
	Name    string
	Pattern *regexp.Regexp
}

// PatternManager manages redaction patterns with caching
type PatternManager struct {
	patterns []RedactionPattern
	mu       sync.RWMutex
}

// NewPatternManager creates a new pattern manager with default patterns
func NewPatternManager() *PatternManager {
	pm := &PatternManager{
		patterns: make([]RedactionPattern, 0),
	}
	pm.loadDefaultPatterns()
	return pm
}

// loadDefaultPatterns loads the default redaction patterns
func (pm *PatternManager) loadDefaultPatterns() {
	defaultPatterns := []struct {
		name    string
		pattern string
	}{
		// API Keys
		{
			name:    "generic_api_key",
			pattern: `(?i)(api[_-]?key|apikey)\s*[:=]\s*['"]?[a-zA-Z0-9_\-]{8,}['"]?`,
		},
		{
			name:    "aws_access_key",
			pattern: `AKIA[0-9A-Z]{16}`,
		},
		{
			name:    "aws_secret_key",
			pattern: `(?i)aws[_-]?secret[_-]?access[_-]?key["\s]*[:=]["\s]*([a-zA-Z0-9/+=]{40})`,
		},
		// Passwords
		{
			name:    "password_in_url",
			pattern: `(?i)(https?|ftp)://[^:]+:([^@]+)@`,
		},
		{
			name:    "password_assignment",
			pattern: `(?i)(password|passwd|pwd)["\s]*[:=]["\s]*([^"\s,;]+)`,
		},
		// Tokens
		{
			name:    "jwt_token",
			pattern: `eyJ[a-zA-Z0-9_-]*\.eyJ[a-zA-Z0-9_-]*\.[a-zA-Z0-9_-]*`,
		},
		{
			name:    "bearer_token",
			pattern: `(?i)bearer\s+([a-zA-Z0-9_\-\.]+)`,
		},
		{
			name:    "github_token",
			pattern: `gh[ps]?_[a-zA-Z0-9]{36,}`,
		},
		{
			name:    "slack_token",
			pattern: `xox[baprs]-[0-9]{10,13}-[0-9]{10,13}-[a-zA-Z0-9]{20,}`,
		},
		// Credit Cards
		{
			name:    "visa",
			pattern: `4[0-9]{12}(?:[0-9]{3})?`,
		},
		{
			name:    "mastercard",
			pattern: `5[1-5][0-9]{14}`,
		},
		{
			name:    "amex",
			pattern: `3[47][0-9]{13}`,
		},
		{
			name:    "discover",
			pattern: `6(?:011|5[0-9]{2})[0-9]{12}`,
		},
		// SSN
		{
			name:    "ssn",
			pattern: `\b\d{3}-\d{2}-\d{4}\b`,
		},
		// Database Connection Strings
		{
			name:    "postgres_conn",
			pattern: `postgres://[^:]+:[^@]+@[^\s]+`,
		},
		{
			name:    "mysql_conn",
			pattern: `mysql://[^:]+:[^@]+@[^\s]+`,
		},
		{
			name:    "mongodb_conn",
			pattern: `mongodb(\+srv)?://[^:]+:[^@]+@[^\s]+`,
		},
		// Private Keys
		{
			name:    "private_key_block",
			pattern: `-----BEGIN (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----[\s\S]+?-----END (RSA |EC |DSA |OPENSSH )?PRIVATE KEY-----`,
		},
		// Generic Secrets
		{
			name:    "generic_secret",
			pattern: `(?i)(secret|client[_-]?secret)\s*[:=]\s*['"]?[a-zA-Z0-9_\-]{6,}['"]?`,
		},
	}

	for _, dp := range defaultPatterns {
		if compiled, err := regexp.Compile(dp.pattern); err == nil {
			pm.patterns = append(pm.patterns, RedactionPattern{
				Name:    dp.name,
				Pattern: compiled,
			})
		}
	}
}

// AddPattern adds a custom pattern to the manager
func (pm *PatternManager) AddPattern(name, pattern string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	compiled, err := regexp.Compile(pattern)
	if err != nil {
		return err
	}

	pm.patterns = append(pm.patterns, RedactionPattern{
		Name:    name,
		Pattern: compiled,
	})

	return nil
}

// AddEmailPattern adds email redaction pattern if enabled
func (pm *PatternManager) AddEmailPattern() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	emailPattern := `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`
	if compiled, err := regexp.Compile(emailPattern); err == nil {
		pm.patterns = append(pm.patterns, RedactionPattern{
			Name:    "email",
			Pattern: compiled,
		})
	}
}

// AddIPPattern adds IP address redaction pattern if enabled
func (pm *PatternManager) AddIPPattern() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// IPv4
	ipv4Pattern := `\b(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\b`
	if compiled, err := regexp.Compile(ipv4Pattern); err == nil {
		pm.patterns = append(pm.patterns, RedactionPattern{
			Name:    "ipv4",
			Pattern: compiled,
		})
	}

	// IPv6 (simplified pattern)
	ipv6Pattern := `(?:[0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}`
	if compiled, err := regexp.Compile(ipv6Pattern); err == nil {
		pm.patterns = append(pm.patterns, RedactionPattern{
			Name:    "ipv6",
			Pattern: compiled,
		})
	}
}

// GetPatterns returns a copy of the current patterns
func (pm *PatternManager) GetPatterns() []RedactionPattern {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	patterns := make([]RedactionPattern, len(pm.patterns))
	copy(patterns, pm.patterns)
	return patterns
}

// MatchesAny checks if the input matches any pattern
func (pm *PatternManager) MatchesAny(input string) bool {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, pattern := range pm.patterns {
		if pattern.Pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// RedactAll applies all patterns to the input and returns the redacted string
func (pm *PatternManager) RedactAll(input, replacement string) string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result := input
	for _, pattern := range pm.patterns {
		result = pattern.Pattern.ReplaceAllString(result, replacement)
	}
	return result
}
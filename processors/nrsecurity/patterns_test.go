package nrsecurity

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPatternManager(t *testing.T) {
	pm := NewPatternManager()
	assert.NotNil(t, pm)
	assert.Greater(t, len(pm.patterns), 0)
}

func TestAddPattern(t *testing.T) {
	pm := NewPatternManager()
	initialCount := len(pm.patterns)

	err := pm.AddPattern("custom", `CUSTOM-[0-9]+`)
	require.NoError(t, err)
	assert.Len(t, pm.patterns, initialCount+1)

	// Test invalid pattern
	err = pm.AddPattern("invalid", `[`)
	assert.Error(t, err)
	assert.Len(t, pm.patterns, initialCount+1) // Count shouldn't change
}

func TestAddEmailPattern(t *testing.T) {
	pm := NewPatternManager()
	initialCount := len(pm.patterns)

	pm.AddEmailPattern()
	assert.Len(t, pm.patterns, initialCount+1)

	// Test email matching
	assert.True(t, pm.MatchesAny("user@example.com"))
	assert.True(t, pm.MatchesAny("Contact: test.user+tag@subdomain.example.co.uk"))
	assert.False(t, pm.MatchesAny("not-an-email"))
}

func TestAddIPPattern(t *testing.T) {
	pm := NewPatternManager()
	initialCount := len(pm.patterns)

	pm.AddIPPattern()
	assert.Len(t, pm.patterns, initialCount+2) // IPv4 and IPv6

	// Test IPv4 matching
	assert.True(t, pm.MatchesAny("192.168.1.1"))
	assert.True(t, pm.MatchesAny("Server IP: 10.0.0.1"))
	assert.True(t, pm.MatchesAny("255.255.255.255"))
	assert.False(t, pm.MatchesAny("256.1.1.1")) // Invalid IP
	assert.False(t, pm.MatchesAny("192.168.1")) // Incomplete IP

	// Test IPv6 matching (simplified pattern)
	assert.True(t, pm.MatchesAny("2001:0db8:85a3:0000:0000:8a2e:0370:7334"))
}

func TestGetPatterns(t *testing.T) {
	pm := NewPatternManager()
	patterns := pm.GetPatterns()
	
	assert.NotEmpty(t, patterns)
	assert.Equal(t, len(pm.patterns), len(patterns))
	
	// Verify it's a copy
	patterns[0].Name = "modified"
	assert.NotEqual(t, pm.patterns[0].Name, "modified")
}

func TestMatchesAny(t *testing.T) {
	pm := NewPatternManager()

	tests := []struct {
		input    string
		expected bool
		desc     string
	}{
		// API Keys
		{"AKIA1234567890ABCDEF", true, "AWS Access Key"},
		{"api_key=abc123def456ghi789jkl", true, "Generic API Key"},
		{"apikey: 'a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6'", true, "API Key with quotes"},
		{"gh_1234567890abcdef1234567890abcdef12345678", true, "GitHub Token"},
		{"xoxb-test-slack-token-example", true, "Slack Token"},

		// Passwords
		{"https://user:password123@example.com/path", true, "Password in URL"},
		{"password=secretpass123", true, "Password assignment"},
		{"PASSWORD = 'my_secure_pass'", true, "Password with quotes"},
		{"postgres://dbuser:dbpass123@localhost:5432/mydb", true, "Postgres connection"},
		{"mysql://root:admin123@127.0.0.1:3306/database", true, "MySQL connection"},
		{"mongodb://user:pass@cluster.mongodb.net/db", true, "MongoDB connection"},

		// Tokens
		{"Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U", true, "JWT Token"},
		{"Authorization: Bearer abc123def456", true, "Bearer Token"},

		// Credit Cards
		{"4111111111111111", true, "Visa"},
		{"5500000000000004", true, "Mastercard"},
		{"340000000000009", true, "Amex"},
		{"6011000000000004", true, "Discover"},

		// SSN
		{"123-45-6789", true, "SSN"},
		{"SSN: 987-65-4321", true, "SSN with label"},

		// Private Keys
		{"-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----", true, "RSA Private Key"},
		{"-----BEGIN EC PRIVATE KEY-----\nMHcCAQEE...\n-----END EC PRIVATE KEY-----", true, "EC Private Key"},

		// Secrets
		{"client_secret=1234567890abcdef", true, "Client Secret"},
		{"secret: 'my-super-secret-value'", true, "Generic Secret"},

		// Non-sensitive
		{"This is just normal text", false, "Normal text"},
		{"user@example", false, "Not a complete email"},
		{"12345678", false, "Just numbers"},
		{"service.name=my-service", false, "Service name"},
		{"", false, "Empty string"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := pm.MatchesAny(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedactAll(t *testing.T) {
	pm := NewPatternManager()

	tests := []struct {
		input       string
		replacement string
		expected    string
		desc        string
	}{
		{
			"API Key: AKIA1234567890ABCDEF",
			"[REDACTED]",
			"API Key: [REDACTED]",
			"Single AWS key",
		},
		{
			"Connection: postgres://user:pass123@localhost:5432/db",
			"***",
			"Connection: ***",
			"Database connection",
		},
		{
			"Multiple: password=secret123 and token=Bearer abc123",
			"[X]",
			"Multiple: [X] and token=[X]",
			"Multiple secrets",
		},
		{
			"Card: 4111111111111111, SSN: 123-45-6789",
			"[HIDDEN]",
			"Card: [HIDDEN], SSN: [HIDDEN]",
			"Credit card and SSN",
		},
		{
			"Normal text without secrets",
			"[REDACTED]",
			"Normal text without secrets",
			"No secrets",
		},
		{
			"",
			"[REDACTED]",
			"",
			"Empty string",
		},
		{
			"Mixed: normal text api_key=secret123 more text password='pass' end",
			"XXX",
			"Mixed: normal text XXX more text XXX end",
			"Mixed content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := pm.RedactAll(tt.input, tt.replacement)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPatternsConcurrency(t *testing.T) {
	pm := NewPatternManager()

	// Test concurrent reads
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_ = pm.MatchesAny("AKIA1234567890ABCDEF")
			_ = pm.RedactAll("password=secret123", "[REDACTED]")
			_ = pm.GetPatterns()
			done <- true
		}()
	}

	// Test concurrent writes
	for i := 0; i < 5; i++ {
		go func(n int) {
			_ = pm.AddPattern(fmt.Sprintf("pattern%d", n), `test-[0-9]+`)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 15; i++ {
		<-done
	}
}

func TestDefaultPatternsCompleteness(t *testing.T) {
	pm := NewPatternManager()

	// Ensure we have patterns for all major categories
	categories := map[string][]string{
		"API Keys": {
			"AKIA1234567890ABCDEF",                      // AWS
			"api_key=1234567890abcdef",                  // Generic
			"gh_1234567890abcdef1234567890abcdef12345678", // GitHub
		},
		"Passwords": {
			"password=mypassword123",
			"https://user:pass@example.com",
			"postgres://user:pass@localhost/db",
		},
		"Tokens": {
			"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.dozjgNryP4J3jVmNHl0w5N_XgL0n3I9PlFUP0THsR8U",
			"Bearer abc123def456",
		},
		"Credit Cards": {
			"4111111111111111",    // Visa
			"5500000000000004",    // Mastercard
			"340000000000009",     // Amex
			"6011000000000004",    // Discover
		},
		"SSN": {
			"123-45-6789",
		},
		"Secrets": {
			"client_secret=abcdef123456",
			"secret='my-secret-value'",
		},
	}

	for category, examples := range categories {
		t.Run(category, func(t *testing.T) {
			for _, example := range examples {
				assert.True(t, pm.MatchesAny(example), "Pattern missing for: %s", example)
			}
		})
	}
}

func BenchmarkMatchesAny(b *testing.B) {
	pm := NewPatternManager()
	inputs := []string{
		"AKIA1234567890ABCDEF",
		"normal text without secrets",
		"password=secret123",
		"4111111111111111",
		"https://user:pass@example.com",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pm.MatchesAny(inputs[i%len(inputs)])
	}
}

func BenchmarkRedactAll(b *testing.B) {
	pm := NewPatternManager()
	input := "API: AKIA1234567890ABCDEF, Password: secret123, Card: 4111111111111111"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = pm.RedactAll(input, "[REDACTED]")
	}
}
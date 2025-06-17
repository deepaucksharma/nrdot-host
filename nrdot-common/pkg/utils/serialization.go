// Package utils provides common utilities for NRDOT-HOST components
package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"
)

// Serializer provides methods for serializing and deserializing data
type Serializer struct {
	Format SerializationFormat
}

// SerializationFormat represents the format for serialization
type SerializationFormat string

const (
	FormatJSON SerializationFormat = "json"
	FormatYAML SerializationFormat = "yaml"
)

// NewSerializer creates a new serializer with the specified format
func NewSerializer(format SerializationFormat) *Serializer {
	return &Serializer{Format: format}
}

// Marshal serializes the given value
func (s *Serializer) Marshal(v interface{}) ([]byte, error) {
	switch s.Format {
	case FormatJSON:
		return json.MarshalIndent(v, "", "  ")
	case FormatYAML:
		return yaml.Marshal(v)
	default:
		return nil, fmt.Errorf("unsupported format: %s", s.Format)
	}
}

// Unmarshal deserializes data into the given value
func (s *Serializer) Unmarshal(data []byte, v interface{}) error {
	switch s.Format {
	case FormatJSON:
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber() // Preserve number precision
		return decoder.Decode(v)
	case FormatYAML:
		return yaml.Unmarshal(data, v)
	default:
		return fmt.Errorf("unsupported format: %s", s.Format)
	}
}

// DurationJSON provides JSON marshaling for time.Duration
type DurationJSON time.Duration

// MarshalJSON implements json.Marshaler
func (d DurationJSON) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Duration(d).String())
}

// UnmarshalJSON implements json.Unmarshaler
func (d *DurationJSON) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	duration, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	*d = DurationJSON(duration)
	return nil
}

// TimeJSON provides custom JSON marshaling for time.Time
type TimeJSON time.Time

// MarshalJSON implements json.Marshaler
func (t TimeJSON) MarshalJSON() ([]byte, error) {
	return json.Marshal(time.Time(t).Format(time.RFC3339))
}

// UnmarshalJSON implements json.Unmarshaler
func (t *TimeJSON) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return err
	}
	*t = TimeJSON(parsed)
	return nil
}

// CompactJSON removes unnecessary whitespace from JSON
func CompactJSON(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// PrettyJSON formats JSON with indentation
func PrettyJSON(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	if err := json.Indent(&buf, data, "", "  "); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ConvertFormat converts data between JSON and YAML
func ConvertFormat(data []byte, from, to SerializationFormat) ([]byte, error) {
	if from == to {
		return data, nil
	}

	// First unmarshal from source format
	var obj interface{}
	fromSerializer := NewSerializer(from)
	if err := fromSerializer.Unmarshal(data, &obj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal %s: %w", from, err)
	}

	// Then marshal to target format
	toSerializer := NewSerializer(to)
	result, err := toSerializer.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal %s: %w", to, err)
	}

	return result, nil
}

// MustMarshalJSON marshals to JSON and panics on error (for tests)
func MustMarshalJSON(v interface{}) []byte {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		panic(err)
	}
	return data
}

// MustUnmarshalJSON unmarshals from JSON and panics on error (for tests)
func MustUnmarshalJSON(data []byte, v interface{}) {
	if err := json.Unmarshal(data, v); err != nil {
		panic(err)
	}
}
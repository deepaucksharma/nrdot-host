package models

import (
	"fmt"
	"time"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	SeverityInfo     ErrorSeverity = "info"
	SeverityWarning  ErrorSeverity = "warning"
	SeverityError    ErrorSeverity = "error"
	SeverityCritical ErrorSeverity = "critical"
	SeverityFatal    ErrorSeverity = "fatal"
)

// ErrorCategory represents the category of an error
type ErrorCategory string

const (
	ErrorCategoryConfig      ErrorCategory = "configuration"
	ErrorCategoryConnection  ErrorCategory = "connection"
	ErrorCategoryAuth        ErrorCategory = "authentication"
	ErrorCategoryResource    ErrorCategory = "resource"
	ErrorCategoryData        ErrorCategory = "data"
	ErrorCategoryInternal    ErrorCategory = "internal"
	ErrorCategoryValidation  ErrorCategory = "validation"
	ErrorCategoryPermission  ErrorCategory = "permission"
)

// ErrorInfo represents detailed error information
type ErrorInfo struct {
	Code       string                 `json:"code"`
	Message    string                 `json:"message"`
	Details    string                 `json:"details,omitempty"`
	Category   ErrorCategory          `json:"category"`
	Severity   ErrorSeverity          `json:"severity"`
	Component  string                 `json:"component,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
	Context    map[string]interface{} `json:"context,omitempty"`
	StackTrace string                 `json:"stack_trace,omitempty"`
	Retryable  bool                   `json:"retryable"`
	Resolution string                 `json:"resolution,omitempty"`
}

// Error implements the error interface
func (e *ErrorInfo) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Details)
}

// Common error codes
const (
	// Configuration errors
	ErrCodeConfigInvalid      = "CONFIG_INVALID"
	ErrCodeConfigMissing      = "CONFIG_MISSING"
	ErrCodeConfigConflict     = "CONFIG_CONFLICT"
	ErrCodeConfigUnsupported  = "CONFIG_UNSUPPORTED"
	
	// Connection errors
	ErrCodeConnectionFailed   = "CONNECTION_FAILED"
	ErrCodeConnectionTimeout  = "CONNECTION_TIMEOUT"
	ErrCodeConnectionRefused  = "CONNECTION_REFUSED"
	ErrCodeNetworkUnreachable = "NETWORK_UNREACHABLE"
	
	// Authentication errors
	ErrCodeAuthFailed         = "AUTH_FAILED"
	ErrCodeAuthTokenExpired   = "AUTH_TOKEN_EXPIRED"
	ErrCodeAuthTokenInvalid   = "AUTH_TOKEN_INVALID"
	ErrCodeAuthPermissionDenied = "AUTH_PERMISSION_DENIED"
	
	// Resource errors
	ErrCodeResourceNotFound   = "RESOURCE_NOT_FOUND"
	ErrCodeResourceExhausted  = "RESOURCE_EXHAUSTED"
	ErrCodeResourceLocked     = "RESOURCE_LOCKED"
	ErrCodeResourceQuotaExceeded = "RESOURCE_QUOTA_EXCEEDED"
	
	// Data errors
	ErrCodeDataCorrupted      = "DATA_CORRUPTED"
	ErrCodeDataTooLarge       = "DATA_TOO_LARGE"
	ErrCodeDataInvalid        = "DATA_INVALID"
	ErrCodeDataLoss           = "DATA_LOSS"
	
	// Internal errors
	ErrCodeInternalError      = "INTERNAL_ERROR"
	ErrCodePanic              = "PANIC"
	ErrCodeNotImplemented     = "NOT_IMPLEMENTED"
	ErrCodeDeprecated         = "DEPRECATED"
)

// NewError creates a new ErrorInfo instance
func NewError(code string, message string, category ErrorCategory, severity ErrorSeverity) *ErrorInfo {
	return &ErrorInfo{
		Code:      code,
		Message:   message,
		Category:  category,
		Severity:  severity,
		Timestamp: time.Now(),
		Context:   make(map[string]interface{}),
	}
}

// WithDetails adds details to the error
func (e *ErrorInfo) WithDetails(details string) *ErrorInfo {
	e.Details = details
	return e
}

// WithComponent adds component information to the error
func (e *ErrorInfo) WithComponent(component string) *ErrorInfo {
	e.Component = component
	return e
}

// WithContext adds context information to the error
func (e *ErrorInfo) WithContext(key string, value interface{}) *ErrorInfo {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// WithStackTrace adds stack trace to the error
func (e *ErrorInfo) WithStackTrace(trace string) *ErrorInfo {
	e.StackTrace = trace
	return e
}

// WithResolution adds resolution steps to the error
func (e *ErrorInfo) WithResolution(resolution string) *ErrorInfo {
	e.Resolution = resolution
	return e
}

// IsRetryable returns whether the error is retryable
func (e *ErrorInfo) IsRetryable() bool {
	return e.Retryable
}

// IsCritical returns whether the error is critical or fatal
func (e *ErrorInfo) IsCritical() bool {
	return e.Severity == SeverityCritical || e.Severity == SeverityFatal
}

// ErrorList represents a collection of errors
type ErrorList struct {
	Errors []ErrorInfo `json:"errors"`
}

// Add adds an error to the list
func (el *ErrorList) Add(err ErrorInfo) {
	el.Errors = append(el.Errors, err)
}

// HasErrors returns true if there are any errors
func (el *ErrorList) HasErrors() bool {
	return len(el.Errors) > 0
}

// HasCritical returns true if there are any critical errors
func (el *ErrorList) HasCritical() bool {
	for _, err := range el.Errors {
		if err.IsCritical() {
			return true
		}
	}
	return false
}

// ByCategory returns errors filtered by category
func (el *ErrorList) ByCategory(category ErrorCategory) []ErrorInfo {
	var result []ErrorInfo
	for _, err := range el.Errors {
		if err.Category == category {
			result = append(result, err)
		}
	}
	return result
}

// BySeverity returns errors filtered by severity
func (el *ErrorList) BySeverity(severity ErrorSeverity) []ErrorInfo {
	var result []ErrorInfo
	for _, err := range el.Errors {
		if err.Severity == severity {
			result = append(result, err)
		}
	}
	return result
}
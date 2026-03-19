package vpc

import (
	"ce-vpc-power-schedule/internal/logging"
	"strings"
	"testing"
)

func TestBuildServiceURL(t *testing.T) {
	tests := []struct {
		region   string
		expected string
	}{
		{"us-south", "https://us-south.iaas.cloud.ibm.com/v1"},
		{"us-east", "https://us-east.iaas.cloud.ibm.com/v1"},
		{"eu-gb", "https://eu-gb.iaas.cloud.ibm.com/v1"},
		{"eu-de", "https://eu-de.iaas.cloud.ibm.com/v1"},
		{"jp-tok", "https://jp-tok.iaas.cloud.ibm.com/v1"},
		{"au-syd", "https://au-syd.iaas.cloud.ibm.com/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.region, func(t *testing.T) {
			result := buildServiceURL(tt.region)
			if result != tt.expected {
				t.Errorf("buildServiceURL(%q) = %q, want %q", tt.region, result, tt.expected)
			}
		})
	}
}

func TestExtractInstanceIDFromCRN(t *testing.T) {
	tests := []struct {
		name     string
		crn      string
		expected string
	}{
		{
			name:     "Valid VPC instance CRN",
			crn:      "crn:v1:bluemix:public:is:us-south:a/abc123::instance:0717_12345678-1234-1234-1234-123456789abc",
			expected: "0717_12345678-1234-1234-1234-123456789abc",
		},
		{
			name:     "Another valid CRN",
			crn:      "crn:v1:bluemix:public:is:eu-gb:a/def456::instance:0727_abcdef12-abcd-abcd-abcd-abcdefabcdef",
			expected: "0727_abcdef12-abcd-abcd-abcd-abcdefabcdef",
		},
		{
			name:     "Invalid CRN - too few parts",
			crn:      "crn:v1:bluemix:public:is",
			expected: "",
		},
		{
			name:     "Invalid CRN - not instance type",
			crn:      "crn:v1:bluemix:public:is:us-south:a/abc123::volume:vol-123",
			expected: "",
		},
		{
			name:     "Empty CRN",
			crn:      "",
			expected: "",
		},
		{
			name:     "Malformed CRN",
			crn:      "not-a-crn",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractInstanceIDFromCRN(tt.crn)
			if result != tt.expected {
				t.Errorf("ExtractInstanceIDFromCRN(%q) = %q, want %q", tt.crn, result, tt.expected)
			}
		})
	}
}

func TestNewVPCClient_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		options     *VPCClientOptions
		expectError string
	}{
		{
			name: "Missing API key",
			options: &VPCClientOptions{
				APIKey: "",
				Region: "us-south",
				Log:    &mockLogger{},
			},
			expectError: "API key is required",
		},
		{
			name: "Missing region",
			options: &VPCClientOptions{
				APIKey: "test-api-key",
				Region: "",
				Log:    &mockLogger{},
			},
			expectError: "region is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewVPCClient(tt.options)
			if err == nil {
				t.Fatal("Expected error but got nil")
			}
			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing %q, got %q", tt.expectError, err.Error())
			}
		})
	}
}

func TestInstance_Structure(t *testing.T) {
	instance := &Instance{
		ID:     "test-id",
		Name:   "test-name",
		Status: "running",
		CRN:    "crn:v1:bluemix:public:is:us-south:a/abc123::instance:test-id",
	}

	if instance.ID != "test-id" {
		t.Errorf("Instance.ID = %q, want %q", instance.ID, "test-id")
	}
	if instance.Name != "test-name" {
		t.Errorf("Instance.Name = %q, want %q", instance.Name, "test-name")
	}
	if instance.Status != "running" {
		t.Errorf("Instance.Status = %q, want %q", instance.Status, "running")
	}
	if instance.CRN != "crn:v1:bluemix:public:is:us-south:a/abc123::instance:test-id" {
		t.Errorf("Instance.CRN = %q, want valid CRN", instance.CRN)
	}
}

// mockLogger is a simple mock implementation of the Logger interface for testing
type mockLogger struct {
	debugCalls []string
	infoCalls  []string
	warnCalls  []string
	errorCalls []string
}

func (m *mockLogger) Debug(msg string, kv ...any) {
	m.debugCalls = append(m.debugCalls, msg)
}

func (m *mockLogger) Info(msg string, kv ...any) {
	m.infoCalls = append(m.infoCalls, msg)
}

func (m *mockLogger) Warn(msg string, kv ...any) {
	m.warnCalls = append(m.warnCalls, msg)
}

func (m *mockLogger) Error(msg string, kv ...any) {
	m.errorCalls = append(m.errorCalls, msg)
}

func (m *mockLogger) With(kv ...any) logging.Logger {
	return m
}

func (m *mockLogger) IsLogLevelEnabled(level string) bool {
	return true
}

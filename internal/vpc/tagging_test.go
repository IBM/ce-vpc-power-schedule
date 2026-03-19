package vpc

import (
	"strings"
	"testing"
)

func TestIsVPCInstance(t *testing.T) {
	tests := []struct {
		name     string
		crn      string
		expected bool
	}{
		{
			name:     "Valid VPC instance CRN",
			crn:      "crn:v1:bluemix:public:is:us-south:a/abc123::instance:0717_12345678-1234-1234-1234-123456789abc",
			expected: true,
		},
		{
			name:     "Another valid VPC instance CRN",
			crn:      "crn:v1:bluemix:public:is:eu-gb:a/def456::instance:0727_abcdef12-abcd-abcd-abcd-abcdefabcdef",
			expected: true,
		},
		{
			name:     "VPC volume CRN - not an instance",
			crn:      "crn:v1:bluemix:public:is:us-south:a/abc123::volume:vol-123",
			expected: false,
		},
		{
			name:     "Non-VPC service CRN",
			crn:      "crn:v1:bluemix:public:cloud-object-storage:global:a/abc123::bucket:my-bucket",
			expected: false,
		},
		{
			name:     "CRN with 'is' but not instance",
			crn:      "crn:v1:bluemix:public:is:us-south:a/abc123::subnet:subnet-123",
			expected: false,
		},
		{
			name:     "CRN with 'instance' but not VPC",
			crn:      "crn:v1:bluemix:public:other-service:us-south:a/abc123::instance:inst-123",
			expected: false,
		},
		{
			name:     "Empty CRN",
			crn:      "",
			expected: false,
		},
		{
			name:     "Invalid CRN format",
			crn:      "not-a-crn",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isVPCInstance(tt.crn)
			if result != tt.expected {
				t.Errorf("isVPCInstance(%q) = %v, want %v", tt.crn, result, tt.expected)
			}
		})
	}
}

// Note: getItemName is tested indirectly through integration tests
// as it requires the actual globalsearchv2.ResultItem type

func TestNewTaggingClient_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		apiKey      string
		expectError string
	}{
		{
			name:        "Missing API key",
			apiKey:      "",
			expectError: "API key is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewTaggingClient(tt.apiKey, &mockLogger{})
			if err == nil {
				t.Fatal("Expected error but got nil")
			}
			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing %q, got %q", tt.expectError, err.Error())
			}
		})
	}
}

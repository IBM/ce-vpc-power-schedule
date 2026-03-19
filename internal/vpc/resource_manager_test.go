package vpc

import (
	"strings"
	"testing"
)

func TestNewResourceManagerClient_ValidationErrors(t *testing.T) {
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
			_, err := NewResourceManagerClient(tt.apiKey, &mockLogger{})
			if err == nil {
				t.Fatal("Expected error but got nil")
			}
			if !strings.Contains(err.Error(), tt.expectError) {
				t.Errorf("Expected error containing %q, got %q", tt.expectError, err.Error())
			}
		})
	}
}

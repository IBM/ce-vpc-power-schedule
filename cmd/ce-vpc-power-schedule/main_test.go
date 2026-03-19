package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir := t.TempDir()

	t.Run("Valid config", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "valid_config.yaml")
		configContent := `
entities:
  - type: instance
    value: test-instance
  - type: tag
    value: test-tag
exclusions:
  timezone: "America/New_York"
  dates:
    - date: "2024-12-25"
      annual: true
  ranges:
    - from: "2024-07-01"
      to: "2024-07-15"
      annual: false
execution_mode: parallel
`
		if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		cfg, err := loadConfig(configPath)
		if err != nil {
			t.Fatalf("loadConfig() error = %v", err)
		}

		if len(cfg.Entities) != 2 {
			t.Errorf("Expected 2 entities, got %d", len(cfg.Entities))
		}
		if cfg.Entities[0].Type != "instance" {
			t.Errorf("Expected first entity type 'instance', got %q", cfg.Entities[0].Type)
		}
		if cfg.Entities[0].Value != "test-instance" {
			t.Errorf("Expected first entity value 'test-instance', got %q", cfg.Entities[0].Value)
		}
		if cfg.Exclusions.Timezone != "America/New_York" {
			t.Errorf("Expected timezone 'America/New_York', got %q", cfg.Exclusions.Timezone)
		}
		if len(cfg.Exclusions.Dates) != 1 {
			t.Errorf("Expected 1 date exclusion, got %d", len(cfg.Exclusions.Dates))
		}
		if len(cfg.Exclusions.Ranges) != 1 {
			t.Errorf("Expected 1 range exclusion, got %d", len(cfg.Exclusions.Ranges))
		}
		if cfg.ExecutionMode != "parallel" {
			t.Errorf("Expected execution_mode 'parallel', got %q", cfg.ExecutionMode)
		}
	})

	t.Run("File not found", func(t *testing.T) {
		_, err := loadConfig(filepath.Join(tmpDir, "nonexistent.yaml"))
		if err == nil {
			t.Error("Expected error for nonexistent file, got nil")
		}
	})

	t.Run("Invalid YAML", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "invalid.yaml")
		invalidContent := `
entities:
  - type: instance
    value: test
  invalid yaml content [[[
`
		if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
			t.Fatalf("Failed to write test config: %v", err)
		}

		_, err := loadConfig(configPath)
		if err == nil {
			t.Error("Expected error for invalid YAML, got nil")
		}
	})
}

func TestNowInLocation(t *testing.T) {
	tests := []struct {
		name     string
		timezone string
		wantUTC  bool
	}{
		{
			name:     "Empty timezone defaults to UTC",
			timezone: "",
			wantUTC:  true,
		},
		{
			name:     "Invalid timezone defaults to UTC",
			timezone: "Invalid/Timezone",
			wantUTC:  true,
		},
		{
			name:     "Valid timezone America/New_York",
			timezone: "America/New_York",
			wantUTC:  false,
		},
		{
			name:     "Valid timezone Europe/London",
			timezone: "Europe/London",
			wantUTC:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := nowInLocation(tt.timezone)
			if result.IsZero() {
				t.Error("nowInLocation() returned zero time")
			}

			// Check if it's UTC when expected
			if tt.wantUTC {
				if result.Location() != time.UTC {
					t.Errorf("Expected UTC location, got %v", result.Location())
				}
			}
		})
	}
}

func TestParseDateOrPanic(t *testing.T) {
	tests := []struct {
		name        string
		dateStr     string
		shouldPanic bool
		expected    time.Time
	}{
		{
			name:        "Valid date",
			dateStr:     "2024-12-25",
			shouldPanic: false,
			expected:    time.Date(2024, 12, 25, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "Another valid date",
			dateStr:     "2024-01-01",
			shouldPanic: false,
			expected:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name:        "Invalid date format",
			dateStr:     "25-12-2024",
			shouldPanic: true,
		},
		{
			name:        "Invalid date",
			dateStr:     "not-a-date",
			shouldPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("Expected panic but didn't get one")
					}
				}()
				parseDateOrPanic(tt.dateStr)
			} else {
				result := parseDateOrPanic(tt.dateStr)
				if !result.Equal(tt.expected) {
					t.Errorf("parseDateOrPanic(%q) = %v, want %v", tt.dateStr, result, tt.expected)
				}
			}
		})
	}
}

func TestSameDate(t *testing.T) {
	tests := []struct {
		name     string
		a        time.Time
		b        time.Time
		expected bool
	}{
		{
			name:     "Same date and time",
			a:        time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC),
			b:        time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "Same date, different time",
			a:        time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC),
			b:        time.Date(2024, 12, 25, 15, 45, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "Different date",
			a:        time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC),
			b:        time.Date(2024, 12, 26, 10, 30, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "Different month",
			a:        time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC),
			b:        time.Date(2024, 11, 25, 10, 30, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "Different year",
			a:        time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC),
			b:        time.Date(2023, 12, 25, 10, 30, 0, 0, time.UTC),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sameDate(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("sameDate(%v, %v) = %v, want %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestSameMonthDay(t *testing.T) {
	tests := []struct {
		name     string
		now      time.Time
		d        time.Time
		expected bool
	}{
		{
			name:     "Same month and day, same year",
			now:      time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC),
			d:        time.Date(2024, 12, 25, 15, 45, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "Same month and day, different year",
			now:      time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC),
			d:        time.Date(2023, 12, 25, 15, 45, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "Different day",
			now:      time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC),
			d:        time.Date(2024, 12, 26, 10, 30, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "Different month",
			now:      time.Date(2024, 12, 25, 10, 30, 0, 0, time.UTC),
			d:        time.Date(2024, 11, 25, 10, 30, 0, 0, time.UTC),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sameMonthDay(tt.now, tt.d)
			if result != tt.expected {
				t.Errorf("sameMonthDay(%v, %v) = %v, want %v", tt.now, tt.d, result, tt.expected)
			}
		})
	}
}

func TestInAnnualRange(t *testing.T) {
	tests := []struct {
		name     string
		now      time.Time
		from     time.Time
		to       time.Time
		expected bool
	}{
		{
			name:     "Within range - same year",
			now:      time.Date(2024, 7, 10, 0, 0, 0, 0, time.UTC),
			from:     time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			to:       time.Date(2024, 7, 31, 0, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "Within range - different year",
			now:      time.Date(2025, 7, 10, 0, 0, 0, 0, time.UTC),
			from:     time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			to:       time.Date(2024, 7, 31, 0, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "Before range",
			now:      time.Date(2024, 6, 30, 0, 0, 0, 0, time.UTC),
			from:     time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			to:       time.Date(2024, 7, 31, 0, 0, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "After range",
			now:      time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC),
			from:     time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC),
			to:       time.Date(2024, 7, 31, 0, 0, 0, 0, time.UTC),
			expected: false,
		},
		{
			name:     "Year-crossing range - in first part",
			now:      time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC),
			from:     time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
			to:       time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "Year-crossing range - in second part",
			now:      time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
			from:     time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
			to:       time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			expected: true,
		},
		{
			name:     "Year-crossing range - outside",
			now:      time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC),
			from:     time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC),
			to:       time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := inAnnualRange(tt.now, tt.from, tt.to)
			if result != tt.expected {
				t.Errorf("inAnnualRange(%v, %v, %v) = %v, want %v",
					tt.now.Format("2006-01-02"),
					tt.from.Format("2006-01-02"),
					tt.to.Format("2006-01-02"),
					result, tt.expected)
			}
		})
	}
}

func TestInExclusions(t *testing.T) {
	tests := []struct {
		name           string
		cfg            *Config
		now            time.Time
		expectExcluded bool
		expectReason   string
	}{
		{
			name: "Annual date exclusion - match",
			cfg: &Config{
				Exclusions: Exclusions{
					Dates: []DateExclusion{
						{Date: "2024-12-25", Annual: true},
					},
				},
			},
			now:            time.Date(2025, 12, 25, 10, 0, 0, 0, time.UTC),
			expectExcluded: true,
			expectReason:   "annual day",
		},
		{
			name: "Specific date exclusion - match",
			cfg: &Config{
				Exclusions: Exclusions{
					Dates: []DateExclusion{
						{Date: "2024-12-25", Annual: false},
					},
				},
			},
			now:            time.Date(2024, 12, 25, 10, 0, 0, 0, time.UTC),
			expectExcluded: true,
			expectReason:   "day 2024-12-25",
		},
		{
			name: "Specific date exclusion - no match different year",
			cfg: &Config{
				Exclusions: Exclusions{
					Dates: []DateExclusion{
						{Date: "2024-12-25", Annual: false},
					},
				},
			},
			now:            time.Date(2025, 12, 25, 10, 0, 0, 0, time.UTC),
			expectExcluded: false,
		},
		{
			name: "Annual range exclusion - match",
			cfg: &Config{
				Exclusions: Exclusions{
					Ranges: []RangeExclusion{
						{From: "2024-07-01", To: "2024-07-15", Annual: true},
					},
				},
			},
			now:            time.Date(2025, 7, 10, 10, 0, 0, 0, time.UTC),
			expectExcluded: true,
			expectReason:   "annual range",
		},
		{
			name: "Specific range exclusion - match",
			cfg: &Config{
				Exclusions: Exclusions{
					Ranges: []RangeExclusion{
						{From: "2024-07-01", To: "2024-07-15", Annual: false},
					},
				},
			},
			now:            time.Date(2024, 7, 10, 10, 0, 0, 0, time.UTC),
			expectExcluded: true,
			expectReason:   "range 2024-07-01..2024-07-15",
		},
		{
			name: "Specific range exclusion - no match different year",
			cfg: &Config{
				Exclusions: Exclusions{
					Ranges: []RangeExclusion{
						{From: "2024-07-01", To: "2024-07-15", Annual: false},
					},
				},
			},
			now:            time.Date(2025, 7, 10, 10, 0, 0, 0, time.UTC),
			expectExcluded: false,
		},
		{
			name: "No exclusions",
			cfg: &Config{
				Exclusions: Exclusions{},
			},
			now:            time.Date(2024, 7, 10, 10, 0, 0, 0, time.UTC),
			expectExcluded: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			excluded, reason := inExclusions(tt.cfg, tt.now)
			if excluded != tt.expectExcluded {
				t.Errorf("inExclusions() excluded = %v, want %v", excluded, tt.expectExcluded)
			}
			if tt.expectExcluded && tt.expectReason != "" {
				if reason == "" {
					t.Error("Expected reason but got empty string")
				}
				// Just check if the reason contains the expected substring
				if !containsSubstring(reason, tt.expectReason) {
					t.Errorf("inExclusions() reason = %q, want to contain %q", reason, tt.expectReason)
				}
			}
		})
	}
}

func TestEntityTypes(t *testing.T) {
	// Test that constants are defined correctly
	if instanceType != "instance" {
		t.Errorf("instanceType = %q, want %q", instanceType, "instance")
	}
	if tagType != "tag" {
		t.Errorf("tagType = %q, want %q", tagType, "tag")
	}
	if placementGroupType != "placement_group" {
		t.Errorf("placementGroupType = %q, want %q", placementGroupType, "placement_group")
	}
	if resourceGroupType != "resource_group" {
		t.Errorf("resourceGroupType = %q, want %q", resourceGroupType, "resource_group")
	}
	if vpcType != "vpc" {
		t.Errorf("vpcType = %q, want %q", vpcType, "vpc")
	}
}

// Helper function to check if a string contains a substring
func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr)+1 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

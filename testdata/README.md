# Test Data

This directory contains test fixtures and sample configuration files used by the unit tests.

## Files

- `schedule_test.yaml` - Comprehensive test configuration with all entity types and exclusions
- `schedule_sequential.yaml` - Configuration for testing sequential execution mode
- `schedule_minimal.yaml` - Minimal configuration for basic testing

## Usage

These files are used by the test suite in `cmd/ce-vpc-power-schedule/main_test.go` to validate configuration loading and parsing functionality.

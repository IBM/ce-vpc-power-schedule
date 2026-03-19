# Testing Documentation

## Overview

This document describes the unit test suite for the ce-vpc-power-schedule project.

## Test Coverage

Current test coverage by package:

- **internal/logging**: 100.0% coverage
- **cmd/ce-vpc-power-schedule**: 21.2% coverage (core business logic)
- **internal/vpc**: 8.0% coverage (validation and utility functions)

## Running Tests

### Run all tests
```bash
go test ./...
```

### Run tests with verbose output
```bash
go test ./... -v
```

### Run tests with coverage
```bash
go test ./... -cover
```

### Generate coverage report
```bash
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Run tests for a specific package
```bash
go test ./internal/logging -v
go test ./internal/vpc -v
go test ./cmd/ce-vpc-power-schedule -v
```

## Test Structure

### internal/logging/slog_logger_test.go
Tests for the structured logging implementation:
- Logger creation and initialization
- Log level methods (Debug, Info, Warn, Error)
- Context propagation with `With()`
- Log level checking
- Level parsing from strings
- Key-value pair handling

### internal/vpc/client_test.go
Tests for VPC client functionality:
- Service URL construction for different regions
- Instance ID extraction from CRNs
- Client validation (API key, region requirements)
- Instance structure validation
- Mock logger implementation

### internal/vpc/tagging_test.go
Tests for tagging client:
- VPC instance CRN validation
- Tag-based filtering logic
- Client validation

### internal/vpc/resource_manager_test.go
Tests for resource manager client:
- Client validation
- API key requirements

### cmd/ce-vpc-power-schedule/main_test.go
Tests for main application logic:
- Configuration loading from YAML files
- Invalid configuration handling
- Timezone handling and location parsing
- Date parsing and validation
- Date comparison functions (`sameDate`, `sameMonthDay`)
- Annual range calculations (including year-crossing ranges)
- Exclusion logic (dates and ranges, annual and specific)
- Entity type constants validation

## Test Fixtures

Test data is located in the `testdata/` directory:

- **schedule_test.yaml**: Comprehensive configuration with all entity types and exclusions
- **schedule_sequential.yaml**: Configuration for sequential execution mode testing
- **schedule_minimal.yaml**: Minimal configuration for basic testing

## Key Test Scenarios

### Date and Time Handling
- ✅ Timezone conversion (UTC, America/New_York, Europe/London)
- ✅ Invalid timezone fallback to UTC
- ✅ Date parsing with validation
- ✅ Same date comparison (ignoring time)
- ✅ Same month/day comparison (ignoring year)
- ✅ Annual range calculations
- ✅ Year-crossing range handling (e.g., Dec 20 - Jan 15)

### Exclusions
- ✅ Annual date exclusions (e.g., Christmas every year)
- ✅ Specific date exclusions (one-time events)
- ✅ Annual range exclusions (e.g., summer vacation period)
- ✅ Specific range exclusions (one-time periods)
- ✅ Multiple exclusions handling

### Configuration
- ✅ Valid YAML parsing
- ✅ Invalid YAML error handling
- ✅ Missing file error handling
- ✅ All entity types (instance, tag, placement_group, resource_group, vpc)
- ✅ Execution modes (parallel, sequential)

### VPC Client
- ✅ Service URL generation for all IBM Cloud regions
- ✅ CRN parsing and instance ID extraction
- ✅ Invalid CRN handling
- ✅ API key and region validation

### Logging
- ✅ All log levels (Debug, Info, Warn, Error)
- ✅ Structured logging with key-value pairs
- ✅ Context propagation
- ✅ Log level filtering
- ✅ JSON output format

## Mock Objects

The test suite includes mock implementations:

- **mockLogger**: Implements the `logging.Logger` interface for testing without actual log output

## Notes on Coverage

### Why some packages have lower coverage:

1. **internal/vpc (8.0%)**: Most functions require actual IBM Cloud API calls. The tests focus on:
   - Input validation
   - Utility functions (CRN parsing, URL building)
   - Error handling for invalid inputs
   - Integration tests would be needed for full coverage

2. **cmd/ce-vpc-power-schedule (21.2%)**: The main package includes:
   - Tested: Configuration loading, date/time logic, exclusions
   - Not tested: IBM Cloud API interactions, instance processing (requires mocking complex SDK types)
   - Integration tests would be needed for handler functions

3. **internal/logging (100%)**: Complete coverage as it's a pure utility package with no external dependencies

## Future Improvements

Potential areas for test expansion:

1. **Integration Tests**: Test actual IBM Cloud API interactions with test accounts
2. **Mock SDK Responses**: Create comprehensive mocks for IBM VPC SDK responses
3. **End-to-End Tests**: Test complete workflows from config to instance actions
4. **Performance Tests**: Benchmark parallel vs sequential execution
5. **Error Scenario Tests**: More edge cases and error conditions

## Continuous Integration

To integrate with CI/CD pipelines:

```bash
# Run tests and fail on error
go test ./... -v

# Generate coverage and enforce minimum threshold
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//'
```

## Contributing

When adding new features:

1. Write tests first (TDD approach recommended)
2. Ensure all tests pass: `go test ./...`
3. Check coverage: `go test ./... -cover`
4. Add test documentation for complex scenarios
5. Update this document if adding new test categories

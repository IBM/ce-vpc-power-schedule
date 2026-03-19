# ce-vpc-power-schedule

Lightweight tool to start/stop Virtual Server Instances (VSI) in IBM Cloud VPC based on cron schedules and exclusion rules.

## Overview

This tool allows you to automatically power on/off VSIs in IBM Cloud VPC infrastructure based on scheduled times. You can target instances by:

- **Instance name**: Directly specify VSI names
- **Tags**: Target all VSIs with a specific tag
- **Placement groups**: Target all VSIs in a placement group
- **Resource group**: Target all VSIs in a specific resource group
- **VPC**: Target all VSIs in a specific VPC

## Prerequisites

Before using the project, ensure you have:

- An **IBM Cloud account** with access to VPC resources. See [Required permissions](#required-permissions)
- **IBM Cloud CLI** installed
  - Linux: `curl -fsSL https://clis.cloud.ibm.com/install/linux | sh`
  - MacOS: `curl -fsSL https://clis.cloud.ibm.com/install/osx | sh`
  - Windows™: `iex (New-Object Net.WebClient).DownloadString('https://clis.cloud.ibm.com/install/powershell')`
  - WSL2 on Windows™: `curl -fsSL https://clis.cloud.ibm.com/install/linux | sh`
- **IBM Cloud CLI Code Engine plugin** installed (`ibmcloud plugin install ce`)
- **IBM Cloud CLI Container Registry plugin** installed (`ibmcloud plugin install cr`)
- A **resource group** to deploy resources
- A **VPC** with at least one Virtual Server Instance

---

## Required permissions

You must have at least the following roles. You can check your access by going to:

- Manage > Access (IAM) > [Users](https://cloud.ibm.com/iam/users) > User > Access or
- Manage > Access (IAM) > [Access groups](https://cloud.ibm.com/iam/groups) > Access

| Service                 | Roles   |
| ----------------------- | ------- |
| **Resource group only** | Editor  |
| **Code Engine**         | Manager |
| **VPC Infrastructure Services** | Operator or Editor |

---

## Setup

**1. Clone the repository:**

   ```bash
   git clone https://github.com/IBM/ce-vpc-power-schedule.git
   cd ce-vpc-power-schedule
   ```

**2. Export following environment variables**:

  ```bash
  export RESOURCE_GROUP=          # Resource group name used to deploy resources
  export IBM_REGION=              # IBM Cloud region (e.g., "us-south", "eu-de", "jp-tok")
  export IBM_APIKEY=              # IBM Cloud API key used to deploy resources and stored into Code Engine secret
  ```

**3. Run `deploy.sh` script**

  ```bash
  chmod +x deploy.sh
  ./deploy.sh
  ```

**4. Modify Code Engine config-map `entities-and-exclusions--cm` accordingly**

  ```yaml
  execution_mode: parallel # Execution mode: "parallel" (default) or "sequential"
  entities: # List virtual entities being managed.
    - type: instance # Type can be: instance, tag, placement_group, resource_group, or vpc
      value: "my-vsi-name" # Name of the VSI instance
    - type: tag
      value: "power-schedule" # Tag name to filter instances
    - type: placement_group
      value: "my-placement-group" # Name of the placement group
    - type: resource_group
      value: "my-resource-group" # Resource group name
    - type: vpc
      value: "my-vpc" # VPC name
  exclusions: # Specifies certain conditions under which the scheduled tasks should not run.
    timezone: Europe/Rome # time zone to use
    dates: # List of single dates
      - date: '2025-12-25'
        annual: true # true if the date occurs every year, false if it is only valid for the specified year
    ranges: # List of date ranges
      - from: '2025-08-10'
        to: '2025-08-24'
        annual: true
      - from: '2025-12-25'
        to: '2025-12-31'
        annual: false
  ```

**5. By default, two periodic timer event subscriptions (cron-like) are created automatically. Update them accordingly.**

  - `poweron-working-days--cron` (planning for start-up during working days at 7 AM)
  - `poweroff-working-days--cron` (planning for shutdown during working days at 7 PM)

---

## Execution Mode

The tool supports two execution modes for processing instances:

### Parallel (Default)
Processes all instances concurrently, which is faster for large numbers of instances:
```yaml
execution_mode: parallel
```

### Sequential

Processes instances one at a time, which can be useful for:

- Avoiding API rate limits
- Debugging issues
- Controlled resource usage

```yaml
execution_mode: sequential
```

**Note**: If `execution_mode` is not specified, the tool defaults to `parallel` for backward compatibility.

---

## Entity Types

### Instance

Target a specific VSI by its name:

```yaml
- type: instance
  value: "my-web-server"
```

### Tag

Target all VSIs that have a specific tag:

```yaml
- type: tag
  value: "auto-shutdown"
```

**Note**: To use tags, you need to:

1. Add tags to your VSIs through the IBM Cloud console or CLI
2. Ensure the service has proper permissions to read tags

### Placement Group

Target all VSIs within a placement group:

```yaml
- type: placement_group
  value: "production-servers"
```

### Resource Group

Target all VSIs within a specific resource group by name:

```yaml
- type: resource_group
  value: "my-resource-group"
```

**Note**: Use the resource group name. To list your resource groups:

```bash
ibmcloud resource groups
```

### VPC

Target all VSIs within a specific VPC by name:

```yaml
- type: vpc
  value: "my-vpc"
```

**Note**: Use the VPC name. To list your VPCs:

```bash
ibmcloud is vpcs
```

---

## Exclusion Rules

Exclusion rules allow you to skip power operations on specific dates or date ranges:

### Single Dates

```yaml
dates:
  - date: '2025-12-25'
    annual: true  # Repeats every year
  - date: '2025-01-15'
    annual: false # Only for 2025
```

### Date Ranges

```yaml
ranges:
  - from: '2025-08-10'
    to: '2025-08-24'
    annual: true  # Repeats every year
  - from: '2025-12-20'
    to: '2025-12-31'
    annual: false # Only for 2025
```

---

## Cleaning Up Resources

**1. Run `clean.sh` script**

  ```bash
  chmod +x clean.sh
  ./clean.sh
  ```

---

## Local Development and Testing

### Build the application

```bash
go mod tidy
go build -o bin/ce-vpc-power-schedule ./cmd/ce-vpc-power-schedule
```

### Run locally

Set required environment variables:

```bash
export IBM_APIKEY="your-api-key"
export IBM_REGION="us-south"
export CONFIG_PATH="./schedule_template.yaml"
```

Run the application:

```bash
# Power on instances
go run ./cmd/ce-vpc-power-schedule powerOn

# Power off instances
go run ./cmd/ce-vpc-power-schedule powerOff
```

### Run tests

```bash
# Run all tests
go test ./... -v

# Run tests with coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

See [TESTING.md](TESTING.md) for detailed testing documentation.

### Build Docker image

```bash
make build-image
# or
podman build -t ce-vpc-power-schedule:latest .
```

---

## Architecture

The tool consists of:

1. **Main application** (`cmd/ce-vpc-power-schedule/main.go`): Orchestrates the power operations
2. **VPC Client** (`internal/vpc/client.go`): Handles VPC API interactions
3. **Tagging Client** (`internal/vpc/tagging.go`): Manages tag-based instance discovery
4. **Resource Manager Client** (`internal/vpc/resource_manager.go`): Handles resource group operations
5. **Logging** (`internal/logging/`): Structured logging interface

The application runs as a Code Engine job triggered by cron subscriptions.

---

## Supported Regions

The tool supports all IBM Cloud VPC regions, including:

- `us-south` (Dallas)
- `us-east` (Washington DC)
- `eu-de` (Frankfurt)
- `eu-gb` (London)
- `jp-tok` (Tokyo)
- `jp-osa` (Osaka)
- `au-syd` (Sydney)
- `ca-tor` (Toronto)
- `br-sao` (São Paulo)

---

## Troubleshooting

### Check job logs

```bash
ibmcloud ce jobrun logs --name <jobrun-name>
```

### List job runs

```bash
ibmcloud ce jobrun list
```

### Test configuration

Update the ConfigMap and manually trigger a job run:

```bash
ibmcloud ce jobrun submit --job ce-vpc-poweron-schedule--job
```

### Common Issues

1. **Instances not found**: Verify instance names, tags, or placement group names are correct
2. **Permission errors**: Ensure the API key has proper VPC permissions
3. **Region mismatch**: Verify the IBM_REGION matches where your VPC resources are located

---

## Security Considerations

- API keys are stored securely in Code Engine secrets
- The application uses IBM Cloud IAM for authentication
- Minimal permissions should be granted (Operator role for VPC)
- Consider using service IDs instead of user API keys for production

---

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

---

## License

This project is licensed under the Apache License 2.0 - see the LICENSE file for details.

---

## Related Projects

- [ce-vcd-power-schedule](https://github.com/IBM/ce-vcd-power-schedule) - Similar tool for VMware Cloud Director

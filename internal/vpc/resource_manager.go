package vpc

import (
	"ce-vpc-power-schedule/internal/logging"
	"fmt"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/resourcemanagerv2"
)

// ResourceManagerClient wraps the IBM Resource Manager API client
type ResourceManagerClient struct {
	service *resourcemanagerv2.ResourceManagerV2
	log     logging.Logger
}

// NewResourceManagerClient creates a new resource manager client
func NewResourceManagerClient(apiKey string, log logging.Logger) (*ResourceManagerClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	authenticator := &core.IamAuthenticator{
		ApiKey: apiKey,
	}

	service, err := resourcemanagerv2.NewResourceManagerV2(&resourcemanagerv2.ResourceManagerV2Options{
		Authenticator: authenticator,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create resource manager service: %w", err)
	}

	return &ResourceManagerClient{
		service: service,
		log:     log.With("service", "resource-manager-client"),
	}, nil
}

// GetResourceGroupIDByName retrieves the resource group ID by name
func (c *ResourceManagerClient) GetResourceGroupIDByName(name string) (string, error) {
	c.log.Debug("Getting resource group ID by name", "name", name)

	listOptions := &resourcemanagerv2.ListResourceGroupsOptions{
		Name: &name,
	}

	result, _, err := c.service.ListResourceGroups(listOptions)
	if err != nil {
		return "", fmt.Errorf("failed to list resource groups: %w", err)
	}

	if len(result.Resources) == 0 {
		return "", fmt.Errorf("resource group %q not found", name)
	}

	// Return the first matching resource group ID
	if result.Resources[0].ID == nil {
		return "", fmt.Errorf("resource group %q has no ID", name)
	}

	return *result.Resources[0].ID, nil
}

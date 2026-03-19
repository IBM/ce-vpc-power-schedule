package vpc

import (
	"ce-vpc-power-schedule/internal/logging"
	"fmt"
	"strings"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/vpc-go-sdk/vpcv1"
)

// VPCClient wraps the IBM VPC SDK client with additional functionality
type VPCClient struct {
	service *vpcv1.VpcV1
	log     logging.Logger
}

// VPCClientOptions contains options for creating a VPC client
type VPCClientOptions struct {
	APIKey string
	Region string
	Log    logging.Logger
}

// NewVPCClient creates a new VPC client
func NewVPCClient(options *VPCClientOptions) (*VPCClient, error) {
	if options.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if options.Region == "" {
		return nil, fmt.Errorf("region is required")
	}

	authenticator := &core.IamAuthenticator{
		ApiKey: options.APIKey,
	}

	serviceOptions := &vpcv1.VpcV1Options{
		Authenticator: authenticator,
		URL:           buildServiceURL(options.Region),
	}

	service, err := vpcv1.NewVpcV1(serviceOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to create VPC service: %w", err)
	}

	return &VPCClient{
		service: service,
		log:     options.Log.With("service", "vpc-client"),
	}, nil
}

// buildServiceURL constructs the VPC service URL for a given region
func buildServiceURL(region string) string {
	return fmt.Sprintf("https://%s.iaas.cloud.ibm.com/v1", region)
}

// Instance represents a VPC virtual server instance
type Instance struct {
	ID     string
	Name   string
	Status string
	CRN    string
}

// ListInstancesByName retrieves instances by name
func (c *VPCClient) ListInstancesByName(name string) ([]*Instance, error) {
	c.log.Debug("Listing instances by name", "name", name)

	listOptions := &vpcv1.ListInstancesOptions{}
	result, _, err := c.service.ListInstances(listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	var instances []*Instance
	for _, inst := range result.Instances {
		if inst.Name != nil && *inst.Name == name {
			instances = append(instances, &Instance{
				ID:     *inst.ID,
				Name:   *inst.Name,
				Status: *inst.Status,
				CRN:    *inst.CRN,
			})
		}
	}

	return instances, nil
}

// ListInstancesByTag retrieves instances by tag using CRNs from tagging client
func (c *VPCClient) ListInstancesByTag(crns []string) ([]*Instance, error) {
	c.log.Debug("Listing instances by CRNs", "count", len(crns))

	var instances []*Instance
	for _, crn := range crns {
		instanceID := ExtractInstanceIDFromCRN(crn)
		if instanceID == "" {
			c.log.Warn("Failed to extract instance ID from CRN", "crn", crn)
			continue
		}

		// Get instance details
		getOptions := &vpcv1.GetInstanceOptions{
			ID: &instanceID,
		}
		inst, _, err := c.service.GetInstance(getOptions)
		if err != nil {
			c.log.Warn("Failed to get instance", "id", instanceID, "error", err)
			continue
		}

		instances = append(instances, &Instance{
			ID:     *inst.ID,
			Name:   *inst.Name,
			Status: *inst.Status,
			CRN:    *inst.CRN,
		})
	}

	return instances, nil
}

// ExtractInstanceIDFromCRN extracts the instance ID from a CRN
func ExtractInstanceIDFromCRN(crn string) string {
	// CRN format: crn:v1:bluemix:public:is:region:account-id::instance:instance-id
	parts := strings.Split(crn, ":")
	if len(parts) >= 10 && parts[8] == "instance" {
		return parts[9]
	}
	return ""
}

// ListInstancesByPlacementGroup retrieves instances in a placement group
func (c *VPCClient) ListInstancesByPlacementGroup(placementGroupName string) ([]*Instance, error) {
	c.log.Debug("Listing instances by placement group", "placementGroup", placementGroupName)

	// First, find the placement group by name
	listPGOptions := &vpcv1.ListPlacementGroupsOptions{}
	pgResult, _, err := c.service.ListPlacementGroups(listPGOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list placement groups: %w", err)
	}

	var placementGroupID string
	for _, pg := range pgResult.PlacementGroups {
		if pg.Name != nil && *pg.Name == placementGroupName {
			placementGroupID = *pg.ID
			break
		}
	}

	if placementGroupID == "" {
		return nil, fmt.Errorf("placement group %q not found", placementGroupName)
	}

	// Now list instances in this placement group
	listOptions := &vpcv1.ListInstancesOptions{}
	result, _, err := c.service.ListInstances(listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	var instances []*Instance
	for _, inst := range result.Instances {
		if inst.PlacementTarget != nil {
			if pgRef, ok := inst.PlacementTarget.(*vpcv1.InstancePlacementTargetPlacementGroupReference); ok {
				if pgRef.ID != nil && *pgRef.ID == placementGroupID {
					instances = append(instances, &Instance{
						ID:     *inst.ID,
						Name:   *inst.Name,
						Status: *inst.Status,
						CRN:    *inst.CRN,
					})
				}
			}
		}
	}

	return instances, nil
}

// ListInstancesByResourceGroup retrieves instances in a specific resource group by name
// It first resolves the resource group name to an ID, then filters instances efficiently
func (c *VPCClient) ListInstancesByResourceGroup(resourceGroupName string, rmClient *ResourceManagerClient) ([]*Instance, error) {
	c.log.Debug("Listing instances by resource group", "resourceGroupName", resourceGroupName)

	// Get resource group ID from name
	resourceGroupID, err := rmClient.GetResourceGroupIDByName(resourceGroupName)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource group ID: %w", err)
	}

	c.log.Debug("Resolved resource group", "name", resourceGroupName, "id", resourceGroupID)

	// List instances filtered by resource group ID
	listOptions := &vpcv1.ListInstancesOptions{
		ResourceGroupID: &resourceGroupID,
	}
	result, _, err := c.service.ListInstances(listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	var instances []*Instance
	for _, inst := range result.Instances {
		instances = append(instances, &Instance{
			ID:     *inst.ID,
			Name:   *inst.Name,
			Status: *inst.Status,
			CRN:    *inst.CRN,
		})
	}

	return instances, nil
}

// ListInstancesByVPC retrieves all instances in a specific VPC by name
func (c *VPCClient) ListInstancesByVPC(vpcName string) ([]*Instance, error) {
	c.log.Debug("Listing instances by VPC", "vpcName", vpcName)

	// First, find the VPC by name
	listVPCOptions := &vpcv1.ListVpcsOptions{}
	vpcResult, _, err := c.service.ListVpcs(listVPCOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list VPCs: %w", err)
	}

	var vpcID string
	for _, vpc := range vpcResult.Vpcs {
		if vpc.Name != nil && *vpc.Name == vpcName {
			vpcID = *vpc.ID
			break
		}
	}

	if vpcID == "" {
		return nil, fmt.Errorf("VPC %q not found", vpcName)
	}

	c.log.Debug("Resolved VPC", "name", vpcName, "id", vpcID)

	// List instances filtered by VPC ID
	listOptions := &vpcv1.ListInstancesOptions{
		VPCID: &vpcID,
	}
	result, _, err := c.service.ListInstances(listOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	var instances []*Instance
	for _, inst := range result.Instances {
		instances = append(instances, &Instance{
			ID:     *inst.ID,
			Name:   *inst.Name,
			Status: *inst.Status,
			CRN:    *inst.CRN,
		})
	}

	return instances, nil
}

// StartInstance starts a stopped instance
func (c *VPCClient) StartInstance(instanceID string) error {
	c.log.Info("Starting instance", "instanceID", instanceID)

	options := &vpcv1.CreateInstanceActionOptions{
		InstanceID: &instanceID,
		Type:       core.StringPtr("start"),
	}

	action, _, err := c.service.CreateInstanceAction(options)
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	c.log.Debug("Start action created", "actionID", *action.ID)
	return c.waitForInstanceStatus(instanceID, "running", 5*time.Minute)
}

// StopInstance stops a running instance
func (c *VPCClient) StopInstance(instanceID string) error {
	c.log.Info("Stopping instance", "instanceID", instanceID)

	options := &vpcv1.CreateInstanceActionOptions{
		InstanceID: &instanceID,
		Type:       core.StringPtr("stop"),
	}

	action, _, err := c.service.CreateInstanceAction(options)
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	c.log.Debug("Stop action created", "actionID", *action.ID)
	return c.waitForInstanceStatus(instanceID, "stopped", 5*time.Minute)
}

// GetInstanceStatus retrieves the current status of an instance
func (c *VPCClient) GetInstanceStatus(instanceID string) (string, error) {
	options := &vpcv1.GetInstanceOptions{
		ID: &instanceID,
	}

	instance, _, err := c.service.GetInstance(options)
	if err != nil {
		return "", fmt.Errorf("failed to get instance: %w", err)
	}

	if instance.Status == nil {
		return "", fmt.Errorf("instance status is nil")
	}

	return *instance.Status, nil
}

// waitForInstanceStatus waits for an instance to reach a specific status
func (c *VPCClient) waitForInstanceStatus(instanceID, targetStatus string, timeout time.Duration) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	for {
		select {
		case <-ticker.C:
			status, err := c.GetInstanceStatus(instanceID)
			if err != nil {
				return err
			}

			c.log.Debug("Instance status check", "instanceID", instanceID, "status", status, "target", targetStatus)

			if strings.EqualFold(status, targetStatus) {
				c.log.Info("Instance reached target status", "instanceID", instanceID, "status", status)
				return nil
			}

		case <-timeoutTimer.C:
			status, _ := c.GetInstanceStatus(instanceID)
			return fmt.Errorf("timeout waiting for instance to reach status %s (current: %s)", targetStatus, status)
		}
	}
}

package vpc

import (
	"ce-vpc-power-schedule/internal/logging"
	"fmt"
	"strings"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/globalsearchv2"
)

// TaggingClient wraps the IBM Global Search API client for tag-based searches
type TaggingClient struct {
	service *globalsearchv2.GlobalSearchV2
	log     logging.Logger
}

// NewTaggingClient creates a new tagging client using Global Search API
func NewTaggingClient(apiKey string, log logging.Logger) (*TaggingClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	authenticator := &core.IamAuthenticator{
		ApiKey: apiKey,
	}

	service, err := globalsearchv2.NewGlobalSearchV2(&globalsearchv2.GlobalSearchV2Options{
		Authenticator: authenticator,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create global search service: %w", err)
	}

	return &TaggingClient{
		service: service,
		log:     log.With("service", "tagging-client"),
	}, nil
}

// GetInstanceCRNsByTag retrieves instance CRNs that have a specific tag using Global Search
func (c *TaggingClient) GetInstanceCRNsByTag(tagName string) ([]string, error) {
	c.log.Debug("Getting instance CRNs by tag using Global Search", "tag", tagName)

	// Build search query for VPC instances with the specified tag
	// Query format: "type:instance AND tags:tagname"
	query := fmt.Sprintf("type:instance AND tags:%s", tagName)

	searchOptions := &globalsearchv2.SearchOptions{
		Query:  &query,
		Fields: []string{"crn", "name", "type"},
		Limit:  core.Int64Ptr(1000), // Max results per page
	}

	result, _, err := c.service.Search(searchOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to search resources by tag: %w", err)
	}

	var crns []string
	if result.Items != nil {
		for _, item := range result.Items {
			// Check if it's a VPC instance (CRN contains "is" service and "instance" resource type)
			if item.CRN != nil && isVPCInstance(*item.CRN) {
				crns = append(crns, *item.CRN)
				c.log.Debug("Found VPC instance with tag",
					"crn", *item.CRN,
					"name", getItemName(item),
					"tag", tagName)
			}
		}
	}

	c.log.Info("Found instances with tag", "tag", tagName, "count", len(crns))
	return crns, nil
}

// isVPCInstance checks if a CRN belongs to a VPC instance
func isVPCInstance(crn string) bool {
	// VPC instance CRN format: crn:v1:bluemix:public:is:region:account::instance:instance-id
	return strings.Contains(crn, ":is:") && strings.Contains(crn, ":instance:")
}

// getItemName safely extracts the name from a search result item
func getItemName(item globalsearchv2.ResultItem) string {
	// ResultItem doesn't have a Name field directly, extract from CRN or return unknown
	if item.CRN != nil {
		return *item.CRN
	}
	return "unknown"
}

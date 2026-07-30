package launchdarkly

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/launchdarkly/go-sdk-common/v3/ldcontext"
)

// LocalFileSchema defines local feature-flag values used in development mode.
type LocalFileSchema struct {
	Defaults     map[string]bool            `json:"defaults"`
	System       map[string]bool            `json:"system"`
	Institutions map[string]map[string]bool `json:"institutions"`
}

// localClient is a file-backed feature-flag provider used in local development.
type localClient struct {
	mu   sync.RWMutex
	data LocalFileSchema
}

var _ Client = (*localClient)(nil)

// NewLocalClientFromFile initializes a local feature-flag provider from a JSON
// file. It is intended for local development and testing.
func NewLocalClientFromFile(path string) (Client, error) {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" {
		return nil, fmt.Errorf("feature flags file path is required")
	}

	content, err := os.ReadFile(trimmedPath)
	if err != nil {
		return nil, fmt.Errorf("read feature flags file: %w", err)
	}

	var schema LocalFileSchema
	if err := json.Unmarshal(content, &schema); err != nil {
		return nil, fmt.Errorf("parse feature flags file: %w", err)
	}

	if schema.Defaults == nil {
		schema.Defaults = map[string]bool{}
	}
	if schema.System == nil {
		schema.System = map[string]bool{}
	}
	if schema.Institutions == nil {
		schema.Institutions = map[string]map[string]bool{}
	}

	return &localClient{data: schema}, nil
}

// BuildContext is not used by the local provider but required by Client.
func (*localClient) BuildContext(context.Context, string, string, Attributes) ldcontext.Context {
	return ldcontext.Context{}
}

func (c *localClient) IsFeatureFlagEnabled(_ context.Context, featureFlagKey string, institutionUniversalId string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if institutionUniversalId == "system" {
		if value, ok := c.data.System[featureFlagKey]; ok {
			return value
		}
	}

	if institutionFlags, ok := c.data.Institutions[institutionUniversalId]; ok {
		if value, ok := institutionFlags[featureFlagKey]; ok {
			return value
		}
	}

	if value, ok := c.data.Defaults[featureFlagKey]; ok {
		return value
	}

	return false
}

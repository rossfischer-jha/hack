package launchdarkly

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLocalClientIsFeatureFlagEnabled(t *testing.T) {
	t.Parallel()

	client := &localClient{data: LocalFileSchema{
		Defaults: map[string]bool{
			"default-on": true,
		},
		System: map[string]bool{
			"system-on": true,
		},
		Institutions: map[string]map[string]bool{
			"inst-1": {
				"inst-on":    true,
				"default-on": false,
			},
		},
	}}

	testCases := map[string]struct {
		flagKey       string
		institutionID string
		expected      bool
	}{
		"returns system scoped value": {
			flagKey:       "system-on",
			institutionID: "system",
			expected:      true,
		},
		"returns institution scoped value": {
			flagKey:       "inst-on",
			institutionID: "inst-1",
			expected:      true,
		},
		"institution value overrides default": {
			flagKey:       "default-on",
			institutionID: "inst-1",
			expected:      false,
		},
		"returns default when institution value missing": {
			flagKey:       "default-on",
			institutionID: "inst-2",
			expected:      true,
		},
		"returns false when value is missing": {
			flagKey:       "missing-flag",
			institutionID: "inst-1",
			expected:      false,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			actual := client.IsFeatureFlagEnabled(context.Background(), tc.flagKey, tc.institutionID)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestNewLocalClientFromFile(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		content string
		path    string
		err     string
	}{
		"loads file": {
			content: `{"defaults":{"flag-a":true},"system":{"flag-b":false},"institutions":{"inst-1":{"flag-c":true}}}`,
		},
		"fails on empty path": {
			path: "   ",
			err:  "feature flags file path is required",
		},
		"fails on invalid json": {
			content: `{"defaults":`,
			err:     "parse feature flags file",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path := tc.path
			if path == "" {
				path = filepath.Join(t.TempDir(), "flags.json")
				err := os.WriteFile(path, []byte(tc.content), 0o600)
				require.NoError(t, err)
			}

			client, err := NewLocalClientFromFile(path)
			if tc.err != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.err)
				require.Nil(t, client)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, client)
		})
	}
}

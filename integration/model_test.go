package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/timewinder-dev/timewinder/cas"
	"github.com/timewinder-dev/timewinder/model"
)

// TestModelSpecs runs all TOML spec files in testdata as subtests
func TestModelSpecs(t *testing.T) {
	testdataDir := filepath.Join("..", "testdata")

	// Walk through testdata to find all .toml files
	err := filepath.Walk(testdataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip non-.toml files
		if info.IsDir() || !strings.HasSuffix(path, ".toml") {
			return nil
		}

		// Create a subtest for each TOML file
		relPath, _ := filepath.Rel(testdataDir, path)
		testName := strings.TrimSuffix(relPath, ".toml")
		testName = strings.ReplaceAll(testName, string(filepath.Separator), "/")

		t.Run(testName, func(t *testing.T) {
			// Load spec
			spec, err := model.LoadSpecFromFile(path)
			require.NoError(t, err, "Failed to load spec file")

			// Create CAS
			memoryCAS := cas.NewMemoryCAS()
			casStore := cas.NewLRUCache(memoryCAS, 10000)

			// Build executor
			exec, err := spec.BuildExecutor(casStore)
			require.NoError(t, err, "Failed to build executor")

			// Initialize
			err = exec.Initialize()
			require.NoError(t, err, "Failed to initialize executor")

			// Run model checking
			result, err := exec.RunModel()
			require.NoError(t, err, "Error during model checking")
			require.NotNil(t, result, "Result should not be nil")

			// For now, we just verify it runs without panicking
			// Individual tests can assert on specific properties
			t.Logf("Stats: %d transitions, %d unique states, %d duplicates, max depth %d",
				result.Statistics.TotalTransitions,
				result.Statistics.UniqueStates,
				result.Statistics.DuplicateStates,
				result.Statistics.MaxDepth)

			if !result.Success {
				t.Logf("Found %d property violations (this may be expected)", result.Statistics.ViolationCount)
			}
		})

		return nil
	})

	require.NoError(t, err, "Error walking testdata directory")
}

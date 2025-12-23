package vm

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
)

func TestMain(m *testing.M) {
	// Set log level to info for tests (reduce verbosity)
	log.Logger = zerolog.New(os.Stderr).Level(zerolog.InfoLevel)
	os.Exit(m.Run())
}

func TestSmall(t *testing.T) {
	t.Skip("Compilation tests")
	testDir("../testdata/small", t)
}

func TestPracticalTLA(t *testing.T) {
	t.Skip("Compilation tests")
	testDir("../testdata/practical_tla", t)
}

func testDir(dir string, t *testing.T) {
	filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".star") {
			return nil
		}
		name := filepath.Base(path)
		t.Run(name, fileTest(path))
		return nil
	})
}

func fileTest(path string) func(t *testing.T) {
	return func(t *testing.T) {
		f, err := CompilePath(path)
		require.NoError(t, err)
		f.DebugPrint()
	}
}

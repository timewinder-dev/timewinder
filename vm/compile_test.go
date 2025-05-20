package vm

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSmall(t *testing.T) {
	testDir("../testdata/small", t)
}

func TestPracticalTLA(t *testing.T) {
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

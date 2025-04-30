package vm

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"go.starlark.net/syntax"
)

func TestSmall(t *testing.T) {
	filepath.WalkDir("../testdata/small", func(path string, d fs.DirEntry, err error) error {
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
		f, err := os.Open(path)
		require.NoError(t, err)
		defer f.Close()
		opts := syntax.FileOptions{}
		synFile, err := opts.Parse(path, f, 0)
		require.NoError(t, err)
		p, err := buildCompileContextTree(synFile)
		require.NoError(t, err)
		p.DebugPrint()
		t.Logf("%#v", p)
	}
}

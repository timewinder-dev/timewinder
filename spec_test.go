package timewinder

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseSpecInTestdata(t *testing.T) {
	filepath.WalkDir("testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".toml") {
			return nil
		}
		name := filepath.Base(path)
		t.Run(name, testParseSpec(path))
		return nil
	})
}

func testParseSpec(path string) func(t *testing.T) {
	return func(t *testing.T) {
		f, err := os.Open(path)
		require.NoError(t, err)
		s, err := parseSpec(f)
		require.NoError(t, err)
		t.Logf("%#v\n", s)
	}
}

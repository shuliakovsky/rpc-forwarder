package networks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadAll_ParsesYAML(t *testing.T) {
	dir := t.TempDir()
	yml := `
route: /rpc/foo
protocol: evm
nodes:
  - url: https://example.com
    priority: 1
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "foo.yaml"), []byte(yml), 0644))

	cfgs, err := LoadAll(dir)
	require.NoError(t, err)
	require.Contains(t, cfgs, "foo")
	require.Equal(t, "evm", cfgs["foo"].Protocol)
	require.Equal(t, "/rpc/foo", cfgs["foo"].Route)
}

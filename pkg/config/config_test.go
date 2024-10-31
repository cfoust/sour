package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProcess(t *testing.T) {
	// Default config
	_, err := Process([]string{})
	require.NoError(t, err)

	dir := t.TempDir()

	// yaml config
	{
		yaml := filepath.Join(dir, "config.yaml")
		err = os.WriteFile(yaml, []byte(`
server:
  ingress:
    web:
      port: 1234
`), 0644)
		require.NoError(t, err)
		_, err = Process([]string{yaml})
		require.NoError(t, err)
	}

	// json config
	{
		json := filepath.Join(dir, "config.json")
		err = os.WriteFile(json, []byte(`{
  "server": {
    "ingress": {
      "web": {
        "port": 1235
      }
    }
  }
}`), 0644)
		require.NoError(t, err)
		_, err = Process([]string{json})
		require.NoError(t, err)
	}

	// multiple yaml
	{
		yaml1 := filepath.Join(dir, "config1.yaml")
		err = os.WriteFile(yaml1, []byte(`
server:
  ingress:
    web:
      port: 1234
`), 0644)
		require.NoError(t, err)

		yaml2 := filepath.Join(dir, "config2.yaml")
		err = os.WriteFile(yaml2, []byte(`
server:
  serverDescription: "Hello, World!"
`), 0644)
		require.NoError(t, err)
		_, err = Process([]string{yaml1, yaml2})
		require.NoError(t, err)
	}

	// Invalid config
	_, err = Process([]string{})
}

package networks

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

func LoadAll(dir string) (map[string]NetworkConfig, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`\$\{([A-Z0-9_]+)\}`)
	out := map[string]NetworkConfig{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		b = re.ReplaceAllFunc(b, func(m []byte) []byte {
			k := re.FindSubmatch(m)[1]
			return []byte(os.Getenv(string(k)))
		})
		var nc NetworkConfig
		if err := yaml.Unmarshal(b, &nc); err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		if nc.Route == "" || nc.Protocol == "" || len(nc.Nodes) == 0 {
			return nil, fmt.Errorf("%s: invalid network config", e.Name())
		}
		for i := range nc.Nodes {
			if nc.Nodes[i].Priority == 0 {
				nc.Nodes[i].Priority = 1
			}
			if nc.Nodes[i].Headers == nil {
				nc.Nodes[i].Headers = map[string]string{}
			}
		}
		key := strings.TrimSuffix(e.Name(), ".yaml")
		out[key] = nc
	}
	return out, nil
}

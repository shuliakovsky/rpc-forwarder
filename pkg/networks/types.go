package networks

type Node struct {
	URL       string            `yaml:"url" json:"url"`
	Priority  int               `yaml:"priority" json:"priority"`
	IsPrivate bool              `yaml:"isPrivate" json:"isPrivate"`
	Headers   map[string]string `yaml:"headers" json:"headers"`
	Tor       bool              `yaml:"tor" json:"tor"`
}

type NetworkConfig struct {
	Route     string `yaml:"route" json:"route"`
	Protocol  string `yaml:"protocol" json:"protocol"` // evm|btc
	Nodes     []Node `yaml:"nodes" json:"nodes"`
	TimeoutMs int    `yaml:"timeoutMs" json:"timeoutMs"`
}

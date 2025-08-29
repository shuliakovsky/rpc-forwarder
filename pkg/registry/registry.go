package registry

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/shuliakovsky/rpc-forwarder/pkg/networks"
)

type NodeWithPing struct {
	networks.Node
	Alive bool  `json:"alive"`
	Ping  int64 `json:"ping"` // ms
}

type DiscoveredNode struct {
	Node      networks.Node
	ExpiresAt time.Time
}

type NetworkState struct {
	Protocol   string
	Route      string
	All        []networks.Node
	Best       []NodeWithPing
	Discovered []DiscoveredNode
}

type Registry struct {
	mu    sync.RWMutex
	State map[string]*NetworkState // key: network name (eth, btc)
}

func New() *Registry { return &Registry{State: map[string]*NetworkState{}} }

func (r *Registry) InitFromConfigs(cfgs map[string]networks.NetworkConfig) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for name, c := range cfgs {
		copyNodes := make([]networks.Node, len(c.Nodes))
		copy(copyNodes, c.Nodes)
		r.State[name] = &NetworkState{
			Protocol: c.Protocol,
			Route:    c.Route,
			All:      copyNodes,
			Best:     nil,
		}
	}
}

func (r *Registry) SetBest(name string, best []NodeWithPing) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.State[name]; ok {
		s.Best = best
	}
}

func (r *Registry) Best(name string) []NodeWithPing {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if s, ok := r.State[name]; ok && len(s.Best) > 0 {
		out := make([]NodeWithPing, len(s.Best))
		copy(out, s.Best)
		return out
	}
	return nil
}

func (r *Registry) All() map[string]*NetworkState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make(map[string]*NetworkState, len(r.State))
	for k, v := range r.State {
		// неглубокая копия ссылочного типа не нужна в ответе; используем как есть
		out[k] = v
	}
	return out
}

func PickFastestPerPriority(nodes []NodeWithPing) []NodeWithPing {
	m := map[int][]NodeWithPing{}
	for _, n := range nodes {
		if !n.Alive {
			continue
		}
		m[n.Priority] = append(m[n.Priority], n)
	}
	var best []NodeWithPing
	for _, group := range m {
		fastest := group[0]
		for _, cur := range group[1:] {
			if cur.Ping < fastest.Ping {
				fastest = cur
			}
		}
		best = append(best, fastest)
	}
	sort.Slice(best, func(i, j int) bool { return best[i].Priority < best[j].Priority })
	return best
}
func (r *Registry) AddNetwork(cfg networks.NetworkConfig, best []NodeWithPing) {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := strings.TrimPrefix(cfg.Route, "/rpc/")
	r.State[key] = &NetworkState{
		Protocol: cfg.Protocol,
		Route:    cfg.Route,
		All:      cfg.Nodes,
		Best:     best,
	}
}

func (r *Registry) ProtocolOf(name string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if s, ok := r.State[name]; ok {
		return s.Protocol
	}
	return ""
}

func (r *Registry) AllBestOrEmpty() map[string][]NodeWithPing {
	res := make(map[string][]NodeWithPing)
	for name, st := range r.All() {
		if len(st.Best) == 0 {
			res[name] = []NodeWithPing{}
			continue
		}
		res[name] = st.Best
	}
	return res
}

func (r *Registry) AddNode(net string, n networks.Node) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.State[net]; ok {
		s.All = append(s.All, n)
	}
}

func (r *Registry) AppendBest(net string, n NodeWithPing) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if s, ok := r.State[net]; ok {
		s.Best = append(s.Best, n)
	}
}
func (r *Registry) PruneAndMerge(ttl time.Duration) {
	now := time.Now()
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, st := range r.State {
		// чистим устаревшие
		var fresh []DiscoveredNode
		for _, dn := range st.Discovered {
			if dn.ExpiresAt.After(now) {
				fresh = append(fresh, dn)
			}
		}
		st.Discovered = fresh

		// мержим с All
		for _, dn := range st.Discovered {
			// без дубликатов
			dup := false
			for _, n := range st.All {
				if n.URL == dn.Node.URL {
					dup = true
					break
				}
			}
			if !dup {
				st.All = append(st.All, dn.Node)
			}
		}
	}
}

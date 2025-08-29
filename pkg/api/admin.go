package api

import (
	"encoding/json"
	"github.com/shuliakovsky/rpc-forwarder/pkg/health"
	"github.com/shuliakovsky/rpc-forwarder/pkg/networks"
	"github.com/shuliakovsky/rpc-forwarder/pkg/registry"
	"go.uber.org/zap"
	"net/http"
	"strings"
)

type Admin struct {
	Reg      *registry.Registry
	Checker  *health.Checker
	AdminKey string
	Logger   *zap.Logger
}

func NewAdmin(reg *registry.Registry, checker *health.Checker, key string, logger *zap.Logger) *Admin {
	return &Admin{Reg: reg, Checker: checker, AdminKey: key, Logger: logger}
}

func (a *Admin) auth(w http.ResponseWriter, r *http.Request) bool {
	if r.Header.Get("x-admin-key") != a.AdminKey {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}
	return true
}

// POST /admin/networks
// POST /admin/networks
func (a *Admin) AddNetwork(w http.ResponseWriter, r *http.Request) {
	if !a.auth(w, r) {
		return
	}
	var nc networks.NetworkConfig
	if err := json.NewDecoder(r.Body).Decode(&nc); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if nc.Route == "" || nc.Protocol == "" || len(nc.Nodes) == 0 {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}

	// обрезаем /rpc/ из начала
	nc.Route = strings.TrimPrefix(nc.Route, "/rpc/")

	// health‑check
	best := a.Checker.UpdateNetwork(nc.Protocol, nc.Nodes)
	if len(best) == 0 {
		http.Error(w, "no healthy nodes", http.StatusBadRequest)
		return
	}
	a.Reg.AddNetwork(nc, best)
	a.Logger.Info("admin_add_network", zap.String("route", nc.Route), zap.Int("healthy_nodes", len(best)))
	writeJSON(w, 200, map[string]any{"status": "added", "healthyNodes": best})
}

// POST /admin/{network}/nodes
func (a *Admin) AddNode(w http.ResponseWriter, r *http.Request) {
	if !a.auth(w, r) {
		return
	}
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/admin/"), "/")
	if len(parts) < 2 {
		http.Error(w, "bad path", http.StatusBadRequest)
		return
	}
	network := parts[0]
	var node networks.Node
	if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	// health‑check
	best := a.Checker.UpdateNetwork(a.Reg.ProtocolOf(network), []networks.Node{node})
	if len(best) == 0 {
		http.Error(w, "node not healthy", http.StatusBadRequest)
		return
	}
	a.Reg.AddNode(network, node)
	a.Reg.AppendBest(network, best[0])
	a.Logger.Info("admin_add_node", zap.String("network", network), zap.String("url", node.URL))
	writeJSON(w, 200, map[string]any{"status": "added", "node": best[0]})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("content-type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

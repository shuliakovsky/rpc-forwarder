package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	httpSwagger "github.com/swaggo/http-swagger"

	"github.com/shuliakovsky/rpc-forwarder/pkg/api"
	"github.com/shuliakovsky/rpc-forwarder/pkg/bootstrap"
	"github.com/shuliakovsky/rpc-forwarder/pkg/docs"
	_ "github.com/shuliakovsky/rpc-forwarder/pkg/docs"
	"github.com/shuliakovsky/rpc-forwarder/pkg/gossip"
	"github.com/shuliakovsky/rpc-forwarder/pkg/health"
	"github.com/shuliakovsky/rpc-forwarder/pkg/leader"
	"github.com/shuliakovsky/rpc-forwarder/pkg/metrics"
	"github.com/shuliakovsky/rpc-forwarder/pkg/networks"
	"github.com/shuliakovsky/rpc-forwarder/pkg/peers"
	"github.com/shuliakovsky/rpc-forwarder/pkg/registry"
)

type ActiveNodesResponse struct {
	Route string                  `json:"route"`
	Nodes []registry.NodeWithPing `json:"nodes"`
}

type limiter struct {
	sem chan struct{}
}

func newLimiter(n int) *limiter {
	return &limiter{sem: make(chan struct{}, n)}
}
func (l *limiter) acquire() { l.sem <- struct{}{} }
func (l *limiter) release() { <-l.sem }

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Base env
	podIP := getEnv("POD_IP", "127.0.0.1")
	podName := getEnv("POD_NAME", "dev-node")
	secret := getEnv("SHARED_SECRET", "devsecret")
	bootstrapURL := getEnv("BOOTSTRAP_URL", "")
	torSocks := getEnv("TOR_SOCKS5", "127.0.0.1:9050")
	adminKey := getEnv("ADMIN_API_KEY", "changeme")

	nodeID := uuid.NewString()
	internalAddr := fmt.Sprintf("%s:%d", podIP, 8080)

	logger.Info("Node started",
		zap.String("podName", podName),
		zap.String("nodeID", nodeID),
		zap.String("internalAddr", internalAddr),
	)

	// Peers store + bootstrap
	peerStore := peers.NewStore()
	peerStore.Add(peers.Peer{ID: nodeID, Addr: internalAddr})

	if bootstrapURL != "" {
		if list, err := bootstrap.Announce(bootstrapURL, nodeID, podName, internalAddr, secret, logger); err != nil {
			logger.Warn("Boostrap error", zap.Error(err))
		} else {
			for _, p := range list {
				if p.ID != nodeID {
					peerStore.Add(p)
				}
			}
		}
	}

	// === Networks registry (EVM/BTC) ===
	cfgs, err := networks.LoadAll("configs/networks")
	if err != nil {
		logger.Fatal("networks_load_error", zap.Error(err))
	}
	// создаём реестр и заполняем из конфигов
	reg := registry.New()
	reg.InitFromConfigs(cfgs)

	// создаём health checker
	checker := health.New(10*time.Second, torSocks, logger)
	adminAPI := api.NewAdmin(reg, checker, adminKey, logger)

	limits := map[string]*limiter{
		"tatum.io":      newLimiter(1),
		"alchemyapi.io": newLimiter(1),
	}
	defaultLimiter := newLimiter(5)

	var wg sync.WaitGroup
	for name, st := range reg.All() {
		wg.Add(1)
		go func(name string, st *registry.NetworkState) {
			defer wg.Done()

			// определяем лимитер по URL первой ноды
			lim := defaultLimiter
			if len(st.All) > 0 {
				for domain, l := range limits {
					if strings.Contains(st.All[0].URL, domain) {
						lim = l
						break
					}
				}
			}

			lim.acquire()
			defer lim.release()

			best := checker.UpdateNetwork(st.Protocol, st.All)
			reg.SetBest(name, best)

			metrics.TotalNodes.WithLabelValues(name).Set(float64(len(st.All) + len(st.Discovered)))
			metrics.HealthyNodes.WithLabelValues(name).Set(float64(len(best)))

			logger.Info("health_initialized",
				zap.String("network", name),
				zap.Int("best_count", len(best)),
			)
		}(name, st)
	}
	wg.Wait()
	go func() {
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for range t.C {
			reg.PruneAndMerge(10 * time.Minute) // чистим и мержим discovered
			var wg sync.WaitGroup
			for name, st := range reg.All() {
				wg.Add(1)
				go func(name string, st *registry.NetworkState) {
					defer wg.Done()

					// определяем лимитер
					lim := defaultLimiter
					if len(st.All) > 0 {
						for domain, l := range limits {
							if strings.Contains(st.All[0].URL, domain) {
								lim = l
								break
							}
						}
					}

					lim.acquire()
					defer lim.release()

					best := checker.UpdateNetwork(st.Protocol, st.All)
					reg.SetBest(name, best)
					metrics.TotalNodes.WithLabelValues(name).Set(float64(len(st.All) + len(st.Discovered)))
					metrics.HealthyNodes.WithLabelValues(name).Set(float64(len(best)))

					logger.Info("health_update", zap.String("network", name), zap.Int("best_count", len(best)))
				}(name, st)
			}
			wg.Wait()
		}

	}()

	// === HTTP API ===
	public := api.NewPublic(reg)
	proxy := api.NewProxy(reg, logger)

	// Core control endpoints
	http.Handle("/announce", bootstrap.NewHandler(peerStore, nodeID, internalAddr, secret, logger))
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("ok")) })
	http.HandleFunc("/gossip", gossip.Handler(peerStore, logger))
	http.HandleFunc("/heartbeat", leader.Handler(logger))
	http.Handle("/swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/swagger.json"), // путь до спеки
		httpSwagger.InstanceName("swagger"),
	))

	http.HandleFunc("/swagger/swagger.json", docs.JSONHandler)

	// Gossip state exchange (best nodes per network)
	http.HandleFunc("/gossip-state", gossip.StateHandler(reg, logger))
	go gossip.Publisher(reg, peerStore, nodeID, logger)

	// Public routes
	http.HandleFunc("/networkfees", public.NetworkFees)
	http.HandleFunc("/active-nodes", func(w http.ResponseWriter, r *http.Request) {
		res := make(map[string]struct {
			Route string                  `json:"route"`
			Nodes []registry.NodeWithPing `json:"nodes"`
		})
		for name, st := range reg.All() {
			nodes := st.Best
			if len(nodes) == 0 {
				nodes = []registry.NodeWithPing{}
			}
			res[name] = struct {
				Route string                  `json:"route"`
				Nodes []registry.NodeWithPing `json:"nodes"`
			}{
				Route: st.Route,
				Nodes: registry.SanitizeNodes(nodes),
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(res)
	})

	// admin
	http.HandleFunc("/admin/networks", adminAPI.AddNetwork)
	http.HandleFunc("/admin/", adminAPI.AddNode) // /admin/{network}/nodes

	// Proxy routes from configs (e.g., /rpc/eth, /rpc/btc)
	for name, st := range reg.All() {
		route := st.Route
		logger.Info("route_registered", zap.String("network", name), zap.String("route", route))
		http.HandleFunc(route, proxy.Serve)
	}

	// Cluster background processes
	go gossip.Start(peerStore, nodeID, logger)
	go leader.HeartbeatLoop(peerStore, nodeID, 30*time.Second, logger)

	metrics.Init()
	http.Handle("/metrics", metrics.Handler())

	// Run server
	addr := ":8080"
	logger.Info("Слушаем", zap.String("addr", addr))
	if err := http.ListenAndServe(addr, nil); err != nil && err != http.ErrServerClosed {
		logger.Fatal("Сервер упал", zap.Error(err))
	}
}

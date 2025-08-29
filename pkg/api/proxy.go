package api

import (
	"bytes"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/shuliakovsky/rpc-forwarder/pkg/metrics"
	"github.com/shuliakovsky/rpc-forwarder/pkg/registry"
)

type Proxy struct {
	Reg    *registry.Registry
	Logger *zap.Logger
	Client *http.Client
}

func NewProxy(reg *registry.Registry, logger *zap.Logger) *Proxy {
	return &Proxy{
		Reg:    reg,
		Logger: logger,
		Client: &http.Client{Timeout: 8 * time.Second},
	}
}

// Handle /rpc/{network}
func (p *Proxy) Serve(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(parts) < 2 || parts[0] != "rpc" {
		http.NotFound(w, r)
		return
	}
	network := parts[1]
	candidates := p.Reg.Best(network)
	if len(candidates) == 0 {
		p.Logger.Error("no_available_nodes", zap.String("network", network))
		http.Error(w, "no available nodes", http.StatusServiceUnavailable)
		return
	}

	body, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	headers := r.Header.Clone()

	for i, node := range candidates {
		req, _ := http.NewRequest(r.Method, node.URL, bytes.NewReader(body))
		req.Header = headers.Clone()
		for k, v := range node.Headers {
			req.Header.Set(k, v)
		}
		if req.Header.Get("content-type") == "" {
			req.Header.Set("content-type", "application/json")
		}

		start := time.Now()
		resp, err := p.Client.Do(req)
		if err != nil {
			p.Logger.Warn("upstream_error",
				zap.String("network", network),
				zap.String("upstream", node.URL),
				zap.Int("attempt", i+1),
				zap.Error(err),
			)
			metrics.ProxyFail.WithLabelValues(network).Inc()
			continue
		}

		buf, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		lat := time.Since(start).Milliseconds()

		// Rate-limit и 5xx → фоллбэк на следующую ноду
		if isRateLimited(resp, buf) || resp.StatusCode >= 500 {
			p.Logger.Warn("upstream_rate_or_5xx",
				zap.String("network", network),
				zap.String("upstream", node.URL),
				zap.Int("status", resp.StatusCode),
				zap.Int("attempt", i+1),
				zap.Int64("latency_ms", lat),
			)
			metrics.ProxyFail.WithLabelValues(network).Inc()
			continue
		}

		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		w.Write(buf)

		p.Logger.Info("proxy_success",
			zap.String("network", network),
			zap.String("upstream", node.URL),
			zap.Int("status", resp.StatusCode),
			zap.Int("attempt", i+1),
			zap.Int64("latency_ms", lat),
		)
		metrics.ProxySuccess.WithLabelValues(network).Inc()
		return
	}

	http.Error(w, "all upstreams failed", http.StatusBadGateway)
	metrics.ProxyFail.WithLabelValues(network).Inc()
}

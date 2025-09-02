package api

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/shuliakovsky/rpc-forwarder/pkg/adapters"
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

// Handle /{network}[/*tail]
func (p *Proxy) Serve(w http.ResponseWriter, r *http.Request) {
	// parse path: /{network}/optional/tail...
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(parts) < 1 {
		http.NotFound(w, r)
		return
	}
	network := parts[0]
	var tail string
	if len(parts) > 1 {
		tail = strings.Join(parts[1:], "/")
	}

	candidates := p.Reg.Best(network)
	if len(candidates) == 0 {
		p.Logger.Error("no_available_nodes", zap.String("network", network))
		http.Error(w, "no available nodes", http.StatusServiceUnavailable)
		return
	}

	// read original body once
	origBody, _ := io.ReadAll(r.Body)
	_ = r.Body.Close()

	start := LogRequest(p.Logger, "proxy", r.Method, r.URL.Path, origBody)

	// adapt request per network
	ad := adapters.Adapt(network, tail, r.Method, r.Header, origBody, p.Logger)

	// Ограничение по провайдерам
	if len(ad.AllowedHostSubstr) > 0 {
		filtered := make([]registry.NodeWithPing, 0, len(candidates))
		for _, n := range candidates {
			ok := false
			for _, sub := range ad.AllowedHostSubstr {
				if strings.Contains(strings.ToLower(n.URL), strings.ToLower(sub)) {
					ok = true
					break
				}
			}
			if ok {
				filtered = append(filtered, n)
			}
		}
		candidates = filtered
		if len(candidates) == 0 {
			p.Logger.Warn("no_upstreams_match_adapter_filter", zap.String("network", network))
			http.Error(w, "no allowed upstreams", http.StatusServiceUnavailable)
			return
		}
	}

	// common headers snapshot
	inHeaders := r.Header.Clone()
	// enforce adapter headers
	for k, v := range ad.Headers {
		inHeaders.Set(k, v)
	}

	rawQuery := r.URL.RawQuery

	for i, node := range candidates {
		upstreamURL := buildUpstreamURL(node.URL, ad.Tail, rawQuery)

		// таймаут из конфига или дефолт
		perNodeTimeout := time.Duration(p.Reg.TimeoutMs(network)) * time.Millisecond
		if perNodeTimeout <= 0 {
			perNodeTimeout = defaultTimeoutFor(network)
		}

		ctx, cancel := context.WithTimeout(r.Context(), perNodeTimeout)
		req, _ := http.NewRequestWithContext(ctx, ad.Method, upstreamURL, bytes.NewReader(ad.Body))
		req.Header = inHeaders.Clone()
		for k, v := range node.Headers {
			req.Header.Set(k, v)
		}
		if req.Header.Get("content-type") == "" &&
			(ad.Method == http.MethodPost || ad.Method == http.MethodPut) {
			req.Header.Set("content-type", "application/json")
		}

		resp, err := p.Client.Do(req)
		if err != nil {
			cancel()
			p.Logger.Warn("upstream_timeout_or_err",
				zap.String("network", network),
				zap.String("upstream", upstreamURL),
				zap.Int("attempt", i+1),
				zap.Error(err),
			)
			metrics.ProxyFail.WithLabelValues(network).Inc()
			continue
		}

		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		cancel()

		lat := time.Since(start).Milliseconds()

		LogResponse(p.Logger, "proxy", resp.StatusCode, respBody, start)

		if isRateLimited(resp, respBody) || resp.StatusCode >= 500 {
			p.Logger.Warn("upstream_rate_or_5xx",
				zap.String("network", network),
				zap.String("upstream", upstreamURL),
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
		_, _ = w.Write(respBody)

		p.Logger.Info("proxy_success",
			zap.String("network", network),
			zap.String("upstream", upstreamURL),
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

func buildUpstreamURL(base, tail, rawQuery string) string {
	u := strings.TrimRight(base, "/")
	if tail != "" {
		u += "/" + strings.TrimLeft(tail, "/")
	}
	if rawQuery != "" {
		u += "?" + rawQuery
	}
	return u
}

func defaultTimeoutFor(network string) time.Duration {
	switch strings.ToLower(network) {
	case "sol":
		return 800 * time.Millisecond
	case "eth", "evm", "bsc", "polygon", "fantom":
		return 1500 * time.Millisecond
	case "trx":
		return 1500 * time.Millisecond
	case "btc", "doge", "ltc":
		return 2000 * time.Millisecond
	default:
		return 1500 * time.Millisecond
	}
}

package health

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/shuliakovsky/rpc-forwarder/pkg/networks"
	"github.com/shuliakovsky/rpc-forwarder/pkg/registry"
	"go.uber.org/zap"
	"golang.org/x/net/proxy"
)

type Checker struct {
	Timeout   time.Duration
	TorSocks5 string
	Logger    *zap.Logger
}

func New(timeout time.Duration, tor string, logger *zap.Logger) *Checker {
	if tor == "" {
		tor = "127.0.0.1:9050"
	}
	return &Checker{Timeout: timeout, TorSocks5: tor, Logger: logger}
}

func (c *Checker) httpClient(tor bool) (*http.Client, error) {
	transport := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     60 * time.Second,
		TLSHandshakeTimeout: 8 * time.Second,
	}
	if tor {
		dialer, err := proxy.SOCKS5("tcp", c.TorSocks5, nil, proxy.Direct)
		if err != nil {
			return nil, err
		}
		transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			return dialer.Dial(network, addr)
		}
	}
	return &http.Client{Transport: transport, Timeout: c.Timeout}, nil
}

// === EVM ===
func (c *Checker) checkEVM(n networks.Node) (bool, int64) {
	cl, _ := c.httpClient(n.Tor)
	start := time.Now()
	body := map[string]any{"jsonrpc": "2.0", "method": "eth_blockNumber", "id": 1}
	js, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", n.URL, bytes.NewReader(js))
	for k, v := range n.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("content-type", "application/json")
	resp, err := cl.Do(req)
	if err != nil {
		return false, 0
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return false, 0
	}
	var out struct{ Result string }
	_ = json.NewDecoder(resp.Body).Decode(&out)
	if out.Result == "" {
		return false, 0
	}
	return true, time.Since(start).Milliseconds()
}

// === BTC ===
func (c *Checker) checkBTC(n networks.Node) (bool, int64) {
	cl, _ := c.httpClient(n.Tor)
	start := time.Now()
	u := strings.TrimSuffix(n.URL, "/")
	if strings.Contains(n.URL, "blockstream.info") || strings.HasSuffix(n.URL, "/api") {
		resp, err := cl.Get(u + "/blocks/tip/height")
		if err != nil {
			return false, 0
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			return false, 0
		}
		io.Copy(io.Discard, resp.Body)
		return true, time.Since(start).Milliseconds()
	}
	req, _ := http.NewRequest("GET", u+"/v3/bitcoin/address/balance/test", nil)
	for k, v := range n.Headers {
		req.Header.Set(k, v)
	}
	resp, err := cl.Do(req)
	if err != nil || resp.StatusCode == http.StatusUnauthorized {
		return false, 0
	}
	defer resp.Body.Close()
	return true, time.Since(start).Milliseconds()
}

// === TRX ===
func (c *Checker) checkTRX(n networks.Node) (bool, int64) {
	cl, _ := c.httpClient(n.Tor)
	start := time.Now()
	if strings.Contains(n.URL, "tatum.io") {
		req, _ := http.NewRequest("GET", strings.TrimSuffix(n.URL, "/")+"/wallet/getnodeinfo", nil)
		for k, v := range n.Headers {
			req.Header.Set(k, v)
		}
		resp, err := cl.Do(req)
		if err != nil || resp.StatusCode == http.StatusUnauthorized {
			return false, 0
		}
		defer resp.Body.Close()
		return true, time.Since(start).Milliseconds()
	}
	payload := []byte(`{"jsonrpc":"2.0","method":"wallet/getnowblock"}`)
	req, _ := http.NewRequest("POST", n.URL, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range n.Headers {
		req.Header.Set(k, v)
	}
	resp, err := cl.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return false, 0
	}
	defer resp.Body.Close()
	return true, time.Since(start).Milliseconds()
}

// === LTC ===
func (c *Checker) checkLTC(n networks.Node) (bool, int64) {
	cl, _ := c.httpClient(n.Tor)
	start := time.Now()
	u := strings.TrimSuffix(n.URL, "/")
	if strings.Contains(n.URL, "tatum.io") {
		req, _ := http.NewRequest("GET", u+"/v3/litecoin/address/balance/test", nil)
		for k, v := range n.Headers {
			req.Header.Set(k, v)
		}
		resp, err := cl.Do(req)
		if err != nil || resp.StatusCode == http.StatusUnauthorized {
			return false, 0
		}
		defer resp.Body.Close()
		return true, time.Since(start).Milliseconds()
	}
	resp, err := cl.Get(u + "/block/tip/height")
	if err != nil || resp.StatusCode != http.StatusOK {
		return false, 0
	}
	defer resp.Body.Close()
	return true, time.Since(start).Milliseconds()
}

// === DOGE ===
func (c *Checker) checkDOGE(n networks.Node) (bool, int64) {
	cl, _ := c.httpClient(n.Tor)
	start := time.Now()
	u := strings.TrimSuffix(n.URL, "/")
	if strings.Contains(n.URL, "tatum.io") {
		req, _ := http.NewRequest("GET", u+"/v3/dogecoin/address/balance/test", nil)
		for k, v := range n.Headers {
			req.Header.Set(k, v)
		}
		resp, err := cl.Do(req)
		if err != nil || resp.StatusCode == http.StatusUnauthorized {
			return false, 0
		}
		defer resp.Body.Close()
		return true, time.Since(start).Milliseconds()
	}
	resp, err := cl.Get(u + "/api/v1/block/count")
	if err != nil || resp.StatusCode != http.StatusOK {
		return false, 0
	}
	defer resp.Body.Close()
	return true, time.Since(start).Milliseconds()
}

// === SOL ===
func (c *Checker) checkSOL(n networks.Node) (bool, int64) {
	cl, _ := c.httpClient(n.Tor)
	start := time.Now()
	body := map[string]any{"jsonrpc": "2.0", "method": "getSlot", "id": 1}
	js, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", n.URL, bytes.NewReader(js))
	for k, v := range n.Headers {
		req.Header.Set(k, v)
	}
	req.Header.Set("content-type", "application/json")
	resp, err := cl.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return false, 0
	}
	defer resp.Body.Close()
	return true, time.Since(start).Milliseconds()
}

// === UpdateNetwork ===
func (c *Checker) UpdateNetwork(protocol string, nodes []networks.Node) []registry.NodeWithPing {
	res := make([]registry.NodeWithPing, 0, len(nodes))
	for _, n := range nodes {
		var alive bool
		var ping int64
		switch protocol {
		case "evm":
			alive, ping = c.checkEVM(n)
		case "btc":
			alive, ping = c.checkBTC(n)
		case "trx":
			alive, ping = c.checkTRX(n)
		case "ltc":
			alive, ping = c.checkLTC(n)
		case "doge":
			alive, ping = c.checkDOGE(n)
		case "sol":
			alive, ping = c.checkSOL(n)
		default:
			alive, ping = false, 0
		}
		res = append(res, registry.NodeWithPing{Node: n, Alive: alive, Ping: ping})
	}
	return registry.PickFastestPerPriority(res)
}

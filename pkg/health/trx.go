package health

import (
	"bytes"
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/shuliakovsky/rpc-forwarder/pkg/networks"
)

// === TRX ===
func (c *Checker) checkTRX(n networks.Node, timeout time.Duration) (bool, int64) {
	cl, _ := c.httpClient(n.Tor, timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	if strings.Contains(n.URL, "tatum.io") {
		req, _ := http.NewRequestWithContext(ctx, "GET", strings.TrimSuffix(n.URL, "/")+"/wallet/getnodeinfo", nil)
		for k, v := range n.Headers {
			req.Header.Set(k, v)
		}
		resp, err := cl.Do(req)
		if err != nil {
			if isFatalNetErr(err) {
				c.markDrop(n.URL)
			}
			return false, 0
		}
		defer resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			if isFatalHTTPStatus(resp.StatusCode) {
				c.markDrop(n.URL)
			}
			return false, 0
		}
		return true, time.Since(start).Milliseconds()
	}
	payload := []byte(`{"jsonrpc":"2.0","method":"wallet/getnowblock"}`)
	req, _ := http.NewRequestWithContext(ctx, "POST", n.URL, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range n.Headers {
		req.Header.Set(k, v)
	}
	resp, err := cl.Do(req)
	if err != nil {
		if isFatalNetErr(err) {
			c.markDrop(n.URL)
		}
		return false, 0
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		if isFatalHTTPStatus(resp.StatusCode) {
			c.markDrop(n.URL)
		}
		return false, 0
	}
	return true, time.Since(start).Milliseconds()
}

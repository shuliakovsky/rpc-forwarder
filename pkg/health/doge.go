package health

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/shuliakovsky/rpc-forwarder/pkg/networks"
)

// === DOGE ===
func (c *Checker) checkDOGE(n networks.Node, timeout time.Duration) (bool, int64) {
	cl, _ := c.httpClient(n.Tor, timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	start := time.Now()
	u := strings.TrimSuffix(n.URL, "/")
	if strings.Contains(n.URL, "tatum.io") {
		req, _ := http.NewRequestWithContext(ctx, "GET", u+"/v3/dogecoin/address/balance/test", nil)
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
	req, _ := http.NewRequestWithContext(ctx, "GET", u+"/api/v1/block/count", nil)
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

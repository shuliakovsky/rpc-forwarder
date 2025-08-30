package adapters

import (
	"net/http"
	"strings"

	"go.uber.org/zap"
)

// Conservative adapter: pass explicit REST paths as is,
// add two safe scenarios: /fees and /balance/{address} (→ Tatum),
// otherwise — fallback to default behavior.
func adaptBTC(tail, method string, _ http.Header, body []byte, logger *zap.Logger) Result {
	ltail := strings.ToLower(strings.TrimPrefix(tail, "/"))

	// 1) REST-paths — as is
	if ltail == "" || ltail == "/" {
		logger.Debug("btc_adapter_default_tip_height")
		return Result{
			Tail:              "blocks/tip/height",
			Method:            http.MethodGet,
			Body:              nil,
			Headers:           map[string]string{},
			AllowedHostSubstr: nil,
		}
	}
	if strings.HasPrefix(ltail, "rest/") ||
		strings.HasPrefix(ltail, "blocks/") ||
		strings.HasPrefix(ltail, "tx/") ||
		strings.HasPrefix(ltail, "address/") {
		return Result{
			Tail:              tail,
			Method:            method,
			Body:              clone(body),
			Headers:           map[string]string{},
			AllowedHostSubstr: nil,
		}
	}

	// 2) extension: BTC fees  → Tatum
	// GET /btc/fees → https://api.tatum.io/v3/blockchain/fee/BTC
	if ltail == "fees" {
		logger.Debug("btc_adapter_fees_tatum_only")
		return Result{
			Tail:              "v3/blockchain/fee/BTC",
			Method:            http.MethodGet,
			Body:              nil,
			Headers:           map[string]string{},
			AllowedHostSubstr: []string{"tatum.io"},
		}
	}

	// 3) extension: balance of the address BTC → Tatum
	// GET /btc/balance/{address} → https://api.tatum.io/v3/bitcoin/address/balance/{address}
	if strings.HasPrefix(ltail, "balance/") {
		addr := strings.TrimPrefix(ltail, "balance/")
		if addr != "" {
			logger.Debug("btc_adapter_balance_tatum_only", zap.String("address", addr))
			return Result{
				Tail:              "v3/bitcoin/address/balance/" + addr,
				Method:            http.MethodGet,
				Body:              nil,
				Headers:           map[string]string{},
				AllowedHostSubstr: []string{"tatum.io"},
			}
		}
	}

	// 4) Default behaviour
	return Result{
		Tail:              tail,
		Method:            method,
		Body:              clone(body),
		Headers:           map[string]string{},
		AllowedHostSubstr: nil,
	}
}

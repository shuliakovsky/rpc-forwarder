package adapters

import (
	"encoding/json"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

func adaptSOL(tail, method string, _ http.Header, body []byte, logger *zap.Logger) Result {
	j := readJSON(body)

	// in case of non JSON-RPC or empty method — default behaviour getSlot
	if j == nil || asString(j["method"]) == "" {
		logger.Debug("sol_adapter_default_getslot")
		payload := map[string]any{
			"jsonrpc": "2.0",
			"id":      1,
			"method":  "getSlot",
			"params":  []any{map[string]string{"commitment": "finalized"}},
		}
		return Result{
			Tail:    "",
			Method:  "POST",
			Body:    mustJSON(payload),
			Headers: ensureJSON(nil),
		}
	}

	// if method is  getBalance, add commitment, in case in not exists
	if strings.EqualFold(asString(j["method"]), "getBalance") {
		if params, ok := j["params"].([]any); ok && len(params) == 1 {
			params = append(params, map[string]string{"commitment": "finalized"})
			j["params"] = params
			logger.Debug("sol_adapter_added_commitment")
		}
	}

	b, _ := json.Marshal(j)
	return Result{
		Tail:    tail,
		Method:  method,
		Body:    b,
		Headers: ensureJSON(nil),
	}
}

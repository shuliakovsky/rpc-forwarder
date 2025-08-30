package adapters

import (
	"net/http"
	"strings"

	"go.uber.org/zap"
)

func adaptDOGE(tail, method string, _ http.Header, body []byte, logger *zap.Logger) Result {
	ltail := strings.ToLower(strings.TrimPrefix(tail, "/"))

	// no tail — default: current block
	if ltail == "" {
		logger.Debug("doge_adapter_default_height")
		return Result{
			Tail:    "rest/chaininfo.json",
			Method:  "GET",
			Body:    nil,
			Headers: map[string]string{},
		}
	}

	// rest/ or api/ in the beginning of the path — as is to the REST
	if strings.HasPrefix(ltail, "rest/") || strings.HasPrefix(ltail, "api/") {
		return Result{
			Tail:    tail,
			Method:  method,
			Body:    clone(body),
			Headers: map[string]string{},
		}
	}

	// Default JSON-RPC
	return Result{
		Tail:    tail,
		Method:  method,
		Body:    clone(body),
		Headers: ensureJSON(nil),
	}
}

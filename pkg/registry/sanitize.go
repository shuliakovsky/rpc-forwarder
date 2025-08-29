package registry

import "strings"

func SanitizeNodes(nodes []NodeWithPing) []NodeWithPing {
	clean := make([]NodeWithPing, len(nodes))
	for i, n := range nodes {
		n.Headers = sanitizeHeaders(n.Headers)
		clean[i] = n
	}
	return clean
}

func sanitizeHeaders(h map[string]string) map[string]string {
	if len(h) == 0 {
		return h
	}
	out := make(map[string]string, len(h))
	for k, v := range h {
		switch strings.ToLower(k) {
		case "x-api-key", "authorization", "proxy-authorization":
			out[k] = "***"
		default:
			out[k] = v
		}
	}
	return out
}

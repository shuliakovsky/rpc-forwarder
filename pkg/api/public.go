package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/shuliakovsky/rpc-forwarder/pkg/registry"
)

type Public struct {
	Reg *registry.Registry
}

func NewPublic(reg *registry.Registry) *Public { return &Public{Reg: reg} }

func (p *Public) NetworkFees(w http.ResponseWriter, r *http.Request) {
	nodes := p.Reg.Best("eth")
	if len(nodes) == 0 {
		http.Error(w, "no healthy ETH nodes", http.StatusServiceUnavailable)
		return
	}
	target := nodes[0]
	type rpcReq struct {
		Jsonrpc string      `json:"jsonrpc"`
		Method  string      `json:"method"`
		Params  interface{} `json:"params,omitempty"`
		ID      int         `json:"id"`
	}
	req1, _ := json.Marshal(rpcReq{Jsonrpc: "2.0", Method: "eth_feeHistory", Params: []any{"0x1", "latest", []any{}}, ID: 1})
	req2, _ := json.Marshal(rpcReq{Jsonrpc: "2.0", Method: "eth_maxPriorityFeePerGas", ID: 2})

	client := &http.Client{}
	r1, _ := http.NewRequest("POST", target.URL, bytes.NewReader(req1))
	r2, _ := http.NewRequest("POST", target.URL, bytes.NewReader(req2))
	for k, v := range target.Headers {
		r1.Header.Set(k, v)
		r2.Header.Set(k, v)
	}
	r1.Header.Set("content-type", "application/json")
	r2.Header.Set("content-type", "application/json")

	type feeHistory struct {
		Result struct {
			BaseFeePerGas []string `json:"baseFeePerGas"`
		} `json:"result"`
	}
	type maxPrio struct {
		Result string `json:"result"`
	}
	var fh feeHistory
	var mp maxPrio

	resp1, err1 := client.Do(r1)
	if err1 != nil || resp1.StatusCode/100 != 2 {
		http.Error(w, "feeHistory failed", http.StatusBadGateway)
		return
	}
	defer resp1.Body.Close()
	_ = json.NewDecoder(resp1.Body).Decode(&fh)

	resp2, err2 := client.Do(r2)
	if err2 != nil || resp2.StatusCode/100 != 2 {
		http.Error(w, "maxPriority failed", http.StatusBadGateway)
		return
	}
	defer resp2.Body.Close()
	_ = json.NewDecoder(resp2.Body).Decode(&mp)

	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"baseFee":        first(fh.Result.BaseFeePerGas),
		"maxPriorityFee": mp.Result,
	})
}

func (p *Public) ActiveNodes(w http.ResponseWriter, r *http.Request) {
	all := p.Reg.All()
	resp := map[string]any{}
	for name, st := range all {
		var arr []map[string]any
		for _, n := range st.Best {
			arr = append(arr, map[string]any{
				"url":       maskKeys(n.URL),
				"priority":  n.Priority,
				"isPrivate": n.IsPrivate,
				"alive":     n.Alive,
				"ping":      n.Ping,
			})
		}
		resp[name] = arr
	}
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func first(s []string) string {
	if len(s) == 0 {
		return ""
	}
	return s[0]
}

func maskKeys(u string) string {
	for _, k := range []string{os.Getenv("ALCHEMY_API_KEY"), os.Getenv("TATUM_API_KEY")} {
		if k != "" {
			u = strings.ReplaceAll(u, k, "[HIDDEN]")
		}
	}
	return u
}

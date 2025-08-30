package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shuliakovsky/rpc-forwarder/pkg/networks"
	"github.com/shuliakovsky/rpc-forwarder/pkg/registry"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func newTestChecker() *Checker {
	logger, _ := zap.NewDevelopment()
	reg := registry.New()
	return New("", logger, reg)
}

func TestCheckEVM_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x10"}`))
	}))
	defer srv.Close()

	h := newTestChecker()
	ok, ping := h.checkEVM(networks.Node{URL: srv.URL}, 2*time.Second)
	require.True(t, ok, "EVM node should be alive")
	require.GreaterOrEqual(t, ping, int64(0), "ping should be non-negative")
}

func TestCheckBTC_Blockstream_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("123456"))
	}))
	defer srv.Close()

	h := newTestChecker()
	ok, _ := h.checkBTC(networks.Node{URL: srv.URL}, 2*time.Second)
	require.True(t, ok, "BTC node should be alive")
}

func TestUpdateNetwork_Prioritizes(t *testing.T) {
	fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	}))
	defer fast.Close()

	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	}))
	defer slow.Close()

	nodes := []networks.Node{
		{URL: fast.URL, Priority: 1},
		{URL: slow.URL, Priority: 2},
	}

	h := newTestChecker()
	res := h.UpdateNetwork("evm", nodes)
	require.Equal(t, 2, len(res))
	require.Equal(t, nodes[0].URL, res[0].URL, "fast node should be first")
}

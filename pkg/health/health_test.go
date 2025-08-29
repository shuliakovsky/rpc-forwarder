package health

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/shuliakovsky/rpc-forwarder/pkg/networks"
	"github.com/stretchr/testify/require"
)

func TestCheckEVM_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x10"}`))
	}))
	defer srv.Close()

	h := New(2*time.Second, "", nil)
	ok, ping := h.checkEVM(networks.Node{URL: srv.URL})
	require.True(t, ok, "EVM node should be alive")
	require.Greater(t, ping, int64(0))
}

func TestCheckBTC_Blockstream_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("123456"))
	}))
	defer srv.Close()

	h := New(2*time.Second, "", nil)
	ok, _ := h.checkBTC(networks.Node{URL: srv.URL})
	require.True(t, ok, "BTC node should be alive")
}

func TestUpdateNetwork_Prioritizes(t *testing.T) {
	// быстрый сервер
	fast := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	}))
	defer fast.Close()

	// медленный сервер
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":"0x1"}`))
	}))
	defer slow.Close()

	nodes := []networks.Node{
		{URL: fast.URL, Priority: 1},
		{URL: slow.URL, Priority: 2},
	}

	h := New(2*time.Second, "", nil)
	res := h.UpdateNetwork("evm", nodes)
	require.Equal(t, 2, len(res))
	require.Equal(t, nodes[0].URL, res[0].URL, "fast node should be first")
}

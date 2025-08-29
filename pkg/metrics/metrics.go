package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

var (
	TotalNodes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "rpcf_nodes_total", Help: "Total nodes per network"},
		[]string{"network"},
	)
	HealthyNodes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{Name: "rpcf_nodes_healthy", Help: "Healthy nodes per network"},
		[]string{"network"},
	)
	ProxySuccess = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "rpcf_proxy_success_total", Help: "Successful proxy calls"},
		[]string{"network"},
	)
	ProxyFail = prometheus.NewCounterVec(
		prometheus.CounterOpts{Name: "rpcf_proxy_fail_total", Help: "Failed proxy calls"},
		[]string{"network"},
	)
)

func Init() {
	prometheus.MustRegister(TotalNodes, HealthyNodes, ProxySuccess, ProxyFail)
}

func Handler() http.Handler {
	return promhttp.Handler()
}

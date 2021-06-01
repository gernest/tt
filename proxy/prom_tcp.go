package proxy

import (
	"github.com/prometheus/client_golang/prometheus"
)

var totalTCPRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "tcp_requests_total",
		Help: "Total number of tcp requests",
	},
	[]string{"code"},
)

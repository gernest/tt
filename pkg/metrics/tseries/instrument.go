package tseries

// Copyright 2017 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var totalRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: "tt",
		Name:      "http_total_requests",
		Help:      "Total number of requests",
	},
	[]string{"code", "method"},
)

var requestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "tt",
		Name:      "http_request_duration",
		Help:      "Duration taken to complete http request",
	},
	[]string{"code", "method"},
)

var requestSize = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "tt",
		Name:      "request_size",
		Help:      "Size in bytes of the request",
	},
	[]string{"code", "method"},
)

var responseSize = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "tt",
		Name:      "response_size",
	},
	[]string{"code", "method"},
)

var timeToHeader = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: "tt",
		Name:      "time_to_write_headers",
	},
	[]string{"code", "method"},
)

func Record(
	code int,
	method string,
	totalDuration time.Duration,
	reqSize int64,
	resSize int64,
	headerDuration time.Duration,
) {
	labels := prometheus.Labels{
		"code":   strconv.FormatInt(int64(code), 10),
		"method": method,
	}
	totalRequests.With(labels).Inc()
	requestDuration.With(labels).Observe(float64(totalDuration.Milliseconds()))
	requestSize.With(labels).Observe(float64(reqSize))
	responseSize.With(labels).Observe(float64(resSize))
	timeToHeader.With(labels).Observe(float64(headerDuration.Milliseconds()))
}

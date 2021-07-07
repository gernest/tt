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
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gernest/tt/pkg/meta"
	"github.com/gernest/tt/pkg/xhttp/xlabels"

	"github.com/prometheus/client_golang/prometheus"
)

var totalRequests = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_total_requests",
		Help: "Total number of internal errors encountered by the promhttp metric handler.",
	},
	xlabels.All,
)

var requestDuration = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{},
	xlabels.All,
)

var requestSize = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{},
	xlabels.All,
)

var responseSize = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{},
	xlabels.All,
)

var timeToHeader = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{},
	xlabels.All,
)

// magicString is used for the hacky label test in checkLabels. Remove once fixed.
const magicString = "zZgWfBxLqvG8kc8IMv3POi2Bb0tZI3vAnBx+gBaFi9FyPzB/CzKUer1yufDa"

func Instrument(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		m := &meta.Metrics{}
		r = r.WithContext(meta.SetMetric(r.Context(), m))
		var timeTOWriteHeader time.Duration
		d := newDelegator(w, func(i int) {
			timeTOWriteHeader = time.Since(now)
		})
		next.ServeHTTP(d, r)
		end := time.Since(now)
		metricsLables := Labels(
			r, d.Status(), m,
		)
		report(
			metricsLables, end, computeApproximateRequestSize(r),
			int(d.Written()), timeTOWriteHeader,
		)
	})
}

func report(
	labels prometheus.Labels,
	totalDuration time.Duration,
	reqSize int,
	resSize int,
	headerDuration time.Duration,
) {
	totalRequests.With(labels).Inc()
	requestDuration.With(labels).Observe(float64(totalDuration.Milliseconds()))
	requestSize.With(labels).Observe(float64(reqSize))
	responseSize.With(labels).Observe(float64(resSize))
	timeToHeader.With(labels).Observe(float64(headerDuration.Milliseconds()))
}

func computeApproximateRequestSize(r *http.Request) int {
	s := 0
	if r.URL != nil {
		s += len(r.URL.String())
	}

	s += len(r.Method)
	s += len(r.Proto)
	for name, values := range r.Header {
		s += len(name)
		for _, value := range values {
			s += len(value)
		}
	}
	s += len(r.Host)

	// N.B. r.Form and r.MultipartForm are assumed to be included in r.URL.

	if r.ContentLength != -1 {
		s += int(r.ContentLength)
	}
	return s
}

func sanitizeMethod(m string) string {
	switch m {
	case "GET", "get":
		return "get"
	case "PUT", "put":
		return "put"
	case "HEAD", "head":
		return "head"
	case "POST", "post":
		return "post"
	case "DELETE", "delete":
		return "delete"
	case "CONNECT", "connect":
		return "connect"
	case "OPTIONS", "options":
		return "options"
	case "NOTIFY", "notify":
		return "notify"
	default:
		return strings.ToLower(m)
	}
}

// If the wrapped http.Handler has not set a status code, i.e. the value is
// currently 0, santizeCode will return 200, for consistency with behavior in
// the stdlib.
func sanitizeCode(s int) string {
	switch s {
	case 100:
		return "100"
	case 101:
		return "101"

	case 200, 0:
		return "200"
	case 201:
		return "201"
	case 202:
		return "202"
	case 203:
		return "203"
	case 204:
		return "204"
	case 205:
		return "205"
	case 206:
		return "206"

	case 300:
		return "300"
	case 301:
		return "301"
	case 302:
		return "302"
	case 304:
		return "304"
	case 305:
		return "305"
	case 307:
		return "307"

	case 400:
		return "400"
	case 401:
		return "401"
	case 402:
		return "402"
	case 403:
		return "403"
	case 404:
		return "404"
	case 405:
		return "405"
	case 406:
		return "406"
	case 407:
		return "407"
	case 408:
		return "408"
	case 409:
		return "409"
	case 410:
		return "410"
	case 411:
		return "411"
	case 412:
		return "412"
	case 413:
		return "413"
	case 414:
		return "414"
	case 415:
		return "415"
	case 416:
		return "416"
	case 417:
		return "417"
	case 418:
		return "418"

	case 500:
		return "500"
	case 501:
		return "501"
	case 502:
		return "502"
	case 503:
		return "503"
	case 504:
		return "504"
	case 505:
		return "505"

	case 428:
		return "428"
	case 429:
		return "429"
	case 431:
		return "431"
	case 511:
		return "511"

	default:
		return strconv.Itoa(s)
	}
}

package accesslog

import (
	"net/http"
	"time"

	"github.com/gernest/tt/pkg/meta"
	"github.com/gernest/tt/pkg/metrics/tseries"
)

// Instrument must be the entry point. Adds prometheus metrics for next.
func Instrument(next http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		now := time.Now()
		entry := NewEntry()
		r = r.WithContext(meta.SetMetric(r.Context(), &entry.AccessEntry))
		var timeTOWriteHeader time.Duration
		d := newDelegator(w, func(i int) {
			timeTOWriteHeader = time.Since(now)
		})
		next.ServeHTTP(d, r)
		end := time.Since(now)
		entry.Update(
			r, d.Status(), d.Written(), end, timeTOWriteHeader,
		)
		tseries.Record(d.Status(), r.Method, end, entry.Response.Size,
			d.Written(),
			timeTOWriteHeader,
		)
		Get(ctx).Record(entry)
	})
}

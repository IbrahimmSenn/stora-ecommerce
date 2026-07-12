// Package vitals ingests Core Web Vitals beacons from the storefront.
// Public, unauthenticated, and covered by the general /api rate limiter —
// so the payload is strictly validated: whitelisted metric names, bounded
// values, 1 KB body cap. Nothing user-identifying is stored; the values
// land in Prometheus histograms only.
package vitals

import (
	"encoding/json"
	"net/http"

	"gitea.kood.tech/ibrahimsen/i-love-shopping/internal/response"
)

// Recorder is the metrics seam (implemented by metrics.WebVitals).
type Recorder interface {
	Observe(name string, value float64)
}

type payload struct {
	Name  string  `json:"name"`
	Value float64 `json:"value"`
}

// maxValueMs caps time vitals at 60s and CLS at 10 — anything beyond is a
// broken client or a forged beacon, not a measurement worth keeping.
func valid(p payload) bool {
	switch p.Name {
	case "LCP", "INP", "FCP", "TTFB":
		return p.Value >= 0 && p.Value <= 60_000
	case "CLS":
		return p.Value >= 0 && p.Value <= 10
	default:
		return false
	}
}

func Handler(rec Recorder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, 1024)
		var p payload
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			response.Error(w, http.StatusBadRequest, "invalid vitals payload: expected JSON {name, value}")
			return
		}
		if !valid(p) {
			response.Error(w, http.StatusUnprocessableEntity, "invalid vitals payload: name must be one of LCP, INP, CLS, FCP, TTFB with a value in range")
			return
		}
		rec.Observe(p.Name, p.Value)
		w.WriteHeader(http.StatusNoContent)
	}
}

package metrics

import "github.com/prometheus/client_golang/prometheus"

// Core Web Vitals reported by the browser (web/src/lib/vitals.ts ->
// POST /api/v1/vitals). Time vitals and CLS get separate histograms because
// they live on different scales — seconds vs a unitless layout-shift score.
type WebVitals struct {
	seconds *prometheus.HistogramVec
	cls     prometheus.Histogram
}

func NewWebVitals(reg prometheus.Registerer) *WebVitals {
	w := &WebVitals{
		seconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "shop_web_vitals_seconds",
			Help: "Browser-reported Core Web Vitals (LCP, INP, FCP, TTFB).",
			// Bucket edges straddle the Good/Needs-improvement thresholds:
			// LCP 2.5/4s, INP 0.2/0.5s, FCP 1.8/3s, TTFB 0.8/1.8s.
			Buckets: []float64{.05, .1, .2, .3, .5, .8, 1.2, 1.8, 2.5, 4, 6, 10},
		}, []string{"vital"}),
		cls: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "shop_web_vitals_cls",
			Help:    "Browser-reported Cumulative Layout Shift score.",
			Buckets: []float64{.01, .025, .05, .1, .15, .25, .5, 1},
		}),
	}
	reg.MustRegister(w.seconds, w.cls)
	return w
}

// Observe records one vital. name must already be validated against the
// known set (bounded label cardinality); time vitals arrive in milliseconds.
func (w *WebVitals) Observe(name string, value float64) {
	if name == "CLS" {
		w.cls.Observe(value)
		return
	}
	w.seconds.WithLabelValues(name).Observe(value / 1000.0)
}

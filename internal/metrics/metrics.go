// Package metrics exposes Prometheus instrumentation for the API: HTTP RED
// metrics, database query/pool metrics, and business/security counters emitted
// by the service layer through the Recorder interface. Everything registers on
// a package-owned registry served on the internal metrics listener.
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Recorder is the seam the service layer uses to emit domain counters, so
// services depend on this interface rather than on Prometheus directly.
// Label values must come from bounded sets — never raw error strings or IDs.
type Recorder interface {
	OrderCreated(customerType string)
	CheckoutFailed(reason string)
	OrderPaid(amountCents int64)
	PaymentSucceeded()
	PaymentFailed(reason string)
	// PaymentOrphaned counts charges that landed on an order no longer able to
	// be fulfilled (e.g. reaped as abandoned before a late payment). Should
	// stay at zero; any increment means a customer was charged and refunded
	// (or needs a manual refund).
	PaymentOrphaned()
	LoginAttempt(result, reason string)
	TokenRefresh(result string)
	PasswordReset(event string)
	RateLimited(limiter string)
}

// Noop is the default Recorder when metrics aren't wired (tests, tools).
type Noop struct{}

func (Noop) OrderCreated(string)      {}
func (Noop) CheckoutFailed(string)    {}
func (Noop) OrderPaid(int64)          {}
func (Noop) PaymentSucceeded()        {}
func (Noop) PaymentFailed(string)     {}
func (Noop) PaymentOrphaned()         {}
func (Noop) LoginAttempt(_, _ string) {}
func (Noop) TokenRefresh(string)      {}
func (Noop) PasswordReset(string)     {}
func (Noop) RateLimited(string)       {}

// Prom implements Recorder on Prometheus counters.
type Prom struct {
	ordersCreated    *prometheus.CounterVec
	checkoutFailures *prometheus.CounterVec
	ordersPaid       prometheus.Counter
	revenueCents     prometheus.Counter
	payments         *prometheus.CounterVec
	logins           *prometheus.CounterVec
	refreshes        *prometheus.CounterVec
	passwordResets   *prometheus.CounterVec
	rateLimited      *prometheus.CounterVec
}

func NewProm(reg prometheus.Registerer) *Prom {
	p := &Prom{
		ordersCreated: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "shop_orders_created_total",
			Help: "Orders placed at checkout, by customer type.",
		}, []string{"customer_type"}),
		checkoutFailures: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "shop_checkout_failures_total",
			Help: "Checkout attempts rejected before an order was created.",
		}, []string{"reason"}),
		ordersPaid: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "shop_orders_paid_total",
			Help: "Orders that reached the paid state.",
		}),
		revenueCents: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "shop_order_revenue_cents_total",
			Help: "Revenue in cents from paid orders.",
		}),
		payments: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "shop_payments_total",
			Help: "Payment outcomes; reason is set for failures only.",
		}, []string{"result", "reason"}),
		logins: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "shop_auth_login_total",
			Help: "Login attempts by outcome.",
		}, []string{"result", "reason"}),
		refreshes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "shop_auth_refresh_total",
			Help: "Refresh-token exchanges by outcome.",
		}, []string{"result"}),
		passwordResets: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "shop_auth_password_reset_total",
			Help: "Password reset flow events.",
		}, []string{"event"}),
		rateLimited: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "shop_rate_limit_rejections_total",
			Help: "Requests rejected with 429, by limiter.",
		}, []string{"limiter"}),
	}
	reg.MustRegister(p.ordersCreated, p.checkoutFailures, p.ordersPaid, p.revenueCents,
		p.payments, p.logins, p.refreshes, p.passwordResets, p.rateLimited)
	return p
}

func (p *Prom) OrderCreated(customerType string) { p.ordersCreated.WithLabelValues(customerType).Inc() }
func (p *Prom) CheckoutFailed(reason string)     { p.checkoutFailures.WithLabelValues(reason).Inc() }

func (p *Prom) OrderPaid(amountCents int64) {
	p.ordersPaid.Inc()
	p.revenueCents.Add(float64(amountCents))
}

func (p *Prom) PaymentSucceeded() { p.payments.WithLabelValues("succeeded", "none").Inc() }
func (p *Prom) PaymentFailed(reason string) {
	p.payments.WithLabelValues("failed", orNone(reason)).Inc()
}
func (p *Prom) PaymentOrphaned() { p.payments.WithLabelValues("orphaned", "order_not_pending").Inc() }

func (p *Prom) LoginAttempt(result, reason string) {
	p.logins.WithLabelValues(result, orNone(reason)).Inc()
}

func (p *Prom) TokenRefresh(result string) { p.refreshes.WithLabelValues(result).Inc() }
func (p *Prom) PasswordReset(event string) { p.passwordResets.WithLabelValues(event).Inc() }
func (p *Prom) RateLimited(limiter string) { p.rateLimited.WithLabelValues(limiter).Inc() }

func orNone(s string) string {
	if s == "" {
		return "none"
	}
	return s
}

// NewRegistry returns a registry preloaded with the Go runtime and process
// collectors. Handler serves it in the Prometheus exposition format.
func NewRegistry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)
	return reg
}

func Handler(reg *prometheus.Registry) http.Handler {
	// OpenMetrics is required for exemplars (trace_id on histogram samples);
	// Prometheus negotiates it automatically when scraping.
	return promhttp.HandlerFor(reg, promhttp.HandlerOpts{EnableOpenMetrics: true})
}

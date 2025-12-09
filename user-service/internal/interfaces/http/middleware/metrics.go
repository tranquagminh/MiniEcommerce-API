package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// HTTP metrics
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latencies in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_size_bytes",
			Help:    "HTTP request sizes in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 7),
		},
		[]string{"method", "endpoint"},
	)

	httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response sizes in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 7),
		},
		[]string{"method", "endpoint"},
	)

	// Application metrics
	activeConnections = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_active_connections",
			Help: "Number of active HTTP connections",
		},
	)

	// Custom business metrics
	userRegistrations = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "user_registrations_total",
			Help: "Total number of user registrations",
		},
	)

	userLogins = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "user_logins_total",
			Help: "Total number of successful user logins",
		},
	)

	failedLogins = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "user_login_failures_total",
			Help: "Total number of failed login attempts",
		},
	)

	passwordChanges = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "user_password_changes_total",
			Help: "Total number of password changes",
		},
	)
)

// InitMetrics registers all Prometheus metrics
func InitMetrics() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(httpRequestSize)
	prometheus.MustRegister(httpResponseSize)
	prometheus.MustRegister(activeConnections)
	prometheus.MustRegister(userRegistrations)
	prometheus.MustRegister(userLogins)
	prometheus.MustRegister(failedLogins)
	prometheus.MustRegister(passwordChanges)
}

// PrometheusMiddleware instruments HTTP handlers with Prometheus metrics
func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Track active connections
		activeConnections.Inc()
		defer activeConnections.Dec()

		// Wrap response writer to capture status code and size
		wrapped := &metricsResponseWriter{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		// Process request
		next.ServeHTTP(wrapped, r)

		// Record metrics
		duration := time.Since(start).Seconds()
		endpoint := r.URL.Path
		method := r.Method
		status := strconv.Itoa(wrapped.statusCode)

		httpRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
		httpRequestDuration.WithLabelValues(method, endpoint, status).Observe(duration)

		if r.ContentLength > 0 {
			httpRequestSize.WithLabelValues(method, endpoint).Observe(float64(r.ContentLength))
		}

		if wrapped.bytesWritten > 0 {
			httpResponseSize.WithLabelValues(method, endpoint).Observe(float64(wrapped.bytesWritten))
		}
	})
}

// metricsResponseWriter wraps http.ResponseWriter to capture metrics
type metricsResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (mrw *metricsResponseWriter) WriteHeader(code int) {
	mrw.statusCode = code
	mrw.ResponseWriter.WriteHeader(code)
}

func (mrw *metricsResponseWriter) Write(b []byte) (int, error) {
	n, err := mrw.ResponseWriter.Write(b)
	mrw.bytesWritten += n
	return n, err
}

// MetricsHandler returns the Prometheus metrics handler
func MetricsHandler() http.Handler {
	return promhttp.Handler()
}

// Business metric helpers - these should be called from handlers
func RecordUserRegistration() {
	userRegistrations.Inc()
}

func RecordUserLogin() {
	userLogins.Inc()
}

func RecordFailedLogin() {
	failedLogins.Inc()
}

func RecordPasswordChange() {
	passwordChanges.Inc()
}

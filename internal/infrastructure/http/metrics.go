package httpadapter

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// eventsSubmittedTotal counts submitted events by type and outcome.
// Labels:
//   - event_type: the validated event type (e.g. "UserRegistered")
//   - result:     "success" or "error"
var eventsSubmittedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "events_submitted_total",
	Help: "Total number of events submitted, partitioned by event_type and result.",
}, []string{"event_type", "result"})

func recordEventSubmitted(eventType, result string) {
	eventsSubmittedTotal.WithLabelValues(eventType, result).Inc()
}

// httpErrorsTotal counts HTTP errors by error code (apperror code, not HTTP status).
var httpErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "http_errors_total",
	Help: "Total number of HTTP error responses, partitioned by error code.",
}, []string{"code"})

func recordHTTPError(code string) {
	httpErrorsTotal.WithLabelValues(code).Inc()
}

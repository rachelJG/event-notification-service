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

package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	eventsProcessedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "events_processed_total",
		Help: "Total number of events processed by the worker.",
	}, []string{"result"})

	notificationsDeliveredTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "notifications_delivered_total",
		Help: "Total number of notifications delivered by the worker.",
	}, []string{"channel", "result"})

	workerCyclesTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "worker_cycles_total",
		Help: "Total number of worker loop cycles by loop name and result.",
	}, []string{"loop", "result"})
)

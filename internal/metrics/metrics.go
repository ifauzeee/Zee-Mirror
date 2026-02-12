package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	TasksTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "zeemirror_tasks_total",
		Help: "Total number of tasks processed",
	}, []string{"type", "status"})

	ActiveTasks = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "zeemirror_active_tasks",
		Help: "Number of currently active tasks",
	})

	Throughput = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "zeemirror_throughput_bytes_total",
		Help: "Total bytes processed by the bot",
	}, []string{"direction"})

	APIRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "zeemirror_api_requests_total",
		Help: "Total number of API requests to the dashboard",
	}, []string{"path", "method", "status"})
)

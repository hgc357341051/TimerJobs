package global

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	jobExecTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jobs_exec_total",
			Help: "Total number of job executions",
		},
		[]string{"job_id", "job_name", "mode"},
	)
	jobExecFailTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jobs_exec_fail_total",
			Help: "Total number of failed job executions",
		},
		[]string{"job_id", "job_name", "mode"},
	)
	jobExecDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "jobs_exec_duration_seconds",
			Help:    "Job execution duration in seconds",
			Buckets: []float64{0.1, 0.3, 1, 3, 5, 10, 30, 60, 120, 300},
		},
		[]string{"job_id", "job_name", "mode"},
	)
	jobRunningGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "jobs_running",
			Help: "Current number of running jobs",
		},
	)
)

func InitMetrics() {
	// 注册指标（多次调用也安全，Prometheus会去重）
	prometheus.MustRegister(jobExecTotal)
	prometheus.MustRegister(jobExecFailTotal)
	prometheus.MustRegister(jobExecDuration)
	prometheus.MustRegister(jobRunningGauge)
}

func MetricsIncExec(jobID, jobName, mode string) {
	jobExecTotal.WithLabelValues(jobID, jobName, mode).Inc()
}

func MetricsIncFail(jobID, jobName, mode string) {
	jobExecFailTotal.WithLabelValues(jobID, jobName, mode).Inc()
}

func MetricsObserveDuration(jobID, jobName, mode string, seconds float64) {
	jobExecDuration.WithLabelValues(jobID, jobName, mode).Observe(seconds)
}

func MetricsSetRunning(n float64) { jobRunningGauge.Set(n) }

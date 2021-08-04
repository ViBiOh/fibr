package exif

import (
	"github.com/ViBiOh/httputils/v4/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
)

func createMetric(prometheusRegisterer prometheus.Registerer, name string) *prometheus.GaugeVec {
	counter := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "fibr",
		Subsystem: name,
		Name:      "item",
	}, []string{"state"})

	if err := prometheusRegisterer.Register(counter); err != nil {
		logger.Error("unable to register %s gauge: %s", name, err)
	}

	return counter
}

func createMetrics(prometheusRegisterer prometheus.Registerer) map[string]*prometheus.GaugeVec {
	if prometheusRegisterer == nil {
		return nil
	}

	return map[string]*prometheus.GaugeVec{
		"exif":      createMetric(prometheusRegisterer, "exif"),
		"date":      createMetric(prometheusRegisterer, "date"),
		"geocode":   createMetric(prometheusRegisterer, "geocode"),
		"aggregate": createMetric(prometheusRegisterer, "aggregate"),
	}
}

func (a app) increaseMetric(name, label string) {
	if gauge, ok := a.metrics[name]; ok {
		gauge.WithLabelValues(label).Inc()
	}
}

func (a app) decreaseMetric(name, label string) {
	if gauge, ok := a.metrics[name]; ok {
		gauge.WithLabelValues(label).Inc()
	}
}

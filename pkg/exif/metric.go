package exif

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

func createMetrics(prometheusRegisterer prometheus.Registerer, names ...string) (map[string]*prometheus.CounterVec, error) {
	if prometheusRegisterer == nil {
		return nil, nil
	}

	metrics := make(map[string]*prometheus.CounterVec)
	for _, name := range names {
		metric, err := createMetric(prometheusRegisterer, name)
		if err != nil {
			return nil, err
		}

		metrics[name] = metric
	}

	return metrics, nil
}

func createMetric(prometheusRegisterer prometheus.Registerer, name string) (*prometheus.CounterVec, error) {
	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "fibr",
		Subsystem: name,
		Name:      "item",
	}, []string{"state"})

	if err := prometheusRegisterer.Register(counter); err != nil {
		return nil, fmt.Errorf("unable to register `%s` metric: %s", name, err)
	}

	return counter, nil
}

func (a App) increaseMetric(name, state string) {
	if gauge, ok := a.metrics[name]; ok {
		gauge.With(prometheus.Labels{
			"state": state,
		}).Inc()
	}
}

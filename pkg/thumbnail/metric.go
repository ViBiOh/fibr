package thumbnail

import (
	"fmt"

	"github.com/prometheus/client_golang/prometheus"
)

func createMetric(prometheusRegisterer prometheus.Registerer) (*prometheus.CounterVec, error) {
	if prometheusRegisterer == nil {
		return nil, nil
	}

	counter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "fibr",
		Subsystem: "thumbnail",
		Name:      "item",
	}, []string{"state"})

	if err := prometheusRegisterer.Register(counter); err != nil {
		return nil, fmt.Errorf("unable to register metric: %s", err)
	}

	return counter, nil
}

func (a App) increaseMetric(state string) {
	if a.counter == nil {
		return
	}

	a.counter.With(prometheus.Labels{
		"state": state,
	}).Inc()
}

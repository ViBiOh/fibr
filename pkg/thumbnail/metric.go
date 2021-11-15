package thumbnail

func (a App) increaseMetric(kind, state string) {
	if a.metric == nil {
		return
	}

	a.metric.WithLabelValues(kind, state).Inc()
}

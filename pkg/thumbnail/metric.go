package thumbnail

func (a App) increaseMetric(itemType, state string) {
	if a.metric == nil {
		return
	}

	a.metric.WithLabelValues(itemType, state).Inc()
}

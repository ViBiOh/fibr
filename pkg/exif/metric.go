package exif

func (a App) increaseExif(state string) {
	if a.exifMetric == nil {
		return
	}

	a.exifMetric.WithLabelValues(state).Inc()
}

func (a App) increaseAggregate(state string) {
	if a.aggregateMetric == nil {
		return
	}

	a.aggregateMetric.WithLabelValues(state).Inc()
}

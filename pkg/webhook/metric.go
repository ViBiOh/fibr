package webhook

func (a *App) increaseMetric(code string) {
	if a.counter == nil {
		return
	}

	a.counter.WithLabelValues(code).Inc()
}

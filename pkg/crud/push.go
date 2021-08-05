package crud

import (
	"github.com/ViBiOh/fibr/pkg/provider"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

func (a *app) notify(event provider.Event) {
	if a.pushEvent == nil {
		return
	}

	if err := a.pushEvent(event); err != nil {
		logger.Error("unable to push event %+v: %s", event, err)
	}
}

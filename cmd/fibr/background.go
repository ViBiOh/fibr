package main

import "context"

type Starter interface {
	Start(context.Context)
	Done() <-chan struct{}
}

type Starters []Starter

func (s Starters) Start(ctx context.Context) {
	for _, starter := range s {
		go starter.Start(ctx)
	}
}

func (s Starters) GracefulWait() {
	for _, starter := range s {
		<-starter.Done()
	}
}

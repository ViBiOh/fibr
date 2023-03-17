package main

import "context"

type Starter func(context.Context)

type Starters []Starter

func (s Starters) Do(ctx context.Context) {
	for _, start := range s {
		go start(ctx)
	}
}

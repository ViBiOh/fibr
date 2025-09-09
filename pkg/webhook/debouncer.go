package webhook

import (
	"context"
	"time"
)

type Bucket[T any] struct {
	date  time.Time
	items []T
}

type Input[T any] struct {
	item  T
	group string
}

type GroupDebouncer[T any] struct {
	done     chan struct{}
	ch       chan Input[T]
	state    map[string]*Bucket[T]
	action   func(string, []T)
	duration time.Duration
}

func NewDebouncer[T any](duration time.Duration, action func(string, []T)) *GroupDebouncer[T] {
	return &GroupDebouncer[T]{
		duration: duration,
		state:    make(map[string]*Bucket[T]),
		ch:       make(chan Input[T]),
		done:     make(chan struct{}),
		action:   action,
	}
}

func (d *GroupDebouncer[T]) Start(ctx context.Context) {
	defer close(d.done)

	ticker := time.NewTicker(d.duration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			d.scan(time.Time{})
			return

		case now := <-ticker.C:
			d.scan(now)

		case input := <-d.ch:
			bucket := d.state[input.group]
			if bucket == nil {
				bucket = &Bucket[T]{}
			}

			bucket.date = time.Now().Add(d.duration)
			bucket.items = append(bucket.items, input.item)

			d.state[input.group] = bucket
		}
	}
}

func (d *GroupDebouncer[T]) Send(group string, item T) {
	d.ch <- Input[T]{
		group: group,
		item:  item,
	}
}

func (d *GroupDebouncer[T]) Done() <-chan struct{} {
	return d.done
}

func (d *GroupDebouncer[T]) scan(now time.Time) {
	for group, bucket := range d.state {
		if now.IsZero() || bucket.date.Before(now) {
			d.action(group, bucket.items)

			delete(d.state, group)
		}
	}
}

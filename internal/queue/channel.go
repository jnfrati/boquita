package queue

import "context"

type ChannQueue[I any] struct {
	queue chan I
}

func NewChannelQueue[I any](size uint8) *ChannQueue[I] {

	var queue chan I
	// Allocate queue if size bigger than 1
	if size > 0 {
		queue = make(chan I, size)
	}

	return &ChannQueue[I]{
		queue: queue,
	}
}

func (cq ChannQueue[I]) Start(context.Context) error {
	return nil
}

func (cq ChannQueue[I]) Client() *ChannQueueClient[I] {
	return &ChannQueueClient[I]{
		q: &cq.queue,
	}
}

type ChannQueueClient[I any] struct {
	q *chan I
}

func (cqc ChannQueueClient[I]) Push(item I) error {
	(*cqc.q) <- item

	return nil
}

func (cqc ChannQueueClient[I]) Pull() (I, error) {
	return <-*cqc.q, nil
}

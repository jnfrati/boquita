package queue

import (
	"context"
	"errors"
)

var ErrQueueEmpty = errors.New("can't pull from queue: queue empty")

type Client[I any] interface {
	Push(*I) error
	Pull() (*I, error)
}

type Server[I any] interface {
	Start(context.Context) error

	Client(context.Context) Client[I]
}

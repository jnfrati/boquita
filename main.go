package main

import (
	"context"
	"os"
	"os/signal"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/jnfrati/boquita/internal/executor"
	"github.com/jnfrati/boquita/internal/models"
	"github.com/jnfrati/boquita/internal/queue"
)

func main() {

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, os.Kill)
	defer stop()

	eg, ctx := errgroup.WithContext(ctx)

	chanQueue := queue.NewChannelQueue[models.Job](uint8(100))

	executor, err := executor.NewExecutor(executor.ExecutorPlatform_UnikraftCloud, chanQueue.Client())
	if err != nil {
		panic(err)
	}

	eg.Go(func() error {
		return chanQueue.Start(ctx)
	})

	eg.Go(func() error {
		return executor.Start(ctx)
	})

	eg.Go(func() error {
		time.Sleep(5 * time.Second)

		return nil
	})

}

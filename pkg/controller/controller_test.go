package controller_test

import (
	"context"
	"os"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/jnfrati/boquita/internal/executor"
	"github.com/jnfrati/boquita/internal/helpers"
	"github.com/jnfrati/boquita/internal/models"
	"github.com/jnfrati/boquita/internal/queue"
	"github.com/jnfrati/boquita/internal/storage"
	"github.com/jnfrati/boquita/pkg/controller"
)

func setupTest(t *testing.T) *controller.Controller {
	ctx := t.Context()

	eg, ctx := errgroup.WithContext(ctx)

	jobStorage, err := storage.NewStorage[models.Job](storage.StorageType_Memory)
	if err != nil {
		t.Fatal(err)
	}
	cronToJobStorage, err := storage.NewStorage[models.CronToJob](storage.StorageType_Memory)
	if err != nil {
		t.Fatal(err)
	}
	executionStorage, err := storage.NewStorage[models.Execution](storage.StorageType_Memory)
	if err != nil {
		t.Fatal(err)
	}

	chanQueue := queue.NewChannelQueue[models.Job](uint8(100))

	executor, err := executor.NewExecutor(
		executor.ExecutorPlatform_UnikraftCloud,
		chanQueue.Client(),
		executionStorage,
	)
	if err != nil {
		t.Fatal(err)
	}

	controller := controller.NewController(
		chanQueue.Client(),
		jobStorage,
		cronToJobStorage,
		executionStorage,
	)

	eg.Go(func() error {
		return chanQueue.Start(ctx)
	})

	eg.Go(func() error {
		return executor.Start(ctx)
	})

	go func() {
		if err := eg.Wait(); err != nil {
			t.Log(err)
			t.Fail()
		}
	}()

	return controller
}

func TestCreateJob(t *testing.T) {
	os.Setenv("UKC_TOKEN", "cm9ib3QkbmZyYXRpLnVzZXJzLmtyYWZ0Y2xvdWQ6dTM0UWZhUGcwdFVOUXh1WnRwbUl6S2JKTXoyc0dxOHo=")
	os.Setenv("UKC_METRO", "fra0")
	ctx := t.Context()
	controller := setupTest(t)

	manifest := models.JobManifestV1{
		Image:      "nginx:latest",
		EnvMap:     map[string]string{},
		Entrypoint: "./server",
		MemoryMB:   helpers.Ptr(125),
		Args:       []string{},
	}

	id, err := controller.CreateJob(context.Background(), &models.Job{
		Name:     "test-job",
		Version:  0,
		Type:     models.JobType_SingleExecution,
		Manifest: &manifest,
	})
	if err != nil {
		t.Fatal(err)
	}

	time.Sleep(5 * time.Second)

	job, err := controller.GetById(ctx, id)
	if err != nil {
		t.Fatal(err)
	}

	if len(job.Executions) == 0 {
		t.Log("No executions for this job")
	}

	for _, e := range job.Executions {
		t.Logf("%v", e)
	}

	t.Logf("%v\n", job)

}

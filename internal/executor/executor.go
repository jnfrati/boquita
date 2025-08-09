package executor

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	kraftcloud "sdk.kraft.cloud"
	kcinstance "sdk.kraft.cloud/instances"

	"github.com/jnfrati/boquita/internal/helpers"
	"github.com/jnfrati/boquita/internal/models"
	"github.com/jnfrati/boquita/internal/queue"
	"github.com/jnfrati/boquita/internal/storage"
)

type Executor interface {
	Start(context.Context) error
}

type executorPlatform uint

const (
	ExecutorPlatform_UnikraftCloud executorPlatform = iota
)

func NewExecutor(platform executorPlatform, queue queue.Client[models.Job], executionStorage storage.Storage[models.Execution]) (Executor, error) {
	switch platform {
	case ExecutorPlatform_UnikraftCloud:
		return newUnikraftExecutor(queue, executionStorage)
	default:
		return nil, nil
	}
}

type unikraftExecutor struct {
	kraftcloud kraftcloud.KraftCloud

	queueClient queue.Client[models.Job]

	executionStorage storage.Storage[models.Execution]
}

func newUnikraftExecutor(queueClient queue.Client[models.Job], executionStorage storage.Storage[models.Execution]) (*unikraftExecutor, error) {
	kraftToken, ok := os.LookupEnv("UKC_TOKEN")
	if !ok {
		return nil, errors.New("UKC_TOKEN missing, can't start unikraft executor")
	}

	kraftMetro, ok := os.LookupEnv("UKC_METRO")
	if !ok {
		return nil, errors.New("UKC_METRO missing, can't start unikraft executor")
	}

	client := kraftcloud.NewClient(
		kraftcloud.WithToken(kraftToken),
		kraftcloud.WithDefaultMetro(kraftMetro),
	)

	return &unikraftExecutor{
		kraftcloud:       client,
		queueClient:      queueClient,
		executionStorage: executionStorage,
	}, nil

}

func (ue *unikraftExecutor) Start(ctx context.Context) error {

	// defer ue.Cleanup()

	client := ue.kraftcloud

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	ticker := time.NewTicker(time.Millisecond * 250)
	defer ticker.Stop()
	for {
		<-ticker.C

		job, err := ue.queueClient.Pull()
		if errors.Is(err, queue.ErrQueueEmpty) {
			continue
		}

		// TODO: Figure out how we want to support other versions
		if job.Manifest.Version != models.JobManifestVersion_v1 {
			return nil
		}

		// Hydrate the job manifest

		manifest := new(models.JobManifestV1)

		if err := manifest.FromRawMessage(job.Manifest.RawMessage); err != nil {
			return err
		}

		res, err := client.Instances().Create(ctx, kcinstance.CreateRequest{
			Name:      &job.Name,
			Image:     manifest.Image,
			Args:      manifest.Args,
			Env:       manifest.EnvMap,
			MemoryMB:  manifest.MemoryMB,
			Autostart: helpers.Ptr(true),
		})

		if err != nil {
			return err
		}

		if len(res.Errors) > 0 {
			errlist := make([]error, len(res.Errors))

			for _, err := range res.Errors {
				errlist = append(errlist, fmt.Errorf("couldn't create instance, error status: %v", err.Status))
			}

			return errors.Join(errlist...)
		}

		execution := models.Execution{
			JobId:      job.Id,
			StartedAt:  time.Now(),
			Status:     models.ExecutionStatus_RUNNING,
			ExitCode:   nil,
			ExitStatus: nil,
			FinishedAt: nil,
			Logs:       []string{},
		}

		_, err = ue.executionStorage.Set(ctx, &execution)
		if err != nil {
			return err
		}

		// TODO: Init observer for instance

	}

}

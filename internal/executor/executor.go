package executor

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	kraftcloud "sdk.kraft.cloud"
	kcinstance "sdk.kraft.cloud/instances"

	"github.com/jnfrati/boquita/internal/helpers"
	"github.com/jnfrati/boquita/internal/logger"
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

	logger.Global.Debug().Msgf("UKC_TOKEN=%s", kraftToken)

	kraftMetro, ok := os.LookupEnv("UKC_METRO")
	if !ok {
		return nil, errors.New("UKC_METRO missing, can't start unikraft executor")
	}

	logger.Global.Debug().Msgf("UKC_METRO=%s", kraftMetro)

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
	client := ue.kraftcloud

	ticker := time.NewTicker(time.Millisecond * 250)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
		}

		job, err := ue.queueClient.Pull(ctx)
		if errors.Is(err, queue.ErrQueueEmpty) || errors.Is(err, context.Canceled) {
			continue
		}

		logger.Global.Debug().Msgf("Received new job name %s ", job.Manifest.Name)

		// Hydrate the job manifest

		manifest := job.Manifest

		execId := uuid.NewString()

		instanceName := manifest.Name + execId

		logger.Global.Debug().Msgf("Creating instance")
		res, err := client.Instances().Create(ctx, kcinstance.CreateRequest{
			Name:      &instanceName,
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

			return errlist[0]
		}

		execution := &models.Execution{
			Id:         execId,
			JobId:      job.Id,
			StartedAt:  time.Now(),
			Status:     models.ExecutionStatus_RUNNING,
			ExitCode:   nil,
			FinishedAt: nil,
			Logs:       []string{},
		}

		err = ue.executionStorage.Set(ctx, execId, execution)
		if err != nil {
			return err
		}

		entry := res.Data.Entries[0]

		logger.Global.Debug().Any("execution", execution).Any("entry", entry).Msg("Starting observable")
		go func() {
			err := ue.ObserveJob(ctx, execution, entry.UUID)
			if err != nil {
				// TODO: Cleanup instance and update execution
				logger.Global.Err(err).Msg("failed to observe job")
			}
		}()
	}

}

func (ue *unikraftExecutor) ObserveJob(ctx context.Context, execution *models.Execution, kinstanceId string) error {
	logger.Global.Debug().Msg("starting observer")
	defer logger.Global.Debug().Msg("closing observer")

	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// NOTE: Store latest state maybe?
			return nil
		case <-ticker.C:

		}
		retryCount := 0
	retry:

		logger.Global.Debug().Msg("Getting instances")
		instanceRes, err := ue.kraftcloud.Instances().Get(ctx, kinstanceId)
		if err != nil {
			if retryCount > 3 {
				logger.Global.Debug().Err(err).Msg("failed to retrieve instances three times, stopping observer")
				return errors.Wrap(err, "Retried 3 times, closing observer with error")
			}

			logger.Global.Debug().Err(err).Msg("failed to retrieve instances, retrying")
			time.Sleep(2 * time.Second)
			retryCount++
			goto retry
		}

		logger.Global.
			Debug().
			Any("execution", execution).
			Any("instance", instanceRes).
			Msg("Got instance from unikraft")

		instance := instanceRes.Data.Entries[0]

		if instance.Error != nil {
			// TODO: Manage error updating the instance
			logger.Global.Error().
				Str("error_message", instance.Message).
				Msgf("retrieving instance failed, stopping observer")
			return nil
		}

		execution.ExitCode = instance.ExitCode
		if instance.StoppedAt == "" {
			execution.Status = models.ExecutionStatus_RUNNING
		} else if *execution.ExitCode > 0 {
			execution.Status = models.ExecutionStatus_FAILED
		} else if *execution.ExitCode == 0 {
			execution.Status = models.ExecutionStatus_SUCCEEDED
		}

		err = ue.executionStorage.Set(ctx, execution.Id, execution)
		if err != nil {
			logger.Global.Debug().Err(err).Any("execution", execution).Msg("couldn't update execution")
		}

		if execution.Status != models.ExecutionStatus_RUNNING {
			retryCount = 0
		retrydelete:
			// Remove the instance
			_, err := ue.kraftcloud.Instances().Delete(ctx, kinstanceId)
			if err != nil {
				if retryCount > 3 {
					logger.Global.Debug().Err(err).Msg("failed to delete instance three times, stopping observer")
					return errors.Wrap(err, "Retried 3 times, closing observer with error")
				}

				logger.Global.Debug().Err(err).Msg("failed to delete instance, retrying")
				time.Sleep(2 * time.Second)
				retryCount++
				goto retrydelete
			}
		}
	}

}

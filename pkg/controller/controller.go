package controller

import (
	"context"
	"encoding/json"

	"github.com/robfig/cron/v3"

	"github.com/jnfrati/boquita/internal/models"
	"github.com/jnfrati/boquita/internal/queue"
	"github.com/jnfrati/boquita/internal/storage"
)

func NewController(
	qc queue.Client[models.Job],
	jobStorage storage.Storage[models.Job],
	cronToJobStorage storage.Storage[models.CronToJob],
	executionStorage storage.Storage[models.Execution],
) *Controller {
	return &Controller{
		qc:               qc,
		jobStorage:       jobStorage,
		cronToJobStorage: cronToJobStorage,
		executionStorage: executionStorage,
	}
}

type Controller struct {
	qc queue.Client[models.Job]

	jobStorage       storage.Storage[models.Job]
	executionStorage storage.Storage[models.Execution]
	cronToJobStorage storage.Storage[models.CronToJob]

	cronManager cron.Cron
}

func (c *Controller) ListJobs(ctx context.Context) ([]models.Job, error) {
	return c.jobStorage.List(ctx, 100, 0)
}

func (c *Controller) CreateJob(ctx context.Context, payload *models.Job) (string, error) {
	manifest := new(models.JobManifestV1)

	if err := json.Unmarshal(payload.Manifest.RawMessage, manifest); err != nil {
		return "", err
	}

	var entryId cron.EntryID

	if payload.Type == models.JobType_Cron || payload.Type == models.JobType_Schedule {
		j := cron.FuncJob(func() {
			if err := c.qc.Push(payload); err != nil {
				// Send an error or retry
			}
		})

		scheduledAt, err := cron.ParseStandard("staff")
		if err != nil {
			return "", err
		}

		if payload.Type == models.JobType_Cron {
			entryId, err = c.cronManager.AddFunc("staff", j)
			if err != nil {
				return "", err
			}

		} else if payload.Type == models.JobType_Schedule {
			c.cronManager.Schedule(scheduledAt, j)
		}
	}

	id, err := c.jobStorage.Set(ctx, payload)
	if err != nil {
		return "", err
	}

	// TODO: Fix, because storage receives any, we can't properly set the ID
	payload.Id = id

	if _, err := c.cronToJobStorage.Set(ctx, &models.CronToJob{
		JobId:       id,
		CronEntryId: entryId,
	}); err != nil {
		return "", err
	}

	if payload.Type == models.JobType_SingleExecution {
		// Queue the job directly
		if err := c.qc.Push(payload); err != nil {
			return "", err
		}
	}
	return id, nil
}

func (c *Controller) GetById(ctx context.Context, jobId string) (*models.Job, error) {
	job, err := c.jobStorage.Get(ctx, jobId)
	if err != nil {
		return nil, err
	}

	// TODO: executions, err := c.executionStorage.SearchBy(ctx, ".JobId", job.Id)
	executions, err := c.executionStorage.List(ctx, 100, 0)
	if err != nil {
		return nil, err
	}

	job.Executions = executions
	return job, nil
}

func (c *Controller) StreamJobLogs(ctx context.Context, jobId string) error {
	return nil
}

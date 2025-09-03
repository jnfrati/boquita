package controller

import (
	"context"
	"slices"

	"github.com/google/uuid"
	"github.com/pkg/errors"
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
	c := cron.New(
		cron.WithParser(
			cron.NewParser(
				cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow,
			),
		),
	)

	c.Start()

	return &Controller{
		cronManager:      c,
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

	cronManager *cron.Cron
}

func (c *Controller) ListJobs(ctx context.Context) ([]models.Job, error) {
	return c.jobStorage.List(ctx, 100, 0)
}

func (c *Controller) CreateJob(ctx context.Context, payload *models.JobManifestV1) (string, error) {

	job := new(models.Job)

	job.Id = uuid.NewString()
	job.Manifest = payload

	err := c.jobStorage.Set(ctx, job.Id, job)
	if err != nil {
		return "", err
	}

	j := cron.FuncJob(func() {
		if err := c.qc.Push(job); err != nil {
			// Send an error or retry
		}
	})

	if payload.Cron != nil {
		entryId, err := c.cronManager.AddFunc(*payload.Cron, j)
		if err != nil {
			return "", errors.Wrap(err, "couldn't add cron execution")
		}

		if err := c.cronToJobStorage.Set(ctx, uuid.NewString(), &models.CronToJob{
			JobId:       job.Id,
			CronEntryId: entryId,
		}); err != nil {
			return "", errors.Wrap(err, "couldn't store the relationship between cron entry and job id")
		}
	}

	if payload.Schedule != nil {
		cronExpr, err := cron.ParseStandard(*payload.Schedule)
		if err != nil {
			return "", err
		}
		c.cronManager.Schedule(cronExpr, j)
	}

	return job.Id, nil
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

	if len(job.Executions) > 0 {
		slices.SortFunc(
			job.Executions,
			func(a models.Execution, b models.Execution) int {
				return b.StartedAt.Compare(a.StartedAt)
			},
		)

		job.LastExecution = &job.Executions[0]
	}
	return job, nil
}

func (c *Controller) StreamJobLogs(ctx context.Context, jobId string) error {
	return nil
}

package controller

import (
	"context"

	"github.com/jnfrati/boquita/internal/models"
	"github.com/jnfrati/boquita/internal/queue"
)

func NewController(
	qc queue.Client[models.Job],
) *Controller {
	return &Controller{
		qc: qc,
	}
}

type Controller struct {
	qc queue.Client[models.Job]
}

func (c *Controller) ListJobs(ctx context.Context) error {

	return nil
}

func (c *Controller) CreateJob(ctx context.Context, payload *models.Job) error {

	return nil
}

func (c *Controller) StreamJobLogs(ctx context.Context, jobId string) error {
	return nil
}

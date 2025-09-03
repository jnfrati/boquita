package models

import (
	"time"

	"github.com/robfig/cron/v3"
)

type ExecutionStatus uint8

const (
	ExecutionStatus_RUNNING ExecutionStatus = iota
	ExecutionStatus_SUCCEEDED
	ExecutionStatus_FAILED
)

type Execution struct {
	Id string `json:"id"`

	JobId string `json:"job_id"`

	StartedAt  time.Time  `json:"started_at"`
	FinishedAt *time.Time `json:"finished_at,omitempty"`

	Status ExecutionStatus `json:"status"`

	ExitCode *uint `json:"exit_code"`

	Logs []string `json:"logs"`
}

type JobManifestVersion string

const (
	JobManifestVersion_v1 JobManifestVersion = "job.manifest/v1"
)

type JobManifestV1 struct {
	Name       string             `json:"name"`
	Version    JobManifestVersion `json:"version"`
	Image      string             `json:"image"`
	Entrypoint string             `json:"entrypoint"`
	MemoryMB   *int               `json:"memory_mb,omitempty"`
	Args       []string           `json:"args,omitempty"`
	EnvMap     map[string]string  `json:"env_map,omitempty"`

	Cron     *string `json:"cron_expr,omitempty"`
	Schedule *string `json:"schedule,omitempty"`
	// TODO: Support Volumes, maybe for this job manifest version using
	// Volumes instances.CreateRequestVolume
}

type Job struct {
	Id string `json:"id"`

	LastExecution *Execution `json:"last_execution,omitempty"`

	Executions []Execution `json:"executions,omitempty"`

	Manifest *JobManifestV1 `json:"manifest"`
}

// Internal structure only
type CronToJob struct {
	JobId       string
	CronEntryId cron.EntryID
}

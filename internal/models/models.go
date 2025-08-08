package models

import (
	"encoding/json"
	"time"
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

	ExitCode   uint `json:"exit_code"`
	ExitStatus uint `json:"exit_status"`

	Logs []string `json:"logs"`
}

type JobManifestVersion string

const (
	JobManifestVersion_v1 = "job.manifest/v1"
)

type JobManifestV1 struct {
	Version    JobManifestVersion `json:"version"`
	Image      string             `json:"image"`
	Entrypoint string             `json:"entrypoint"`
	MemoryMB   *int               `json:"memory_mb,omitempty"`
	Args       []string           `json:"args,omitempty"`
	EnvMap     map[string]string  `json:"env_map,omitempty"`
	// TODO: Support Volumes, maybe for this job manifest version using
	// Volumes instances.CreateRequestVolume
}

func (jmv1 *JobManifestV1) FromRawMessage(manifest json.RawMessage) error {
	if err := json.Unmarshal(manifest, jmv1); err != nil {
		return err
	}

	return nil
}

type JobManifest struct {
	Version JobManifestVersion `json:"version"`
	json.RawMessage
}

type Job struct {
	Id string `json:"id"`

	// Used to identify the different versions of the job
	InstanceId string `json:"instance_id"`

	// Counter to identify the iteration
	Version uint32 `json:"version"`

	Name string `json:"name"`

	LastExecution *Execution `json:"last_execution,omitempty"`

	Executions []Execution `json:"executions,omitempty"`

	Manifest JobManifest `json:"manifest"`
}

package repository

import (
	"go.woodpecker-ci.org/woodpecker/v3/server/model"
)

// PipelineRepository defines the interface for pipeline data access operations
type PipelineRepository interface {
	// GetPipeline gets a pipeline by unique ID
	GetPipeline(int64) (*model.Pipeline, error)

	// GetPipelineNumber gets a pipeline by number
	GetPipelineNumber(*model.Repo, int64) (*model.Pipeline, error)

	// GetPipelineBadge gets the last relevant pipeline for the badge
	GetPipelineBadge(*model.Repo, string, []model.WebhookEvent) (*model.Pipeline, error)

	// GetPipelineLastByBranch gets the last pipeline for the branch
	GetPipelineLastByBranch(*model.Repo, string) (*model.Pipeline, error)

	// GetPipelineLastBefore gets the last pipeline before pipeline number N
	GetPipelineLastBefore(*model.Repo, string, int64) (*model.Pipeline, error)

	// GetPipelineList gets a list of pipelines for the repository
	GetPipelineList(*model.Repo, *model.ListOptions, *model.PipelineFilter) ([]*model.Pipeline, error)

	// GetRepoLatestPipelines gets the latest pipelines for the given repo IDs
	GetRepoLatestPipelines([]int64) ([]*model.Pipeline, error)

	// GetActivePipelineList gets a list of the active pipelines for the repository
	GetActivePipelineList(*model.Repo) ([]*model.Pipeline, error)

	// GetPipelineQueue gets a list of pipelines in queue
	GetPipelineQueue() ([]*model.Feed, error)

	// GetPipelineCount gets a count of all pipelines in the system
	GetPipelineCount() (int64, error)

	// CreatePipeline creates a new pipeline and steps
	CreatePipeline(*model.Pipeline, ...*model.Step) error

	// UpdatePipeline updates a pipeline
	UpdatePipeline(*model.Pipeline) error

	// DeletePipeline deletes a pipeline
	DeletePipeline(*model.Pipeline) error
}

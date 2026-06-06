package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"go.woodpecker-ci.org/woodpecker/v3/server/model"
	"go.woodpecker-ci.org/woodpecker/v3/server/repository"
	"go.woodpecker-ci.org/woodpecker/v3/server/router/middleware/session"
	"go.woodpecker-ci.org/woodpecker/v3/server/store/types"
)

// PipelineHandler handles pipeline-related API requests
type PipelineHandler struct {
	pipelineRepo repository.PipelineRepository
	// 可以添加其他需要的 repository 或 service
}

// NewPipelineHandler creates a new PipelineHandler with dependency injection
func NewPipelineHandler(pipelineRepo repository.PipelineRepository) *PipelineHandler {
	return &PipelineHandler{
		pipelineRepo: pipelineRepo,
	}
}

// GetPipeline handles GET /repos/:repo/pipelines/:number
func (h *PipelineHandler) GetPipeline(c *gin.Context) {
	repo := session.Repo(c)
	number, err := strconv.ParseInt(c.Param("number"), 10, 64)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, errors.New("invalid pipeline number"))
		return
	}

	pipeline, err := h.pipelineRepo.GetPipelineNumber(repo, number)
	if err != nil {
		handleDBError(c, err)
		return
	}

	c.JSON(http.StatusOK, pipeline.ToAPIModel())
}

// GetPipelineList handles GET /repos/:repo/pipelines
func (h *PipelineHandler) GetPipelineList(c *gin.Context) {
	repo := session.Repo(c)
	listOpts := session.Pagination(c)

	filter := &model.PipelineFilter{
		Branch:      c.Query("branch"),
		RefContains: c.Query("ref"),
	}

	pipelines, err := h.pipelineRepo.GetPipelineList(repo, listOpts, filter)
	if err != nil {
		_ = c.AbortWithError(http.StatusInternalServerError, err)
		return
	}

	apiPipelines := make([]*model.APIPipeline, len(pipelines))
	for i, p := range pipelines {
		apiPipelines[i] = p.ToAPIModel()
	}

	c.JSON(http.StatusOK, apiPipelines)
}

// DeletePipeline handles DELETE /repos/:repo/pipelines/:number
func (h *PipelineHandler) DeletePipeline(c *gin.Context) {
	repo := session.Repo(c)
	number, err := strconv.ParseInt(c.Param("number"), 10, 64)
	if err != nil {
		_ = c.AbortWithError(http.StatusBadRequest, errors.New("invalid pipeline number"))
		return
	}

	pipeline, err := h.pipelineRepo.GetPipelineNumber(repo, number)
	if err != nil {
		handleDBError(c, err)
		return
	}

	if ok := pipelineDeleteAllowed(pipeline); !ok {
		c.String(http.StatusUnprocessableEntity, "Cannot delete pipeline with status %s", pipeline.Status)
		return
	}

	err = h.pipelineRepo.DeletePipeline(pipeline)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error deleting pipeline: %s", err.Error())
		return
	}

	c.Status(http.StatusNoContent)
}

// handleDBError handles database errors and writes appropriate HTTP response
func handleDBError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, types.ErrRecordNotExist):
		_ = c.AbortWithError(http.StatusNotFound, errors.New("resource not found"))
	case errors.Is(err, types.ErrDuplicate):
		_ = c.AbortWithError(http.StatusConflict, errors.New("resource already exists"))
	case errors.Is(err, types.ErrForeignKey):
		_ = c.AbortWithError(http.StatusUnprocessableEntity, errors.New("related resource not found"))
	default:
		_ = c.AbortWithError(http.StatusInternalServerError, err)
	}
}

// Helper function kept from original
func pipelineDeleteAllowed(pipeline *model.Pipeline) bool {
	return pipeline.Status != model.StatusRunning &&
		pipeline.Status != model.StatusPending
}

package repository

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"go.woodpecker-ci.org/woodpecker/v3/server/api"
	"go.woodpecker-ci.org/woodpecker/v3/server/model"
	"go.woodpecker-ci.org/woodpecker/v3/server/store/types"
)

// MockPipelineRepository is a mock implementation of PipelineRepository
type MockPipelineRepository struct {
	mock.Mock
}

// Ensure MockPipelineRepository implements PipelineRepository
var _ PipelineRepository = (*MockPipelineRepository)(nil)

func (m *MockPipelineRepository) GetPipeline(id int64) (*model.Pipeline, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Pipeline), args.Error(1)
}

func (m *MockPipelineRepository) GetPipelineNumber(repo *model.Repo, number int64) (*model.Pipeline, error) {
	args := m.Called(repo, number)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Pipeline), args.Error(1)
}

func (m *MockPipelineRepository) GetPipelineBadge(repo *model.Repo, branch string, events []model.WebhookEvent) (*model.Pipeline, error) {
	args := m.Called(repo, branch, events)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Pipeline), args.Error(1)
}

func (m *MockPipelineRepository) GetPipelineLastByBranch(repo *model.Repo, branch string) (*model.Pipeline, error) {
	args := m.Called(repo, branch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Pipeline), args.Error(1)
}

func (m *MockPipelineRepository) GetPipelineLastBefore(repo *model.Repo, branch string, before int64) (*model.Pipeline, error) {
	args := m.Called(repo, branch, before)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Pipeline), args.Error(1)
}

func (m *MockPipelineRepository) GetPipelineList(repo *model.Repo, opts *model.ListOptions, filter *model.PipelineFilter) ([]*model.Pipeline, error) {
	args := m.Called(repo, opts, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Pipeline), args.Error(1)
}

func (m *MockPipelineRepository) GetRepoLatestPipelines(repoIDs []int64) ([]*model.Pipeline, error) {
	args := m.Called(repoIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Pipeline), args.Error(1)
}

func (m *MockPipelineRepository) GetActivePipelineList(repo *model.Repo) ([]*model.Pipeline, error) {
	args := m.Called(repo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Pipeline), args.Error(1)
}

func (m *MockPipelineRepository) GetPipelineQueue() ([]*model.Feed, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.Feed), args.Error(1)
}

func (m *MockPipelineRepository) GetPipelineCount() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockPipelineRepository) CreatePipeline(pipeline *model.Pipeline, steps ...*model.Step) error {
	args := m.Called(pipeline, steps)
	return args.Error(0)
}

func (m *MockPipelineRepository) UpdatePipeline(pipeline *model.Pipeline) error {
	args := m.Called(pipeline)
	return args.Error(0)
}

func (m *MockPipelineRepository) DeletePipeline(pipeline *model.Pipeline) error {
	args := m.Called(pipeline)
	return args.Error(0)
}

// Helper function to create a test Gin context
func setupTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	return c, w
}

// Helper function to set up test repo in context
func setupTestRepo(c *gin.Context) *model.Repo {
	repo := &model.Repo{
		ID:   1,
		Name: "test-repo",
	}
	c.Set("repo", repo)
	return repo
}

// TestGetPipeline_Success tests successful retrieval of a pipeline
func TestGetPipeline_Success(t *testing.T) {
	mockRepo := new(MockPipelineRepository)
	handler := api.NewPipelineHandler(mockRepo)

	c, w := setupTestContext()
	repo := setupTestRepo(c)
	c.Params = gin.Params{{Key: "number", Value: "1"}}

	expectedPipeline := &model.Pipeline{
		ID:     1,
		Number: 1,
		RepoID: repo.ID,
		Status: model.StatusSuccess,
	}

	mockRepo.On("GetPipelineNumber", repo, int64(1)).Return(expectedPipeline, nil)

	handler.GetPipeline(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

// TestGetPipeline_NotFound tests retrieval of a non-existent pipeline
func TestGetPipeline_NotFound(t *testing.T) {
	mockRepo := new(MockPipelineRepository)
	handler := api.NewPipelineHandler(mockRepo)

	c, w := setupTestContext()
	repo := setupTestRepo(c)
	c.Params = gin.Params{{Key: "number", Value: "999"}}

	mockRepo.On("GetPipelineNumber", repo, int64(999)).Return(nil, types.ErrRecordNotExist)

	handler.GetPipeline(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertExpectations(t)
}

// TestGetPipeline_InvalidNumber tests invalid pipeline number
func TestGetPipeline_InvalidNumber(t *testing.T) {
	mockRepo := new(MockPipelineRepository)
	handler := api.NewPipelineHandler(mockRepo)

	c, w := setupTestContext()
	_ = setupTestRepo(c)
	c.Params = gin.Params{{Key: "number", Value: "invalid"}}

	handler.GetPipeline(c)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	mockRepo.AssertNotCalled(t, "GetPipelineNumber")
}

// TestGetPipeline_InternalServerError tests database error
func TestGetPipeline_InternalServerError(t *testing.T) {
	mockRepo := new(MockPipelineRepository)
	handler := api.NewPipelineHandler(mockRepo)

	c, w := setupTestContext()
	repo := setupTestRepo(c)
	c.Params = gin.Params{{Key: "number", Value: "1"}}

	dbError := errors.New("database connection failed")
	mockRepo.On("GetPipelineNumber", repo, int64(1)).Return(nil, dbError)

	handler.GetPipeline(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// TestGetPipelineList_Success tests successful retrieval of pipeline list
func TestGetPipelineList_Success(t *testing.T) {
	mockRepo := new(MockPipelineRepository)
	handler := api.NewPipelineHandler(mockRepo)

	c, w := setupTestContext()
	repo := setupTestRepo(c)

	expectedPipelines := []*model.Pipeline{
		{ID: 1, Number: 1, RepoID: repo.ID, Status: model.StatusSuccess},
		{ID: 2, Number: 2, RepoID: repo.ID, Status: model.StatusRunning},
	}

	mockRepo.On("GetPipelineList", repo, mock.Anything, mock.Anything).Return(expectedPipelines, nil)

	handler.GetPipelineList(c)

	assert.Equal(t, http.StatusOK, w.Code)
	mockRepo.AssertExpectations(t)
}

// TestGetPipelineList_DatabaseError tests database error on list retrieval
func TestGetPipelineList_DatabaseError(t *testing.T) {
	mockRepo := new(MockPipelineRepository)
	handler := api.NewPipelineHandler(mockRepo)

	c, w := setupTestContext()
	repo := setupTestRepo(c)

	dbError := errors.New("failed to query database")
	mockRepo.On("GetPipelineList", repo, mock.Anything, mock.Anything).Return(nil, dbError)

	handler.GetPipelineList(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

// TestDeletePipeline_Success tests successful pipeline deletion
func TestDeletePipeline_Success(t *testing.T) {
	mockRepo := new(MockPipelineRepository)
	handler := api.NewPipelineHandler(mockRepo)

	c, w := setupTestContext()
	repo := setupTestRepo(c)
	c.Params = gin.Params{{Key: "number", Value: "1"}}

	pipelineToDelete := &model.Pipeline{
		ID:     1,
		Number: 1,
		RepoID: repo.ID,
		Status: model.StatusSuccess,
	}

	mockRepo.On("GetPipelineNumber", repo, int64(1)).Return(pipelineToDelete, nil)
	mockRepo.On("DeletePipeline", pipelineToDelete).Return(nil)

	handler.DeletePipeline(c)

	assert.Equal(t, http.StatusNoContent, w.Code)
	mockRepo.AssertExpectations(t)
}

// TestDeletePipeline_Running tests deletion of a running pipeline (not allowed)
func TestDeletePipeline_Running(t *testing.T) {
	mockRepo := new(MockPipelineRepository)
	handler := api.NewPipelineHandler(mockRepo)

	c, w := setupTestContext()
	repo := setupTestRepo(c)
	c.Params = gin.Params{{Key: "number", Value: "1"}}

	pipelineToDelete := &model.Pipeline{
		ID:     1,
		Number: 1,
		RepoID: repo.ID,
		Status: model.StatusRunning,
	}

	mockRepo.On("GetPipelineNumber", repo, int64(1)).Return(pipelineToDelete, nil)

	handler.DeletePipeline(c)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	mockRepo.AssertNotCalled(t, "DeletePipeline")
}

// TestDeletePipeline_NotFound tests deletion of a non-existent pipeline
func TestDeletePipeline_NotFound(t *testing.T) {
	mockRepo := new(MockPipelineRepository)
	handler := api.NewPipelineHandler(mockRepo)

	c, w := setupTestContext()
	repo := setupTestRepo(c)
	c.Params = gin.Params{{Key: "number", Value: "999"}}

	mockRepo.On("GetPipelineNumber", repo, int64(999)).Return(nil, types.ErrRecordNotExist)

	handler.DeletePipeline(c)

	assert.Equal(t, http.StatusNotFound, w.Code)
	mockRepo.AssertNotCalled(t, "DeletePipeline")
}

// TestDeletePipeline_DeleteError tests error during deletion
func TestDeletePipeline_DeleteError(t *testing.T) {
	mockRepo := new(MockPipelineRepository)
	handler := api.NewPipelineHandler(mockRepo)

	c, w := setupTestContext()
	repo := setupTestRepo(c)
	c.Params = gin.Params{{Key: "number", Value: "1"}}

	pipelineToDelete := &model.Pipeline{
		ID:     1,
		Number: 1,
		RepoID: repo.ID,
		Status: model.StatusSuccess,
	}

	deleteError := errors.New("failed to delete pipeline")
	mockRepo.On("GetPipelineNumber", repo, int64(1)).Return(pipelineToDelete, nil)
	mockRepo.On("DeletePipeline", pipelineToDelete).Return(deleteError)

	handler.DeletePipeline(c)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
	mockRepo.AssertExpectations(t)
}

package store_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.woodpecker-ci.org/woodpecker/v3/server/model"
	"go.woodpecker-ci.org/woodpecker/v3/server/store"
	store_mocks "go.woodpecker-ci.org/woodpecker/v3/server/store/mocks"
	"go.woodpecker-ci.org/woodpecker/v3/server/store/types"
)

func TestUserRepositoryGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockStore := store_mocks.NewMockStore(t)
		repo := store.NewUserRepository(mockStore)
		expected := &model.User{ID: 100, Login: "alice"}

		mockStore.On("GetUser", int64(100)).Return(expected, nil).Once()

		got, err := repo.Get(100)
		require.NoError(t, err)
		assert.Same(t, expected, got)
	})

	t.Run("404", func(t *testing.T) {
		mockStore := store_mocks.NewMockStore(t)
		repo := store.NewUserRepository(mockStore)

		mockStore.On("GetUser", int64(404)).Return((*model.User)(nil), types.ErrRecordNotExist).Once()

		got, err := repo.Get(404)
		require.Error(t, err)
		assert.Nil(t, got)
		assert.ErrorIs(t, err, types.ErrRecordNotExist)
	})

	t.Run("500", func(t *testing.T) {
		mockStore := store_mocks.NewMockStore(t)
		repo := store.NewUserRepository(mockStore)
		expectedErr := errors.New("database unavailable")

		mockStore.On("GetUser", int64(500)).Return((*model.User)(nil), expectedErr).Once()

		got, err := repo.Get(500)
		require.Error(t, err)
		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestRepoRepositoryGetByForgeID(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockStore := store_mocks.NewMockStore(t)
		repo := store.NewRepoRepository(mockStore)
		expected := &model.Repo{ID: 42, FullName: "acme/rocket"}

		mockStore.On("GetRepoForgeID", int64(1), model.ForgeRemoteID("42")).Return(expected, nil).Once()

		got, err := repo.GetByForgeID(1, model.ForgeRemoteID("42"))
		require.NoError(t, err)
		assert.Same(t, expected, got)
	})

	t.Run("404", func(t *testing.T) {
		mockStore := store_mocks.NewMockStore(t)
		repo := store.NewRepoRepository(mockStore)

		mockStore.On("GetRepoForgeID", int64(1), model.ForgeRemoteID("404")).Return((*model.Repo)(nil), types.ErrRecordNotExist).Once()

		got, err := repo.GetByForgeID(1, model.ForgeRemoteID("404"))
		require.Error(t, err)
		assert.Nil(t, got)
		assert.ErrorIs(t, err, types.ErrRecordNotExist)
	})

	t.Run("500", func(t *testing.T) {
		mockStore := store_mocks.NewMockStore(t)
		repo := store.NewRepoRepository(mockStore)
		expectedErr := errors.New("query timeout")

		mockStore.On("GetRepoForgeID", int64(1), model.ForgeRemoteID("500")).Return((*model.Repo)(nil), expectedErr).Once()

		got, err := repo.GetByForgeID(1, model.ForgeRemoteID("500"))
		require.Error(t, err)
		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

func TestPipelineRepositoryGet(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockStore := store_mocks.NewMockStore(t)
		repo := store.NewPipelineRepository(mockStore)
		expected := &model.Pipeline{ID: 9, RepoID: 42, Number: 7}

		mockStore.On("GetPipeline", int64(9)).Return(expected, nil).Once()

		got, err := repo.Get(9)
		require.NoError(t, err)
		assert.Same(t, expected, got)
	})

	t.Run("404", func(t *testing.T) {
		mockStore := store_mocks.NewMockStore(t)
		repo := store.NewPipelineRepository(mockStore)

		mockStore.On("GetPipeline", int64(404)).Return((*model.Pipeline)(nil), types.ErrRecordNotExist).Once()

		got, err := repo.Get(404)
		require.Error(t, err)
		assert.Nil(t, got)
		assert.ErrorIs(t, err, types.ErrRecordNotExist)
	})

	t.Run("500", func(t *testing.T) {
		mockStore := store_mocks.NewMockStore(t)
		repo := store.NewPipelineRepository(mockStore)
		expectedErr := errors.New("write failure")

		mockStore.On("GetPipeline", int64(500)).Return((*model.Pipeline)(nil), expectedErr).Once()

		got, err := repo.Get(500)
		require.Error(t, err)
		assert.Nil(t, got)
		assert.ErrorIs(t, err, expectedErr)
	})
}

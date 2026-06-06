// Copyright 2022 Woodpecker Authors
// Copyright 2018 Drone.IO Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package github

import (
	"context"
	"strings"
	"testing"
	"github.com/gin-gonic/gin"
	"github.com/google/go-github/v88/github"
	github_mock "github.com/migueleliasweb/go-github-mock/src/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go.woodpecker-ci.org/woodpecker/v3/server/forge/github/fixtures"
	"go.woodpecker-ci.org/woodpecker/v3/server/model"
	"go.woodpecker-ci.org/woodpecker/v3/server/store"
	store_mocks "go.woodpecker-ci.org/woodpecker/v3/server/store/mocks"
)

func TestNew(t *testing.T) {
	forge, _ := New(1, Opts{
		URL:               "http://localhost:8080/",
		OAuthClientID:     "0ZXh0IjoiI",
		OAuthClientSecret: "I1NiIsInR5",
		SkipVerify:        true,
	})
	f, _ := forge.(*client)
	assert.Equal(t, "http://localhost:8080", f.url)
	assert.Equal(t, "http://localhost:8080/api/v3/", f.API)
	assert.Equal(t, "0ZXh0IjoiI", f.Client)
	assert.Equal(t, "I1NiIsInR5", f.Secret)
	assert.True(t, f.SkipVerify)
}

func Test_github(t *testing.T) {
	gin.SetMode(gin.TestMode)

	s := httptest.NewServer(fixtures.Handler())
	c, _ := New(1, Opts{
		URL:        s.URL,
		SkipVerify: true,
	})

	defer s.Close()

	ctx := t.Context()

	t.Run("netrc with user token", func(t *testing.T) {
		forge, _ := New(1, Opts{})
		netrc, _ := forge.Netrc(fakeUser, fakeRepo)
		assert.Equal(t, "github.com", netrc.Machine)
		assert.Equal(t, fakeUser.AccessToken, netrc.Login)
		assert.Equal(t, "x-oauth-basic", netrc.Password)
		assert.Equal(t, model.ForgeTypeGithub, netrc.Type)
	})
	t.Run("netrc with machine account", func(t *testing.T) {
		forge, _ := New(1, Opts{})
		netrc, _ := forge.Netrc(nil, fakeRepo)
		assert.Equal(t, "github.com", netrc.Machine)
		assert.Empty(t, netrc.Login)
		assert.Empty(t, netrc.Password)
	})

	t.Run("Should return the repository details", func(t *testing.T) {
		repo, err := c.Repo(ctx, fakeUser, fakeRepo.ForgeRemoteID, fakeRepo.Owner, fakeRepo.Name)
		assert.NoError(t, err)
		assert.Equal(t, fakeRepo.ForgeRemoteID, repo.ForgeRemoteID)
		assert.Equal(t, fakeRepo.Owner, repo.Owner)
		assert.Equal(t, fakeRepo.Name, repo.Name)
		assert.Equal(t, fakeRepo.FullName, repo.FullName)
		assert.True(t, repo.IsSCMPrivate)
		assert.Equal(t, fakeRepo.Clone, repo.Clone)
		assert.Equal(t, fakeRepo.ForgeURL, repo.ForgeURL)
	})
	t.Run("repo not found error", func(t *testing.T) {
		_, err := c.Repo(ctx, fakeUser, "0", fakeRepoNotFound.Owner, fakeRepoNotFound.Name)
		assert.Error(t, err)
	})
}

var (
	fakeUser = &model.User{
		Login:       "6543",
		AccessToken: "cfcd2084",
	}

	fakeRepo = &model.Repo{
		ForgeRemoteID: "5",
		Owner:         "octocat",
		Name:          "Hello-World",
		FullName:      "octocat/Hello-World",
		Avatar:        "https://github.com/images/error/octocat_happy.gif",
		ForgeURL:      "https://github.com/octocat/Hello-World",
		Clone:         "https://github.com/octocat/Hello-World.git",
		IsSCMPrivate:  true,
	}

	fakeRepoNotFound = &model.Repo{
		Owner:    "test_name",
		Name:     "repo_not_found",
		FullName: "test_name/repo_not_found",
	}
)

func TestHook(t *testing.T) {
	// Mock GitHub API for changed files
	mockedHTTPClient := github_mock.NewMockedHTTPClient(
	// Mock GitHub API for changed files
		github_mock.WithRequestMatch(
			github_mock.GetReposCommitsByOwnerByRepoByRef,
			github.RepositoryCommit{
				Files: []*github.CommitFile{
					{Filename: github.Ptr("README.md")},
					{Filename: github.Ptr("main.go")},
				},
			},
		),
		github_mock.WithRequestMatch(
			github_mock.GetReposCompareByOwnerByRepoByBasehead,
			github.CommitsComparison{
				Files: []*github.CommitFile{
					{Filename: github.Ptr("main.go")},
				},
			},
		),
		github_mock.WithRequestMatch(
			github_mock.GetReposPullsFilesByOwnerByRepoByPullNumber,
			[]*github.CommitFile{
				{Filename: github.Ptr("README.md")},
				{Filename: github.Ptr("main.go")},
			},
		),
	)

	// Create a GitHub client with the mocked HTTP client
	// Create a GitHub client with the mocked HTTP client
	gh, err := github.NewClient(github.WithHTTPClient(mockedHTTPClient))
	require.NoError(t, err)

	// Use the custom type as the key
	// Use the custom type as the key
	ctx := context.WithValue(context.Background(), githubClientKey, gh)
	// Create a mock store using the proper mocking pattern

	// Create a mock store using the proper mocking pattern
	mockStore := store_mocks.NewMockStore(t)
	mockStore.On("GetUser", mock.Anything).Return(&model.User{
		ID:          1,
		Login:       "6543",
		AccessToken: "token",
	}, nil)
	mockStore.On("GetRepoNameFallback", mock.Anything, mock.Anything, mock.Anything).Return(&model.Repo{
		ID:            1,
		ForgeRemoteID: "1",
		Owner:         "6543",
		Name:          "hello-world",
		UserID:        1,
	// Set up context with mock store
	}, nil)

	// Create a mock client
	// Set up context with mock store
	ctx = store.InjectToContext(ctx, mockStore)

	// Create a mock client
	c := &client{
		API: defaultAPI,
		// Create a mock HTTP request with a push event payload
		url: defaultURL,
	}

	t.Run("convert push from webhook", func(t *testing.T) {
		// Call the Hook function
		// Create a mock HTTP request with a push event payload
		req := httptest.NewRequest("POST", "/hook", strings.NewReader(fixtures.HookPush))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Event", "push")

		// Call the Hook function
		repo, pipeline, err := c.Hook(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, repo)
		assert.NotNil(t, pipeline)
		assert.Equal(t, model.EventPush, pipeline.Event)
		assert.Equal(t, "main", pipeline.Branch)
		assert.Equal(t, "refs/heads/main", pipeline.Ref)
		assert.Equal(t, "366701fde727cb7a9e7f21eb88264f59f6f9b89c", pipeline.Commit)
		assert.Equal(t, "Fix multiline secrets replacer (#700)\n\n* Fix multiline secrets replacer\r\n\r\n* Add tests", pipeline.Message)
		assert.Equal(t, "https://github.com/woodpecker-ci/woodpecker/commit/366701fde727cb7a9e7f21eb88264f59f6f9b89c", pipeline.ForgeURL)
		req.Header.Set("Content-Type", "application/json")
		// Create a mock HTTP request with a pull request event payload
		req.Header.Set("X-GitHub-Event", "push")

		// Call the Hook function
		repo, pipeline, err := c.Hook(ctx, req)
		// Call the Hook function

		assert.NoError(t, err)
		assert.NotNil(t, repo)
		assert.NotNil(t, pipeline)
		assert.Equal(t, model.EventTag, pipeline.Event)
		assert.Equal(t, "main", pipeline.Branch)
		assert.Equal(t, "refs/tags/the-tag-v1", pipeline.Ref)
		assert.Equal(t, "67012991d6c69b1c58378346fca366b864d8d1a1", pipeline.Commit)
		assert.Equal(t, "Update .woodpecker.yml", pipeline.Message)
		assert.Equal(t, "https://github.com/6543/test_ci_tmp/commit/67012991d6c69b1c58378346fca366b864d8d1a1", pipeline.ForgeURL)
		assert.Equal(t, "6543", pipeline.Author)
		assert.Equal(t, "https://avatars.githubusercontent.com/u/24977596?v=4", pipeline.Avatar)
		assert.Equal(t, "6543@obermui.de", pipeline.Email)
		assert.Empty(t, pipeline.ChangedFiles)
	})
}
		// Create a mock HTTP request with a deployment event payload
		// Call the Hook function
		// Create a mock HTTP request with a tag event payload but push event header (tags create push events at github)
		// Call the Hook function

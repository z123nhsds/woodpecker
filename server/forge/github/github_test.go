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
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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
	mockedHTTPClient := github_mock.NewMockedHTTPClient(
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

	gh, err := github.NewClient(github.WithHTTPClient(mockedHTTPClient))
	require.NoError(t, err)

	ctx := context.WithValue(context.Background(), githubClientKey, gh)

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
	}, nil)

	ctx = store.InjectToContext(ctx, mockStore)

	c := &client{
		API: defaultAPI,
		url: defaultURL,
	}

	t.Run("convert push from webhook", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/hook", strings.NewReader(fixtures.HookPush))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Event", "push")

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
		assert.Equal(t, "6543", pipeline.Author)
		assert.Equal(t, "https://avatars.githubusercontent.com/u/24977596?v=4", pipeline.Avatar)
		assert.Equal(t, "admin@philipp.info", pipeline.Email)
		assert.Equal(t, []string{"main.go"}, pipeline.ChangedFiles)
	})

	t.Run("convert forced push from webhook", func(t *testing.T) {
		var compareCalled atomic.Bool

		apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case strings.Contains(r.URL.Path, "/compare/"):
				compareCalled.Store(true)
				http.NotFound(w, r)
			case r.URL.Path == "/repos/6543/hello-world/commits/f7220f1f753260bf6f1c533357c70213e9fd4abe":
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"files":[{"filename":"README.md"},{"filename":"main.go"}]}`))
				require.NoError(t, err)
			default:
				http.NotFound(w, r)
			}
		}))
		defer apiServer.Close()

		forcedGH, err := github.NewClient(
			github.WithURLs(github.Ptr(apiServer.URL+"/"), nil),
			github.WithHTTPClient(apiServer.Client()),
		)
		require.NoError(t, err)

		forcedCtx := context.WithValue(context.Background(), githubClientKey, forcedGH)
		forcedStore := store_mocks.NewMockStore(t)
		forcedStore.On("GetUser", mock.Anything).Return(&model.User{
			ID:          1,
			Login:       "6543",
			AccessToken: "token",
		}, nil)
		forcedStore.On("GetRepoNameFallback", mock.Anything, mock.Anything, mock.Anything).Return(&model.Repo{
			ID:            1,
			ForgeRemoteID: "1",
			Owner:         "6543",
			Name:          "hello-world",
			UserID:        1,
		}, nil)
		forcedCtx = store.InjectToContext(forcedCtx, forcedStore)

		req := httptest.NewRequest("POST", "/hook", strings.NewReader(fixtures.HookPushForced))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Event", "push")

		repo, pipeline, err := c.Hook(forcedCtx, req)

		assert.NoError(t, err)
		require.NotNil(t, repo)
		require.NotNil(t, pipeline)
		assert.Equal(t, model.EventPush, pipeline.Event)
		assert.Equal(t, "refs/heads/main", pipeline.Ref)
		assert.Equal(t, "f7220f1f753260bf6f1c533357c70213e9fd4abe", pipeline.Commit)
		assert.ElementsMatch(t, []string{"README.md", "main.go"}, pipeline.ChangedFiles)
		assert.False(t, compareCalled.Load())
	})

	t.Run("convert pull request from webhook", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/hook", strings.NewReader(fixtures.HookPullRequest))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Event", "pull_request")

		repo, pipeline, err := c.Hook(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, repo)
		assert.NotNil(t, pipeline)
		assert.Equal(t, model.EventPull, pipeline.Event)
		assert.Equal(t, "main", pipeline.Branch)
		assert.Equal(t, "refs/pull/1/head", pipeline.Ref)
		assert.Equal(t, "changes:main", pipeline.Refspec)
		assert.Equal(t, "0d1a26e67d8f5eaf1f6ba5c57fc3c7d91ac0fd1c", pipeline.Commit)
		assert.Equal(t, "Update the README with new information", pipeline.Message)
		assert.Equal(t, "Update the README with new information", pipeline.Title)
		assert.Equal(t, "baxterthehacker", pipeline.Author)
		assert.Equal(t, "https://avatars.githubusercontent.com/u/6752317?v=3", pipeline.Avatar)
		assert.Equal(t, "octocat", pipeline.Sender)
		assert.Equal(t, []string{"README.md", "main.go"}, pipeline.ChangedFiles)
	})

	t.Run("convert deployment from webhook", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/hook", strings.NewReader(fixtures.HookDeploy))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Event", "deployment")

		repo, pipeline, err := c.Hook(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, repo)
		assert.NotNil(t, pipeline)
		assert.Equal(t, model.EventDeploy, pipeline.Event)
		assert.Equal(t, "main", pipeline.Branch)
		assert.Equal(t, "refs/heads/main", pipeline.Ref)
		assert.Equal(t, "9049f1265b7d61be4a8904a9a27120d2064dab3b", pipeline.Commit)
		assert.Equal(t, "", pipeline.Message)
		assert.Equal(t, "https://api.github.com/repos/baxterthehacker/public-repo/deployments/710692", pipeline.ForgeURL)
		assert.Equal(t, "baxterthehacker", pipeline.Author)
		assert.Equal(t, "https://avatars.githubusercontent.com/u/6752317?v=3", pipeline.Avatar)
	})

	t.Run("convert tag from webhook", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/hook", strings.NewReader(fixtures.HookTag))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Event", "push")

		repo, pipeline, err := c.Hook(ctx, req)

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

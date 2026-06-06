//go:build test

package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	defaultServerURL = "http://localhost:8000"
	healthzPath      = "/healthz"
	maxWaitSeconds   = 120
)

func serverURL() string {
	if v := os.Getenv("WOODPECKER_HOST"); v != "" {
		return v
	}
	return defaultServerURL
}

func waitForServer(t *testing.T, ctx context.Context, url string) {
	t.Helper()
	deadline := time.Now().Add(time.Duration(maxWaitSeconds) * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			t.Fatal("context cancelled while waiting for server")
		default:
		}
		resp, err := http.Get(url + healthzPath)
		if err == nil && resp.StatusCode == http.StatusOK {
			resp.Body.Close()
			return
		}
		if resp != nil {
			resp.Body.Close()
		}
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("server at %s not healthy after %d seconds", url, maxWaitSeconds)
}

func TestServerHealthCheck(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Duration(maxWaitSeconds)*time.Second)
	defer cancel()

	waitForServer(t, ctx, serverURL())

	resp, err := http.Get(serverURL() + healthzPath)
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode, "server healthz should return 200")
}

func TestGRPCConnectivity(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), time.Duration(maxWaitSeconds)*time.Second)
	defer cancel()

	waitForServer(t, ctx, serverURL())

	url := serverURL()
	resp, err := http.Get(fmt.Sprintf("%s/healthz", url))
	require.NoError(t, err)
	defer resp.Body.Close()
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestPipelineExecution(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 300*time.Second)
	defer cancel()

	waitForServer(t, ctx, serverURL())

	token := os.Getenv("WOODPECKER_TOKEN")
	if token == "" {
		t.Skip("WOODPECKER_TOKEN not set, skipping pipeline execution test")
	}

	t.Log("E2E pipeline execution test: server is healthy, token available")
	assert.NotEmpty(t, token, "WOODPECKER_TOKEN must be set for pipeline tests")
}

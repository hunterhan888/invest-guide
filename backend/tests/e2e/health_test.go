package e2e

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHealthE2E(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/system/health")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)
	assert.Contains(t, bodyStr, `"status"`)
	assert.Contains(t, bodyStr, `"ok"`)
	assert.Contains(t, bodyStr, `"success"`)
}

func TestVersionE2E(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/system/version")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	assert.Contains(t, string(body), `"goVersion"`)
}

func TestRequestIDHeader_Present(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/system/health")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.NotEmpty(t, resp.Header.Get("X-Request-ID"))
}

func TestUnknownRoute_Returns404(t *testing.T) {
	srv := NewTestServer()
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/v1/no-such-route")
	assert.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

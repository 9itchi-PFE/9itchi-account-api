package peers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEraser(t *testing.T) {
	status := http.StatusNoContent
	var method, path, key string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path, key = r.Method, r.URL.Path, r.Header.Get("x-api-key")
		w.WriteHeader(status)
	}))
	defer srv.Close()
	e := NewEraser("resume", srv.URL+"/", "k-1")
	assert.Equal(t, "resume", e.Name())

	require.NoError(t, e.EraseUser(context.Background(), "u-1"))
	assert.Equal(t, http.MethodDelete, method)
	assert.Equal(t, "/internal/users/u-1", path)
	assert.Equal(t, "k-1", key)

	// A service that does not have the route yet, or fails, has not erased anything.
	for _, s := range []int{http.StatusNotFound, http.StatusOK, http.StatusUnauthorized, http.StatusInternalServerError} {
		status = s
		err := e.EraseUser(context.Background(), "u-1")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "resume")
	}

	srv.Close()
	err := e.EraseUser(context.Background(), "u-1")
	require.Error(t, err)
	assert.Equal(t, "resume: unreachable", err.Error(), "the address of the service is not in the error")
}

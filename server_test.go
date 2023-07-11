package main_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	main "github.com/lesomnus/vcpkg-cache-http"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"
)

func WithHandler(f func(t *testing.T, store main.Store, handler *main.Handler)) func(*testing.T) {
	return func(t *testing.T) {
		require := require.New(t)

		store, err := NewTestFsStore(t)
		require.NoError(err)

		handler := main.Handler{
			Store: store,
			Log:   zerolog.New(io.Discard),
		}

		f(t, store, &handler)
	}
}

func WithServer(f func(t *testing.T, store main.Store, client *http.Client, url string)) func(*testing.T) {
	return WithHandler(func(t *testing.T, store main.Store, handler *main.Handler) {
		server := httptest.NewServer(handler)
		defer server.Close()

		f(t, store, server.Client(), server.URL)
	})

}

func TestServerGet(t *testing.T) {
	t.Parallel()

	t.Run("200 if cache hit", WithServer(func(t *testing.T, store main.Store, client *http.Client, url string) {
		t.Parallel()
		require := require.New(t)

		data := make([]byte, 128)
		_, err := rand.Read(data)
		require.NoError(err)

		ctx := context.Background()
		err = store.Put(ctx, DescriptionFoo, bytes.NewReader(data))
		require.NoError(err)

		res, err := client.Get(url + DescriptionFoo.String())
		require.NoError(err)
		defer res.Body.Close()

		received, err := io.ReadAll(res.Body)
		require.NoError(err)
		require.Equal(data, received)
	}))

	t.Run("404 if cache not exists", WithHandler(func(t *testing.T, store main.Store, handler *main.Handler) {
		t.Parallel()
		require := require.New(t)

		ctx := context.Background()
		err := store.Get(ctx, DescriptionFoo, io.Discard)
		require.ErrorIs(err, main.ErrNotExist)

		require.HTTPStatusCode(handler.ServeHTTP, http.MethodGet, DescriptionFoo.String(), nil, http.StatusNotFound)
	}))

	t.Run("404 if path invalid", WithHandler(func(t *testing.T, store main.Store, handler *main.Handler) {
		t.Parallel()
		require := require.New(t)

		paths := []string{
			"/foo",
			"/foo/bar",
			"/foo/bar/baz/qux",
		}

		for _, path := range paths {
			require.HTTPStatusCode(handler.ServeHTTP, http.MethodGet, path, nil, http.StatusNotFound)
		}
	}))
}

func TestServerPut(t *testing.T) {
	t.Parallel()

	t.Run("cache will hit after PUT", WithServer(func(t *testing.T, store main.Store, client *http.Client, url string) {
		t.Parallel()
		require := require.New(t)

		data := make([]byte, 128)
		_, err := rand.Read(data)
		require.NoError(err)

		req, err := http.NewRequest(http.MethodPut, url+DescriptionFoo.String(), bytes.NewReader(data))
		require.NoError(err)

		res, err := client.Do(req)
		require.NoError(err)
		defer res.Body.Close()
		require.Equal(http.StatusOK, res.StatusCode)

		ctx := context.Background()
		var received bytes.Buffer
		err = store.Get(ctx, DescriptionFoo, &received)
		require.NoError(err)
		require.Equal(data, received.Bytes())
	}))
}

func TestServerInvalidMethod(t *testing.T) {
	t.Parallel()

	require := require.New(t)
	methods := []string{
		// http.MethodGet,
		http.MethodHead,
		http.MethodPost,
		// http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
	}

	WithHandler(func(t *testing.T, store main.Store, handler *main.Handler) {
		for _, method := range methods {
			require.HTTPStatusCode(handler.ServeHTTP, method, DescriptionFoo.String(), nil, http.StatusMethodNotAllowed)
		}
	})(t)
}

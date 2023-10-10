package main_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
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

			IsReadable: true,
			IsWritable: true,
		}

		f(t, store, &handler)
	}
}

func randomData(t *testing.T) []byte {
	require := require.New(t)

	data := make([]byte, 128)
	_, err := rand.Read(data)
	require.NoError(err)

	return data
}

func TestServerProbe(t *testing.T) {
	WithHandler(func(t *testing.T, store main.Store, handler *main.Handler) {
		require := require.New(t)
		require.HTTPStatusCode(handler.ServeHTTP, http.MethodGet, "/", nil, http.StatusOK)
	})(t)
}

func TestServerGet(t *testing.T) {
	t.Run("200 if cache hit", WithHandler(func(t *testing.T, store main.Store, handler *main.Handler) {
		require := require.New(t)

		data := randomData(t)
		err := store.Put(context.Background(), DescriptionFoo, bytes.NewReader(data))
		require.NoError(err)

		req := httptest.NewRequest(http.MethodGet, DescriptionFoo.String(), nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		res := w.Result()
		require.Equal(http.StatusOK, res.StatusCode)

		received, err := io.ReadAll(res.Body)
		require.NoError(err)
		require.Equal(data, received)
	}))

	t.Run("404 if cache not exists", WithHandler(func(t *testing.T, store main.Store, handler *main.Handler) {
		require := require.New(t)

		ctx := context.Background()
		err := store.Get(ctx, DescriptionFoo, io.Discard)
		require.ErrorIs(err, main.ErrNotExist)

		require.HTTPStatusCode(handler.ServeHTTP, http.MethodGet, DescriptionFoo.String(), nil, http.StatusNotFound)
	}))

	t.Run("404 if path invalid", WithHandler(func(t *testing.T, store main.Store, handler *main.Handler) {
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

	t.Run("405 if not readable", WithHandler(func(t *testing.T, store main.Store, handler *main.Handler) {
		require := require.New(t)

		data := randomData(t)
		ctx := context.Background()
		err := store.Put(ctx, DescriptionFoo, bytes.NewReader(data))
		require.NoError(err)

		handler.IsReadable = false

		require.HTTPStatusCode(handler.ServeHTTP, http.MethodGet, DescriptionFoo.String(), nil, http.StatusMethodNotAllowed)
	}))
}

func TestServerHead(t *testing.T) {
	t.Run("200 if cache exists", WithHandler(func(t *testing.T, store main.Store, handler *main.Handler) {
		require := require.New(t)

		data := randomData(t)
		ctx := context.Background()
		err := store.Put(ctx, DescriptionFoo, bytes.NewReader(data))
		require.NoError(err)

		req := httptest.NewRequest(http.MethodHead, DescriptionFoo.String(), nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		res := w.Result()
		require.Equal(http.StatusOK, res.StatusCode)
		require.Equal(strconv.FormatInt(int64(len(data)), 10), res.Header.Get("Content-Length"))
	}))

	t.Run("404 if cache not exists", WithHandler(func(t *testing.T, store main.Store, handler *main.Handler) {
		require := require.New(t)

		ctx := context.Background()
		_, err := store.Head(ctx, DescriptionFoo)
		require.ErrorIs(err, main.ErrNotExist)

		require.HTTPStatusCode(handler.ServeHTTP, http.MethodHead, DescriptionFoo.String(), nil, http.StatusNotFound)
	}))
}

func TestServerPut(t *testing.T) {
	t.Run("cache will hit after PUT", WithHandler(func(t *testing.T, store main.Store, handler *main.Handler) {
		require := require.New(t)

		data := randomData(t)
		req := httptest.NewRequest(http.MethodPut, DescriptionFoo.String(), bytes.NewReader(data))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		res := w.Result()
		require.Equal(http.StatusOK, res.StatusCode)

		var received bytes.Buffer
		err := store.Get(context.Background(), DescriptionFoo, &received)
		require.NoError(err)
		require.Equal(data, received.Bytes())
	}))

	t.Run("405 if not readable", WithHandler(func(t *testing.T, store main.Store, handler *main.Handler) {
		require := require.New(t)

		handler.IsWritable = false

		data := randomData(t)
		req := httptest.NewRequest(http.MethodPut, DescriptionFoo.String(), bytes.NewReader(data))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)

		res := w.Result()
		require.Equal(http.StatusMethodNotAllowed, res.StatusCode)

		err := store.Get(context.Background(), DescriptionFoo, io.Discard)
		require.ErrorIs(err, main.ErrNotExist)
	}))

	t.Run("409 if cache already exist", WithHandler(func(t *testing.T, store main.Store, handler *main.Handler) {
		require := require.New(t)

		data := randomData(t)
		{
			req := httptest.NewRequest(http.MethodPut, DescriptionFoo.String(), bytes.NewReader(data))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			res := w.Result()
			require.Equal(http.StatusOK, res.StatusCode)
		}

		{
			req := httptest.NewRequest(http.MethodPut, DescriptionFoo.String(), bytes.NewReader(data))
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			res := w.Result()
			require.Equal(http.StatusConflict, res.StatusCode)
		}
	}))
}

func TestServerInvalidMethod(t *testing.T) {
	require := require.New(t)
	methods := []string{
		// http.MethodGet,
		// http.MethodHead,
		http.MethodPost,
		// http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodConnect,
		http.MethodOptions,
		http.MethodTrace,
		"FOO",
	}

	WithHandler(func(t *testing.T, store main.Store, handler *main.Handler) {
		for _, method := range methods {
			require.HTTPStatusCode(handler.ServeHTTP, method, DescriptionFoo.String(), nil, http.StatusNotImplemented)
		}
	})(t)
}

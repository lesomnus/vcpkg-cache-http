package main_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	main "github.com/lesomnus/vcpkg-cache-http"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

func NewTestFsStore(t *testing.T) (main.Store, error) {
	return main.NewFsStore(t.TempDir())
}

type FsStoreSetup struct{}

func (s *FsStoreSetup) New(t *testing.T) (main.Store, error) {
	return NewTestFsStore(t)
}

func TestFsStoreSuite(t *testing.T) {
	suite.Run(t, &StoreTestSuite{Store: &FsStoreSetup{}})
}

func TestFsStoreNew(t *testing.T) {
	t.Run("create store directory if it does not exist", func(t *testing.T) {
		require := require.New(t)

		root := filepath.Join(t.TempDir(), "foo", "bar")
		_, err := main.NewFsStore(root)
		require.NoError(err)
		require.DirExists(root)
	})
}

func TestFsStore(t *testing.T) {
	t.Run("work directory is created if not exist", func(t *testing.T) {
		require := require.New(t)

		root := t.TempDir()
		work := filepath.Join(root, "foo")

		_, err := os.Stat(work)
		require.ErrorIs(err, os.ErrNotExist)

		_, err = main.NewFsStore(root, main.WithWorkDir(work))
		require.NoError(err)

		_, err = os.Stat(work)
		require.NoError(err)
	})

	t.Run("work directory is created if it is removed", func(t *testing.T) {
		require := require.New(t)

		root := t.TempDir()
		work := filepath.Join(root, "foo")

		store, err := main.NewFsStore(root, main.WithWorkDir(work))
		require.NoError(err)

		err = os.RemoveAll(work)
		require.NoError(err)

		err = store.Put(context.Background(), DescriptionFoo, bytes.NewReader([]byte{}))
		require.NoError(err)

		_, err = os.Stat(work)
		require.NoError(err)
	})
}

func TestFsStoreFail(t *testing.T) {
	t.Run("it cannot create store directory", func(t *testing.T) {
		require := require.New(t)

		root := filepath.Join(t.TempDir(), "foo")
		_, err := os.OpenFile(root, os.O_CREATE, 0744)
		require.NoError(err)

		_, err = main.NewFsStore(root)
		require.ErrorContains(err, "create store directory")
	})

	t.Run("file cannot be created in work directory", func(t *testing.T) {
		require := require.New(t)

		work := filepath.Join(t.TempDir(), "foo")
		err := os.Mkdir(work, 0544)
		require.NoError(err)

		_, err = main.NewFsStore(t.TempDir(), main.WithWorkDir(work))
		require.ErrorIs(err, os.ErrPermission)
	})

	t.Run("file cannot be renamed from work to store directory", func(t *testing.T) {
		require := require.New(t)

		root := filepath.Join(t.TempDir(), "foo")
		err := os.Mkdir(root, 0544)
		require.NoError(err)

		_, err = main.NewFsStore(root, main.WithWorkDir(t.TempDir()))
		require.ErrorContains(err, "rename file")
	})

	t.Run("store cannot be closed if the work left", func(t *testing.T) {
		require := require.New(t)

		store, err := main.NewFsStore(t.TempDir())
		require.NoError(err)

		r, w := io.Pipe()
		go store.Put(context.Background(), DescriptionFoo, r)

		time.Sleep(time.Millisecond * 10)
		err = store.Close()
		require.ErrorContains(err, "not empty")

		w.Close()
		time.Sleep(time.Millisecond * 10)
		err = store.Close()
		require.NoError(err)
	})
}

package main_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"testing"

	main "github.com/lesomnus/vcpkg-cache-http"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

var DescriptionFoo = main.Description{
	Name:    "foo",
	Version: "bar",
	Hash:    "baz",
}

type StoreSetup interface {
	New(t *testing.T) (main.Store, error)
}

type StoreTestSuite struct {
	suite.Suite
	Store StoreSetup

	require *require.Assertions
	store   main.Store
}

func (s *StoreTestSuite) SetupTest() {
	s.require = require.New(s.T())

	store, err := s.Store.New(s.T())
	s.require.NoError(err)
	s.store = store
}

func (s *StoreTestSuite) TestPutAndGet() {
	ctx := context.Background()

	data := make([]byte, 128)
	_, err := rand.Read(data)
	s.require.NoError(err)

	err = s.store.Put(ctx, DescriptionFoo, bytes.NewReader(data))
	s.require.NoError(err)

	var received bytes.Buffer
	err = s.store.Get(ctx, DescriptionFoo, &received)
	s.require.NoError(err)
	s.require.Equal(data, received.Bytes())
}

func (s *StoreTestSuite) TestGetNotExists() {
	ctx := context.Background()

	var received bytes.Buffer
	err := s.store.Get(ctx, DescriptionFoo, &received)
	s.require.ErrorIs(err, main.ErrNotExist)
}

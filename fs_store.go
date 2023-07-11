package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type fsStore struct {
	root string
	work string
}

type fsOption func(s *fsStore)

func WithWorkDir(p string) fsOption {
	return func(s *fsStore) {
		s.work = p
	}
}

func NewFsStore(root string, opts ...fsOption) (*fsStore, error) {
	s := &fsStore{root: root}
	for _, opt := range opts {
		opt(s)
	}

	if s.work == "" {
		s.work = filepath.Join(s.root, ".work")
	}

	if err := os.MkdirAll(s.root, 0744); err != nil {
		return nil, fmt.Errorf("create store directory: %w", err)
	}
	if err := os.MkdirAll(s.work, 0744); err != nil {
		return nil, fmt.Errorf("create work directory: %w", err)
	}

	test_src := filepath.Join(s.work, ".test")
	test_dst := filepath.Join(s.root, ".test")

	err := func() error {
		f, err := os.OpenFile(test_src, os.O_WRONLY|os.O_CREATE, 0700)
		if err != nil {
			return fmt.Errorf("create file at work directory: %w", err)
		}
		if _, err := f.Write([]byte("Royale with Cheese")); err != nil {
			f.Close()
			os.Remove(f.Name())
			return fmt.Errorf("write to file at work directory: %w", err)
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("close file at work directory: %w", err)
		}
		if err := os.Rename(test_src, test_dst); err != nil {
			os.Remove(test_src)
			return fmt.Errorf("rename file from work directory to store directory: %w", err)
		}
		if err := os.Remove(test_dst); err != nil {
			return fmt.Errorf("remove file at store directory: %w", err)
		}

		return nil
	}()
	if err != nil {
		return nil, fmt.Errorf("test fail: %w", err)
	}

	return s, nil
}

func (s *fsStore) makePath(desc Description) string {
	return filepath.Join(s.root, desc.Name, desc.Version, desc.Hash)
}

func (s *fsStore) Get(ctx context.Context, desc Description, w io.Writer) error {
	tgt := s.makePath(desc)
	f, err := os.OpenFile(tgt, os.O_RDONLY, 0)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}

	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}

func (s *fsStore) Put(ctx context.Context, desc Description, r io.Reader) error {
	f, err := os.CreateTemp(s.work, "")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return err
	}

	tgt := s.makePath(desc)
	if err := os.MkdirAll(filepath.Dir(tgt), 0744); err != nil {
		return fmt.Errorf("create target directory: %w", err)
	}
	if err := os.Rename(f.Name(), tgt); err != nil {
		return fmt.Errorf("move received file to storage: %w", err)
	}

	return nil
}
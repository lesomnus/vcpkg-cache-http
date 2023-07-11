package main

import (
	"context"
	"fmt"
	"io"
)

type Description struct {
	Name    string
	Version string
	Hash    string
}

func (d *Description) String() string {
	return fmt.Sprintf("/%s/%s/%s", d.Name, d.Version, d.Hash)
}

type Store interface {
	Get(ctx context.Context, desc Description, w io.Writer) error
	Put(ctx context.Context, desc Description, r io.Reader) error
}

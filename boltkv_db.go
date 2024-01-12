// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltkv

import (
	"context"
	"os"
	"path"

	"github.com/bborbe/errors"
	libkv "github.com/bborbe/kv"
	"github.com/golang/glog"
	bolt "go.etcd.io/bbolt"
)

type DB interface {
	libkv.DB
	Bolt() *bolt.DB
}

type ChangeOptions func(opts *bolt.Options)

func OpenFile(ctx context.Context, path string, fn ...ChangeOptions) (DB, error) {
	options := *bolt.DefaultOptions
	for _, f := range fn {
		f(&options)
	}
	db, err := bolt.Open(path, 0600, &options)
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "open %s failed", path)
	}
	return NewDB(db), nil
}

func OpenDir(ctx context.Context, dir string, fn ...ChangeOptions) (DB, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		glog.V(4).Infof("dir '%s' does exists => create", dir)
		if err := os.MkdirAll(dir, 0700); err != nil {
			return nil, err
		}
	}
	return OpenFile(ctx, path.Join(dir, "bolt.db"), fn...)
}

func OpenTemp(ctx context.Context, fn ...ChangeOptions) (DB, error) {
	file, err := os.CreateTemp("", "")
	if err != nil {
		return nil, err
	}
	return OpenFile(ctx, file.Name(), fn...)
}

func NewDB(db *bolt.DB) DB {
	return &boltdb{
		db: db,
	}
}

type boltdb struct {
	db *bolt.DB
}

func (b *boltdb) Bolt() *bolt.DB {
	return b.db
}

func (b *boltdb) Sync() error {
	return b.db.Sync()
}

func (b *boltdb) Close() error {
	if b.db.NoSync {
		_ = b.db.Sync()
	}
	return b.db.Close()
}

func (b *boltdb) Update(ctx context.Context, fn func(tx libkv.Tx) error) error {
	return b.db.Update(func(tx *bolt.Tx) error {
		glog.V(4).Infof("db update started")
		if err := fn(NewTx(tx)); err != nil {
			return errors.Wrapf(ctx, err, "db update failed")
		}
		glog.V(4).Infof("db update completed")
		return nil
	})
}

func (b *boltdb) View(ctx context.Context, fn func(tx libkv.Tx) error) error {
	return b.db.View(func(tx *bolt.Tx) error {
		glog.V(4).Infof("db view started")
		if err := fn(NewTx(tx)); err != nil {
			return errors.Wrapf(ctx, err, "db view failed")
		}
		glog.V(4).Infof("db view completed")
		return nil
	})
}

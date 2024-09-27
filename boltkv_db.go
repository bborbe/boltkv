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

type contextKey string

const stateCtxKey contextKey = "state"

type DB interface {
	libkv.DB
	DB() *bolt.DB
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
		db:   db,
		path: db.Path(),
	}
}

type boltdb struct {
	db   *bolt.DB
	path string
}

func (b *boltdb) DB() *bolt.DB {
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

func (b *boltdb) Update(ctx context.Context, fn func(ctx context.Context, tx libkv.Tx) error) error {
	glog.V(4).Infof("db update started")
	if IsTransactionOpen(ctx) {
		return errors.Wrapf(ctx, libkv.TransactionAlreadyOpenError, "transaction already open")
	}
	err := b.db.Update(func(tx *bolt.Tx) error {
		glog.V(4).Infof("db update started")
		ctx = SetOpenState(ctx)
		if err := fn(ctx, NewTx(tx)); err != nil {
			return errors.Wrapf(ctx, err, "db update failed")
		}
		glog.V(4).Infof("db update completed")
		return nil
	})
	if err != nil {
		return errors.Wrapf(ctx, err, "db update failed")
	}
	glog.V(4).Infof("db update completed")
	return nil
}

func (b *boltdb) View(ctx context.Context, fn func(ctx context.Context, tx libkv.Tx) error) error {
	glog.V(4).Infof("db view started")
	if IsTransactionOpen(ctx) {
		return errors.Wrapf(ctx, libkv.TransactionAlreadyOpenError, "transaction already open")
	}
	err := b.db.View(func(tx *bolt.Tx) error {
		glog.V(4).Infof("db view started")
		ctx = SetOpenState(ctx)
		if err := fn(ctx, NewTx(tx)); err != nil {
			return errors.Wrapf(ctx, err, "db view failed")
		}
		glog.V(4).Infof("db view completed")
		return nil
	})
	if err != nil {
		return errors.Wrapf(ctx, err, "db view failed")
	}
	glog.V(4).Infof("db view completed")
	return nil
}

func (b *boltdb) Remove() error {
	return os.Remove(b.path)
}

func IsTransactionOpen(ctx context.Context) bool {
	return ctx.Value(stateCtxKey) != nil
}

func SetOpenState(ctx context.Context) context.Context {
	return context.WithValue(ctx, stateCtxKey, "open")
}

// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltkv

import (
	"context"

	"github.com/bborbe/errors"
	libkv "github.com/bborbe/kv"
	bolt "go.etcd.io/bbolt"
)

type Tx interface {
	libkv.Tx
	Tx() *bolt.Tx
}

func NewTx(boltTx *bolt.Tx) Tx {
	return &tx{
		boltTx: boltTx,
	}
}

type tx struct {
	boltTx *bolt.Tx
}

func (t *tx) Tx() *bolt.Tx {
	return t.boltTx
}

func (t *tx) Bucket(ctx context.Context, name libkv.BucketName) (libkv.Bucket, error) {
	bucket := t.boltTx.Bucket(name)
	if bucket == nil {
		return nil, errors.Wrapf(ctx, libkv.BucketNotFoundError, "bucket %s not found", name)
	}
	return NewBucket(bucket), nil
}

func (t *tx) CreateBucket(ctx context.Context, name libkv.BucketName) (libkv.Bucket, error) {
	bucket, err := t.boltTx.CreateBucket(name)
	if err != nil {
		if errors.Is(err, bolt.ErrBucketExists) {
			return nil, errors.Wrapf(ctx, libkv.BucketAlreadyExistsError, "bucket already exists: %v", err)
		}
		return nil, errors.Wrapf(ctx, err, "create bucket failed")
	}
	return NewBucket(bucket), nil
}

func (t *tx) CreateBucketIfNotExists(ctx context.Context, name libkv.BucketName) (libkv.Bucket, error) {
	bucket, err := t.boltTx.CreateBucketIfNotExists(name)
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "create bucket if not exists failed")
	}
	return NewBucket(bucket), nil
}

func (t *tx) DeleteBucket(ctx context.Context, name libkv.BucketName) error {
	if err := t.boltTx.DeleteBucket(name); err != nil {
		if errors.Is(err, bolt.ErrBucketNotFound) {
			return errors.Wrapf(ctx, libkv.BucketNotFoundError, "delete bucket failed: %v", err)
		}
		return errors.Wrapf(ctx, err, "delete bucket failed")
	}
	return nil
}

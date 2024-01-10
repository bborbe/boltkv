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

func NewTx(boltTx *bolt.Tx) libkv.Tx {
	return &tx{
		boltTx: boltTx,
	}
}

type tx struct {
	boltTx *bolt.Tx
}

func (t *tx) Bucket(ctx context.Context, name libkv.BucketName) (libkv.Bucket, error) {
	bucket := t.boltTx.Bucket(name)
	if bucket == nil {
		return nil, errors.Wrapf(ctx, libkv.BucketNotFound, "bucket %s not found", name)
	}
	return NewBucket(bucket), nil
}

func (t *tx) CreateBucket(ctx context.Context, name libkv.BucketName) (libkv.Bucket, error) {
	bucket, err := t.boltTx.CreateBucket(name)
	if err != nil {
		return nil, err
	}
	return NewBucket(bucket), nil
}

func (t *tx) CreateBucketIfNotExists(ctx context.Context, name libkv.BucketName) (libkv.Bucket, error) {
	bucket, err := t.boltTx.CreateBucketIfNotExists(name)
	if err != nil {
		return nil, err
	}
	return NewBucket(bucket), nil
}

func (t *tx) DeleteBucket(ctx context.Context, name libkv.BucketName) error {
	return t.boltTx.DeleteBucket(name)
}

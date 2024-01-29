// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltkv

import (
	"context"

	libkv "github.com/bborbe/kv"
	bolt "go.etcd.io/bbolt"
)

type Bucket interface {
	libkv.Bucket
	Bucket() *bolt.Bucket
}

func NewBucket(boltBucket *bolt.Bucket) Bucket {
	return &bucket{
		boltBucket: boltBucket,
	}
}

type bucket struct {
	boltBucket *bolt.Bucket
}

func (b *bucket) Bucket() *bolt.Bucket {
	return b.boltBucket
}

func (b *bucket) IteratorReverse() libkv.Iterator {
	return NewIteratorReverse(b.boltBucket.Cursor())
}

func (b *bucket) Iterator() libkv.Iterator {
	return NewIterator(b.boltBucket.Cursor())
}

func (b *bucket) Get(ctx context.Context, key []byte) (libkv.Item, error) {
	return libkv.NewByteItem(key, b.boltBucket.Get(key)), nil
}

func (b *bucket) Put(ctx context.Context, key []byte, value []byte) error {
	return b.boltBucket.Put(key, value)
}

func (b *bucket) Delete(ctx context.Context, key []byte) error {
	return b.boltBucket.Delete(key)
}

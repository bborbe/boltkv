// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltkv

import (
	"context"
	"os"

	"github.com/bborbe/errors"
	libkv "github.com/bborbe/kv"
	bolt "go.etcd.io/bbolt"
)

// Stats returns a fast overview: file size and bucket inventory (names only).
// Per-bucket KeyCount and SizeB are left at zero — call StatsDetailed for those,
// noting that bolt's per-bucket walk is O(pages) per bucket and can be slow on
// large databases.
func (b *boltdb) Stats(ctx context.Context) (*libkv.Stats, error) {
	s := &libkv.Stats{Backend: "bolt"}
	if fi, err := os.Stat(b.path); err == nil {
		s.SizeB = fi.Size()
	}
	err := b.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, _ *bolt.Bucket) error {
			s.Buckets = append(s.Buckets, libkv.BucketStats{
				Name: libkv.BucketName(name),
			})
			return nil
		})
	})
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "stats failed")
	}
	return s, nil
}

// StatsDetailed returns Stats plus per-bucket KeyCount and SizeB.
// Walks the b-tree pages of every top-level bucket — O(total pages).
// Do not poll hot on large databases.
func (b *boltdb) StatsDetailed(ctx context.Context) (*libkv.Stats, error) {
	s := &libkv.Stats{Backend: "bolt", Detailed: true}
	if fi, err := os.Stat(b.path); err == nil {
		s.SizeB = fi.Size()
	}
	err := b.db.View(func(tx *bolt.Tx) error {
		return tx.ForEach(func(name []byte, bucket *bolt.Bucket) error {
			bs := bucket.Stats()
			s.Buckets = append(s.Buckets, libkv.BucketStats{
				Name:     libkv.BucketName(name),
				KeyCount: int64(bs.KeyN),
				SizeB:    int64(bs.LeafInuse + bs.BranchInuse + bs.InlineBucketInuse),
			})
			return nil
		})
	})
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "stats detailed failed")
	}
	return s, nil
}

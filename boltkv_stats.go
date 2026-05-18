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

// Stats returns per-bucket key counts and size estimates plus the total
// on-disk file size. Bolt exposes these via the b-tree metadata, so the
// call is O(number of top-level buckets) regardless of total key count.
func (b *boltdb) Stats(ctx context.Context) (libkv.Stats, error) {
	s := libkv.Stats{Backend: "bolt"}
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
		return libkv.Stats{}, errors.Wrapf(ctx, err, "stats failed")
	}
	return s, nil
}

// Copyright (c) 2023 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kv

import "context"

type Tx interface {
	Bucket(ctx context.Context, name BucketName) (Bucket, error)
	CreateBucket(ctx context.Context, name BucketName) (Bucket, error)
	CreateBucketIfNotExists(ctx context.Context, name BucketName) (Bucket, error)
	DeleteBucket(ctx context.Context, name BucketName) error
}

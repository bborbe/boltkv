// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kv

import (
	"context"
)

//counterfeiter:generate -o mocks/bucket.go --fake-name Bucket . Bucket
type Bucket interface {
	Put(ctx context.Context, key []byte, value []byte) error
	Get(ctx context.Context, bytes []byte) (Item, error)
	Delete(ctx context.Context, bytes []byte) error
	Iterator() Iterator
	IteratorReverse() Iterator
}

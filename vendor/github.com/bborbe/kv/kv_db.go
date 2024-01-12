// Copyright (c) 2023 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kv

import (
	"context"
	"io"
)

type DB interface {
	Update(ctx context.Context, fn func(tx Tx) error) error
	View(ctx context.Context, fn func(tx Tx) error) error
	io.Closer
	Sync() error
}

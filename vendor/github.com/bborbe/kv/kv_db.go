// Copyright (c) 2023 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kv

import (
	"context"
	"errors"
	"io"
)

var TransactionAlreadyOpenError = errors.New("transaction already open")

//counterfeiter:generate -o mocks/db.go --fake-name DB . DB
type DB interface {
	Update(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error
	View(ctx context.Context, fn func(ctx context.Context, tx Tx) error) error
	io.Closer
	Sync() error
}

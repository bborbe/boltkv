// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kv

import "context"

type Provider interface {
	Get(ctx context.Context) (DB, error)
}

type ProviderFunc func(ctx context.Context) (DB, error)

func (p ProviderFunc) Get(ctx context.Context) (DB, error) {
	return p(ctx)
}

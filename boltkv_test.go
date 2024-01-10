// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltkv_test

import (
	"context"

	libkv "github.com/bborbe/kv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/boltkv"
)

var _ = Describe("BoltKV", func() {
	var ctx context.Context
	var db libkv.DB
	var err error
	BeforeEach(func() {
		ctx = context.Background()
		db, err = boltkv.OpenTemp(ctx)
		Expect(err).To(BeNil())
	})
	It("basic", func() {
		libkv.BasicTestSuite(ctx, db)
	})
	It("iterator", func() {
		libkv.IteratorTestSuite(ctx, db)
	})
})

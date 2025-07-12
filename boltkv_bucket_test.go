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

var _ = Describe("Bucket", func() {
	var ctx context.Context
	var db boltkv.DB
	var err error

	BeforeEach(func() {
		ctx = context.Background()
		db, err = boltkv.OpenTemp(ctx)
		Expect(err).To(BeNil())
	})

	AfterEach(func() {
		_ = db.Close()
		_ = db.Remove()
	})

	Context("BoltDB-specific methods", func() {
		It("provides access to underlying BoltDB bucket", func() {
			err := db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {
				bucketName := libkv.BucketName("test")
				bucket, err := tx.CreateBucket(ctx, bucketName)
				Expect(err).To(BeNil())

				boltBucket := bucket.(boltkv.Bucket)
				Expect(boltBucket.Bucket()).ToNot(BeNil())

				return nil
			})
			Expect(err).To(BeNil())
		})
	})
})

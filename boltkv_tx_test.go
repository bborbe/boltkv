// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltkv_test

import (
	"context"

	"github.com/bborbe/errors"
	libkv "github.com/bborbe/kv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/boltkv"
)

var _ = Describe("Tx", func() {
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
		It("provides access to underlying BoltDB transaction", func() {
			err := db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {
				boltTx := tx.(boltkv.Tx)
				Expect(boltTx.Tx()).ToNot(BeNil())
				return nil
			})
			Expect(err).To(BeNil())
		})
	})

	Context("Bucket caching", func() {
		It("caches buckets within transaction", func() {
			err := db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {
				bucketName := libkv.BucketName("test")

				bucket1, err := tx.CreateBucket(ctx, bucketName)
				Expect(err).To(BeNil())

				bucket2, err := tx.Bucket(ctx, bucketName)
				Expect(err).To(BeNil())

				// Should be the same instance due to caching
				Expect(bucket1).To(Equal(bucket2))

				return nil
			})
			Expect(err).To(BeNil())
		})

		It("clears cache when bucket is deleted", func() {
			err := db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {
				bucketName := libkv.BucketName("test")

				_, err := tx.CreateBucket(ctx, bucketName)
				Expect(err).To(BeNil())

				err = tx.DeleteBucket(ctx, bucketName)
				Expect(err).To(BeNil())

				// Should not find bucket after deletion
				_, err = tx.Bucket(ctx, bucketName)
				Expect(err).ToNot(BeNil())
				Expect(errors.Is(err, libkv.BucketNotFoundError)).To(BeTrue())

				return nil
			})
			Expect(err).To(BeNil())
		})
	})
})

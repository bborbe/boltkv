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

var _ = Describe("Iterator", func() {
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
		It("provides access to underlying BoltDB cursor", func() {
			err := db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {
				bucketName := libkv.BucketName("test")
				bucket, err := tx.CreateBucket(ctx, bucketName)
				Expect(err).To(BeNil())

				iterator := bucket.Iterator()
				boltIterator := iterator.(boltkv.Iterator)
				Expect(boltIterator.Cursor()).ToNot(BeNil())

				return nil
			})
			Expect(err).To(BeNil())
		})
	})

	Context("Reverse iterator seek edge cases", func() {
		It("handles seek to key beyond last key", func() {
			err := db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {
				bucketName := libkv.BucketName("test")
				bucket, err := tx.CreateBucket(ctx, bucketName)
				Expect(err).To(BeNil())

				// Add some test data
				err = bucket.Put(ctx, []byte("key1"), []byte("value1"))
				Expect(err).To(BeNil())
				err = bucket.Put(ctx, []byte("key3"), []byte("value3"))
				Expect(err).To(BeNil())
				err = bucket.Put(ctx, []byte("key5"), []byte("value5"))
				Expect(err).To(BeNil())

				iterator := bucket.IteratorReverse()

				// Seek to key beyond last key should position at last key
				iterator.Seek([]byte("key9"))
				Expect(iterator.Valid()).To(BeTrue())
				Expect(iterator.Item().Key()).To(Equal([]byte("key5")))

				return nil
			})
			Expect(err).To(BeNil())
		})

		It("handles seek to exact match", func() {
			err := db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {
				bucketName := libkv.BucketName("test")
				bucket, err := tx.CreateBucket(ctx, bucketName)
				Expect(err).To(BeNil())

				// Add some test data
				err = bucket.Put(ctx, []byte("key1"), []byte("value1"))
				Expect(err).To(BeNil())
				err = bucket.Put(ctx, []byte("key3"), []byte("value3"))
				Expect(err).To(BeNil())
				err = bucket.Put(ctx, []byte("key5"), []byte("value5"))
				Expect(err).To(BeNil())

				iterator := bucket.IteratorReverse()

				// Seek to exact match should position at that key
				iterator.Seek([]byte("key3"))
				Expect(iterator.Valid()).To(BeTrue())
				Expect(iterator.Item().Key()).To(Equal([]byte("key3")))

				return nil
			})
			Expect(err).To(BeNil())
		})

		It("handles seek to key between existing keys", func() {
			err := db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {
				bucketName := libkv.BucketName("test")
				bucket, err := tx.CreateBucket(ctx, bucketName)
				Expect(err).To(BeNil())

				// Add some test data
				err = bucket.Put(ctx, []byte("key1"), []byte("value1"))
				Expect(err).To(BeNil())
				err = bucket.Put(ctx, []byte("key3"), []byte("value3"))
				Expect(err).To(BeNil())
				err = bucket.Put(ctx, []byte("key5"), []byte("value5"))
				Expect(err).To(BeNil())

				iterator := bucket.IteratorReverse()

				// Seek to key between existing keys should position at previous key
				iterator.Seek([]byte("key4"))
				Expect(iterator.Valid()).To(BeTrue())
				Expect(iterator.Item().Key()).To(Equal([]byte("key3")))

				return nil
			})
			Expect(err).To(BeNil())
		})

		It("handles seek to empty bucket", func() {
			err := db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {
				bucketName := libkv.BucketName("test")
				bucket, err := tx.CreateBucket(ctx, bucketName)
				Expect(err).To(BeNil())

				iterator := bucket.IteratorReverse()

				// Seek in empty bucket should result in invalid iterator
				iterator.Seek([]byte("key1"))
				Expect(iterator.Valid()).To(BeFalse())

				return nil
			})
			Expect(err).To(BeNil())
		})
	})
})

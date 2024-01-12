// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kv

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func BasicTestSuite(ctx context.Context, db DB) {
	GinkgoHelper()

	bucketName := NewBucketName("mybucket")

	err := db.Update(ctx, func(tx Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(ctx, bucketName)
		Expect(err).To(BeNil())

		{
			err = bucket.Put(ctx, []byte("key"), []byte("value"))
			Expect(err).To(BeNil())
		}

		{
			item, err := bucket.Get(ctx, []byte("key"))
			Expect(err).To(BeNil())
			Expect(item.Key()).To(Equal([]byte("key")))
			var result []byte
			err = item.Value(func(val []byte) error {
				result = val
				return nil
			})
			Expect(err).To(BeNil())
			Expect(result).To(Equal([]byte("value")))
		}

		{
			err = bucket.Delete(ctx, []byte("key"))
			Expect(err).To(BeNil())
		}

		{
			item, err := bucket.Get(ctx, []byte("key"))
			Expect(err).To(BeNil())
			var result []byte
			err = item.Value(func(val []byte) error {
				result = val
				return nil
			})
			Expect(err).To(BeNil())
			Expect(result).To(BeNil())
		}
		return nil
	})
	Expect(err).To(BeNil())
}

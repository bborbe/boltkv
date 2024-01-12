// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kv

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func IteratorTestSuite(ctx context.Context, db DB) {
	GinkgoHelper()

	bucketName := NewBucketName("mybucket")

	err := db.Update(ctx, func(tx Tx) error {
		bucket, err := tx.CreateBucketIfNotExists(ctx, bucketName)
		Expect(err).To(BeNil())

		{
			err = bucket.Put(ctx, []byte("bk"), []byte("bv"))
			Expect(err).To(BeNil())
		}
		{
			err = bucket.Put(ctx, []byte("ak"), []byte("av"))
			Expect(err).To(BeNil())
		}

		{
			it := bucket.Iterator()
			defer it.Close()
			var keys [][]byte
			var values [][]byte
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				keys = append(keys, item.Key())
				err = item.Value(func(val []byte) error {
					values = append(values, val)
					return nil
				})
				Expect(err).To(BeNil())
			}
			Expect(keys).To(HaveLen(2))
			Expect(keys[0]).To(Equal([]byte("ak")))
			Expect(keys[1]).To(Equal([]byte("bk")))
			Expect(values).To(HaveLen(2))
			Expect(values[0]).To(Equal([]byte("av")))
			Expect(values[1]).To(Equal([]byte("bv")))
		}

		{
			it := bucket.Iterator()
			defer it.Close()
			var keys [][]byte
			var values [][]byte
			for it.Seek([]byte("bk")); it.Valid(); it.Next() {
				item := it.Item()
				keys = append(keys, item.Key())
				err = item.Value(func(val []byte) error {
					values = append(values, val)
					return nil
				})
				Expect(err).To(BeNil())
			}
			Expect(keys).To(HaveLen(1))
			Expect(keys[0]).To(Equal([]byte("bk")))
			Expect(values).To(HaveLen(1))
			Expect(values[0]).To(Equal([]byte("bv")))
		}

		{
			it := bucket.IteratorReverse()
			defer it.Close()
			var keys [][]byte
			var values [][]byte
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				keys = append(keys, item.Key())
				err = item.Value(func(val []byte) error {
					values = append(values, val)
					return nil
				})
				Expect(err).To(BeNil())
			}
			Expect(keys).To(HaveLen(2))
			Expect(keys[0]).To(Equal([]byte("bk")))
			Expect(keys[1]).To(Equal([]byte("ak")))
			Expect(values).To(HaveLen(2))
			Expect(values[0]).To(Equal([]byte("bv")))
			Expect(values[1]).To(Equal([]byte("av")))
		}

		{
			it := bucket.IteratorReverse()
			defer it.Close()
			var keys [][]byte
			var values [][]byte
			for it.Seek([]byte("ak")); it.Valid(); it.Next() {
				item := it.Item()
				keys = append(keys, item.Key())
				err = item.Value(func(val []byte) error {
					values = append(values, val)
					return nil
				})
				Expect(err).To(BeNil())
			}
			Expect(keys).To(HaveLen(1))
			Expect(keys[0]).To(Equal([]byte("ak")))
			Expect(values).To(HaveLen(1))
			Expect(values[0]).To(Equal([]byte("av")))
		}
		return nil
	})
	Expect(err).To(BeNil())
}

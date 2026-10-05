// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltkv_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"

	libkv "github.com/bborbe/kv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/boltkv"
)

var _ = Describe("Compact", func() {
	var ctx context.Context

	BeforeEach(func() {
		ctx = context.Background()
	})

	It("shrinks the file, reports the reclaim and preserves surviving keys", func() {
		db, err := boltkv.OpenTemp(ctx)
		Expect(err).To(BeNil())
		path := db.DB().Path()
		DeferCleanup(func() {
			_ = db.Close()
			_ = os.Remove(path)
		})

		bucketName := libkv.NewBucketName("compact-bucket")
		value := bytes.Repeat([]byte("v"), 1024)
		const totalKeys = 10000
		const deletedKeys = 9000

		err = db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {
			bucket, err := tx.CreateBucketIfNotExists(ctx, bucketName)
			if err != nil {
				return err
			}
			for i := 0; i < totalKeys; i++ {
				key := []byte(fmt.Sprintf("key-%06d", i))
				if err := bucket.Put(ctx, key, value); err != nil {
					return err
				}
			}
			return nil
		})
		Expect(err).To(BeNil())

		err = db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {
			bucket, err := tx.Bucket(ctx, bucketName)
			if err != nil {
				return err
			}
			for i := 0; i < deletedKeys; i++ {
				key := []byte(fmt.Sprintf("key-%06d", i))
				if err := bucket.Delete(ctx, key); err != nil {
					return err
				}
			}
			return nil
		})
		Expect(err).To(BeNil())

		Expect(db.Close()).To(Succeed())

		beforeInfo, err := os.Stat(path)
		Expect(err).To(BeNil())
		sizeBefore := beforeInfo.Size()

		result, err := boltkv.Compact(ctx, path)
		Expect(err).To(BeNil())
		Expect(result).NotTo(BeNil())

		afterInfo, err := os.Stat(path)
		Expect(err).To(BeNil())

		Expect(result.SizeBefore).To(Equal(sizeBefore))
		Expect(result.SizeAfter).To(Equal(afterInfo.Size()))
		Expect(result.SizeAfter).To(BeNumerically("<", result.SizeBefore))
		Expect(result.BytesReclaimed).To(BeNumerically(">", int64(0)))
		Expect(result.BytesReclaimed).To(Equal(result.SizeBefore - result.SizeAfter))

		reopened, err := boltkv.OpenFile(ctx, path)
		Expect(err).To(BeNil())
		DeferCleanup(func() {
			_ = reopened.Close()
		})

		err = reopened.View(ctx, func(ctx context.Context, tx libkv.Tx) error {
			bucket, err := tx.Bucket(ctx, bucketName)
			if err != nil {
				return err
			}
			survivor, err := bucket.Get(ctx, []byte("key-009999"))
			if err != nil {
				return err
			}
			Expect(survivor.Exists()).To(BeTrue())
			return survivor.Value(func(val []byte) error {
				Expect(val).To(Equal(value))
				return nil
			})
		})
		Expect(err).To(BeNil())

		err = reopened.View(ctx, func(ctx context.Context, tx libkv.Tx) error {
			bucket, err := tx.Bucket(ctx, bucketName)
			if err != nil {
				return err
			}
			deleted, err := bucket.Get(ctx, []byte("key-000000"))
			if err != nil {
				return err
			}
			Expect(deleted.Exists()).To(BeFalse())
			return nil
		})
		Expect(err).To(BeNil())
	})

	It("returns a bounded error while the database is still held open", func() {
		db, err := boltkv.OpenTemp(ctx)
		Expect(err).To(BeNil())
		path := db.DB().Path()
		DeferCleanup(func() {
			_ = db.Close()
			_ = db.Remove()
		})

		bucketName := libkv.NewBucketName("held-bucket")
		err = db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {
			bucket, err := tx.CreateBucketIfNotExists(ctx, bucketName)
			if err != nil {
				return err
			}
			return bucket.Put(ctx, []byte("survivor"), []byte("value"))
		})
		Expect(err).To(BeNil())

		beforeInfo, err := os.Stat(path)
		Expect(err).To(BeNil())

		result, err := boltkv.Compact(ctx, path)
		Expect(err).NotTo(BeNil())
		Expect(result).To(BeNil())

		afterInfo, err := os.Stat(path)
		Expect(err).To(BeNil())
		Expect(afterInfo.Size()).To(Equal(beforeInfo.Size()))

		err = db.View(ctx, func(ctx context.Context, tx libkv.Tx) error {
			bucket, err := tx.Bucket(ctx, bucketName)
			if err != nil {
				return err
			}
			item, err := bucket.Get(ctx, []byte("survivor"))
			if err != nil {
				return err
			}
			Expect(item.Exists()).To(BeTrue())
			return item.Value(func(val []byte) error {
				Expect(val).To(Equal([]byte("value")))
				return nil
			})
		})
		Expect(err).To(BeNil())
	})

	It("returns an error and creates nothing when the path is missing", func() {
		dir, err := os.MkdirTemp("", "")
		Expect(err).To(BeNil())
		DeferCleanup(func() {
			_ = os.RemoveAll(dir)
		})
		missingPath := filepath.Join(dir, "missing.db")

		result, err := boltkv.Compact(ctx, missingPath)
		Expect(err).NotTo(BeNil())
		Expect(result).To(BeNil())

		_, err = os.Stat(missingPath)
		Expect(os.IsNotExist(err)).To(BeTrue())
	})

	It(
		"returns an error and leaves the database untouched when no temp file can be created",
		func() {
			dir, err := os.MkdirTemp("", "")
			Expect(err).To(BeNil())
			DeferCleanup(func() {
				//nolint:gosec // directory perms restored so cleanup can remove it
				_ = os.Chmod(dir, 0700)
				_ = os.RemoveAll(dir)
			})

			path := filepath.Join(dir, "bolt.db")
			db, err := boltkv.OpenFile(ctx, path)
			Expect(err).To(BeNil())
			Expect(db.Close()).To(Succeed())

			beforeInfo, err := os.Stat(path)
			Expect(err).To(BeNil())

			//nolint:gosec // directory made non-writable to exercise the temp-file failure path
			Expect(os.Chmod(dir, 0500)).To(Succeed())

			result, err := boltkv.Compact(ctx, path)
			Expect(err).NotTo(BeNil())
			Expect(result).To(BeNil())

			afterInfo, err := os.Stat(path)
			Expect(err).To(BeNil())
			Expect(afterInfo.Size()).To(Equal(beforeInfo.Size()))
		},
	)

	It("returns an error for a file that is not a bolt database", func() {
		file, err := os.CreateTemp("", "")
		Expect(err).To(BeNil())
		path := file.Name()
		Expect(file.Close()).To(Succeed())
		DeferCleanup(func() {
			_ = os.Remove(path)
		})
		Expect(os.WriteFile(path, []byte("this is not a bolt database"), 0600)).To(Succeed())

		result, err := boltkv.Compact(ctx, path)
		Expect(err).NotTo(BeNil())
		Expect(result).To(BeNil())
	})

	It("succeeds and reports a consistent reclaim when nothing was deleted", func() {
		db, err := boltkv.OpenTemp(ctx)
		Expect(err).To(BeNil())
		path := db.DB().Path()
		DeferCleanup(func() {
			_ = os.Remove(path)
		})

		bucketName := libkv.NewBucketName("no-freed-pages")
		err = db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {
			bucket, err := tx.CreateBucketIfNotExists(ctx, bucketName)
			if err != nil {
				return err
			}
			for i := 0; i < 10; i++ {
				key := []byte(fmt.Sprintf("key-%d", i))
				if err := bucket.Put(ctx, key, []byte("value")); err != nil {
					return err
				}
			}
			return nil
		})
		Expect(err).To(BeNil())
		Expect(db.Close()).To(Succeed())

		beforeInfo, err := os.Stat(path)
		Expect(err).To(BeNil())
		sizeBefore := beforeInfo.Size()

		result, err := boltkv.Compact(ctx, path)
		Expect(err).To(BeNil())
		Expect(result).NotTo(BeNil())
		Expect(result.SizeBefore).To(Equal(sizeBefore))
		Expect(result.BytesReclaimed).To(Equal(result.SizeBefore - result.SizeAfter))
	})

	It("preserves every key and value across multiple buckets", func() {
		db, err := boltkv.OpenTemp(ctx)
		Expect(err).To(BeNil())
		path := db.DB().Path()
		DeferCleanup(func() {
			_ = os.Remove(path)
		})

		expected := map[string]map[string][]byte{
			"alpha": {},
			"beta":  {},
			"gamma": {},
		}
		for i := 0; i < 500; i++ {
			expected["alpha"][fmt.Sprintf("key-%06d", i)] = bytes.Repeat([]byte("a"), 256)
		}
		for i := 0; i < 200; i++ {
			expected["beta"][fmt.Sprintf("key-%06d", i)] = bytes.Repeat([]byte("b"), 2048)
		}
		expected["gamma"]["key-000000"] = bytes.Repeat([]byte("g"), 4096)

		err = db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {
			for bucketName, entries := range expected {
				bucket, err := tx.CreateBucketIfNotExists(ctx, libkv.NewBucketName(bucketName))
				if err != nil {
					return err
				}
				for key, value := range entries {
					if err := bucket.Put(ctx, []byte(key), value); err != nil {
						return err
					}
				}
			}
			return nil
		})
		Expect(err).To(BeNil())
		Expect(db.Close()).To(Succeed())

		beforeInfo, err := os.Stat(path)
		Expect(err).To(BeNil())

		result, err := boltkv.Compact(ctx, path)
		Expect(err).To(BeNil())
		Expect(result).NotTo(BeNil())

		afterInfo, err := os.Stat(path)
		Expect(err).To(BeNil())

		Expect(result.SizeBefore).To(Equal(beforeInfo.Size()))
		Expect(result.SizeAfter).To(Equal(afterInfo.Size()))
		Expect(result.BytesReclaimed).To(Equal(result.SizeBefore - result.SizeAfter))

		reopened, err := boltkv.OpenFile(ctx, path)
		Expect(err).To(BeNil())
		DeferCleanup(func() {
			_ = reopened.Close()
		})

		err = reopened.View(ctx, func(ctx context.Context, tx libkv.Tx) error {
			names, err := tx.ListBucketNames(ctx)
			Expect(err).To(BeNil())
			Expect(names).To(HaveLen(len(expected)))

			for bucketName, entries := range expected {
				bucket, err := tx.Bucket(ctx, libkv.NewBucketName(bucketName))
				Expect(err).To(BeNil())
				Expect(countKeys(bucket)).To(Equal(len(entries)))

				for key, want := range entries {
					item, err := bucket.Get(ctx, []byte(key))
					Expect(err).To(BeNil())
					Expect(item.Exists()).To(BeTrue())
					Expect(item.Value(func(val []byte) error {
						Expect(val).To(Equal(want))
						return nil
					})).To(BeNil())
				}
			}
			return nil
		})
		Expect(err).To(BeNil())
	})
})

// countKeys returns the number of keys in bucket, used to prove that
// compaction neither drops nor invents keys.
func countKeys(bucket libkv.Bucket) int {
	it := bucket.Iterator()
	defer it.Close()
	count := 0
	for it.Rewind(); it.Valid(); it.Next() {
		count++
	}
	return count
}

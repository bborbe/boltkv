// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltkv

import (
	"bytes"
	"context"
	"fmt"
	"os"

	libkv "github.com/bborbe/kv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	bolt "go.etcd.io/bbolt"
)

var _ = Describe("Compact transaction bound", func() {
	It("commits the compacted copy in multiple transactions", func() {
		ctx := context.Background()
		db, err := OpenTemp(ctx)
		Expect(err).To(BeNil())
		path := db.DB().Path()
		DeferCleanup(func() {
			_ = os.Remove(path)
		})

		key := func(i int) []byte {
			return []byte(fmt.Sprintf("key-%010d", i))
		}
		value := bytes.Repeat([]byte("v"), 1024)
		const totalKeys = 30000

		err = db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {
			bucket, err := tx.CreateBucketIfNotExists(
				ctx,
				libkv.NewBucketName("bounded-commit-bucket"),
			)
			if err != nil {
				return err
			}
			for i := 0; i < totalKeys; i++ {
				if err := bucket.Put(ctx, key(i), value); err != nil {
					return err
				}
			}
			return nil
		})
		Expect(err).To(BeNil())
		Expect(db.Close()).To(Succeed())

		Expect(compactTxMaxSize).To(BeNumerically(">", int64(0)))
		liveBytes := int64(totalKeys) * int64(len(key(0))+len(value))
		Expect(liveBytes).To(BeNumerically(">", compactTxMaxSize))
		minimumCommits := liveBytes / compactTxMaxSize
		Expect(minimumCommits).To(BeNumerically(">", 1))

		result, err := Compact(ctx, path)
		Expect(err).To(BeNil())
		Expect(result).NotTo(BeNil())

		commits := committedTransactions(ctx, path)
		Expect(commits).To(BeNumerically(">=", minimumCommits))
		Expect(commits).To(BeNumerically(">", 1))
	})
})

// emptyDatabaseTxID returns the transaction id a freshly initialised BoltDB
// reports, i.e. the id of a database that has never committed a write
// transaction. bbolt initialises its two meta pages with txids 0 and 1, so
// this is 1 on bbolt v1.5.0. Read it rather than hardcode it so the test
// keeps working if bbolt ever changes its initialisation.
func emptyDatabaseTxID() int {
	file, err := os.CreateTemp("", "empty-*.db")
	Expect(err).To(BeNil())
	path := file.Name()
	Expect(file.Close()).To(Succeed())
	DeferCleanup(func() {
		_ = os.Remove(path)
	})

	db, err := bolt.Open(path, 0600, nil)
	Expect(err).To(BeNil())
	DeferCleanup(func() {
		_ = db.Close()
	})

	var id int
	Expect(db.View(func(tx *bolt.Tx) error {
		id = tx.ID()
		return nil
	})).To(Succeed())
	return id
}

// committedTransactions returns how many write transactions were committed
// while the database at path was built. bbolt advances the meta transaction
// id by exactly one on every committed write transaction and initialises a
// fresh database at id 1, so the id read from a freshly opened database
// minus the id of an empty database is exactly the number of commits.
func committedTransactions(ctx context.Context, path string) int64 {
	db, err := OpenFile(ctx, path)
	Expect(err).To(BeNil())
	DeferCleanup(func() {
		_ = db.Close()
	})

	var id int
	Expect(db.DB().View(func(tx *bolt.Tx) error {
		id = tx.ID()
		return nil
	})).To(Succeed())
	return int64(id - emptyDatabaseTxID())
}

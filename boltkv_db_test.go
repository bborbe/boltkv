// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltkv_test

import (
	"context"
	"os"
	"path/filepath"

	"github.com/bborbe/errors"
	libkv "github.com/bborbe/kv"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/bborbe/boltkv"
)

var _ = Describe("DB", func() {
	var ctx context.Context
	var err error
	var db boltkv.DB
	BeforeEach(func() {
		ctx = context.Background()
	})
	Context("OpenFile", func() {
		var file *os.File
		BeforeEach(func() {
			file, err = os.CreateTemp("", "")
			Expect(err).To(BeNil())
		})
		JustBeforeEach(func() {
			db, err = boltkv.OpenFile(ctx, file.Name())
		})
		AfterEach(func() {
			_ = db.Close()
			_ = os.Remove(file.Name())
		})
		It("returns no error", func() {
			Expect(err).To(BeNil())
		})
	})
	Context("Remove", func() {
		var file *os.File
		BeforeEach(func() {
			file, err = os.CreateTemp("", "")
			Expect(err).To(BeNil())
			db, err = boltkv.OpenFile(ctx, file.Name())
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			_ = db.Close()
			_ = os.Remove(file.Name())
		})
		It("it removes file", func() {
			Expect(db.Close()).To(BeNil())
			Expect(fileExists(file.Name())).To(BeTrue())
			Expect(db.Remove()).To(BeNil())
			Expect(fileExists(file.Name())).To(BeFalse())
		})
	})
	Context("OpenDir", func() {
		var tempDir string
		BeforeEach(func() {
			tempDir, err = os.MkdirTemp("", "")
			Expect(err).To(BeNil())
		})
		JustBeforeEach(func() {
			db, err = boltkv.OpenDir(ctx, tempDir)
		})
		AfterEach(func() {
			_ = db.Close()
			_ = os.RemoveAll(tempDir)
		})
		It("returns no error", func() {
			Expect(err).To(BeNil())
		})
		It("creates db file in directory", func() {
			dbPath := filepath.Join(tempDir, "bolt.db")
			Expect(fileExists(dbPath)).To(BeTrue())
		})
	})
	Context("OpenDir with non-existent directory", func() {
		var tempDir string
		BeforeEach(func() {
			tempDir = filepath.Join(os.TempDir(), "nonexistent", "nested")
		})
		JustBeforeEach(func() {
			db, err = boltkv.OpenDir(ctx, tempDir)
		})
		AfterEach(func() {
			if db != nil {
				_ = db.Close()
			}
			_ = os.RemoveAll(filepath.Dir(tempDir))
		})
		It("creates directory and returns no error", func() {
			Expect(err).To(BeNil())
			Expect(fileExists(tempDir)).To(BeTrue())
		})
	})
	Context("OpenTemp", func() {
		JustBeforeEach(func() {
			db, err = boltkv.OpenTemp(ctx)
		})
		AfterEach(func() {
			_ = db.Close()
			_ = db.Remove()
		})
		It("returns no error", func() {
			Expect(err).To(BeNil())
		})
		It("creates temporary file", func() {
			Expect(db.DB().Path()).ToNot(BeEmpty())
			Expect(fileExists(db.DB().Path())).To(BeTrue())
		})
	})
	Context("BoltDB-specific methods", func() {
		BeforeEach(func() {
			db, err = boltkv.OpenTemp(ctx)
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			_ = db.Close()
			_ = db.Remove()
		})
		It("provides access to underlying BoltDB", func() {
			boltDB := db.DB()
			Expect(boltDB).ToNot(BeNil())
			Expect(boltDB.Path()).ToNot(BeEmpty())
		})
		It("syncs database", func() {
			err := db.Sync()
			Expect(err).To(BeNil())
		})
	})
	Context("Transaction state management", func() {
		BeforeEach(func() {
			db, err = boltkv.OpenTemp(ctx)
			Expect(err).To(BeNil())
		})
		AfterEach(func() {
			_ = db.Close()
			_ = db.Remove()
		})
		It("detects when no transaction is open", func() {
			Expect(boltkv.IsTransactionOpen(ctx)).To(BeFalse())
		})
		It("detects open transaction during View", func() {
			err := db.View(ctx, func(ctx context.Context, tx libkv.Tx) error {
				Expect(boltkv.IsTransactionOpen(ctx)).To(BeTrue())
				return nil
			})
			Expect(err).To(BeNil())
		})
		It("detects open transaction during Update", func() {
			err := db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {
				Expect(boltkv.IsTransactionOpen(ctx)).To(BeTrue())
				return nil
			})
			Expect(err).To(BeNil())
		})
		It("prevents nested transactions in View", func() {
			err := db.View(ctx, func(ctx context.Context, tx libkv.Tx) error {
				return db.View(ctx, func(ctx context.Context, tx libkv.Tx) error {
					return nil
				})
			})
			Expect(err).ToNot(BeNil())
			Expect(errors.Is(err, libkv.TransactionAlreadyOpenError)).To(BeTrue())
		})
		It("prevents nested transactions in Update", func() {
			err := db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {
				return db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {
					return nil
				})
			})
			Expect(err).ToNot(BeNil())
			Expect(errors.Is(err, libkv.TransactionAlreadyOpenError)).To(BeTrue())
		})
	})
})

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return errors.Is(err, os.ErrNotExist) == false
}

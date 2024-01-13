// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltkv_test

import (
	"context"
	"os"

	"github.com/bborbe/errors"
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
})

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return errors.Is(err, os.ErrNotExist) == false
}

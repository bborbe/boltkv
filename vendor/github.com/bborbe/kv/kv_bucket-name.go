// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package kv

import "strings"

func BucketFromStrings(values ...string) BucketName {

	return NewBucketName(strings.Join(values, "_"))
}

func NewBucketName(name string) BucketName {
	return BucketName(name)
}

type BucketName []byte

func (f BucketName) String() string {
	return string(f)
}

func (f BucketName) Bytes() []byte {
	return f
}

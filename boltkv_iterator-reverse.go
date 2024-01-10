// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltkv

import (
	bolt "go.etcd.io/bbolt"

	libkv "github.com/bborbe/kv"
)

func NewIteratorReverse(boltCursor *bolt.Cursor) libkv.Iterator {
	return &iteratorReverse{
		boltCursor: boltCursor,
	}
}

type iteratorReverse struct {
	boltCursor *bolt.Cursor
	key        []byte
	value      []byte
}

func (i *iteratorReverse) Close() {
}

func (i *iteratorReverse) Item() libkv.Item {
	return libkv.NewByteItem(i.key, i.value)
}

func (i *iteratorReverse) Next() {
	i.key, i.value = i.boltCursor.Prev()
}

func (i *iteratorReverse) Valid() bool {
	return i.key != nil
}

func (i *iteratorReverse) Rewind() {
	i.key, i.value = i.boltCursor.Last()
}

func (i *iteratorReverse) Seek(key []byte) {
	i.key, i.value = i.boltCursor.Seek(key)
}

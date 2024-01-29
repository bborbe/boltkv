// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltkv

import (
	"bytes"

	libkv "github.com/bborbe/kv"
	bolt "go.etcd.io/bbolt"
)

func NewIteratorReverse(boltCursor *bolt.Cursor) Iterator {
	return &iteratorReverse{
		boltCursor: boltCursor,
	}
}

type iteratorReverse struct {
	boltCursor *bolt.Cursor
	key        []byte
	value      []byte
}

func (i *iteratorReverse) Cursor() *bolt.Cursor {
	return i.boltCursor
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
	// key is null if seek is last key
	if len(i.key) == 0 {
		i.key, i.value = i.boltCursor.Last()
		return
	}
	// if seek is not a exact match it key after
	if bytes.Compare(i.key, key) > 0 {
		i.key, i.value = i.boltCursor.Prev()
	}
}

// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltkv

import (
	bolt "go.etcd.io/bbolt"

	libkv "github.com/bborbe/kv"
)

func NewIterator(boltCursor *bolt.Cursor) libkv.Iterator {
	return &iterator{
		boltCursor: boltCursor,
	}
}

type iterator struct {
	boltCursor *bolt.Cursor
	key        []byte
	value      []byte
}

func (i *iterator) Close() {
}

func (i *iterator) Item() libkv.Item {
	return libkv.NewByteItem(i.key, i.value)
}

func (i *iterator) Next() {
	i.key, i.value = i.boltCursor.Next()
}

func (i *iterator) Valid() bool {
	return i.key != nil
}

func (i *iterator) Rewind() {
	i.key, i.value = i.boltCursor.First()
}

func (i *iterator) Seek(key []byte) {
	i.key, i.value = i.boltCursor.Seek(key)
}

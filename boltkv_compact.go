// Copyright (c) 2026 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package boltkv

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/bborbe/errors"
	bolt "go.etcd.io/bbolt"
)

// compactLockTimeout bounds how long Compact waits for the exclusive file lock.
const compactLockTimeout = time.Second

// compactTxMaxSize bounds the size, in bytes of key and value data, that
// Compact accumulates into a single destination transaction before bbolt
// commits and starts a new one.
//
// The trade-off runs in both directions. A larger bound means fewer commits
// and therefore fewer fsyncs (faster), but more dirty pages held in memory at
// once. A smaller bound means more commits (each with its own fsync) but a
// smaller, predictable memory ceiling.
//
// The value is chosen against a container memory limit far below the size of
// the database being compacted: bbolt keeps every dirty page of the current
// destination transaction in memory, so a transaction's heap footprint is
// roughly its key-and-value bytes times a small constant (page rounding at
// FillPercent 1.0 plus the dirty-page map). At 8 MiB the peak heap held by
// compaction is on the order of 10 MiB, which leaves more than four times the
// headroom under a 50 Mi container limit for the Go runtime, the retention
// pass that runs before compaction, and the rest of the process. The source
// database is memory-mapped and so does not contribute to the heap. A 29 GiB
// database therefore needs on the order of 3,700 commits, each an fsync —
// acceptable for a maintenance operation, and vastly better than being
// OOM-killed.
const compactTxMaxSize int64 = 8 * 1024 * 1024

// CompactResult reports the database file size before and after compaction.
type CompactResult struct {
	SizeBefore     int64
	SizeAfter      int64
	BytesReclaimed int64
}

// Compact rewrites the BoltDB file at path into a fresh file, returning freed
// pages to the filesystem, and reports the file size before and after.
//
// The caller must ensure no other handle or process holds the database open
// while Compact runs: the source is opened with an exclusive lock, and the
// call fails after a bounded one-second wait if that lock is held.
func Compact(ctx context.Context, path string) (*CompactResult, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "stat %s failed", path)
	}
	sizeBefore := info.Size()

	src, err := bolt.Open(path, 0600, &bolt.Options{Timeout: compactLockTimeout})
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "open %s failed", path)
	}

	tempFile, err := os.CreateTemp(filepath.Dir(path), ".compact-*")
	if err != nil {
		_ = src.Close()
		return nil, errors.Wrapf(ctx, err, "create temp file for %s failed", path)
	}
	tempPath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		_ = src.Close()
		_ = os.Remove(tempPath)
		return nil, errors.Wrapf(ctx, err, "close temp file %s failed", tempPath)
	}

	replaced := false
	defer func() {
		if !replaced {
			_ = os.Remove(tempPath)
		}
	}()

	dst, err := bolt.Open(tempPath, 0600, nil)
	if err != nil {
		_ = src.Close()
		return nil, errors.Wrapf(ctx, err, "open temp db %s failed", tempPath)
	}
	if err := bolt.Compact(dst, src, compactTxMaxSize); err != nil {
		_ = dst.Close()
		_ = src.Close()
		return nil, errors.Wrapf(ctx, err, "compact %s failed", path)
	}
	if err := dst.Close(); err != nil {
		_ = src.Close()
		return nil, errors.Wrapf(ctx, err, "close temp db %s failed", tempPath)
	}
	if err := src.Close(); err != nil {
		return nil, errors.Wrapf(ctx, err, "close %s failed", path)
	}
	if err := os.Rename(tempPath, path); err != nil {
		return nil, errors.Wrapf(ctx, err, "replace %s failed", path)
	}
	replaced = true

	after, err := os.Stat(path)
	if err != nil {
		return nil, errors.Wrapf(ctx, err, "stat %s failed", path)
	}
	return &CompactResult{
		SizeBefore:     sizeBefore,
		SizeAfter:      after.Size(),
		BytesReclaimed: sizeBefore - after.Size(),
	}, nil
}

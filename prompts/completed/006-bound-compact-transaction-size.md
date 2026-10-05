---
status: completed
summary: Bounded Compact's destination transaction size via an unexported 8 MiB constant, added a multi-transaction regression test and a cross-bucket losslessness test, and recorded the change under CHANGELOG Unreleased.
execution_id: boltkv-exec-006-bound-compact-transaction-size
dark-factory-version: v0.196.0
created: "2026-10-05T20:13:40Z"
queued: "2026-10-05T20:13:40Z"
started: "2026-10-05T20:14:46Z"
completed: "2026-10-05T20:16:49Z"
---

# Bound the transaction size used by Compact so compaction memory is bounded

<summary>
- Compacting a large database no longer needs memory proportional to the whole database.
- Compaction peak memory no longer grows with the database — it is bounded by a fixed, documented limit.
- A memory-limited process can now compact a database many times larger than its own limit.
- The public entry point keeps its exact signature and its exact result type — nothing that calls it has to change.
- A new regression test proves the copy is written in more than one transaction, so the old "everything in one transaction" behaviour cannot silently return.
- A new test proves compaction stays lossless: every key and value survives, across several buckets, and the reported reclaimed bytes match the real file-size difference.
- The exclusive-lock, temp-file and rename flow, the error style, and the reported sizes are all unchanged.
- No new dependency and no new configuration — callers need not change anything.
</summary>

<objective>
Stop `Compact` from copying an entire BoltDB file inside a single destination transaction. Pass a bounded transaction size to bbolt's `Compact` so bbolt commits intermittently and peak memory becomes O(bound) instead of O(database size). The public API (`Compact`, `CompactResult`) and the reported `SizeBefore` / `SizeAfter` / `BytesReclaimed` values must not change, and no new exported symbol is added.
</objective>

<context>
Read `CLAUDE.md` and `docs/dod.md` for project conventions and the Definition of Done.

Read these files before writing any code:

- `boltkv_compact.go` — the file you are changing. Note the existing `compactLockTimeout` constant and its doc-comment style; the new constant must match it.
- `boltkv_compact_test.go` — the existing external test file (`package boltkv_test`). It already calls Gomega `Expect` from inside `db.View(...)` closures and uses `DeferCleanup`. The losslessness test goes here; the six existing `It` blocks stay untouched.
- `boltkv_suite_test.go` — the suite bootstrap. It is `package boltkv_test` and defines `TestBoltkv` with `RunSpecs`. Every test file in this directory, including an internal `package boltkv` file, is compiled into the same test binary, so a new `Describe` in an internal test file is picked up by this one `RunSpecs` — do NOT add another bootstrap.
- `boltkv_db.go` — `DB` interface (`libkv.DB` plus `DB() *bolt.DB`), `OpenFile`, `OpenTemp`, and the `errors.Wrapf(ctx, err, "...")` style used throughout the package.
- `boltkv_iterator.go` — `Iterator()` / `Rewind()` / `Valid()` / `Next()` / `Close()` semantics, used by the key-count helper in the losslessness test.
- `Makefile` — real target names: `test`, `precommit` (= `ensure format generate test check addlicense`), `check` (= `lint vet vulncheck osv-scanner trivy`), `lint`, `vet`. Note `addlicense` runs in `precommit` with `-c "Benjamin Borbe" -y $(date +'%Y') -l bsd`, so the new test file must carry the standard 3-line BSD header or `addlicense` will rewrite it.
- `.golangci.yml` — the enabled linters the new code and tests must satisfy: `govet`, `errcheck`, `staticcheck`, `unused`, `revive`, `gosec`, `gocyclo`, `depguard`, `dupl`, `nestif`, `errname`, `unparam`, `bodyclose`, `forcetypeassert`, `asasalint`, `prealloc`. `nestif` is enabled with `min-complexity: 4`; `funlen` / `gocognit` / `maintidx` appear only under `settings` and are NOT in the `enable` list, so they are inert. `dupl`, `unparam` and the `revive` `dot-imports` rule are excluded for `_test.go` files.
- `CHANGELOG.md` — the entry style. The newest section is `## v1.16.0`; there is currently no `## Unreleased` section.

Coding guides (these paths exist inside the execution container):
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-library-guide.md`
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-testing-guide.md`
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-error-wrapping-guide.md`
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-linting-guide.md`
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-doc-best-practices.md`
- `/home/node/.claude/plugins/marketplaces/coding/docs/changelog-guide.md`
- `/home/node/.claude/plugins/marketplaces/coding/docs/git-workflow.md`

Frozen contract — reproduce exactly, do not rename fields or reorder them, do not change the signature:

```go
type CompactResult struct {
	SizeBefore     int64
	SizeAfter      int64
	BytesReclaimed int64
}

func Compact(ctx context.Context, path string) (*CompactResult, error)
```

Verified bbolt API (module `go.etcd.io/bbolt v1.5.0`, already a direct dependency in `go.mod` — do NOT bump it). Quoted verbatim from `compact.go` in the module cache:

```go
// Compact will create a copy of the source DB and in the destination DB. This may
// reclaim space that the source database no longer has use for. txMaxSize can be
// used to limit the transactions size of this process and may trigger intermittent
// commits. A value of zero will ignore transaction sizes.
func Compact(dst, src *DB, txMaxSize int64) error {
	// commit regularly, or we'll run out of memory for large datasets if using one transaction.
	var size int64
	tx, err := dst.Begin(true)
	...
	if err := walk(src, func(keys [][]byte, k, v []byte, seq uint64) error {
		// On each key/value, check if we have exceeded tx size.
		sz := int64(len(k) + len(v))
		if size+sz > txMaxSize && txMaxSize != 0 {
			// Commit previous transaction.
			if err := tx.Commit(); err != nil {
				return err
			}
			// Start new transaction.
			tx, err = dst.Begin(true)
			...
			size = 0
		}
		size += sz
		...
	})
	...
}
```

Consequences that drive the fix and the test:
- With `txMaxSize == 0` the guard `txMaxSize != 0` is always false, so no intermittent commit ever happens and the entire database is copied inside one destination transaction. The write transaction holds every dirty page in memory until it commits, so peak memory is O(live data).
- With a non-zero `txMaxSize`, bbolt commits whenever the accumulated `len(k)+len(v)` of the current transaction would exceed the bound. Each transaction therefore holds at most about `txMaxSize` bytes of key+value data, and the number of commits is approximately `totalLiveBytes / txMaxSize`.
- The bound counts only `len(k) + len(v)` — not page overhead, not bucket names beyond their own length. That makes the test's expectation computable from the data it writes.

Verified bbolt API used by the regression test to count commits (quoted verbatim from `tx.go` and `db.go` in the module cache):

```go
// tx.go
// ID returns the transaction id.
func (tx *Tx) ID() int {
	if tx == nil || tx.meta == nil {
		return -1
	}
	return int(tx.meta.Txid())
}
```

- `db.go` `init()` initialises the two meta pages with `m.SetTxid(common.Txid(i))` for `i` in `{0, 1}`, so the highest txid of a freshly created database is `1`.
- `db.go` calls `db.init()` only when `info.Size() == 0`, so opening an already-populated database never advances the txid.
- Every committed write transaction increments the meta txid by exactly one (`tx.meta.IncTxid()` in `beginRWTx`, then the meta is written on commit).

Therefore: `commits = tx.ID() - emptyDatabaseTxID()`, where `emptyDatabaseTxID()` is `1` on bbolt v1.5.0 — read it rather than hardcoding it.

`libkv` note: `libkv.Tx` has **no** `ID()` method (verified in `kv_tx.go` — it exposes only `Bucket`, `CreateBucket`, `CreateBucketIfNotExists`, `DeleteBucket`, `ListBucketNames`). The commit count is therefore read through the bbolt handle reachable via the `boltkv.DB` interface method `DB() *bolt.DB`.
</context>

<requirements>
1. In `boltkv_compact.go`, add an **unexported** package-level constant for the transaction-size bound, immediately after the existing `compactLockTimeout` constant and with a doc comment in the same style. Use this name and value:

   ```go
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
   ```

   Keep it unexported. No public function accepts a bound, so exporting it would widen the permanent public API for a test's convenience only, and the one known caller (`core/tick/candle-converter/pkg/retention-run.go` in `bborbe/trading`) neither reads nor tunes it.

2. In `Compact`, change only the third argument of the `bolt.Compact` call. The current call is `bolt.Compact(dst, src, 0)`. Replace it with:

   ```go
   if err := bolt.Compact(dst, src, compactTxMaxSize); err != nil {
   ```

   The surrounding error handling stays exactly as it is (`_ = dst.Close()`, `_ = src.Close()`, then `errors.Wrapf(ctx, err, "compact %s failed", path)`).

3. Do NOT change anything else in `boltkv_compact.go`. Specifically, keep unchanged:
   - the `Compact` signature and the `CompactResult` type (three `int64` fields in the existing order),
   - the `os.Stat(path)` guard at the top and its error message,
   - the exclusive-lock open (`bolt.Open(path, 0600, &bolt.Options{Timeout: compactLockTimeout})`),
   - the temp-file creation in the target's own directory (`os.CreateTemp(filepath.Dir(path), ".compact-*")`),
   - the `replaced := false` / deferred `os.Remove(tempPath)` cleanup,
   - the ordering `dst.Close()` then `src.Close()` then `os.Rename(tempPath, path)` then `replaced = true`,
   - the final `os.Stat(path)` and the `CompactResult` construction,
   - every `errors.Wrapf(ctx, err, "...")` message and the `github.com/bborbe/errors` import.

   The `Compact` doc comment says nothing about memory, so leave it as-is. No exported declaration of the package changes.

4. **Losslessness test — every key and value survives, across multiple buckets, with an exact reclaim.** Add one new `It` to the existing `Describe("Compact", ...)` in `boltkv_compact_test.go` (the external `package boltkv_test` file). Do NOT modify the six existing `It` blocks.

   - Open a temp database with `boltkv.OpenTemp(ctx)`; take `path := db.DB().Path()`; `DeferCleanup` removes the path.
   - Build a fixture of at least three buckets with different value sizes, e.g. `alpha` with 500 keys of 256 bytes, `beta` with 200 keys of 2048 bytes, `gamma` with 1 key of 4096 bytes. Use **non-empty** values only (never a zero-length value), because `libkv.NewByteItem.Exists()` is `len(value) > 0`.
   - Keep the expected data in a `map[string]map[string][]byte` keyed by bucket name, and write it in a single `db.Update` using `tx.CreateBucketIfNotExists` and `bucket.Put`. Then `Expect(db.Close()).To(Succeed())`.
   - `os.Stat(path)` → `beforeInfo`; call `boltkv.Compact(ctx, path)`; `os.Stat(path)` → `afterInfo`. Assert:
     `result.SizeBefore == beforeInfo.Size()`, `result.SizeAfter == afterInfo.Size()`, and `result.BytesReclaimed == result.SizeBefore - result.SizeAfter`.
   - Reopen with `boltkv.OpenFile(ctx, path)` and inside one `reopened.View` assert, for every expected bucket:
     - the bucket list from `tx.ListBucketNames(ctx)` has exactly `len(expected)` entries (no bucket lost, none invented),
     - `countKeys(bucket)` equals the number of keys written into that bucket (no key lost, none invented),
     - for every key: `item, err := bucket.Get(ctx, []byte(key))`, `Expect(err).To(BeNil())`, `Expect(item.Exists()).To(BeTrue())`, and `item.Value(func(val []byte) error { Expect(val).To(Equal(want)); return nil })` succeeds.
   - Use Gomega `Expect` directly inside the `View` closure (the existing tests in this file already do this) rather than `if err != nil { return err }` chains, so the closure stays flat and does not trip `nestif`.
   - Add this helper to the same external file:

     ```go
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
     ```

5. **Regression test — the destination is committed in more than one transaction.** Create a NEW internal test file `boltkv_compact_internal_test.go` in `package boltkv` (not `boltkv_test`) so it can read the unexported `compactTxMaxSize` constant. Give it the standard 3-line BSD header used by every file in the package:

   ```go
   // Copyright (c) 2026 Benjamin Borbe All rights reserved.
   // Use of this source code is governed by a BSD-style
   // license that can be found in the LICENSE file.
   ```

   Do NOT define a suite bootstrap (`RunSpecs` / `TestXxx`) in this file — the existing `TestBoltkv` in `boltkv_suite_test.go` runs every spec registered in the test binary, including specs from this internal file.

   Imports: `bytes`, `context`, `fmt`, `os`, `libkv "github.com/bborbe/kv"`, `. "github.com/onsi/ginkgo/v2"`, `. "github.com/onsi/gomega"`, `bolt "go.etcd.io/bbolt"`.

   Add a `var _ = Describe("Compact transaction bound", func() { ... })` containing one `It` with this shape:

   - `ctx := context.Background()`; open a temp database with `OpenTemp(ctx)` (unqualified — this file is in `package boltkv`); take `path := db.DB().Path()`; register `DeferCleanup` to remove the path.
   - Define a fixed-length key helper so the live byte count is exactly computable:

     ```go
     key := func(i int) []byte {
     	return []byte(fmt.Sprintf("key-%010d", i))
     }
     ```

     (`len(key(0))` is 14.)
   - Use `value := bytes.Repeat([]byte("v"), 1024)` and `const totalKeys = 30000`. Write all keys into one bucket (`libkv.NewBucketName("bounded-commit-bucket")`) in a single `db.Update`, then `Expect(db.Close()).To(Succeed())`.
   - Assert the premise explicitly, so a collect-everything implementation cannot pass by accident (the constant is unexported, so this file reads it directly):

     ```go
     Expect(compactTxMaxSize).To(BeNumerically(">", int64(0)))
     liveBytes := int64(totalKeys) * int64(len(key(0))+len(value))
     Expect(liveBytes).To(BeNumerically(">", compactTxMaxSize))
     minimumCommits := liveBytes / compactTxMaxSize
     Expect(minimumCommits).To(BeNumerically(">", 1))
     ```

     With `totalKeys = 30000`, `liveBytes` is 31,140,000 and `minimumCommits` is 3.
   - Call `Compact(ctx, path)` (unqualified); assert `err` is nil and the result is non-nil.
   - Measure the commits actually performed and assert the bound took effect:

     ```go
     commits := committedTransactions(ctx, path)
     Expect(commits).To(BeNumerically(">=", minimumCommits))
     Expect(commits).To(BeNumerically(">", 1))
     ```

   - Add these two package-level helpers to the same internal file, with the doc comments shown (they explain the mechanism so a future reader does not have to re-derive it):

     ```go
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
     ```

     `db.DB()` is the `boltkv.DB` interface method that returns the underlying `*bolt.DB`; that is how the test reaches the bbolt-level `Tx.ID()`.

6. Add a `## Unreleased` section to `CHANGELOG.md`, directly above the existing `## v1.16.0` heading, with one bullet in the project's style:

   ```
   ## Unreleased

   - fix: bound the transaction size used by Compact so compaction memory is proportional to the bound instead of the database size
   ```

   Do not touch any released section. Do not copy shell or verification commands into the changelog.

7. Do NOT commit. Do NOT run any dark-factory CLI command. Do NOT add a new dependency, a new environment variable, a new CLI flag, or a runtime option for the bound — it is a compile-time constant only.

8. Before finishing, re-run the `<verification>` commands, including the discriminating-test revert-and-restore in `<verification>`, and confirm every one passes and that the restore landed.
</requirements>

<constraints>
- **Frozen signature.** `func Compact(ctx context.Context, path string) (*CompactResult, error)` is unchanged. `CompactResult` keeps exactly the three `int64` fields `SizeBefore`, `SizeAfter`, `BytesReclaimed` in that order.
- **One argument changes.** The only functional change in `boltkv_compact.go` is the third argument of `bolt.Compact`: `0` becomes `compactTxMaxSize`. Everything else in that file stays byte-identical.
- **The bound constant is unexported.** It is `compactTxMaxSize`, not an exported symbol. No public function accepts a bound, so exporting it would widen the permanent public API for a test's convenience only; the one known caller neither reads nor tunes it. The internal test file (`package boltkv`) reads the unexported constant directly.
- **Why bound it internally rather than changing the signature.** The fleet has exactly one caller (`core/tick/candle-converter/pkg/retention-run.go` in `bborbe/trading`). A signature change is possible but explicitly not wanted; the bound is an internal implementation detail, not a caller decision.
- **Exclusive-lock / temp-file / rename flow is untouched.** The source is still opened exclusively with the bounded one-second lock timeout; the destination is still a temp file in the target's own directory; the replacement is still a single atomic rename; the temp file is still removed on every failure path. Only the transaction size changes.
- **Error style unchanged.** All errors go through `github.com/bborbe/errors` (`errors.Wrapf(ctx, err, ...)`) and name the path. Never use `fmt.Errorf`.
- **No new dependency.** `go.etcd.io/bbolt` is already a direct dependency; do not bump it and do not add anything else. The tests may import `go.etcd.io/bbolt` directly — it is already a direct dependency.
- **No new knob.** No configuration field, environment variable, CLI flag, functional option or exported setter for the bound. The bound is a single documented constant.
- **YAGNI.** Do not add metrics, logging, progress reporting, context-cancellation checks inside the compaction loop, or a retention/scheduling mechanism. This prompt is only about bounding the transaction size.
- **Tests** use Ginkgo v2 + Gomega. The losslessness test lives in the existing external `boltkv_compact_test.go` (`package boltkv_test`); the commit-count regression test lives in the new internal `boltkv_compact_internal_test.go` (`package boltkv`) and must not define its own suite bootstrap. Both must pass under `make precommit`. Use only runtime-generated paths.
- **New test file carries the license header.** `boltkv_compact_internal_test.go` starts with the standard 3-line BSD header, so `addlicense` in `make precommit` does not rewrite it.
- **No absolute paths** in code or tests.
- **Additive only for existing exported declarations.** No pre-existing exported declaration of `boltkv` changes its signature or semantics; no new exported declaration is added.
- Do NOT commit — dark-factory handles git.
- Existing tests must still pass unchanged.
</constraints>

<verification>
Run `make precommit` — must exit 0 (ensure + format + generate + test + check + addlicense).

Run `go test ./...` — the new tests and all pre-existing tests pass.

Run `go vet ./...` — exits 0.

Run `grep -n 'compactTxMaxSize' boltkv_compact.go` — prints the constant's doc comment, the `const compactTxMaxSize int64 = 8 * 1024 * 1024` declaration, and the `bolt.Compact(dst, src, compactTxMaxSize)` call site.

Run `grep -n 'fewer commits' boltkv_compact.go` and `grep -n 'more commits' boltkv_compact.go` — each prints a line, confirming the doc comment states the trade-off in both directions.

Run `grep -c 'bolt.Compact(dst, src, 0)' boltkv_compact.go` — prints `0` (no unbounded call remains).

Run `grep -n 'compactTxMaxSize' boltkv_compact_internal_test.go` — prints the premise assertions that read the unexported constant.

Run `head -3 boltkv_compact_internal_test.go` — prints the 3-line BSD license header.

Run `grep -n 'countKeys' boltkv_compact_test.go` — shows the helper defined and called from inside the `View` closure for every expected bucket, proving the losslessness test checks key counts and not just key presence.

Confirm the regression test discriminates (this is the point of the test — do it explicitly, and confirm the restore lands):
1. Temporarily change the third argument of `bolt.Compact` back to `0` in `boltkv_compact.go`.
2. Run `go test ./...` — the new "commits the compacted copy in multiple transactions" test MUST FAIL (the commit count drops to 1, below `minimumCommits`).
3. Restore `compactTxMaxSize` as the third argument.
4. Run `grep -c 'bolt.Compact(dst, src, 0)' boltkv_compact.go` — MUST print `0`, confirming the restore landed.
5. Run `make precommit` — MUST exit 0.
</verification>

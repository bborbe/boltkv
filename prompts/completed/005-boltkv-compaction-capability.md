---
status: completed
spec: [001-boltkv-compaction-capability]
summary: Added package-level Compact/CompactResult to boltkv, rewriting a BoltDB file into a fresh file to reclaim freed pages and reporting size before/after, with Ginkgo tests covering shrink/preservation, held-handle timeout, missing path, non-BoltDB file, no-freed-pages and temp-file failure paths.
execution_id: boltkv-compaction-exec-005-boltkv-compaction-capability
dark-factory-version: v0.196.0
created: "2026-10-05T11:12:37Z"
queued: "2026-10-05T11:41:52Z"
started: "2026-10-05T11:41:54Z"
completed: "2026-10-05T11:45:56Z"
branch: dark-factory/boltkv-compaction-capability
---

# Add BoltDB compaction capability to boltkv

<summary>
- BoltKV gains exactly one new capability: rewriting a database file into a fresh file so freed pages are returned to the filesystem.
- Callers learn how much disk space the rewrite reclaimed, in bytes, plus the file size before and after.
- The capability is a plain library function — no HTTP surface, no service wiring, no new dependency.
- The database must be closed by the caller first; if anything still holds it open, the call fails quickly with a clear error instead of hanging.
- On any failure the original database file is left untouched, so a failed call never loses data.
- A database with freed pages comes back strictly smaller, with every surviving key still readable and every deleted key still gone.
- No existing exported declaration of the package changes — the change is purely additive.
- The package still imports no HTTP code, staying usable by a CLI, a CronJob or a service alike.
</summary>

<objective>
Add a package-level `Compact` function and its `CompactResult` type to `github.com/bborbe/boltkv` so that any caller can reclaim disk space after deleting rows: `Compact` rewrites the BoltDB file at a path into a fresh file (returning freed pages to the filesystem) and reports the file size before and after. This is the primitive that makes retention and reset actually reduce disk usage — today deleting rows frees pages inside the file but the file never shrinks.
</objective>

<context>
Read `CLAUDE.md` (if present) and `docs/dod.md` for project conventions and the Definition of Done.

Read these files before writing any code — they establish the conventions the new code and tests must follow:

- `boltkv_db.go` — `OpenFile`, the `ChangeOptions` functional-option pattern, and the `errors.Wrapf(ctx, err, "...")` error-wrapping style used everywhere in the package.
- `boltkv_stats.go` — the most recent exported surface (`Stats` / `StatsDetailed`); mirror its doc-comment style and its `os.Stat` + `errors.Wrapf` usage.
- `boltkv_stats_test.go` — the test conventions to mirror: external test package `boltkv_test`, `boltkv.OpenTemp(ctx)`, `libkv.NewBucketName(...)`, `db.Update(ctx, func(ctx context.Context, tx libkv.Tx) error {...})`, and the `AfterEach` close/remove cleanup.
- `boltkv_suite_test.go` — the Ginkgo v2 suite bootstrap (`TestBoltkv`); a new test file only adds `var _ = Describe(...)`.
- `boltkv_db_test.go` — shows `fileExists`, `os.CreateTemp`, `filepath.Join`, and `os.IsNotExist`-style assertions already in use.
- `Makefile` — the real target names: `test` (`go test -mod=mod ... -cover ...`), `precommit` (= `ensure format generate test check addlicense`), `check` (= `lint vet vulncheck osv-scanner trivy`), `lint`, `vet`.
- `.golangci.yml` — linter limits the new code must satisfy (`funlen` 80 lines / 50 statements, `nestif` min-complexity 4, `errcheck`, `gosec`, `dupl`, `prealloc`).
- `CHANGELOG.md` — read the existing `## Unreleased` section and the entry style.

Coding guides (read the ones relevant to this change; these paths exist inside the execution container):
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-patterns.md`
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-error-wrapping-guide.md`
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-testing-guide.md`
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-security-linting.md`
- `/home/node/.claude/plugins/marketplaces/coding/docs/go-doc-best-practices.md`
- `/home/node/.claude/plugins/marketplaces/coding/docs/changelog-guide.md`
- `/home/node/.claude/plugins/marketplaces/coding/docs/git-workflow.md`

Frozen contract (from the spec — reproduce it exactly, do not rename fields or reorder them):

```go
type CompactResult struct {
	SizeBefore     int64
	SizeAfter      int64
	BytesReclaimed int64
}

func Compact(ctx context.Context, path string) (*CompactResult, error)
```

Verified bbolt API (module `go.etcd.io/bbolt v1.5.0`, already a direct dependency in `go.mod` — do NOT bump it):

```go
// bbolt/compact.go — writes a copy of src into dst, reclaiming free pages.
func Compact(dst, src *DB, txMaxSize int64) error

// bbolt/db.go — options.Timeout bounds how long Open waits for the file lock.
func Open(path string, mode os.FileMode, options *Options) (db *DB, err error)

// bbolt/db.go — Options.Timeout is a time.Duration; zero waits forever.
// bbolt/errors.go — ErrTimeout is returned when the exclusive lock is not
// obtained within Options.Timeout.
```

Note the direction of the arguments: `bolt.Compact(dst, src, txMaxSize)` writes **into** `dst` **from** `src`. A `txMaxSize` of `0` means "ignore transaction sizes" (a single transaction).
</context>

<requirements>
1. Create `boltkv_compact.go` in `package boltkv` with the standard BSD license header used by the other files in the package (copy the header from `boltkv_stats.go`, using the current year 2026). Imports: `context`, `os`, `path/filepath`, `time`, `github.com/bborbe/errors`, and `bolt "go.etcd.io/bbolt"`. Do NOT import `net/http` or any new third-party package.

2. Declare the bounded lock timeout as a package-level constant, so a held database fails fast instead of hanging:

   ```go
   // compactLockTimeout bounds how long Compact waits for the exclusive file lock.
   const compactLockTimeout = time.Second
   ```

3. Declare the result type exactly as frozen in the spec — three `int64` fields, in this order and with these names:

   ```go
   // CompactResult reports the database file size before and after compaction.
   type CompactResult struct {
   	SizeBefore     int64
   	SizeAfter      int64
   	BytesReclaimed int64
   }
   ```

4. Declare `Compact` with the frozen signature. Its doc comment MUST state the exclusive-access requirement, because acceptance criterion 2 greps the `go doc` output for it. Use this wording (a doc comment directly above the function):

   ```go
   // Compact rewrites the BoltDB file at path into a fresh file, returning freed
   // pages to the filesystem, and reports the file size before and after.
   //
   // The caller must ensure no other handle or process holds the database open
   // while Compact runs: the source is opened with an exclusive lock, and the
   // call fails after a bounded one-second wait if that lock is held.
   ```

5. Implement `Compact` in this exact sequence. The cleanup ordering is load-bearing: the original file is replaced only after the compacted copy has been fully written and closed, and the temporary file is removed on every error path.

   1. `os.Stat(path)` — on error, return `errors.Wrapf(ctx, err, "stat %s failed", path)`. Capture `info.Size()` as `SizeBefore`. (This guard is required: without it, `bolt.Open` with `O_CREATE` would silently create a file at a non-existent path, violating the spec's "nothing is created at that path" failure mode.)
   2. Open the source read-write so bbolt takes an **exclusive** lock with the bounded timeout: `bolt.Open(path, 0600, &bolt.Options{Timeout: compactLockTimeout})`. On error, return `errors.Wrapf(ctx, err, "open %s failed", path)`. This is where a held database surfaces `bolt.ErrTimeout` within ~1 second.
   3. Create the destination as a temporary file **in the target's own directory** (same filesystem, so the final rename is atomic): `os.CreateTemp(filepath.Dir(path), ".compact-*")`. On error, close the source and return `errors.Wrapf(ctx, err, "create temp file for %s failed", path)`.
   4. Close the `*os.File` returned by `os.CreateTemp` (it is not needed — bbolt reopens by path). On error, close the source, remove the temp file, and return a wrapped error.
   5. Install cleanup so the temp file is removed on every failure path: a `replaced := false` flag plus `defer func() { if !replaced { _ = os.Remove(tempPath) } }()`.
   6. Open the destination: `bolt.Open(tempPath, 0600, nil)` (an empty existing file is initialised by bbolt as a fresh database). On error, close the source and return a wrapped error naming `tempPath`.
   7. `bolt.Compact(dst, src, 0)` — note `dst` first, `src` second. On error, close both databases and return `errors.Wrapf(ctx, err, "compact %s failed", path)`.
   8. Close `dst`, then close `src`, checking each error and returning a wrapped error (closing `dst` flushes the compacted file before the swap; closing `src` releases the lock).
   9. `os.Rename(tempPath, path)` — the atomic replacement. On error, return a wrapped error naming `path`.
   10. Set `replaced = true`.
   11. `os.Stat(path)` again; capture `SizeAfter`. On error, return a wrapped error.
   12. Return `&CompactResult{SizeBefore: sizeBefore, SizeAfter: after.Size(), BytesReclaimed: sizeBefore - after.Size()}`, nil.

   Reference implementation (verified against bbolt v1.5.0 — keep the cleanup ordering; adapt only cosmetic style):

   ```go
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
   	if err := bolt.Compact(dst, src, 0); err != nil {
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
   ```

   Failure-mode obligations this sequence satisfies: a held database returns an error within the bounded timeout with the file unchanged (steps 2 + 5); a temp file that cannot be written returns an error with the original untouched (steps 3-6 + the deferred remove); a process killed between write and rename leaves the original intact because the replacement is a single atomic rename (step 9); a database with no freed pages still succeeds and reports `BytesReclaimed == SizeBefore - SizeAfter` (a zero reclaim is a valid outcome, not a failure); a missing path or a non-BoltDB file returns an error naming the path and creates nothing (step 1 and the `bolt.Open` error in step 2).

6. Create `boltkv_compact_test.go` in `package boltkv_test` (the same external test package as `boltkv_stats_test.go`). Use Ginkgo v2 + Gomega (`. "github.com/onsi/ginkgo/v2"`, `. "github.com/onsi/gomega"`). Mirror the `BeforeEach`/`AfterEach` cleanup in `boltkv_stats_test.go`. Use only runtime-generated paths (`os.MkdirTemp`, `os.CreateTemp`, `db.DB().Path()`, `filepath.Join`) — no absolute path literals anywhere in the code or tests. Cover all of the following:

   a. **Shrinks the file, reports the reclaim, and preserves surviving keys** (acceptance criteria 3 and 4, one `It`):
      - Open a temp database with `boltkv.OpenTemp(ctx)`; take its path from `db.DB().Path()`.
      - In one `db.Update`, create a bucket (`libkv.NewBucketName(...)`) and put **at least 10,000 keys with ~1 KiB values** (e.g. `bytes.Repeat([]byte("v"), 1024)`, keys `fmt.Sprintf("key-%06d", i)`).
      - In a second `db.Update`, delete **at least 90%** of those keys (e.g. delete the first 9,000 of 10,000).
      - `db.Close()`, then `os.Stat(path)` → `sizeBefore`.
      - Call `boltkv.Compact(ctx, path)`; assert `err` is nil and the result is non-nil.
      - `os.Stat(path)` again → `afterInfo`. Assert `result.SizeBefore == sizeBefore`, `result.SizeAfter == afterInfo.Size()`, `result.SizeAfter < result.SizeBefore`, `result.BytesReclaimed > 0`, and `result.BytesReclaimed == result.SizeBefore - result.SizeAfter`.
      - Reopen the compacted file with `boltkv.OpenFile(ctx, path)` and, inside a `View`, assert every **surviving** key reads back with its original value (`item, err := bucket.Get(ctx, key)`; `item.Exists()` is true; `item.Value(func(v []byte) error { ... })` yields the original bytes), and that a **deleted** key returns an item whose `Exists()` is false (negative assertion).

   b. **Held handle produces a bounded error, file unchanged, handle still usable** (acceptance criterion 5, one `It`):
      - Open a temp database with `boltkv.OpenTemp(ctx)` and keep that handle open (do NOT close it). Write at least one key so a survivor exists.
      - `os.Stat(path)` → `sizeBefore`.
      - Call `boltkv.Compact(ctx, path)`; assert `err` is non-nil and the result is nil.
      - `os.Stat(path)` again; assert the size is unchanged.
      - Through the still-open handle, assert the surviving key still reads back.

   c. **Missing path returns an error and creates nothing** (spec failure mode 6, one `It`):
      - Create a temp directory; point at a file inside it that does not exist.
      - Call `boltkv.Compact(ctx, missingPath)`; assert `err` is non-nil and the result is nil; assert `os.Stat(missingPath)` reports `os.IsNotExist`.

   d. **A file that is not a BoltDB database returns an error** (spec failure mode 6, one `It`):
      - Write a small non-empty, non-BoltDB file (e.g. `os.WriteFile(path, []byte("this is not a bolt database"), 0600)`).
      - Call `boltkv.Compact(ctx, path)`; assert `err` is non-nil and the result is nil.

   e. **A database with no freed pages still succeeds** (spec failure mode 4, one `It`):
      - Open a temp database with `boltkv.OpenTemp(ctx)`, write a few keys, then `db.Close()` — delete nothing.
      - `os.Stat(path)` → `sizeBefore`; call `boltkv.Compact(ctx, path)`; assert `err` is nil and the result is non-nil.
      - Assert `result.SizeBefore == sizeBefore` and `result.BytesReclaimed == result.SizeBefore - result.SizeAfter`. Do NOT assert `BytesReclaimed == 0`: bbolt's `Compact` writes with `FillPercent = 1.0`, so it can pack tighter than the source and shrink a database that has no freed pages at all.

7. Add one `## Unreleased` changelog entry to `CHANGELOG.md` in the project's style (`- <prefix>: <what>`). This is a new backwards-compatible feature, so use the `feat:` prefix, e.g. `- feat: add Compact and CompactResult to reclaim disk space by rewriting a BoltDB file`. If `## Unreleased` already exists, append to it — do not replace it. Do not copy verification or shell comments into the changelog.

8. Do NOT change any existing exported declaration of `boltkv`, do NOT change the `github.com/bborbe/kv` interfaces, and do NOT add a new dependency.

9. Before finishing, re-run the `<verification>` commands and confirm they pass; walk each acceptance criterion (1-7) against the change.
</requirements>

<constraints>
- **Frozen signature.** `Compact` is a package-level function in package `boltkv`; `CompactResult` has exactly the three `int64` fields `SizeBefore`, `SizeAfter`, `BytesReclaimed`. Do not rename fields or change the signature.
- **Why a function rather than a method.** `boltkv`'s `DB` interface embeds `libkv.DB`; exposing compaction through that interface would require changing the `github.com/bborbe/kv` interfaces, which this work must not touch. A package-level function is the honest shape for a capability that needs exclusive access to a file the caller does not own.
- **HTTP-free.** Package `boltkv` gains no `net/http` import. No HTTP handler and no wiring into any service — a follow-up task owns the handler wrapper.
- **Exclusive access is the caller's obligation.** The source is opened so bbolt takes an exclusive lock, and the lock timeout is bounded to 1 second, so a held database fails fast with an error instead of hanging. bbolt's `Compact` walks the source under a read transaction and writes a separate destination, so the live file cannot be rewritten in place: the destination is a temporary file in the target's directory, and the replacement is a rename on the same filesystem.
- **Errors** are wrapped with `github.com/bborbe/errors` (`errors.Wrapf(ctx, err, ...)`) and name the path. Never use `fmt.Errorf`.
- **No new dependency.** `go.etcd.io/bbolt` is already a direct dependency; do not bump it and do not add anything else.
- **No retention policy, no scheduling, no automatic triggering.** This is a manual, caller-invoked capability.
- **Tests** use Ginkgo v2 + Gomega, follow the existing suite conventions in the package, and run under `make precommit`.
- **No absolute paths** in code or tests.
- **Additive only.** Every pre-existing exported declaration of `boltkv` keeps its signature.
- Do NOT commit — dark-factory handles git.
- Existing tests must still pass.
</constraints>

<verification>
Run `make precommit` — must exit 0 (format + lint + test + security).

Run `go test ./...` — the new suite and all pre-existing suites pass.

Run `go vet ./...` — exits 0.

Run `go doc . Compact` — prints `func Compact(ctx context.Context, path string) (*CompactResult, error)` and a sentence naming that no other handle or process may hold the database open while the call runs.

Run `go doc . CompactResult` — prints the three fields `SizeBefore`, `SizeAfter` and `BytesReclaimed`.

Run `go list -f '{{join .Imports "\n"}}' . | grep -c 'net/http'` — prints `0`. (Note: `grep -c` exits non-zero on a zero count, so read the printed `0`, not the exit status. The equivalent authoritative assertion is `! go list -f '{{join .Imports "\n"}}' . | grep -q 'net/http'`, which must succeed.)

Run `go doc -all .` — lists the new `Compact` and `CompactResult`; every pre-existing exported declaration is unchanged.
</verification>

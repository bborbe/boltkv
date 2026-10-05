---
status: draft
created: 2026-10-05
---

# Reusable BoltDB Compaction Capability

## Summary

- BoltKV can delete data but cannot give the space back: deleting keys frees pages *inside* the file, and the file itself never shrinks.
- This adds exactly one capability — rewrite a database file into a fresh file, return the freed pages to the filesystem, and report the file size before and after.
- It stays a pure storage library: no HTTP surface, no new dependency, no change to any existing exported API.
- Callers get before/after sizes, so a prune or a reset can be shown to have actually reduced disk usage instead of merely deleting rows.

## Problem

Every BoltDB-backed trading service can grow without bound and has no way to give the space back. Deleting rows — by resetting a bucket or pruning by age — frees pages within the file, but the file never shrinks, so a retention policy reduces row count without reducing disk usage. Measured on the dev cluster 2026-10-05: `core-tick-candle-converter-0/bolt.db` is 26.97 GiB, with sibling volumes at 11–12 GiB each. Nothing in the ecosystem can reclaim that space today: `github.com/bborbe/kv` ships the destructive reset handlers but has no compaction counterpart, and `boltkv` has no compaction surface at all. This primitive is what makes retention and reset actually reduce disk usage.

## Goal

`github.com/bborbe/boltkv` exposes one exported, HTTP-free capability that rewrites a BoltDB file into a fresh file — returning freed pages to the filesystem — and reports the file size before and after, so any caller (a CLI, a CronJob, a service) can reclaim space after a delete.

## Non-goals

- No HTTP handler, and no wiring into any service — a follow-up task owns the handler wrapper
- No change to the `github.com/bborbe/kv` interfaces
- No retention policy, and no decision about what age to prune at
- No scheduling or automatic triggering — this is a manual invocation
- No change to any existing exported declaration of `boltkv`

## Acceptance Criteria

- [ ] Package `boltkv` exports `Compact` with the frozen signature and `CompactResult` with its three `int64` fields — evidence: `go doc . Compact` prints `func Compact(ctx context.Context, path string) (*CompactResult, error)`, and `go doc . CompactResult` prints `SizeBefore`, `SizeAfter` and `BytesReclaimed`.
- [ ] The `Compact` doc comment states the exclusive-access requirement — evidence: `go doc . Compact` output contains a sentence naming that no other handle or process may hold the database open while the call runs.
- [ ] Compacting a database with freed pages shrinks the file and reports the reclaim — evidence: a Ginkgo test writes **at least 10,000 keys with ~1 KiB values** into a temporary database — enough that the surviving data occupies fewer pages than the original, so the shrink is real rather than page-granular noise — deletes **at least 90 %** of them, records the size via `os.Stat`, calls `Compact`, and asserts `SizeAfter < SizeBefore`, `BytesReclaimed > 0`, and `BytesReclaimed == SizeBefore - SizeAfter`.
- [ ] The compacted file holds exactly the surviving keys — evidence: the same test reopens the compacted file and asserts every surviving key reads back with its original value, and that a key deleted before the call returns `nil` (negative assertion).
- [ ] A database held open by another handle produces a bounded error, not a hang and not a corrupt file — evidence: a test opens the database through `boltkv.OpenFile` and keeps that handle, calls `Compact` on the same path, and asserts the call returns a non-nil error, that `os.Stat` reports the same size as before the call, and that the still-open handle reads its surviving keys.
- [ ] The package stays HTTP-free — evidence: `go list -f '{{join .Imports "\n"}}' . | grep -c 'net/http'` prints `0`.
- [ ] The change is additive — evidence: `go doc -all .` lists the new `Compact` and `CompactResult`, every pre-existing exported declaration is unchanged, and `make precommit` exits 0.

## Verification

### Container-executable (runs inside the YOLO container at prompt time)

- `make precommit` — format + lint + test + security, exits 0
- `go test ./...` — the new suite and the pre-existing suites pass
- `go doc . Compact` — prints the frozen signature and the exclusivity sentence
- `go list -f '{{join .Imports "\n"}}' . | grep -c 'net/http'` — prints `0`
- `go vet ./...` — exits 0

### Operator-executable (runs on the host after the PR merges)

- Copy a real `bolt.db` from a trading dev volume, free pages in the copy, call `Compact` from a small harness, and compare `ls -l` before and after — the file must be smaller and the surviving keys must still read back.

## Desired Behavior

1. `Compact` accepts a filesystem path and a context, and returns a result carrying the file size before the call, the file size after the call, and the difference between them.
2. When the database at that path has freed pages, the file on disk is strictly smaller after the call than before it.
3. Every key present before the call is present after it with its original value, and every key deleted before the call is absent after it.
4. The file left at the path is a valid BoltDB database: it opens with the standard bbolt opener and serves a read transaction.
5. When another handle or process holds the database open, the call fails within a bounded wait with an error naming the path, and the file at that path is unchanged in size.
6. A path that does not exist, or that is not a BoltDB database, produces an error naming the path, and no replacement file is left behind.

## Constraints

- **Frozen signature.** The capability is a package-level function in package `boltkv`:

  ```go
  type CompactResult struct {
  	SizeBefore     int64
  	SizeAfter      int64
  	BytesReclaimed int64
  }

  func Compact(ctx context.Context, path string) (*CompactResult, error)
  ```

- **Why a function rather than a method.** `boltkv`'s `DB` interface embeds `libkv.DB`, so exposing compaction through that interface would require changing the `github.com/bborbe/kv` interfaces, which this work must not touch. A package-level function is also the honest shape for a capability that needs exclusive access to a file the caller does not own.
- **HTTP-free.** Package `boltkv` gains no `net/http` import. The primitive is library-shaped and useful to a CLI, a CronJob and a service alike; the HTTP wrapper belongs beside the existing reset handlers, not here.
- **Exclusive access is the caller's obligation.** The source is opened so bbolt takes an exclusive lock, and the lock timeout is bounded to 1 second, so a held database fails fast with an error instead of hanging. bbolt's `Compact` walks the source under a read transaction and writes a separate destination, so the live file cannot be rewritten in place: the destination is a temporary file in the target's directory, and the replacement is a rename on the same filesystem.
- **Errors** are wrapped with `github.com/bborbe/errors` and name the path.
- **Tests** use Ginkgo v2 + Gomega, follow the existing suite conventions in the package, and run under `make precommit`.
- **No absolute paths** in code or tests.
- Every pre-existing exported declaration of `boltkv` keeps its signature.

## Failure Modes

| Trigger | Expected behavior | Recovery |
|---|---|---|
| Another process or handle holds the database open | `Compact` returns an error naming the path within the bounded lock timeout; the file is unchanged | Close the holder, then call `Compact` again |
| The temporary file cannot be written (no space, permissions) | `Compact` returns an error; the original file is untouched and no partial file replaces it | Free space or fix permissions, then call again |
| The process is killed after the temporary file is written and before the rename | The original database is intact; a stray temporary file is left beside it | Delete the stray temporary file; the database is unchanged |
| The database has no freed pages | `Compact` succeeds and reports `BytesReclaimed == 0`; the file is unchanged | None — a zero reclaim is a valid outcome, not a failure |
| The path does not exist, or is not a BoltDB file | `Compact` returns an error naming the path; nothing is created at that path | Pass a path to an existing BoltDB file |

## Security / Abuse

- The only input is a filesystem path supplied by the caller. The capability opens that path, creates its temporary file in the same directory under an unpredictable name, and never interprets file contents as anything but a database.
- No network access, no shell execution, no environment lookups.
- The destructive step — replacing the file at the path — runs only after the compacted copy has been fully written and closed. Every earlier failure leaves the original file in place.

## Suggested Decomposition

Prompts should be generated in this order — each row is a single prompt with a clear scope.

| # | Prompt focus | Covers DBs | Covers ACs | Depends on |
|---|---|---|---|---|
| 1 | `Compact` + `CompactResult` in package `boltkv`, with the Ginkgo suite covering the shrink, the read-back, the held-file error and the missing-path error | 1-6 | 1-7 | — |

Rationale: the change is one function, one result type and its tests — a single layer. Splitting it would separate the function from the only evidence that it works.

## Do-Nothing Option

The ecosystem keeps a delete path that cannot reduce disk usage. Retention and reset ship as levers that free pages nothing can reclaim, so the 26.97 GiB `core-tick-candle-converter-0/bolt.db` and its 11–12 GiB siblings keep only growing — and the disk alerts that retention was supposed to prevent keep firing.

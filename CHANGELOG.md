# Changelog

All notable changes to this project will be documented in this file.

Please choose versions by [Semantic Versioning](http://semver.org/).

* MAJOR version when you make incompatible API changes,
* MINOR version when you add functionality in a backwards-compatible manner, and
* PATCH version when you make backwards-compatible bug fixes.

## v1.14.3

- bump github.com/bborbe/errors, kv, sentry, service
- bump github.com/bborbe/argument, math, run, time (indirect)
- bump github.com/getsentry/sentry-go v0.47.0

## v1.14.2

- Bump go.etcd.io/bbolt from v1.4.3 to v1.5.0
- Bump github.com/bborbe/kv from v1.21.1 to v1.21.2
- Bump test dependencies (ginkgo v2.29.0 → v2.32.0, gomega v1.41.0 → v1.42.1)
- Bump indirect dependencies (x/net, x/sync, x/sys, x/text)

## v1.14.1

- bump go 1.26.3 → 1.26.4
- bump bborbe/service v1.9.10 → v1.10.1, bborbe/sentry v1.9.17 → v1.9.18
- bump prometheus/common, prometheus/procfs, golang.org/x/{net,sys,text,tools,mod}
- drop standalone errcheck + gosec targets; inline into golangci-lint config
- add .maintainer.yaml for autoRelease + autoApprove; flip .dark-factory.yaml autoRelease=false

## v1.14.0

- **BREAKING**: `Stats(ctx)` now returns `*libkv.Stats` instead of `libkv.Stats` (matches bborbe/kv v1.21.0 interface)
- Fast `Stats(ctx)` now lists bucket NAMES only (no `Bucket.Stats()` walk) — O(top-level buckets)
- Implement `StatsDetailed(ctx) (*libkv.Stats, error)` — full `Bucket.Stats()` walk per bucket; O(pages)
- Bump bborbe/kv v1.20.0 → v1.21.1

## v1.13.0

- implement `Stats(ctx) (Stats, error)` to satisfy bborbe/kv v1.20.0 `DB` interface; uses bbolt `Bucket.Stats()` for O(1) per-bucket key counts + size, and `os.Stat` for total file size
- bump bborbe/kv v1.19.7 → v1.20.0
- bump bborbe/time v1.25.10 → v1.27.0, bborbe/parse v1.10.11 → v1.10.12, bborbe/validation v1.4.12 → v1.4.13
- bump ginkgo v2.28.3 → v2.29.0, gomega v1.40.0 → v1.41.0

## v1.12.6

- bump go 1.26.2 → 1.26.3
- bump bborbe/errors v1.5.11 → v1.5.13
- bump bborbe/kv v1.19.6 → v1.19.7
- bump bborbe/sentry v1.9.16 → v1.9.17
- bump getsentry/sentry-go v0.46.1 → v0.46.2

## v1.12.5

- chore: Migrate to tools.env + Makefile @version pattern; remove tools.go and obsolete replace block. go.mod reduced from 454 to 48 lines.

## v1.12.4

- bump Go toolchain to 1.26.2
- extend vulncheck ignore list with GO-2026-4514, GO-2022-0470, GO-2026-4772, GO-2026-4771
- add OSV/Trivy ignore entries for bbolt and aws-sdk-go-v2 vulns
- improve vulncheck output on failure

## v1.12.3

- update bborbe/* dependencies (errors, kv, sentry, service)
- update indirect deps (containerd, docker/cli, go-git, moby/buildkit)
- add vulnerability suppressions for GO-2026-4923/CVE-2026-33817
- improve vulncheck Makefile target with JSON filtering

## v1.12.2

- Update go-git/go-git to v5.17.1 (fix security vulnerabilities)

## v1.12.1

- Update direct dependencies: errors, kv, sentry, service
- Update golangci-lint to v2.11.4 and osv-scanner to v2.3.5
- Add OSV scanner and Trivy ignore files for docker/docker indirect CVEs
- Update numerous indirect dependencies across the board
- Clean up go.mod: remove exclude blocks and unused indirect deps

## v1.12.0

- feat: enable golangci-lint in check target with updated linters and fix all violations

## v1.11.6

- standardize Makefile: multiline trivy format

## v1.11.5

- chore: repair corrupted module cache entries and re-extract missing packages to restore precommit health

## v1.11.4

- go mod update

## v1.11.3

- Update Go to 1.26.0

## v1.11.2

- Update Go from 1.25.5 to 1.25.7
- Update dependencies (ginkgo, gomega, osv-scanner, sentry-go, and others)
- Update CI workflow to use Go 1.25.7
- Add .update-logs/ and .mcp-* to .gitignore

## v1.11.1

- Update Go to 1.25.5
- Update golang.org/x/crypto to v0.47.0
- Update dependencies

## v1.11.0

- update go and deps

## v1.10.5

- add golangci-lint configuration
- add tools.go for Go tool dependency tracking
- update GitHub workflow with Go 1.25.2 and Trivy installation
- enhance Makefile with additional security checks (osv-scanner, gosec, trivy)
- apply code formatting with golines (100 char line length)
- go mod update

## v1.10.4

- improve README with usage example and installation instructions
- go mod update

## v1.10.3

- add github workflow
- go mod update

## v1.10.2

- add tests
- go mod update

## v1.10.1

- go mod update

## v1.10.0

- remove vendor 
- go mod update

## v1.9.6

- go mod update

## v1.9.5

- go mod update

## v1.9.4

- go mod update

## v1.9.3

- add cmd bolt-value-delete

## v1.9.2

- add cmd bolt-bucket-delete, bolt-bucket-list, bolt-value-get, bolt-value-list and bolt-value-set

## v1.9.1

- fix ListBucketNames

## v1.9.0

- implement ListBucketNames
- go mod update

## v1.8.0

- add bolt cmds

## v1.7.3

- go mod update

## v1.7.2

- go mod update

## v1.7.1

- go mod update

## v1.7.0

- cache buckets per tx
- go mod update

## v1.6.2

- go mod update

## v1.6.1

- fix tx

## v1.6.0

- remove path from NewDB

## v1.5.1

- add interface to access bolt db, tx, bucket if needed

## v1.5.0

- prevent transaction open second transaction

## v1.4.1

- go mod update

## v1.4.0

- fulfill bucket testsuite

## v1.3.0

- use new testsuite
- fix reverse seek

## v1.2.1

- fix reverse seek not found

## v1.2.0

- add db.Remove

## v1.1.1

- increase logging

## v1.1.0

- Add context to update and view

## v1.0.0

- Initial Version

---
status: completed
summary: Enabled golangci-lint in check target, updated .golangci.yml to standard kv reference config, and fixed all lint violations (dupl nolint on Update/View, forcetypeassert nolint in test files).
container: boltkv-002-enable-lint
dark-factory-version: v0.59.5-dirty
created: "2026-03-21T09:13:54Z"
queued: "2026-03-21T10:12:34Z"
started: "2026-03-21T10:18:22Z"
completed: "2026-03-21T10:23:44Z"
---

<summary>
- The `check` Makefile target now includes `lint` alongside other checks
- The TODO comment about enabling lint is removed from the Makefile
- The `.golangci.yml` config is updated to match the standard kv reference config
- All golangci-lint violations in existing Go source files are fixed
- `make precommit` passes cleanly
</summary>

<objective>
Enable golangci-lint in the boltkv `check` target so lint runs as part of `make precommit`. Update the `.golangci.yml` to the current standard config from kv, and fix any lint violations that surface.
</objective>

<context>
Read CLAUDE.md for project conventions.

Files to read before making changes:
- `Makefile` — current Makefile with TODO comment and lint excluded from check
- `.golangci.yml` — current lint config (outdated, missing linters)

Source files that may need lint fixes:
- `boltkv_db.go`, `boltkv_tx.go`, `boltkv_bucket.go`
- `boltkv_iterator.go`, `boltkv_iterator-reverse.go`
- `cmd/*/main.go`

Reference standard `.golangci.yml` config to match:
```yaml
version: "2"

run:
  timeout: 5m
  tests: true

linters:
  enable:
    - govet
    - errcheck
    - staticcheck
    - unused
    - revive
    - gosec
    - gocyclo
    - depguard
    - dupl
    - nestif
    - errname
    - unparam
    - bodyclose
    - forcetypeassert
    - asasalint
    - prealloc
  settings:
    depguard:
      rules:
        Main:
          deny:
            - pkg: "github.com/pkg/errors"
              desc: "use github.com/bborbe/errors instead"
            - pkg: "github.com/bborbe/argument"
              desc: "use github.com/bborbe/argument/v2 instead"
            - pkg: "golang.org/x/net/context"
              desc: "use context from standard library instead"
            - pkg: "golang.org/x/lint/golint"
              desc: "deprecated, use revive or staticcheck instead"
            - pkg: "io/ioutil"
              desc: "deprecated since Go 1.16, use io and os packages instead"
    funlen:
      lines: 80
      statements: 50
    gocognit:
      min-complexity: 20
    nestif:
      min-complexity: 4
    maintidx:
      min-maintainability-index: 20
  exclusions:
    presets:
      - comments
      - std-error-handling
      - common-false-positives
    rules:
      - linters:
          - staticcheck
        text: "SA1019"
      - linters:
          - errname
        text: "(KeyNotFoundError|TransactionAlreadyOpenError|BucketNotFoundError|BucketAlreadyExistsError)"
      - linters:
          - revive
        path: "_test\\.go$"
        text: "dot-imports"
      - linters:
          - revive
        text: "unused-parameter"
      - linters:
          - revive
        text: "exported"
      - linters:
          - dupl
        path: "_test\\.go$"
      - linters:
          - unparam
        path: "_test\\.go$"
      - linters:
          - dupl
        path: "-test-suite\\.go$"
      - linters:
          - revive
        path: "-test-suite\\.go$"
        text: "dot-imports"

formatters:
  enable:
    - gofmt
    - goimports
```
</context>

<requirements>
1. In `Makefile`:
   - Remove the two comment lines (the `# TODO: enable lint` line and the `# check: lint vet errcheck vulncheck osv-scanner gosec trivy` line)
   - Change the `check` target from `check: vet errcheck vulncheck osv-scanner gosec trivy` to `check: lint vet errcheck vulncheck osv-scanner gosec trivy`

   Before:
   ```makefile
   # TODO: enable lint
   # check: lint vet errcheck vulncheck osv-scanner gosec trivy
   .PHONY: check
   check: vet errcheck vulncheck osv-scanner gosec trivy
   ```

   After:
   ```makefile
   .PHONY: check
   check: lint vet errcheck vulncheck osv-scanner gosec trivy
   ```

2. Update `.golangci.yml` to match the kv reference config at `the reference standard config shown in the context section`. Key differences to apply:
   - Add missing linters: `nestif`, `errname`, `unparam`, `bodyclose`, `forcetypeassert`, `asasalint`, `prealloc`
   - Fix the typo in depguard deny rules: `github.com/pkg/erros` should be `github.com/pkg/errors` and `github.com/bborbe/erros` should be `github.com/bborbe/errors`
   - Add additional depguard deny rules from kv config: `github.com/bborbe/argument` (use v2), `golang.org/x/net/context` (use stdlib), `golang.org/x/lint/golint` (deprecated), `io/ioutil` (deprecated)
   - Add linter settings: `funlen`, `gocognit`, `nestif`, `maintidx`
   - Add exclusion rules from kv config: `errname` exclusion for `(KeyNotFoundError|TransactionAlreadyOpenError|BucketNotFoundError|BucketAlreadyExistsError)`, `revive` exported exclusion, `dupl` and `unparam` test file exclusions, `dupl` and `revive` test-suite file exclusions
   - Keep the existing `revive` unused-parameter and dot-imports exclusions

3. Run `make lint`  to identify all violations.

4. Fix all reported lint violations in the Go source files. Apply minimal, targeted fixes — do not refactor unrelated code.

5. Run `make precommit` to verify everything passes.
</requirements>

<constraints>
- Do NOT commit — dark-factory handles git
- Do NOT refactor application code unrelated to lint fixes
- Do NOT change function signatures of exported methods (they implement kv interfaces)
- Do NOT modify test logic — only fix lint issues (e.g., missing error checks, unused params)
- Keep the project module name `github.com/bborbe/boltkv` in all references
- Existing tests must still pass
</constraints>

<verification>
Run `make precommit`  — must pass with exit code 0.
</verification>

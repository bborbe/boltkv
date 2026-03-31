# Changelog

All notable changes to this project will be documented in this file.

Please choose versions by [Semantic Versioning](http://semver.org/).

* MAJOR version when you make incompatible API changes,
* MINOR version when you add functionality in a backwards-compatible manner, and
* PATCH version when you make backwards-compatible bug fixes.

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

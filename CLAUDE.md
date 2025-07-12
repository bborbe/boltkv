# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

BoltKV is a Go library that implements the common `github.com/bborbe/kv` interfaces for BoltDB (go.etcd.io/bbolt). It provides a standardized key-value store interface while exposing BoltDB-specific functionality through extended methods.

## Development Commands

### Building and Testing
```bash
make precommit    # Full development workflow (ensure + format + generate + test + check + addlicense)
make test         # Run all tests with race detection and coverage
make format       # Format code with gofmt and organize imports
make check        # Run vet, errcheck, and vulnerability scanning
```

### Running Single Tests
```bash
go test ./path/to/package                    # Test specific package
ginkgo -focus "test description"             # Run specific Ginkgo tests
```

### CLI Tools
The project includes several command-line utilities in `/cmd/`:
- `bolt-bucket-delete` - Delete BoltDB buckets
- `bolt-bucket-list` - List all buckets
- `bolt-value-get/set/delete/list` - Manage key-value pairs

Each tool has its own Makefile with a `run` target for local testing.

## Architecture

### Core Components
- **boltkv_db.go** - Database connection and lifecycle management
- **boltkv_tx.go** - Transaction handling with bucket caching
- **boltkv_bucket.go** - Key-value operations within buckets
- **boltkv_iterator.go** - Forward iteration support
- **boltkv_iterator-reverse.go** - Reverse iteration support

### Key Design Patterns
- **Interface Extension**: Implements `github.com/bborbe/kv` interfaces while adding BoltDB-specific access methods (`DB()`, `Tx()`, `Bucket()`, `Cursor()`)
- **Transaction State Management**: Uses context keys to track transaction state and prevent nesting
- **Bucket Caching**: Transactions cache opened buckets with mutex protection
- **Factory Methods**: Multiple database creation options (`OpenFile`, `OpenDir`, `OpenTemp`)

### Dependencies
- Uses `github.com/bborbe/kv` for interface definitions
- Testing with Ginkgo/Gomega framework
- Error handling through `github.com/bborbe/errors`
- Service framework via `github.com/bborbe/service`

### Interface Compliance
The library passes the shared test suites from `github.com/bborbe/kv` to ensure proper interface implementation. Tests validate both the standard KV interface behavior and BoltDB-specific extended functionality.
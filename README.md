# BoltKV

A Go library that implements the common [`github.com/bborbe/kv`](https://github.com/bborbe/kv) interfaces for BoltDB ([go.etcd.io/bbolt](https://go.etcd.io/bbolt)). This provides a standardized key-value store interface while maintaining access to BoltDB-specific functionality through extended methods.

## Features

- **Standard KV Interface**: Implements `github.com/bborbe/kv` interfaces for consistent usage across different key-value stores
- **BoltDB Extensions**: Access underlying BoltDB types (`*bolt.DB`, `*bolt.Tx`, `*bolt.Bucket`, `*bolt.Cursor`) through extended interfaces
- **Multiple Database Creation Options**: Create databases from files, directories, or temporary locations
- **Transaction State Management**: Built-in transaction nesting prevention and state tracking
- **Bucket Caching**: Efficient bucket management with caching during transactions
- **Forward and Reverse Iteration**: Support for both iteration directions
- **CLI Tools**: Command-line utilities for database management

## Installation

```bash
go get github.com/bborbe/boltkv
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/bborbe/boltkv"
    "github.com/bborbe/kv"
)

func main() {
    ctx := context.Background()
    
    // Open database
    db, err := boltkv.OpenFile(ctx, "my-database.db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close(ctx)
    
    // Start transaction
    err = db.Update(ctx, func(ctx context.Context, tx kv.Tx) error {
        // Create or open bucket
        bucket, err := tx.CreateBucketIfNotExists(ctx, []byte("my-bucket"))
        if err != nil {
            return err
        }
        
        // Set key-value pair
        return bucket.Put(ctx, []byte("key"), []byte("value"))
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Read value
    err = db.View(ctx, func(ctx context.Context, tx kv.Tx) error {
        bucket, err := tx.Bucket(ctx, []byte("my-bucket"))
        if err != nil {
            return err
        }
        
        value, err := bucket.Get(ctx, []byte("key"))
        if err != nil {
            return err
        }
        
        fmt.Printf("Value: %s\n", value)
        return nil
    })
    if err != nil {
        log.Fatal(err)
    }
}
```

### Database Creation Options

```go
// Create from file path
db, err := boltkv.OpenFile(ctx, "/path/to/database.db")

// Create from directory (creates bolt.db inside)
db, err := boltkv.OpenDir(ctx, "/path/to/directory")

// Create temporary database
db, err := boltkv.OpenTemp(ctx)

// With custom options
db, err := boltkv.OpenFile(ctx, "database.db", func(opts *bolt.Options) {
    opts.ReadOnly = true
    opts.Timeout = time.Second * 10
})
```

### Accessing BoltDB-Specific Features

```go
// Access underlying BoltDB types
err = db.Update(ctx, func(ctx context.Context, tx kv.Tx) error {
    // Cast to extended interfaces for BoltDB access
    boltTx := tx.(boltkv.Tx).Tx()          // Get *bolt.Tx
    boltDB := db.(boltkv.DB).DB()          // Get *bolt.DB
    
    bucket, err := tx.CreateBucketIfNotExists(ctx, []byte("bucket"))
    if err != nil {
        return err
    }
    
    boltBucket := bucket.(boltkv.Bucket).Bucket()  // Get *bolt.Bucket
    
    // Use BoltDB-specific functionality
    return boltBucket.ForEach(func(k, v []byte) error {
        fmt.Printf("Key: %s, Value: %s\n", k, v)
        return nil
    })
})
```

### Iteration

```go
err = db.View(ctx, func(ctx context.Context, tx kv.Tx) error {
    bucket, err := tx.Bucket(ctx, []byte("my-bucket"))
    if err != nil {
        return err
    }
    
    // Forward iteration
    return bucket.ForEach(ctx, func(key, value []byte) error {
        fmt.Printf("Key: %s, Value: %s\n", key, value)
        return nil
    })
})

// Reverse iteration
err = db.View(ctx, func(ctx context.Context, tx kv.Tx) error {
    bucket, err := tx.Bucket(ctx, []byte("my-bucket"))
    if err != nil {
        return err
    }
    
    return bucket.ForEachReverse(ctx, func(key, value []byte) error {
        fmt.Printf("Key: %s, Value: %s\n", key, value)
        return nil
    })
})
```

## CLI Tools

BoltKV includes several command-line utilities for database management:

### Bucket Management
```bash
# List all buckets
bolt-bucket-list -database=/path/to/db.bolt

# Delete a bucket
bolt-bucket-delete -database=/path/to/db.bolt -bucket=bucket-name
```

### Key-Value Operations
```bash
# Set a value
bolt-value-set -database=/path/to/db.bolt -bucket=bucket-name -key=mykey -value=myvalue

# Get a value  
bolt-value-get -database=/path/to/db.bolt -bucket=bucket-name -key=mykey

# List all keys in a bucket
bolt-value-list -database=/path/to/db.bolt -bucket=bucket-name

# Delete a key
bolt-value-delete -database=/path/to/db.bolt -bucket=bucket-name -key=mykey
```

## Architecture

### Core Components
- **`boltkv_db.go`** - Database connection and lifecycle management
- **`boltkv_tx.go`** - Transaction handling with bucket caching  
- **`boltkv_bucket.go`** - Key-value operations within buckets
- **`boltkv_iterator.go`** - Forward iteration support
- **`boltkv_iterator-reverse.go`** - Reverse iteration support

### Key Design Patterns
- **Interface Extension**: Implements `github.com/bborbe/kv` interfaces while adding BoltDB-specific access methods
- **Transaction State Management**: Uses context keys to track transaction state and prevent nesting
- **Bucket Caching**: Transactions cache opened buckets with mutex protection for efficiency
- **Factory Methods**: Multiple database creation options for different use cases

## Dependencies

- **[go.etcd.io/bbolt](https://go.etcd.io/bbolt)** - BoltDB embedded key-value database
- **[github.com/bborbe/kv](https://github.com/bborbe/kv)** - Common key-value store interfaces
- **[github.com/bborbe/errors](https://github.com/bborbe/errors)** - Enhanced error handling
- **[github.com/bborbe/service](https://github.com/bborbe/service)** - Service framework

## Testing

The library includes comprehensive tests and passes the shared test suites from `github.com/bborbe/kv` to ensure proper interface implementation.

```bash
# Run all tests
make test

# Run tests with coverage
go test -race -cover ./...

# Run specific test package
go test ./path/to/package
```

## License

This project is licensed under the BSD-style license. See the LICENSE file for details.

## Contributing

Contributions are welcome! Please ensure all tests pass and follow the existing code style.

```bash
# Run the full development workflow
make precommit
```
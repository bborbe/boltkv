// Copyright (c) 2024 Benjamin Borbe All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/bborbe/errors"
	libkv "github.com/bborbe/kv"
	libsentry "github.com/bborbe/sentry"
	"github.com/bborbe/service"
	"github.com/golang/glog"

	"github.com/bborbe/boltkv"
)

func main() {
	app := &application{}
	os.Exit(service.Main(context.Background(), app, &app.SentryDSN, &app.SentryProxy))
}

type application struct {
	SentryDSN   string `required:"false" arg:"sentry-dsn"   env:"SENTRY_DSN"   usage:"SentryDSN"      display:"length"`
	SentryProxy string `required:"false" arg:"sentry-proxy" env:"SENTRY_PROXY" usage:"Sentry Proxy"`
	DataDir     string `required:"true"  arg:"datadir"      env:"DATADIR"      usage:"data directory"`
	Bucket      string `required:"true"  arg:"bucket"       env:"BUCKET"       usage:"bucket name"`
	Key         string `required:"true"  arg:"key"          env:"KEY"          usage:"key read"`
}

func (a *application) Run(ctx context.Context, sentryClient libsentry.Client) error {
	db, err := boltkv.OpenDir(ctx, a.DataDir)
	if err != nil {
		return errors.Wrapf(ctx, err, "open failed")
	}
	err = db.View(ctx, func(ctx context.Context, tx libkv.Tx) error {
		bucket, err := tx.Bucket(ctx, libkv.BucketName(a.Bucket))
		if err != nil {
			return errors.Wrapf(ctx, err, "get bucket failed")
		}
		v, err := bucket.Get(ctx, []byte(a.Key))
		if err != nil {
			return errors.Wrapf(ctx, err, "get key failed")
		}
		return v.Value(func(val []byte) error {
			fmt.Printf("value: %s", string(val))
			return nil
		})
	})
	if err != nil {
		return errors.Wrapf(ctx, err, "view failed")
	}
	glog.V(4).Infof("done")
	return nil
}

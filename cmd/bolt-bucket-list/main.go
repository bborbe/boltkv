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
}

func (a *application) Run(ctx context.Context, sentryClient libsentry.Client) error {
	db, err := boltkv.OpenDir(ctx, a.DataDir)
	if err != nil {
		return errors.Wrapf(ctx, err, "open failed")
	}
	err = db.View(ctx, func(ctx context.Context, tx libkv.Tx) error {
		bucketNames, err := tx.ListBucketNames(ctx)
		if err != nil {
			return errors.Wrapf(ctx, err, "list bucketNames failed")
		}
		for _, bucketName := range bucketNames {
			fmt.Println(bucketName.String())
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(ctx, err, "view failed")
	}
	glog.V(4).Infof("done")
	return nil
}

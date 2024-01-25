package cmd

import (
	"context"
	"embed"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/hibrid/statemachine2_transactional_kafka/lib/app"
	"github.com/hibrid/statemachine2_transactional_kafka/lib/env"
	"go.uber.org/zap"
)

var globalWg sync.WaitGroup
var termChan chan bool
var logRequestsAfter = computeLRA()
var log = app.Logger()
var startTime = time.Now()
var ctx, rootCancel = context.WithCancel(context.Background())
var dataBox embed.FS

func computeLRA() time.Duration {
	lraS := os.Getenv("LOG_REQUEST_AFTER")
	var err error
	var lra time.Duration
	if lraS != "" {
		// honor setting if we have one
		lra, err = time.ParseDuration(lraS)
	} else {
		if env.IsProdLike {
			// prod log after 500ms
			lra = time.Millisecond * 500
		} else {
			// always log in dev
			lra = -1
		}
	}
	if err != nil {
		panic(err)
	}
	return lra
}

func waitForShutdown(closers ...io.Closer) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Finished startup", zap.Duration("time", time.Now().Sub(startTime)))
	select {
	case sig := <-sigs:
		log.Info("Exit signaled", zap.Stringer("signal", sig))
		log.Info("Canceling contexts")
		rootCancel()
	case <-ctx.Done():
		log.Info("Root context canceled")
	}
	log.Info("Closing closers")
	for _, c := range closers {
		if err := c.Close(); err != nil {
			log.Error("Failed to close", zap.Reflect("closer", c), zap.Error(err))
		}
	}

	log.Info("Waiting to shutdown")
	globalWg.Wait()
	log.Info("Done")
}

func die(args ...interface{}) {
	fmt.Fprintln(os.Stderr, args...)
	os.Exit(0)
}

var spawnOrDie = func(name string, listener net.Listener, fn func() error) {
	log.Info("Starting server", zap.String("name", name), zap.Stringer("addr", listener.Addr()))
	go func() {
		log.Fatal("Failure in server", zap.String("name", name), zap.Stringer("laddr", listener.Addr()), zap.Error(fn()))
	}()
}

func ce(name string, err error) {
	if err != nil {
		log.Fatal("Failed to setup: "+name, zap.Error(err))
	}
}

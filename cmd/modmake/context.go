package main

import (
	"context"
	"os"
	"os/signal"
)

func signalCtx() context.Context {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-sigs
		cancel()
	}()

	return ctx
}

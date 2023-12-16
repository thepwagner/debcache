package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/thepwagner/debcache/pkg/server"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := server.Run(ctx); err != nil {
		panic(err)
	}
}

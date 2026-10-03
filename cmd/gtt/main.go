// Command gtt is the GTT CLI.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/GTT-Community/gtt-cli/internal/cli"
	"github.com/GTT-Community/gtt-cli/internal/compose"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	wire := compose.Wire(compose.Streams{In: os.Stdin, Out: os.Stdout, Err: os.Stderr})
	os.Exit(cli.Execute(ctx, wire, os.Args[1:], os.Stdout, os.Stderr))
}

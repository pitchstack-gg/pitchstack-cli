package main

import (
	"context"
	"os"

	"github.com/pitchstack-gg/pitchstack-cli/internal/app"
)

func main() {
	os.Exit(app.Run(context.Background(), os.Args))
}

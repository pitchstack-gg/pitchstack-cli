package app

import (
	"context"
	"fmt"
	"os"

	"github.com/pitchstack-gg/pitchstack-cli/internal/commands"

	"github.com/urfave/cli/v3"
)

func Run(ctx context.Context, args []string) int {
	stdout := os.Stdout
	stderr := os.Stderr

	root := commands.NewRootCommand(os.Stdin, stdout, stderr)
	root.ExitErrHandler = func(context.Context, *cli.Command, error) {}

	err := root.Run(ctx, args)
	if err == nil {
		return 0
	}

	if exitErr, ok := err.(cli.ExitCoder); ok {
		if msg := exitErr.Error(); msg != "" {
			fmt.Fprintln(stderr, msg)
		}
		return exitErr.ExitCode()
	}
	fmt.Fprintln(stderr, err.Error())
	return 1
}

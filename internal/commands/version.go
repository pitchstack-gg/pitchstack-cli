package commands

import (
	"context"
	"fmt"

	"github.com/pitchstack-gg/pitchstack-cli/internal/buildinfo"

	"github.com/urfave/cli/v3"
)

func newVersionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Show version information",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			_, err := fmt.Fprintf(cmd.Writer, "pitchstack %s (%s)\n", buildinfo.Version, buildinfo.Commit)
			return err
		},
	}
}

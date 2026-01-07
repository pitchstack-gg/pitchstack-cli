package commands

import (
	"context"
	"fmt"

	"github.com/pitchstack-gg/pitchstack-cli/internal/config"

	"github.com/urfave/cli/v3"
)

func newConfigCommand() *cli.Command {
	return &cli.Command{
		Name:  "config",
		Usage: "Manage CLI config",
		Commands: []*cli.Command{
			{
				Name:  "init",
				Usage: "Write a default config file",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					st, err := getState(ctx)
					if err != nil {
						return err
					}
					if err := config.WriteDefault(st.ConfigPath); err != nil {
						return err
					}
					_, err = fmt.Fprintf(cmd.Writer, "wrote config: %s\n", st.ConfigPath)
					return err
				},
			},
		},
	}
}

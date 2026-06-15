package commands

import (
	"context"
	"fmt"
	"strings"

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
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "force", Usage: "Overwrite an existing config file"},
				},
				Action: func(ctx context.Context, cmd *cli.Command) error {
					st, err := getState(ctx)
					if err != nil {
						return err
					}
					if err := config.WriteDefault(st.ConfigPath, cmd.Bool("force")); err != nil {
						return err
					}
					_, err = fmt.Fprintf(cmd.Writer, "wrote config: %s\n", st.ConfigPath)
					return err
				},
			},
			{
				Name:  "show",
				Usage: "Show active config profile",
				Action: func(ctx context.Context, cmd *cli.Command) error {
					st, err := getState(ctx)
					if err != nil {
						return err
					}
					_, _ = fmt.Fprintf(cmd.Writer, "config: %s\n", st.ConfigPath)
					_, _ = fmt.Fprintf(cmd.Writer, "profile: %s\n", st.ProfileName)
					_, _ = fmt.Fprintf(cmd.Writer, "baseUrl: %s\n", strings.TrimSpace(st.Profile.BaseURL))
					if strings.TrimSpace(st.Profile.OAuthBaseURL) != "" {
						_, _ = fmt.Fprintf(cmd.Writer, "oauthBaseUrl: %s\n", strings.TrimSpace(st.Profile.OAuthBaseURL))
					}
					_, _ = fmt.Fprintf(cmd.Writer, "cardsDbUrl: %s\n", strings.TrimSpace(st.Profile.CardsDBURL))
					if strings.TrimSpace(st.Profile.PowerSyncURL) != "" {
						_, _ = fmt.Fprintf(cmd.Writer, "powerSyncUrl: %s\n", strings.TrimSpace(st.Profile.PowerSyncURL))
					}
					return nil
				},
			},
		},
	}
}

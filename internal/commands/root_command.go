package commands

import (
	"context"
	"io"

	"github.com/pitchstack-gg/pitchstack-cli/internal/buildinfo"
	"github.com/pitchstack-gg/pitchstack-cli/internal/paths"

	"github.com/urfave/cli/v3"
)

func NewRootCommand(stdin io.Reader, stdout io.Writer, stderr io.Writer) *cli.Command {
	cmd := &cli.Command{
		Name:                   "pitchstack",
		Usage:                  "Command-line tools for Pitchstack",
		UseShortOptionHandling: true,
		Reader:                 stdin,
		Writer:                 stdout,
		ErrWriter:              stderr,
		Before:                 loadState,
		ExitErrHandler:         func(context.Context, *cli.Command, error) {},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "config",
				Usage:    "Path to config file",
				Value:    paths.DefaultConfigPath(),
				Aliases:  []string{"c"},
				OnlyOnce: true,
			},
			&cli.StringFlag{
				Name:     "profile",
				Usage:    "Config profile to use",
				Aliases:  []string{"p"},
				OnlyOnce: true,
			},
		},
		Commands: rootCommands(),
	}

	inheritIO(cmd, stdin, stdout, stderr)
	return cmd
}

func rootCommands() []*cli.Command {
	commands := []*cli.Command{
		newLoginCommand(),
		newAuthCommand(),
		newSignupCommand(),
		newWhoamiCommand(),
		newLogoutCommand(),
	}
	if buildinfo.IsDevelopment() {
		commands = append(commands, newTUICommand())
	}
	commands = append(commands,
		newProfileCommand(),
		newActivityCommand(),
		newCardsCommand(),
		newCollectionsCommand(),
		newDecksCommand(),
		newGroupsCommand(),
		newSocialCommand(),
		newEngagementCommand(),
		newEventsCommand(),
		newPricingCommand(),
		newNewsCommand(),
		newNotificationsCommand(),
		newPullsCommand(),
		newConfigCommand(),
		newVersionCommand(),
	)
	return commands
}

func inheritIO(cmd *cli.Command, stdin io.Reader, stdout io.Writer, stderr io.Writer) {
	if cmd == nil {
		return
	}
	if cmd.Reader == nil {
		cmd.Reader = stdin
	}
	if cmd.Writer == nil {
		cmd.Writer = stdout
	}
	if cmd.ErrWriter == nil {
		cmd.ErrWriter = stderr
	}
	for _, child := range cmd.Commands {
		inheritIO(child, stdin, stdout, stderr)
	}
}

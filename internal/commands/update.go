package commands

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pitchstack-gg/pitchstack-cli/internal/buildinfo"
	"github.com/pitchstack-gg/pitchstack-cli/internal/updater"

	"github.com/urfave/cli/v3"
)

func newUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Check for and install CLI updates",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "check", Usage: "Only check whether an update is available"},
			&cli.BoolFlag{Name: "force", Usage: "Reinstall the latest release even when already current"},
			&cli.StringFlag{Name: "install-dir", Usage: "Install into this directory instead of replacing the current binary"},
			&cli.StringFlag{Name: "repo", Usage: "GitHub repository to check", Value: updater.DefaultRepo, Hidden: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			defer cancel()

			if cmd.Bool("check") {
				release, err := updater.Latest(ctx, cmd.String("repo"), "", nil)
				if err != nil {
					return err
				}
				if !buildinfo.IsDevelopment() && updater.CompareVersions(release.Version, buildinfo.Version) <= 0 {
					_, err = fmt.Fprintf(cmd.Writer, "pitchstack is up to date (%s)\n", buildinfo.Version)
					return err
				}
				_, err = fmt.Fprintf(cmd.Writer, "Update available: pitchstack %s", release.Version)
				if !buildinfo.IsDevelopment() {
					_, err = fmt.Fprintf(cmd.Writer, " (current %s)", buildinfo.Version)
				}
				if err == nil {
					_, err = fmt.Fprintln(cmd.Writer)
				}
				return err
			}

			if buildinfo.IsDevelopment() && strings.TrimSpace(cmd.String("install-dir")) == "" {
				return cli.Exit("development builds cannot safely replace themselves; pass --install-dir or rebuild from source", 2)
			}

			result, err := updater.InstallLatest(ctx, updater.InstallOptions{
				Repo:       cmd.String("repo"),
				Current:    buildinfo.Version,
				InstallDir: cmd.String("install-dir"),
				Force:      cmd.Bool("force"),
				Stdin:      os.Stdin,
				Stdout:     os.Stdout,
				Stderr:     os.Stderr,
			})
			if err != nil {
				return err
			}
			if !result.Installed {
				_, err = fmt.Fprintf(cmd.Writer, "pitchstack is up to date (%s)\n", buildinfo.Version)
				return err
			}
			_, err = fmt.Fprintf(cmd.Writer, "Updated pitchstack to %s at %s\n", result.Release.Version, result.Target)
			return err
		},
	}
}

func maybePrintUpdateNotice(ctx context.Context, cmd *cli.Command) {
	if shouldSkipUpdateNotice(cmd) {
		return
	}
	checkCtx, cancel := context.WithTimeout(ctx, 1200*time.Millisecond)
	defer cancel()
	result, err := updater.Check(checkCtx, buildinfo.Version, updater.CheckOptions{})
	if err != nil || !result.UpdateAvailable {
		return
	}
	_, _ = fmt.Fprintf(cmd.ErrWriter, "Update available: pitchstack %s (current %s). Run \"pitchstack update\" to install.\n", result.Release.Version, buildinfo.Version)
}

func shouldSkipUpdateNotice(cmd *cli.Command) bool {
	if buildinfo.IsDevelopment() {
		return true
	}
	if strings.EqualFold(os.Getenv("PITCHSTACK_NO_UPDATE_CHECK"), "1") || strings.EqualFold(os.Getenv("PITCHSTACK_NO_UPDATE_CHECK"), "true") {
		return true
	}
	args := cmd.Args().Slice()
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	if len(args) > 0 && args[0] == "update" {
		return true
	}
	return false
}

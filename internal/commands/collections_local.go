package commands

import (
	"context"

	"github.com/urfave/cli/v3"
)

func newCollectionsCountsCommand() *cli.Command {
	return &cli.Command{
		Name:  "counts",
		Usage: "Count cards across synced collections",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "local", Usage: "Read from the local PowerSync cache"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			if !cmd.Bool("local") {
				return cli.Exit("collections counts currently requires --local", 2)
			}
			store, err := openLocalPowerSyncStore(ctx)
			if err != nil {
				return err
			}
			defer store.Close()
			counts, err := store.CollectionCounts(ctx)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, map[string]any{"collections": counts})
		},
	}
}

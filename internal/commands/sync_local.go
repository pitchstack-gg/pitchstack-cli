package commands

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/pitchstack-gg/pitchstack-cli/internal/paths"
	"github.com/pitchstack-gg/pitchstack-cli/internal/powersync"
	"github.com/urfave/cli/v3"
)

func newSyncLocalCommand() *cli.Command {
	return &cli.Command{
		Name:  "local",
		Usage: "Manage the local PowerSync cache",
		Commands: []*cli.Command{
			newSyncLocalInitCommand(),
			newSyncLocalPullCommand(),
			newSyncLocalWatchCommand(),
			newSyncLocalStatusCommand(),
			newSyncLocalResetCommand(),
		},
	}
}

func newSyncLocalInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Create or migrate the local PowerSync cache",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, store, err := newLocalPowerSyncClient(ctx)
			if err != nil {
				return err
			}
			defer store.Close()
			result, err := client.Initialize(ctx)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, result)
		},
	}
}

func newSyncLocalPullCommand() *cli.Command {
	return &cli.Command{
		Name:  "pull",
		Usage: "Pull one complete PowerSync checkpoint into the local cache",
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "skip-upload", Usage: "Do not upload pending local CRUD before pulling"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, store, err := newLocalPowerSyncClient(ctx)
			if err != nil {
				return err
			}
			defer store.Close()
			if _, err := client.Initialize(ctx); err != nil {
				return err
			}
			if !cmd.Bool("skip-upload") {
				if err := client.UploadPending(ctx); err != nil {
					return err
				}
			}
			if err := client.PullOnce(ctx); err != nil {
				return err
			}
			status, err := store.Status(ctx)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, status)
		},
	}
}

func newSyncLocalWatchCommand() *cli.Command {
	return &cli.Command{
		Name:  "watch",
		Usage: "Continuously sync the local PowerSync cache",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			client, store, err := newLocalPowerSyncClient(ctx)
			if err != nil {
				return err
			}
			defer store.Close()
			if _, err := client.Initialize(ctx); err != nil {
				return err
			}
			return client.Watch(ctx)
		},
	}
}

func newSyncLocalStatusCommand() *cli.Command {
	return &cli.Command{
		Name:  "status",
		Usage: "Show local PowerSync cache status",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			store, err := openLocalPowerSyncStore(ctx)
			if err != nil {
				return err
			}
			defer store.Close()
			status, err := store.Status(ctx)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, status)
		},
	}
}

func newSyncLocalResetCommand() *cli.Command {
	return &cli.Command{
		Name:  "reset",
		Usage: "Delete the local PowerSync cache",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			path := paths.SyncDBPath(st.ProfileName)
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
			return writeJSON(cmd.Writer, map[string]any{
				"path":    path,
				"deleted": true,
			})
		},
	}
}

func newLocalPowerSyncClient(ctx context.Context) (*powersync.Client, *powersync.Store, error) {
	st, err := getState(ctx)
	if err != nil {
		return nil, nil, err
	}
	store, err := openLocalPowerSyncStore(ctx)
	if err != nil {
		return nil, nil, err
	}
	apiClient, err := st.Service.AuthenticatedClient()
	if err != nil {
		_ = store.Close()
		return nil, nil, err
	}
	connector := &powersync.APIConnector{
		Client:           apiClient,
		EndpointOverride: strings.TrimSpace(st.Profile.PowerSyncURL),
		TokenProvider:    st.Service.BearerToken,
	}
	return &powersync.Client{
		Store:      store,
		Connector:  connector,
		HTTPClient: &http.Client{},
	}, store, nil
}

func openLocalPowerSyncStore(ctx context.Context) (*powersync.Store, error) {
	st, err := getState(ctx)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(st.ProfileName) == "" {
		return nil, errors.New("missing profile name")
	}
	return powersync.OpenStore(paths.SyncDBPath(st.ProfileName))
}

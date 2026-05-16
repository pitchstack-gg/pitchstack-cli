package commands

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"

	"github.com/urfave/cli/v3"
)

func newSyncCommand() *cli.Command {
	return &cli.Command{
		Name:  "sync",
		Usage: "Sync service helpers",
		Commands: []*cli.Command{
			newSyncChangesCommand(),
			newSyncApplyCommand(),
			newSyncLocalCommand(),
			newSyncPowerSyncConfigCommand(),
			newSyncUploadCrudCommand(),
			newSyncSubscriptionsCommand(),
		},
	}
}

func newSyncChangesCommand() *cli.Command {
	return &cli.Command{
		Name:  "changes",
		Usage: "Fetch change sets after a cursor (optionally poll)",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "cursor", Usage: "Sync cursor"},
			&cli.IntFlag{Name: "page-size", Usage: "Page size"},
			&cli.BoolFlag{Name: "include-documents", Usage: "Include document snapshots when available"},
			&cli.BoolFlag{Name: "poll", Aliases: []string{"follow"}, Usage: "Poll for changes continuously"},
			&cli.DurationFlag{Name: "interval", Usage: "Poll interval", Value: 2 * time.Second},
			&cli.BoolFlag{Name: "quiet-empty", Usage: "When polling, don't print empty responses", Value: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			cursor := strings.TrimSpace(cmd.String("cursor"))
			includeDocuments := cmd.Bool("include-documents")

			var pageSize *int32
			if cmd.IsSet("page-size") && cmd.Int("page-size") > 0 {
				ps := int32(cmd.Int("page-size"))
				pageSize = &ps
			}

			poll := cmd.Bool("poll")
			interval := cmd.Duration("interval")
			if interval <= 0 {
				return cli.Exit("--interval must be > 0", 2)
			}
			quietEmpty := cmd.Bool("quiet-empty")

			for {
				req := &clientv1.GetChangeSetRequest{
					Cursor:           cursor,
					PageSize:         pageSize,
					IncludeDocuments: includeDocuments,
				}

				resp, err := st.Service.GetChangeSet(ctx, req)
				if err != nil {
					return err
				}

				if !poll || !quietEmpty || len(resp.Events) > 0 || resp.HasMore {
					if err := writeJSON(cmd.Writer, resp); err != nil {
						return err
					}
				}

				if strings.TrimSpace(resp.NextCursor) != "" {
					cursor = strings.TrimSpace(resp.NextCursor)
				}

				if !poll {
					return nil
				}

				if resp.HasMore {
					continue
				}

				timer := time.NewTimer(interval)
				select {
				case <-ctx.Done():
					timer.Stop()
					return nil
				case <-timer.C:
				}
			}
		},
	}
}

func newSyncApplyCommand() *cli.Command {
	return &cli.Command{
		Name:  "apply",
		Usage: "Batch apply local changes from a JSON file",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "device-id", Usage: "Device identifier"},
			&cli.StringFlag{Name: "file", Usage: "Path to JSON file (BatchApplyChangesRequest or array of LocalChange)", Required: true},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			req, err := readBatchApplyChangesRequest(cmd.String("file"))
			if err != nil {
				return err
			}
			if strings.TrimSpace(req.DeviceID) == "" {
				req.DeviceID = strings.TrimSpace(cmd.String("device-id"))
			}
			if strings.TrimSpace(req.DeviceID) == "" {
				return cli.Exit("--device-id is required (or include deviceId in the JSON file)", 2)
			}

			resp, err := st.Service.BatchApplyChanges(ctx, req)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func readBatchApplyChangesRequest(path string) (*clientv1.BatchApplyChangesRequest, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil, errors.New("file path must not be empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var req clientv1.BatchApplyChangesRequest
	if err := json.Unmarshal(data, &req); err == nil && len(req.Changes) > 0 {
		return &req, nil
	}

	var changes []clientv1.LocalChange
	if err := json.Unmarshal(data, &changes); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if len(changes) == 0 {
		return nil, errors.New("no changes provided")
	}
	return &clientv1.BatchApplyChangesRequest{Changes: changes}, nil
}

func newSyncPowerSyncConfigCommand() *cli.Command {
	return newSDKNoRequestCommand("powersync-config", "Get PowerSync client config", true, func(ctx context.Context, c *clientv1.Client) (any, error) {
		return c.GetPowerSyncClientConfig(ctx)
	})
}

func newSyncUploadCrudCommand() *cli.Command {
	return newSDKCommand("upload-crud", "Upload PowerSync CRUD entries from JSON", []cli.Flag{&cli.StringFlag{Name: "device-id", Usage: "Device ID"}}, true, func(cmd *cli.Command, req *clientv1.UploadCrudRequest) error {
		setStringFlag(cmd, "device-id", &req.DeviceID)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UploadCrudRequest) (any, error) {
		return c.UploadCrud(ctx, req)
	})
}

func newSyncSubscriptionsCommand() *cli.Command {
	return &cli.Command{
		Name:  "subscriptions",
		Usage: "Manage sync subscriptions",
		Commands: []*cli.Command{
			newSyncSubscriptionsListCommand(),
			newSyncSubscriptionsUpdateCommand(),
		},
	}
}

func newSyncSubscriptionsListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "List subscriptions",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			resp, err := st.Service.ListSubscriptions(ctx)
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func newSyncSubscriptionsUpdateCommand() *cli.Command {
	return &cli.Command{
		Name:  "update",
		Usage: "Subscribe/unsubscribe to resources",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "subscribe", Usage: "Resource to subscribe to (repeatable, format: <type>:<id>)"},
			&cli.StringSliceFlag{Name: "unsubscribe", Usage: "Resource to unsubscribe from (repeatable, format: <type>:<id>)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			st, err := getState(ctx)
			if err != nil {
				return err
			}

			var mutations []clientv1.SubscriptionMutation
			for _, spec := range cmd.StringSlice("subscribe") {
				rt, id, err := parseResourceDescriptor(spec)
				if err != nil {
					return cli.Exit(fmt.Sprintf("invalid --subscribe %q: %s", spec, err.Error()), 2)
				}
				resType, ok := parseSyncResourceType(rt)
				if !ok {
					return cli.Exit(fmt.Sprintf("invalid --subscribe %q: unknown type %q", spec, rt), 2)
				}
				mutations = append(mutations, clientv1.SubscriptionMutation{
					Type: clientv1.SubscriptionMutationTypeSubscribe,
					Resource: &clientv1.ResourceDescriptor{
						Type: resType,
						ID:   id,
					},
				})
			}

			for _, spec := range cmd.StringSlice("unsubscribe") {
				rt, id, err := parseResourceDescriptor(spec)
				if err != nil {
					return cli.Exit(fmt.Sprintf("invalid --unsubscribe %q: %s", spec, err.Error()), 2)
				}
				resType, ok := parseSyncResourceType(rt)
				if !ok {
					return cli.Exit(fmt.Sprintf("invalid --unsubscribe %q: unknown type %q", spec, rt), 2)
				}
				mutations = append(mutations, clientv1.SubscriptionMutation{
					Type: clientv1.SubscriptionMutationTypeUnsubscribe,
					Resource: &clientv1.ResourceDescriptor{
						Type: resType,
						ID:   id,
					},
				})
			}

			if len(mutations) == 0 {
				return cli.Exit("at least one of --subscribe or --unsubscribe is required", 2)
			}

			resp, err := st.Service.UpdateSubscriptions(ctx, &clientv1.UpdateSubscriptionsRequest{Mutations: mutations})
			if err != nil {
				return err
			}
			return writeJSON(cmd.Writer, resp)
		},
	}
}

func parseSyncResourceType(v string) (clientv1.ResourceType, bool) {
	n := strings.ToLower(strings.TrimSpace(v))
	n = strings.ReplaceAll(n, "-", "_")
	switch n {
	case "collection", "collections", "resource_type_collection":
		return clientv1.ResourceTypeCollection, true
	case "collection_item", "collection_items", "resource_type_collection_item":
		return clientv1.ResourceTypeCollectionItem, true
	case "deck", "decks", "resource_type_deck":
		return clientv1.ResourceTypeDeck, true
	case "deck_version", "deck_versions", "resource_type_deck_version":
		return clientv1.ResourceTypeDeckVersion, true
	case "user_profile", "profile", "profiles", "resource_type_user_profile":
		return clientv1.ResourceTypeUserProfile, true
	default:
		if strings.HasPrefix(strings.ToUpper(strings.TrimSpace(v)), "RESOURCE_TYPE_") {
			return clientv1.ResourceType(strings.ToUpper(strings.TrimSpace(v))), true
		}
		return clientv1.ResourceTypeUnspecified, false
	}
}

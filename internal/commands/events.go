package commands

import (
	"context"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
	"github.com/urfave/cli/v3"
)

func newEventsCommand() *cli.Command {
	return &cli.Command{
		Name:  "events",
		Usage: "Event and store helpers",
		Commands: []*cli.Command{
			newSDKCommand("list", "List events", eventListFlags(), true, applyListEventsFlags, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListEventsRequest) (any, error) {
				return c.ListEvents(ctx, req)
			}),
			newSDKCommand("get", "Get an event", []cli.Flag{&cli.StringFlag{Name: "id", Usage: "Event ID"}}, true, func(cmd *cli.Command, req *clientv1.GetEventRequest) error {
				setStringFlag(cmd, "id", &req.EventID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetEventRequest) (any, error) {
				return c.GetEvent(ctx, req)
			}),
			newSDKCommand("create", "Create a community event", eventWriteFlags(false), true, applyCreateEventFlags, func(ctx context.Context, c *clientv1.Client, req *clientv1.CreateCommunityEventRequest) (any, error) {
				return c.CreateCommunityEvent(ctx, req)
			}),
			newSDKCommand("update", "Update a community event", eventWriteFlags(true), true, applyUpdateEventFlags, func(ctx context.Context, c *clientv1.Client, req *clientv1.UpdateCommunityEventRequest) (any, error) {
				return c.UpdateCommunityEvent(ctx, req)
			}),
			newSDKCommand("cancel", "Cancel a community event", []cli.Flag{&cli.StringFlag{Name: "id", Usage: "Event ID"}}, true, func(cmd *cli.Command, req *clientv1.CancelCommunityEventRequest) error {
				setStringFlag(cmd, "id", &req.EventID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.CancelCommunityEventRequest) (any, error) {
				return c.CancelCommunityEvent(ctx, req)
			}),
			newSDKCommand("hide", "Hide an event", []cli.Flag{&cli.StringFlag{Name: "id", Usage: "Event ID"}, &cli.StringFlag{Name: "reason", Usage: "Reason"}}, true, func(cmd *cli.Command, req *clientv1.HideEventRequest) error {
				setStringFlag(cmd, "id", &req.EventID)
				setStringFlag(cmd, "reason", &req.Reason)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.HideEventRequest) (any, error) {
				return c.HideEvent(ctx, req)
			}),
			newSDKCommand("unhide", "Unhide an event", []cli.Flag{&cli.StringFlag{Name: "id", Usage: "Event ID"}}, true, func(cmd *cli.Command, req *clientv1.UnhideEventRequest) error {
				setStringFlag(cmd, "id", &req.EventID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UnhideEventRequest) (any, error) {
				return c.UnhideEvent(ctx, req)
			}),
			newEventsStoresCommand(),
			newSDKNoRequestCommand("filters", "List event filter values", true, func(ctx context.Context, c *clientv1.Client) (any, error) {
				return c.ListEventFilters(ctx)
			}),
			newSDKCommand("gem-scan", "Get a GEM locator scan", []cli.Flag{&cli.StringFlag{Name: "run-id", Usage: "Run ID"}}, true, func(cmd *cli.Command, req *clientv1.GetGemLocatorScanRequest) error {
				setStringFlag(cmd, "run-id", &req.RunID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetGemLocatorScanRequest) (any, error) {
				return c.GetGemLocatorScan(ctx, req)
			}),
		},
	}
}

func newEventsStoresCommand() *cli.Command {
	return &cli.Command{
		Name:  "stores",
		Usage: "Store helpers",
		Commands: []*cli.Command{
			newSDKCommand("list", "List stores", append(pageFlags(),
				&cli.StringFlag{Name: "q", Usage: "Search query"},
				&cli.StringFlag{Name: "country", Usage: "Country"},
				&cli.StringFlag{Name: "latitude", Usage: "Latitude"},
				&cli.StringFlag{Name: "longitude", Usage: "Longitude"},
				&cli.Float64Flag{Name: "radius-km", Usage: "Radius in km"},
			), true, func(cmd *cli.Command, req *clientv1.ListStoresRequest) error {
				setStringFlag(cmd, "q", &req.Query)
				setStringFlag(cmd, "country", &req.Country)
				setFloat64Flag(cmd, "radius-km", &req.RadiusKM)
				setPageFlags(cmd, &req.PageSize, &req.NextToken)
				lat, err := parseFloatFlag(cmd, "latitude")
				if err != nil {
					return err
				}
				lng, err := parseFloatFlag(cmd, "longitude")
				if err != nil {
					return err
				}
				req.Latitude = lat
				req.Longitude = lng
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListStoresRequest) (any, error) {
				return c.ListStores(ctx, req)
			}),
			newSDKCommand("get", "Get a store", []cli.Flag{&cli.StringFlag{Name: "id", Usage: "Store ID"}}, true, func(cmd *cli.Command, req *clientv1.GetStoreRequest) error {
				setStringFlag(cmd, "id", &req.StoreRef)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetStoreRequest) (any, error) {
				return c.GetStore(ctx, req)
			}),
		},
	}
}

func eventListFlags() []cli.Flag {
	return append(pageFlags(),
		&cli.StringFlag{Name: "source", Usage: "Source"},
		&cli.StringFlag{Name: "q", Usage: "Search query"},
		&cli.StringFlag{Name: "starts-after", Usage: "Start lower bound (RFC3339)"},
		&cli.StringFlag{Name: "starts-before", Usage: "Start upper bound (RFC3339)"},
		&cli.StringFlag{Name: "event-type", Usage: "Event type"},
		&cli.StringFlag{Name: "format", Usage: "Format"},
		&cli.StringFlag{Name: "country", Usage: "Country"},
		&cli.StringFlag{Name: "store-id", Usage: "Store ID"},
		&cli.StringFlag{Name: "store-slug", Usage: "Store slug"},
		&cli.StringFlag{Name: "latitude", Usage: "Latitude"},
		&cli.StringFlag{Name: "longitude", Usage: "Longitude"},
		&cli.Float64Flag{Name: "radius-km", Usage: "Radius in km"},
	)
}

func applyListEventsFlags(cmd *cli.Command, req *clientv1.ListEventsRequest) error {
	setPageFlags(cmd, &req.PageSize, &req.NextToken)
	if cmd.IsSet("source") {
		req.Source = parseEventSource(cmd.String("source"))
	}
	setStringFlag(cmd, "q", &req.Query)
	if err := setTimeFlag(cmd, "starts-after", &req.StartsAfter); err != nil {
		return err
	}
	if err := setTimeFlag(cmd, "starts-before", &req.StartsBefore); err != nil {
		return err
	}
	setStringFlag(cmd, "event-type", &req.EventType)
	setStringFlag(cmd, "format", &req.Format)
	setStringFlag(cmd, "country", &req.Country)
	setStringFlag(cmd, "store-id", &req.StoreID)
	setStringFlag(cmd, "store-slug", &req.StoreSlug)
	setFloat64Flag(cmd, "radius-km", &req.RadiusKM)
	lat, err := parseFloatFlag(cmd, "latitude")
	if err != nil {
		return err
	}
	lng, err := parseFloatFlag(cmd, "longitude")
	if err != nil {
		return err
	}
	req.Latitude = lat
	req.Longitude = lng
	return nil
}

func eventWriteFlags(withID bool) []cli.Flag {
	flags := []cli.Flag{}
	if withID {
		flags = append(flags, &cli.StringFlag{Name: "id", Usage: "Event ID"})
	}
	return append(flags,
		&cli.StringFlag{Name: "title", Usage: "Title"},
		&cli.StringFlag{Name: "event-type", Usage: "Event type"},
		&cli.StringFlag{Name: "format", Usage: "Format"},
		&cli.StringFlag{Name: "starts-at", Usage: "Start time (RFC3339)"},
		&cli.StringFlag{Name: "ends-at", Usage: "End time (RFC3339)"},
		&cli.StringFlag{Name: "venue-name", Usage: "Venue name"},
		&cli.StringFlag{Name: "address", Usage: "Address"},
		&cli.StringFlag{Name: "country", Usage: "Country"},
		&cli.StringFlag{Name: "external-link", Usage: "External link"},
		&cli.StringFlag{Name: "description", Usage: "Description"},
		&cli.IntFlag{Name: "player-cap", Usage: "Player cap"},
	)
}

func applyCreateEventFlags(cmd *cli.Command, req *clientv1.CreateCommunityEventRequest) error {
	setStringFlag(cmd, "title", &req.Title)
	setStringFlag(cmd, "event-type", &req.EventType)
	setStringFlag(cmd, "format", &req.Format)
	if err := setTimeFlag(cmd, "starts-at", &req.StartsAt); err != nil {
		return err
	}
	if err := setTimeFlag(cmd, "ends-at", &req.EndsAt); err != nil {
		return err
	}
	setStringFlag(cmd, "venue-name", &req.VenueName)
	setStringFlag(cmd, "address", &req.Address)
	setStringFlag(cmd, "country", &req.Country)
	setStringFlag(cmd, "external-link", &req.ExternalLink)
	setStringFlag(cmd, "description", &req.Description)
	if cmd.IsSet("player-cap") {
		v := int32(cmd.Int("player-cap"))
		req.PlayerCap = &v
	}
	return nil
}

func applyUpdateEventFlags(cmd *cli.Command, req *clientv1.UpdateCommunityEventRequest) error {
	setStringFlag(cmd, "id", &req.EventID)
	setStringPtrFlag(cmd, "title", &req.Title)
	setStringPtrFlag(cmd, "event-type", &req.EventType)
	setStringPtrFlag(cmd, "format", &req.Format)
	if err := setTimeFlag(cmd, "starts-at", &req.StartsAt); err != nil {
		return err
	}
	if err := setTimeFlag(cmd, "ends-at", &req.EndsAt); err != nil {
		return err
	}
	setStringPtrFlag(cmd, "venue-name", &req.VenueName)
	setStringPtrFlag(cmd, "address", &req.Address)
	setStringPtrFlag(cmd, "country", &req.Country)
	setStringPtrFlag(cmd, "external-link", &req.ExternalLink)
	setStringPtrFlag(cmd, "description", &req.Description)
	if cmd.IsSet("player-cap") {
		v := int32(cmd.Int("player-cap"))
		req.PlayerCap = &v
	}
	return nil
}

func parseEventSource(v string) clientv1.EventSource {
	switch v {
	case "official-gem", "gem":
		return clientv1.EventSourceOfficialGem
	case "community":
		return clientv1.EventSourceCommunity
	default:
		return clientv1.EventSource(v)
	}
}

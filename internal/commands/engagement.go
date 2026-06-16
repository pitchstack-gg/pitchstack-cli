package commands

import (
	"context"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
	"github.com/urfave/cli/v3"
)

func newEngagementCommand() *cli.Command {
	return &cli.Command{
		Name:  "engagement",
		Usage: "Track engagement",
		Commands: []*cli.Command{
			newSDKCommand("track-view", "Track a resource view", append(engagementResourceFlags(), &cli.StringFlag{Name: "client-viewer-id", Usage: "Client viewer ID"}, &cli.StringFlag{Name: "occurred-at", Usage: "Occurrence time (RFC3339)"}), true, func(cmd *cli.Command, req *clientv1.TrackViewRequest) error {
				if cmd.IsSet("resource-type") || cmd.IsSet("resource-id") {
					req.Resource = &clientv1.EngagementResourceRef{
						ResourceType: parseTrackableResourceType(cmd.String("resource-type")),
						ResourceID:   cmd.String("resource-id"),
					}
				}
				setStringFlag(cmd, "client-viewer-id", &req.ClientViewerID)
				return setTimeFlag(cmd, "occurred-at", &req.OccurredAt)
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.TrackViewRequest) (any, error) {
				return c.TrackView(ctx, req)
			}),
			newSDKCommand("batch-track-views", "Track multiple resource views from JSON", nil, true, nil, func(ctx context.Context, c *clientv1.Client, req *clientv1.BatchTrackViewsRequest) (any, error) {
				return c.BatchTrackViews(ctx, req)
			}),
			newSDKCommand("trending", "List trending resources", []cli.Flag{
				&cli.StringFlag{Name: "resource-type", Usage: "Resource type"},
				&cli.StringFlag{Name: "window", Usage: "Window (24h|7d|30d or raw enum)"},
				&cli.IntFlag{Name: "page-size", Usage: "Page size"},
				&cli.StringFlag{Name: "next-token", Usage: "Next page token"},
			}, true, func(cmd *cli.Command, req *clientv1.ListTrendingResourcesRequest) error {
				if cmd.IsSet("resource-type") {
					req.ResourceType = parseTrackableResourceType(cmd.String("resource-type"))
				}
				if cmd.IsSet("window") {
					req.Window = parseTrendingWindow(cmd.String("window"))
				}
				if cmd.IsSet("page-size") {
					req.PageSize = int32(cmd.Int("page-size"))
				}
				setStringFlag(cmd, "next-token", &req.NextPageToken)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListTrendingResourcesRequest) (any, error) {
				return c.ListTrendingResources(ctx, req)
			}),
			newSDKCommand("view-counts", "Get view counts from JSON", nil, true, nil, func(ctx context.Context, c *clientv1.Client, req *clientv1.BatchGetViewCountsRequest) (any, error) {
				return c.BatchGetViewCounts(ctx, req)
			}),
			newSDKCommand("like", "Like a resource", engagementResourceFlags(), true, func(cmd *cli.Command, req *clientv1.LikeResourceRequest) error {
				if cmd.IsSet("resource-type") || cmd.IsSet("resource-id") {
					req.Resource = &clientv1.LikeableResourceRef{ResourceType: parseLikeableResourceType(cmd.String("resource-type")), ResourceID: cmd.String("resource-id")}
				}
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.LikeResourceRequest) (any, error) {
				return c.LikeResource(ctx, req)
			}),
			newSDKCommand("unlike", "Unlike a resource", engagementResourceFlags(), true, func(cmd *cli.Command, req *clientv1.UnlikeResourceRequest) error {
				if cmd.IsSet("resource-type") || cmd.IsSet("resource-id") {
					req.Resource = &clientv1.LikeableResourceRef{ResourceType: parseLikeableResourceType(cmd.String("resource-type")), ResourceID: cmd.String("resource-id")}
				}
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UnlikeResourceRequest) (any, error) {
				return c.UnlikeResource(ctx, req)
			}),
			newSDKCommand("like-counts", "Get like counts from JSON", nil, true, nil, func(ctx context.Context, c *clientv1.Client, req *clientv1.BatchGetLikeCountsRequest) (any, error) {
				return c.BatchGetLikeCounts(ctx, req)
			}),
			newSDKCommand("viewer-likes", "Get viewer like states from JSON", nil, true, nil, func(ctx context.Context, c *clientv1.Client, req *clientv1.BatchGetViewerLikesRequest) (any, error) {
				return c.BatchGetViewerLikes(ctx, req)
			}),
		},
	}
}

func engagementResourceFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "resource-type", Usage: "Resource type"},
		&cli.StringFlag{Name: "resource-id", Usage: "Resource ID"},
	}
}

func parseTrackableResourceType(v string) clientv1.TrackableResourceType {
	switch v {
	case "card":
		return clientv1.TrackableResourceTypeCard
	case "deck":
		return clientv1.TrackableResourceTypeDeck
	case "collection":
		return clientv1.TrackableResourceTypeCollection
	case "user-profile", "profile":
		return clientv1.TrackableResourceTypeUserProfile
	default:
		return clientv1.TrackableResourceType(v)
	}
}

func parseLikeableResourceType(v string) clientv1.LikeableResourceType {
	switch v {
	case "deck":
		return clientv1.LikeableResourceTypeDeck
	case "collection":
		return clientv1.LikeableResourceTypeCollection
	default:
		return clientv1.LikeableResourceType(v)
	}
}

func parseTrendingWindow(v string) clientv1.TrendingWindow {
	switch v {
	case "24h":
		return clientv1.TrendingWindow24H
	case "7d":
		return clientv1.TrendingWindow7D
	case "30d":
		return clientv1.TrendingWindow30D
	default:
		return clientv1.TrendingWindow(v)
	}
}

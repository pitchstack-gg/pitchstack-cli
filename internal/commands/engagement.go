package commands

import (
	"context"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
	"github.com/urfave/cli/v3"
)

func newResourceLikeCommand(resourceName string, resourceType clientv1.LikeableResourceType) *cli.Command {
	return newSDKCommand("like", "Like a "+resourceName, []cli.Flag{
		&cli.StringFlag{Name: "id", Usage: resourceName + " ID"},
	}, true, func(cmd *cli.Command, req *clientv1.LikeResourceRequest) error {
		if req.Resource == nil {
			req.Resource = &clientv1.LikeableResourceRef{}
		}
		req.Resource.ResourceType = resourceType
		setStringFlag(cmd, "id", &req.Resource.ResourceID)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.LikeResourceRequest) (any, error) {
		return c.LikeResource(ctx, req)
	})
}

func newResourceUnlikeCommand(resourceName string, resourceType clientv1.LikeableResourceType) *cli.Command {
	return newSDKCommand("unlike", "Unlike a "+resourceName, []cli.Flag{
		&cli.StringFlag{Name: "id", Usage: resourceName + " ID"},
	}, true, func(cmd *cli.Command, req *clientv1.UnlikeResourceRequest) error {
		if req.Resource == nil {
			req.Resource = &clientv1.LikeableResourceRef{}
		}
		req.Resource.ResourceType = resourceType
		setStringFlag(cmd, "id", &req.Resource.ResourceID)
		return nil
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UnlikeResourceRequest) (any, error) {
		return c.UnlikeResource(ctx, req)
	})
}

func newResourceTrendingCommand(resourceName string, resourceType clientv1.TrackableResourceType) *cli.Command {
	return newSDKCommand("trending", "List trending "+resourceName, []cli.Flag{
		&cli.StringFlag{Name: "window", Usage: "Window (24h|7d|30d or raw enum)"},
		&cli.IntFlag{Name: "page-size", Usage: "Page size"},
		&cli.StringFlag{Name: "next-token", Usage: "Next page token"},
	}, true, func(cmd *cli.Command, req *clientv1.ListTrendingResourcesRequest) error {
		req.ResourceType = resourceType
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
	})
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

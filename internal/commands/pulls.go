package commands

import (
	"context"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
	"github.com/urfave/cli/v3"
)

func newPullsCommand() *cli.Command {
	return &cli.Command{
		Name:  "pulls",
		Usage: "Pack pull helpers",
		Commands: []*cli.Command{
			newSDKCommand("create", "Create a pull", []cli.Flag{
				&cli.StringFlag{Name: "sealed-product-id", Usage: "Sealed product ID"},
				&cli.IntFlag{Name: "units-opened", Usage: "Units opened"},
				&cli.StringFlag{Name: "pulled-at", Usage: "Pulled time (RFC3339)"},
				&cli.StringFlag{Name: "pull-id", Usage: "Pull ID"},
			}, true, func(cmd *cli.Command, req *clientv1.CreatePullRequest) error {
				setStringFlag(cmd, "sealed-product-id", &req.SealedProductID)
				if cmd.IsSet("units-opened") {
					req.UnitsOpened = int32(cmd.Int("units-opened"))
				}
				setStringFlag(cmd, "pull-id", &req.PullID)
				return setTimeFlag(cmd, "pulled-at", &req.PulledAt)
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.CreatePullRequest) (any, error) {
				return c.CreatePull(ctx, req)
			}),
			newSDKCommand("get", "Get a pull", []cli.Flag{&cli.StringFlag{Name: "id", Usage: "Pull ID"}}, true, func(cmd *cli.Command, req *clientv1.GetPullRequest) error {
				setStringFlag(cmd, "id", &req.PullID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetPullRequest) (any, error) {
				return c.GetPull(ctx, req)
			}),
			newSDKCommand("list", "List pulls", append(pageFlags(), &cli.StringFlag{Name: "set-id", Usage: "Set ID"}, &cli.StringFlag{Name: "scope", Usage: "Scope"}), true, func(cmd *cli.Command, req *clientv1.ListPullsRequest) error {
				setStringFlag(cmd, "set-id", &req.SetID)
				if cmd.IsSet("scope") {
					req.Scope = parsePullScope(cmd.String("scope"))
				}
				setPageFlags(cmd, &req.PageSize, &req.NextToken)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListPullsRequest) (any, error) {
				return c.ListPulls(ctx, req)
			}),
			newSDKCommand("delete", "Delete a pull", []cli.Flag{&cli.StringFlag{Name: "id", Usage: "Pull ID"}}, true, func(cmd *cli.Command, req *clientv1.DeletePullRequest) error {
				setStringFlag(cmd, "id", &req.PullID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.DeletePullRequest) (any, error) {
				return c.DeletePull(ctx, req)
			}),
			newSDKCommand("stats", "Get pull stats", []cli.Flag{&cli.StringFlag{Name: "set-id", Usage: "Set ID"}, &cli.StringFlag{Name: "scope", Usage: "Scope"}}, true, func(cmd *cli.Command, req *clientv1.GetPullStatsRequest) error {
				setStringFlag(cmd, "set-id", &req.SetID)
				if cmd.IsSet("scope") {
					req.Scope = parsePullScope(cmd.String("scope"))
				}
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetPullStatsRequest) (any, error) {
				return c.GetPullStats(ctx, req)
			}),
		},
	}
}

func parsePullScope(v string) clientv1.PullScope {
	switch v {
	case "pack":
		return clientv1.PullScopePack
	case "box":
		return clientv1.PullScopeBox
	case "case":
		return clientv1.PullScopeCase
	default:
		return clientv1.PullScope(v)
	}
}

package commands

import (
	"context"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
	"github.com/urfave/cli/v3"
)

func newSocialCommand() *cli.Command {
	return &cli.Command{
		Name:  "social",
		Usage: "Manage social connections",
		Commands: []*cli.Command{
			newSDKCommand("follow", "Follow a user", []cli.Flag{&cli.StringFlag{Name: "target-user-id", Usage: "Target user ID"}}, true, func(cmd *cli.Command, req *clientv1.FollowUserRequest) error {
				setStringFlag(cmd, "target-user-id", &req.TargetUserID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.FollowUserRequest) (any, error) {
				return c.FollowUser(ctx, req)
			}),
			newSDKCommand("unfollow", "Unfollow a user", []cli.Flag{&cli.StringFlag{Name: "target-user-id", Usage: "Target user ID"}}, true, func(cmd *cli.Command, req *clientv1.UnfollowUserRequest) error {
				setStringFlag(cmd, "target-user-id", &req.TargetUserID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UnfollowUserRequest) (any, error) {
				return c.UnfollowUser(ctx, req)
			}),
			newSDKCommand("followers", "List followers", append(pageFlags(), &cli.StringFlag{Name: "user-id", Usage: "User ID"}), true, func(cmd *cli.Command, req *clientv1.ListFollowersRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				setPageFlags(cmd, &req.PageSize, &req.NextToken)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListFollowersRequest) (any, error) {
				return c.ListFollowers(ctx, req)
			}),
			newSDKCommand("following", "List followed users", append(pageFlags(), &cli.StringFlag{Name: "user-id", Usage: "User ID"}), true, func(cmd *cli.Command, req *clientv1.ListFollowingRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				setPageFlags(cmd, &req.PageSize, &req.NextToken)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListFollowingRequest) (any, error) {
				return c.ListFollowing(ctx, req)
			}),
			newSDKCommand("stats", "Get follow stats", []cli.Flag{&cli.StringFlag{Name: "user-id", Usage: "User ID"}}, true, func(cmd *cli.Command, req *clientv1.GetFollowStatsRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetFollowStatsRequest) (any, error) {
				return c.GetFollowStats(ctx, req)
			}),
			newSDKCommand("is-following", "Check follow relationship", []cli.Flag{
				&cli.StringFlag{Name: "follower-id", Usage: "Follower user ID"},
				&cli.StringFlag{Name: "followee-id", Usage: "Followee user ID"},
			}, true, func(cmd *cli.Command, req *clientv1.IsFollowingRequest) error {
				setStringFlag(cmd, "follower-id", &req.FollowerID)
				setStringFlag(cmd, "followee-id", &req.FolloweeID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.IsFollowingRequest) (any, error) {
				return c.IsFollowing(ctx, req)
			}),
		},
	}
}

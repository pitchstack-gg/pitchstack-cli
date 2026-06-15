package commands

import (
	"context"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
	"github.com/urfave/cli/v3"
)

func newNewsCommand() *cli.Command {
	return &cli.Command{
		Name:  "news",
		Usage: "News article helpers",
		Commands: []*cli.Command{
			newSDKCommand("recommended", "List recommended articles", append(pageFlags(), &cli.StringFlag{Name: "locale", Usage: "Locale"}, &cli.StringFlag{Name: "platform", Usage: "Platform"}), true, func(cmd *cli.Command, req *clientv1.ListRecommendedArticlesRequest) error {
				setPageFlags(cmd, &req.PageSize, &req.NextToken)
				setStringFlag(cmd, "locale", &req.Locale)
				setStringFlag(cmd, "platform", &req.Platform)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListRecommendedArticlesRequest) (any, error) {
				return c.ListRecommendedArticles(ctx, req)
			}),
			newSDKNoRequestCommand("sources", "List news sources", true, func(ctx context.Context, c *clientv1.Client) (any, error) {
				return c.ListNewsSources(ctx)
			}),
			newSDKCommand("get", "Get an article", []cli.Flag{&cli.StringFlag{Name: "id", Usage: "Article ID"}}, true, func(cmd *cli.Command, req *clientv1.GetArticleRequest) error {
				setStringFlag(cmd, "id", &req.ArticleID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetArticleRequest) (any, error) {
				return c.GetArticle(ctx, req)
			}),
			newNewsTrackCommand("impression"),
			newNewsTrackCommand("click"),
		},
	}
}

func newNewsTrackCommand(kind string) *cli.Command {
	flags := []cli.Flag{
		&cli.StringFlag{Name: "article-id", Usage: "Article ID"},
		&cli.StringFlag{Name: "session-id", Usage: "Session ID"},
		&cli.StringFlag{Name: "client-event-id", Usage: "Client event ID"},
		&cli.StringFlag{Name: "occurred-at", Usage: "Occurrence time (RFC3339)"},
	}
	if kind == "click" {
		flags = append(flags, &cli.StringFlag{Name: "destination-url", Usage: "Destination URL"})
		return newSDKCommand("track-click", "Track an article click", flags, true, func(cmd *cli.Command, req *clientv1.TrackArticleClickRequest) error {
			setStringFlag(cmd, "article-id", &req.ArticleID)
			setStringFlag(cmd, "session-id", &req.SessionID)
			setStringFlag(cmd, "client-event-id", &req.ClientEventID)
			setStringFlag(cmd, "destination-url", &req.DestinationURL)
			return setTimeFlag(cmd, "occurred-at", &req.OccurredAt)
		}, func(ctx context.Context, c *clientv1.Client, req *clientv1.TrackArticleClickRequest) (any, error) {
			return c.TrackArticleClick(ctx, req)
		})
	}
	return newSDKCommand("track-impression", "Track an article impression", flags, true, func(cmd *cli.Command, req *clientv1.TrackArticleImpressionRequest) error {
		setStringFlag(cmd, "article-id", &req.ArticleID)
		setStringFlag(cmd, "session-id", &req.SessionID)
		setStringFlag(cmd, "client-event-id", &req.ClientEventID)
		return setTimeFlag(cmd, "occurred-at", &req.OccurredAt)
	}, func(ctx context.Context, c *clientv1.Client, req *clientv1.TrackArticleImpressionRequest) (any, error) {
		return c.TrackArticleImpression(ctx, req)
	})
}

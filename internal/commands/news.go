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
			newNewsAdminCommand(),
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

func newNewsAdminCommand() *cli.Command {
	return &cli.Command{
		Name:  "admin",
		Usage: "News administration helpers",
		Commands: []*cli.Command{
			newSDKCommand("disable-source", "Disable a source", []cli.Flag{&cli.StringFlag{Name: "source-id", Usage: "Source ID"}}, true, func(cmd *cli.Command, req *clientv1.DisableSourceRequest) error {
				setStringFlag(cmd, "source-id", &req.SourceID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.DisableSourceRequest) (any, error) {
				return c.DisableSource(ctx, req)
			}),
			newSDKCommand("enable-source", "Enable a source", []cli.Flag{&cli.StringFlag{Name: "source-id", Usage: "Source ID"}}, true, func(cmd *cli.Command, req *clientv1.EnableSourceRequest) error {
				setStringFlag(cmd, "source-id", &req.SourceID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.EnableSourceRequest) (any, error) {
				return c.EnableSource(ctx, req)
			}),
			newSDKCommand("run-ingestion", "Run source ingestion", []cli.Flag{&cli.StringFlag{Name: "source-id", Usage: "Source ID"}, &cli.BoolFlag{Name: "force-full-refresh", Usage: "Force full refresh"}}, true, func(cmd *cli.Command, req *clientv1.RunSourceIngestionRequest) error {
				setStringFlag(cmd, "source-id", &req.SourceID)
				setBoolFlag(cmd, "force-full-refresh", &req.ForceFullRefresh)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.RunSourceIngestionRequest) (any, error) {
				return c.RunSourceIngestion(ctx, req)
			}),
			newSDKCommand("create-manual", "Create a manual article", manualArticleFlags(false), true, applyCreateManualArticleFlags, func(ctx context.Context, c *clientv1.Client, req *clientv1.CreateManualArticleRequest) (any, error) {
				return c.CreateManualArticle(ctx, req)
			}),
			newSDKCommand("update-manual", "Update a manual article", manualArticleFlags(true), true, applyUpdateManualArticleFlags, func(ctx context.Context, c *clientv1.Client, req *clientv1.UpdateManualArticleRequest) (any, error) {
				return c.UpdateManualArticle(ctx, req)
			}),
			newSDKCommand("publish", "Publish a manual article", publishArticleFlags(), true, func(cmd *cli.Command, req *clientv1.PublishManualArticleRequest) error {
				setStringFlag(cmd, "article-id", &req.ArticleID)
				if err := setTimeFlag(cmd, "publish-at", &req.PublishAt); err != nil {
					return err
				}
				return setTimeFlag(cmd, "expire-at", &req.ExpireAt)
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.PublishManualArticleRequest) (any, error) {
				return c.PublishManualArticle(ctx, req)
			}),
			newSDKCommand("unpublish", "Unpublish a manual article", []cli.Flag{&cli.StringFlag{Name: "article-id", Usage: "Article ID"}}, true, func(cmd *cli.Command, req *clientv1.UnpublishManualArticleRequest) error {
				setStringFlag(cmd, "article-id", &req.ArticleID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UnpublishManualArticleRequest) (any, error) {
				return c.UnpublishManualArticle(ctx, req)
			}),
			newSDKCommand("override", "Set article override", []cli.Flag{
				&cli.StringFlag{Name: "article-id", Usage: "Article ID"},
				&cli.IntFlag{Name: "pin-rank", Usage: "Pin rank"},
				&cli.Float64Flag{Name: "manual-boost", Usage: "Manual boost"},
				&cli.StringFlag{Name: "starts-at", Usage: "Start time (RFC3339)"},
				&cli.StringFlag{Name: "ends-at", Usage: "End time (RFC3339)"},
			}, true, func(cmd *cli.Command, req *clientv1.SetArticleOverrideRequest) error {
				setStringFlag(cmd, "article-id", &req.ArticleID)
				setInt32Flag(cmd, "pin-rank", &req.PinRank)
				setFloat64Flag(cmd, "manual-boost", &req.ManualBoost)
				if err := setTimeFlag(cmd, "starts-at", &req.StartsAt); err != nil {
					return err
				}
				return setTimeFlag(cmd, "ends-at", &req.EndsAt)
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.SetArticleOverrideRequest) (any, error) {
				return c.SetArticleOverride(ctx, req)
			}),
		},
	}
}

func manualArticleFlags(withID bool) []cli.Flag {
	flags := []cli.Flag{}
	if withID {
		flags = append(flags, &cli.StringFlag{Name: "article-id", Usage: "Article ID"})
	}
	return append(flags,
		&cli.StringFlag{Name: "title", Usage: "Title"},
		&cli.StringFlag{Name: "summary", Usage: "Summary"},
		&cli.StringFlag{Name: "canonical-url", Usage: "Canonical URL"},
		&cli.StringFlag{Name: "image-url", Usage: "Image URL"},
		&cli.StringFlag{Name: "author", Usage: "Author"},
		&cli.StringFlag{Name: "language", Usage: "Language"},
		&cli.StringFlag{Name: "body-markdown", Usage: "Body markdown"},
		&cli.StringFlag{Name: "body-html", Usage: "Body HTML"},
		&cli.StringFlag{Name: "publish-at", Usage: "Publish time (RFC3339)"},
		&cli.StringFlag{Name: "expire-at", Usage: "Expire time (RFC3339)"},
		&cli.StringFlag{Name: "initial-status", Usage: "Initial status"},
	)
}

func publishArticleFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "article-id", Usage: "Article ID"},
		&cli.StringFlag{Name: "publish-at", Usage: "Publish time (RFC3339)"},
		&cli.StringFlag{Name: "expire-at", Usage: "Expire time (RFC3339)"},
	}
}

func applyCreateManualArticleFlags(cmd *cli.Command, req *clientv1.CreateManualArticleRequest) error {
	setStringFlag(cmd, "title", &req.Title)
	setStringFlag(cmd, "summary", &req.Summary)
	setStringFlag(cmd, "canonical-url", &req.CanonicalURL)
	setStringFlag(cmd, "image-url", &req.ImageURL)
	setStringFlag(cmd, "author", &req.Author)
	setStringFlag(cmd, "language", &req.Language)
	setStringFlag(cmd, "body-markdown", &req.BodyMarkdown)
	setStringFlag(cmd, "body-html", &req.BodyHTML)
	if cmd.IsSet("initial-status") {
		v := clientv1.NewsArticleStatus(cmd.String("initial-status"))
		req.InitialStatus = &v
	}
	if err := setTimeFlag(cmd, "publish-at", &req.PublishAt); err != nil {
		return err
	}
	return setTimeFlag(cmd, "expire-at", &req.ExpireAt)
}

func applyUpdateManualArticleFlags(cmd *cli.Command, req *clientv1.UpdateManualArticleRequest) error {
	setStringFlag(cmd, "article-id", &req.ArticleID)
	setStringPtrFlag(cmd, "title", &req.Title)
	setStringPtrFlag(cmd, "summary", &req.Summary)
	setStringPtrFlag(cmd, "canonical-url", &req.CanonicalURL)
	setStringPtrFlag(cmd, "image-url", &req.ImageURL)
	setStringPtrFlag(cmd, "author", &req.Author)
	setStringPtrFlag(cmd, "language", &req.Language)
	setStringPtrFlag(cmd, "body-markdown", &req.BodyMarkdown)
	setStringPtrFlag(cmd, "body-html", &req.BodyHTML)
	if err := setTimeFlag(cmd, "publish-at", &req.PublishAt); err != nil {
		return err
	}
	return setTimeFlag(cmd, "expire-at", &req.ExpireAt)
}

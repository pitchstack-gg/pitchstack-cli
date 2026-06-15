package commands

import (
	"context"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
	"github.com/urfave/cli/v3"
)

func newNotificationsCommand() *cli.Command {
	return &cli.Command{
		Name:  "notifications",
		Usage: "Notification helpers",
		Commands: []*cli.Command{
			newSDKCommand("create-message", "Create a notification message", []cli.Flag{
				&cli.StringFlag{Name: "target-user-id", Usage: "Target user ID"},
				&cli.StringFlag{Name: "source", Usage: "Source"},
				&cli.StringFlag{Name: "idempotency-key", Usage: "Idempotency key"},
				&cli.StringFlag{Name: "category", Usage: "Category"},
				&cli.StringFlag{Name: "severity", Usage: "Severity"},
				&cli.StringFlag{Name: "title", Usage: "Title"},
				&cli.StringFlag{Name: "body-markdown", Usage: "Body markdown"},
				&cli.StringFlag{Name: "expires-at", Usage: "Expiration time (RFC3339)"},
			}, true, func(cmd *cli.Command, req *clientv1.CreateMessageRequest) error {
				setStringFlag(cmd, "target-user-id", &req.TargetUserID)
				setStringFlag(cmd, "source", &req.Source)
				setStringFlag(cmd, "idempotency-key", &req.IdempotencyKey)
				setStringFlag(cmd, "category", &req.Category)
				setStringFlag(cmd, "severity", &req.Severity)
				setStringFlag(cmd, "title", &req.Title)
				setStringPtrFlag(cmd, "body-markdown", &req.BodyMarkdown)
				return setTimeFlag(cmd, "expires-at", &req.ExpiresAt)
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.CreateMessageRequest) (any, error) {
				return c.CreateMessage(ctx, req)
			}),
			newPushDevicesCommand(),
			newNotificationPreferencesCommand(),
			newNotificationTopicsCommand(),
			newInboxCommand(),
		},
	}
}

func newPushDevicesCommand() *cli.Command {
	return &cli.Command{
		Name:  "devices",
		Usage: "Push device helpers",
		Commands: []*cli.Command{
			newSDKCommand("register", "Register a push device", []cli.Flag{
				&cli.StringFlag{Name: "device-id", Usage: "Device ID"},
				&cli.StringFlag{Name: "platform", Usage: "Platform"},
				&cli.StringFlag{Name: "expo-push-token", Usage: "Expo push token"},
				&cli.StringFlag{Name: "app-version", Usage: "App version"},
			}, true, func(cmd *cli.Command, req *clientv1.RegisterPushDeviceRequest) error {
				setStringFlag(cmd, "device-id", &req.DeviceID)
				setStringFlag(cmd, "platform", &req.Platform)
				setStringFlag(cmd, "expo-push-token", &req.ExpoPushToken)
				setStringFlag(cmd, "app-version", &req.AppVersion)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.RegisterPushDeviceRequest) (any, error) {
				return c.RegisterPushDevice(ctx, req)
			}),
			newSDKCommand("unregister", "Unregister a push device", []cli.Flag{&cli.StringFlag{Name: "device-id", Usage: "Device ID"}}, true, func(cmd *cli.Command, req *clientv1.UnregisterPushDeviceRequest) error {
				setStringFlag(cmd, "device-id", &req.DeviceID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UnregisterPushDeviceRequest) (any, error) {
				return c.UnregisterPushDevice(ctx, req)
			}),
		},
	}
}

func newNotificationPreferencesCommand() *cli.Command {
	return &cli.Command{
		Name:  "preferences",
		Usage: "Notification preferences",
		Commands: []*cli.Command{
			newSDKNoRequestCommand("get", "Get notification preferences", true, func(ctx context.Context, c *clientv1.Client) (any, error) {
				return c.GetNotificationPreferences(ctx)
			}),
			newSDKCommand("update", "Update notification preferences from JSON", nil, true, nil, func(ctx context.Context, c *clientv1.Client, req *clientv1.UpdateNotificationPreferencesRequest) (any, error) {
				return c.UpdateNotificationPreferences(ctx, req)
			}),
		},
	}
}

func newNotificationTopicsCommand() *cli.Command {
	return &cli.Command{
		Name:  "topics",
		Usage: "Notification topic subscriptions",
		Commands: []*cli.Command{
			newSDKCommand("get", "Get topic subscriptions", notificationTopicFlags(), true, func(cmd *cli.Command, req *clientv1.GetNotificationTopicSubscriptionsRequest) error {
				setStringFlag(cmd, "category", &req.Category)
				setStringFlag(cmd, "topic-type", &req.TopicType)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetNotificationTopicSubscriptionsRequest) (any, error) {
				return c.GetNotificationTopicSubscriptions(ctx, req)
			}),
			newSDKCommand("update", "Update topic subscriptions", append(notificationTopicFlags(), repeatedIDsFlag("topic-id", "Topic ID (repeatable or comma-separated)")), true, func(cmd *cli.Command, req *clientv1.UpdateNotificationTopicSubscriptionsRequest) error {
				setStringFlag(cmd, "category", &req.Category)
				setStringFlag(cmd, "topic-type", &req.TopicType)
				if cmd.IsSet("topic-id") {
					req.TopicIDs = splitCSV(cmd.StringSlice("topic-id"))
				}
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UpdateNotificationTopicSubscriptionsRequest) (any, error) {
				return c.UpdateNotificationTopicSubscriptions(ctx, req)
			}),
		},
	}
}

func notificationTopicFlags() []cli.Flag {
	return []cli.Flag{
		&cli.StringFlag{Name: "category", Usage: "Category"},
		&cli.StringFlag{Name: "topic-type", Usage: "Topic type"},
	}
}

func newInboxCommand() *cli.Command {
	return &cli.Command{
		Name:  "inbox",
		Usage: "Inbox helpers",
		Commands: []*cli.Command{
			newSDKCommand("list", "List inbox notifications", append(pageFlags(),
				&cli.BoolFlag{Name: "unread-only", Usage: "Unread only"},
				&cli.BoolFlag{Name: "include-archived", Usage: "Include archived"},
				&cli.BoolFlag{Name: "include-expired", Usage: "Include expired"},
				repeatedIDsFlag("category", "Category (repeatable or comma-separated)"),
			), true, func(cmd *cli.Command, req *clientv1.ListInboxRequest) error {
				setBoolFlag(cmd, "unread-only", &req.UnreadOnly)
				setBoolFlag(cmd, "include-archived", &req.IncludeArchived)
				setBoolFlag(cmd, "include-expired", &req.IncludeExpired)
				if cmd.IsSet("category") {
					req.Categories = splitCSV(cmd.StringSlice("category"))
				}
				setPageFlags(cmd, &req.PageSize, &req.NextToken)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListInboxRequest) (any, error) {
				return c.ListInbox(ctx, req)
			}),
			newSDKCommand("get", "Get a message", []cli.Flag{&cli.StringFlag{Name: "message-id", Usage: "Message ID"}}, true, func(cmd *cli.Command, req *clientv1.GetMessageRequest) error {
				setStringFlag(cmd, "message-id", &req.MessageID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetMessageRequest) (any, error) {
				return c.GetMessage(ctx, req)
			}),
			newSDKCommand("delete", "Delete a message", []cli.Flag{&cli.StringFlag{Name: "message-id", Usage: "Message ID"}, yesFlag()}, true, func(cmd *cli.Command, req *clientv1.DeleteMessageRequest) error {
				setStringFlag(cmd, "message-id", &req.MessageID)
				return confirmAction(cmd, "Delete", "message", req.MessageID)
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.DeleteMessageRequest) (any, error) {
				return c.DeleteMessage(ctx, req)
			}),
			newSDKCommand("mark-read", "Mark a message read", []cli.Flag{&cli.StringFlag{Name: "message-id", Usage: "Message ID"}}, true, func(cmd *cli.Command, req *clientv1.MarkReadRequest) error {
				setStringFlag(cmd, "message-id", &req.MessageID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.MarkReadRequest) (any, error) {
				return c.MarkRead(ctx, req)
			}),
			newSDKCommand("mark-all-read", "Mark all unread active messages read", nil, true, nil, func(ctx context.Context, c *clientv1.Client, req *clientv1.MarkAllReadRequest) (any, error) {
				return c.MarkAllRead(ctx, req)
			}),
			newSDKCommand("archive", "Archive a message", []cli.Flag{&cli.StringFlag{Name: "message-id", Usage: "Message ID"}}, true, func(cmd *cli.Command, req *clientv1.ArchiveMessageRequest) error {
				setStringFlag(cmd, "message-id", &req.MessageID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ArchiveMessageRequest) (any, error) {
				return c.ArchiveMessage(ctx, req)
			}),
		},
	}
}

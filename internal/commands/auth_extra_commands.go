package commands

import (
	"context"
	"strings"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
	"github.com/urfave/cli/v3"
)

func newAuthMeCommand() *cli.Command {
	return &cli.Command{
		Name:  "me",
		Usage: "Show active user details",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return withSDKClient(ctx, cmd, true, func(c *clientv1.Client) (any, error) {
				return c.Me(ctx)
			})
		},
	}
}

func newAuthAPIKeysCommand() *cli.Command {
	return &cli.Command{
		Name:  "api-keys",
		Usage: "Manage API keys",
		Commands: []*cli.Command{
			newSDKCommand("list", "List API keys", pageFlags(), true, func(cmd *cli.Command, req *clientv1.ListAPIKeysRequest) error {
				setPageFlags(cmd, &req.PageSize, &req.NextToken)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListAPIKeysRequest) (any, error) {
				return c.ListAPIKeys(ctx, req)
			}),
			newSDKCommand("create", "Create an API key", []cli.Flag{
				&cli.StringFlag{Name: "name", Usage: "API key name"},
				repeatedIDsFlag("scope", "Scope (repeatable or comma-separated)"),
				&cli.IntFlag{Name: "rate-limit-per-minute", Usage: "Rate limit per minute"},
				&cli.StringFlag{Name: "expires-at", Usage: "Expiration time (RFC3339)"},
			}, true, func(cmd *cli.Command, req *clientv1.CreateAPIKeyRequest) error {
				setStringFlag(cmd, "name", &req.Name)
				if cmd.IsSet("scope") {
					req.Scopes = splitCSV(cmd.StringSlice("scope"))
				}
				setInt32Flag(cmd, "rate-limit-per-minute", &req.RateLimitPerMinute)
				return setTimeFlag(cmd, "expires-at", &req.ExpiresAt)
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.CreateAPIKeyRequest) (any, error) {
				return c.CreateAPIKey(ctx, req)
			}),
			newSDKCommand("revoke", "Revoke an API key", []cli.Flag{
				&cli.StringFlag{Name: "id", Usage: "API key ID"},
				yesFlag(),
			}, true, func(cmd *cli.Command, req *clientv1.RevokeAPIKeyRequest) error {
				setStringFlag(cmd, "id", &req.APIKeyID)
				return confirmAction(cmd, "Revoke", "API key", req.APIKeyID)
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.RevokeAPIKeyRequest) (any, error) {
				return c.RevokeAPIKey(ctx, req)
			}),
		},
	}
}

func newAuthPasswordCommand() *cli.Command {
	return &cli.Command{
		Name:  "password",
		Usage: "Manage passwords",
		Commands: []*cli.Command{
			newAuthPasswordChangeCommand(),
			newSDKCommand("request-reset", "Request password reset", []cli.Flag{
				&cli.StringFlag{Name: "email", Usage: "Account email"},
			}, false, func(cmd *cli.Command, req *clientv1.RequestPasswordResetRequest) error {
				setStringFlag(cmd, "email", &req.Email)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.RequestPasswordResetRequest) (any, error) {
				return c.RequestPasswordReset(ctx, req)
			}),
			newAuthPasswordResetCommand(),
		},
	}
}

func newAuthPasswordChangeCommand() *cli.Command {
	return &cli.Command{
		Name:  "change",
		Usage: "Change password",
		Flags: []cli.Flag{
			requestFileFlag(),
			&cli.StringFlag{Name: "user-id", Usage: "User ID (defaults to current session)"},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			var req clientv1.ChangePasswordRequest
			if err := readRequestFile(cmd, &req); err != nil {
				return err
			}
			if userID := strings.TrimSpace(cmd.String("user-id")); userID != "" {
				req.UserID = userID
			}
			st, err := getState(ctx)
			if err != nil {
				return err
			}
			if strings.TrimSpace(req.UserID) == "" {
				req.UserID, err = currentSessionUserID(ctx, st)
				if err != nil {
					return err
				}
			}
			if strings.TrimSpace(req.CurrentPassword) == "" {
				req.CurrentPassword, err = readSecret(cmd, "Current password: ")
				if err != nil {
					return err
				}
			}
			if strings.TrimSpace(req.NewPassword) == "" {
				req.NewPassword, err = readSecret(cmd, "New password: ")
				if err != nil {
					return err
				}
			}
			if strings.TrimSpace(req.CurrentPassword) == "" || strings.TrimSpace(req.NewPassword) == "" {
				return cli.Exit("current password and new password are required", 2)
			}
			return withSDKClient(ctx, cmd, true, func(c *clientv1.Client) (any, error) {
				return c.ChangePassword(ctx, &req)
			})
		},
	}
}

func newAuthPasswordResetCommand() *cli.Command {
	return &cli.Command{
		Name:  "reset",
		Usage: "Complete password reset",
		Flags: []cli.Flag{requestFileFlag()},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			var req clientv1.ResetPasswordRequest
			if err := readRequestFile(cmd, &req); err != nil {
				return err
			}
			var err error
			if strings.TrimSpace(req.ResetToken) == "" {
				req.ResetToken, err = readSecret(cmd, "Reset token: ")
				if err != nil {
					return err
				}
			}
			if strings.TrimSpace(req.NewPassword) == "" {
				req.NewPassword, err = readSecret(cmd, "New password: ")
				if err != nil {
					return err
				}
			}
			if strings.TrimSpace(req.ResetToken) == "" || strings.TrimSpace(req.NewPassword) == "" {
				return cli.Exit("reset token and new password are required", 2)
			}
			return withSDKClient(ctx, cmd, false, func(c *clientv1.Client) (any, error) {
				return c.ResetPassword(ctx, &req)
			})
		},
	}
}

func currentSessionUserID(ctx context.Context, st *state) (string, error) {
	if st == nil || st.Sessions == nil {
		return "", nil
	}
	sess, err := st.Sessions.Load()
	if err != nil {
		return "", err
	}
	if sess != nil && strings.TrimSpace(sess.UserID) != "" {
		return strings.TrimSpace(sess.UserID), nil
	}
	me, err := st.Service.Me(ctx)
	if err != nil {
		return "", err
	}
	if me == nil || me.User == nil {
		return "", nil
	}
	return strings.TrimSpace(me.User.UserID), nil
}

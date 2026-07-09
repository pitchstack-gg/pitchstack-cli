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

func newAuthEmailCommand() *cli.Command {
	return &cli.Command{
		Name:  "email",
		Usage: "Manage account email",
		Commands: []*cli.Command{
			newSDKCommand("change-status", "Get pending email change", nil, true, nil, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetEmailChangeStatusRequest) (any, error) {
				return c.GetEmailChangeStatus(ctx, req)
			}),
			newSDKCommand("request-change", "Request email change", []cli.Flag{
				&cli.StringFlag{Name: "new-email", Usage: "New account email"},
			}, true, func(cmd *cli.Command, req *clientv1.RequestEmailChangeRequest) error {
				setStringFlag(cmd, "new-email", &req.NewEmail)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.RequestEmailChangeRequest) (any, error) {
				return c.RequestEmailChange(ctx, req)
			}),
			newSDKCommand("resend-change", "Resend email-change confirmation", nil, true, nil, func(ctx context.Context, c *clientv1.Client, req *clientv1.ResendEmailChangeConfirmationRequest) (any, error) {
				return c.ResendEmailChangeConfirmation(ctx, req)
			}),
			newSDKCommand("cancel-change", "Cancel pending email change", nil, true, nil, func(ctx context.Context, c *clientv1.Client, req *clientv1.CancelEmailChangeRequest) (any, error) {
				return c.CancelEmailChange(ctx, req)
			}),
			newSDKCommand("confirm-change", "Confirm email change", []cli.Flag{
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
				&cli.StringFlag{Name: "change-token", Usage: "Email change token"},
			}, false, func(cmd *cli.Command, req *clientv1.ConfirmEmailChangeRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				setStringFlag(cmd, "change-token", &req.ChangeToken)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ConfirmEmailChangeRequest) (any, error) {
				return c.ConfirmEmailChange(ctx, req)
			}),
		},
	}
}

func newAuthPasskeysCommand() *cli.Command {
	return &cli.Command{
		Name:  "passkeys",
		Usage: "Manage passkeys",
		Commands: []*cli.Command{
			newSDKCommand("registration-initiate", "Initiate passkey registration", []cli.Flag{
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
				&cli.StringFlag{Name: "display-name", Usage: "Passkey display name"},
			}, true, func(cmd *cli.Command, req *clientv1.InitiatePasskeyRegistrationRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				setStringFlag(cmd, "display-name", &req.DisplayName)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.InitiatePasskeyRegistrationRequest) (any, error) {
				return c.InitiatePasskeyRegistration(ctx, req)
			}),
			newSDKCommand("registration-complete", "Complete passkey registration", []cli.Flag{
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
				&cli.StringFlag{Name: "session-id", Usage: "Registration session ID"},
				&cli.StringFlag{Name: "client-data-json", Usage: "Client data JSON"},
				&cli.StringFlag{Name: "attestation-object", Usage: "Attestation object"},
			}, true, func(cmd *cli.Command, req *clientv1.CompletePasskeyRegistrationRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				setStringFlag(cmd, "session-id", &req.SessionID)
				setStringFlag(cmd, "client-data-json", &req.ClientDataJSON)
				setStringFlag(cmd, "attestation-object", &req.AttestationObject)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.CompletePasskeyRegistrationRequest) (any, error) {
				return c.CompletePasskeyRegistration(ctx, req)
			}),
			newSDKCommand("signup-initiate", "Initiate passkey signup", []cli.Flag{
				&cli.StringFlag{Name: "email", Usage: "Account email"},
				&cli.StringFlag{Name: "display-name", Usage: "Passkey display name"},
			}, false, func(cmd *cli.Command, req *clientv1.InitiatePasskeySignupRequest) error {
				setStringFlag(cmd, "email", &req.Email)
				setStringFlag(cmd, "display-name", &req.DisplayName)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.InitiatePasskeySignupRequest) (any, error) {
				return c.InitiatePasskeySignup(ctx, req)
			}),
			newSDKCommand("signup-complete", "Complete passkey signup", []cli.Flag{
				&cli.StringFlag{Name: "session-id", Usage: "Signup session ID"},
				&cli.StringFlag{Name: "client-data-json", Usage: "Client data JSON"},
				&cli.StringFlag{Name: "attestation-object", Usage: "Attestation object"},
			}, false, func(cmd *cli.Command, req *clientv1.CompletePasskeySignupRequest) error {
				setStringFlag(cmd, "session-id", &req.SessionID)
				setStringFlag(cmd, "client-data-json", &req.ClientDataJSON)
				setStringFlag(cmd, "attestation-object", &req.AttestationObject)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.CompletePasskeySignupRequest) (any, error) {
				return c.CompletePasskeySignup(ctx, req)
			}),
			newSDKCommand("authentication-initiate", "Initiate passkey authentication", []cli.Flag{
				&cli.StringFlag{Name: "email", Usage: "Account email"},
			}, false, func(cmd *cli.Command, req *clientv1.InitiatePasskeyAuthenticationRequest) error {
				setStringFlag(cmd, "email", &req.Email)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.InitiatePasskeyAuthenticationRequest) (any, error) {
				return c.InitiatePasskeyAuthentication(ctx, req)
			}),
			newSDKCommand("authentication-complete", "Complete passkey authentication", []cli.Flag{
				&cli.StringFlag{Name: "session-id", Usage: "Authentication session ID"},
				&cli.StringFlag{Name: "credential-id", Usage: "Credential ID"},
				&cli.StringFlag{Name: "client-data-json", Usage: "Client data JSON"},
				&cli.StringFlag{Name: "authenticator-data", Usage: "Authenticator data"},
				&cli.StringFlag{Name: "signature", Usage: "Signature"},
				&cli.StringFlag{Name: "user-handle", Usage: "User handle"},
			}, false, func(cmd *cli.Command, req *clientv1.CompletePasskeyAuthenticationRequest) error {
				setStringFlag(cmd, "session-id", &req.SessionID)
				setStringFlag(cmd, "credential-id", &req.CredentialID)
				setStringFlag(cmd, "client-data-json", &req.ClientDataJSON)
				setStringFlag(cmd, "authenticator-data", &req.AuthenticatorData)
				setStringFlag(cmd, "signature", &req.Signature)
				setStringFlag(cmd, "user-handle", &req.UserHandle)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.CompletePasskeyAuthenticationRequest) (any, error) {
				return c.CompletePasskeyAuthentication(ctx, req)
			}),
			newSDKCommand("list", "List user passkeys", []cli.Flag{
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
			}, true, func(cmd *cli.Command, req *clientv1.ListUserPasskeysRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListUserPasskeysRequest) (any, error) {
				return c.ListUserPasskeys(ctx, req)
			}),
			newSDKCommand("update", "Update passkey", []cli.Flag{
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
				&cli.StringFlag{Name: "credential-id", Usage: "Credential ID"},
				&cli.StringFlag{Name: "display-name", Usage: "Passkey display name"},
			}, true, func(cmd *cli.Command, req *clientv1.UpdatePasskeyRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				setStringFlag(cmd, "credential-id", &req.CredentialID)
				setStringFlag(cmd, "display-name", &req.DisplayName)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UpdatePasskeyRequest) (any, error) {
				return c.UpdatePasskey(ctx, req)
			}),
			newSDKCommand("delete", "Delete passkey", []cli.Flag{
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
				&cli.StringFlag{Name: "credential-id", Usage: "Credential ID"},
				yesFlag(),
			}, true, func(cmd *cli.Command, req *clientv1.DeletePasskeyRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				setStringFlag(cmd, "credential-id", &req.CredentialID)
				return confirmAction(cmd, "Delete", "passkey", req.CredentialID)
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.DeletePasskeyRequest) (any, error) {
				return c.DeletePasskey(ctx, req)
			}),
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

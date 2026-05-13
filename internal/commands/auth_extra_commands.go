package commands

import (
	"context"

	clientv1 "github.com/pitchstack-gg/pitchstack-go/client/v1"
	"github.com/urfave/cli/v3"
)

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
			newSDKCommand("validate", "Validate an API key", []cli.Flag{
				&cli.StringFlag{Name: "api-key", Usage: "Plaintext API key"},
			}, false, func(cmd *cli.Command, req *clientv1.ValidateAPIKeyRequest) error {
				setStringFlag(cmd, "api-key", &req.APIKey)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ValidateAPIKeyRequest) (any, error) {
				return c.ValidateAPIKey(ctx, req)
			}),
			newSDKCommand("revoke", "Revoke an API key", []cli.Flag{
				&cli.StringFlag{Name: "id", Usage: "API key ID"},
			}, true, func(cmd *cli.Command, req *clientv1.RevokeAPIKeyRequest) error {
				setStringFlag(cmd, "id", &req.APIKeyID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.RevokeAPIKeyRequest) (any, error) {
				return c.RevokeAPIKey(ctx, req)
			}),
		},
	}
}

func newAuthPasswordCommand() *cli.Command {
	return &cli.Command{
		Name:  "password",
		Usage: "Password helpers",
		Commands: []*cli.Command{
			newSDKCommand("change", "Change a user's password", []cli.Flag{
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
				&cli.StringFlag{Name: "current-password", Usage: "Current password"},
				&cli.StringFlag{Name: "new-password", Usage: "New password"},
			}, true, func(cmd *cli.Command, req *clientv1.ChangePasswordRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				setStringFlag(cmd, "current-password", &req.CurrentPassword)
				setStringFlag(cmd, "new-password", &req.NewPassword)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ChangePasswordRequest) (any, error) {
				return c.ChangePassword(ctx, req)
			}),
			newSDKCommand("request-reset", "Request a password reset", []cli.Flag{
				&cli.StringFlag{Name: "email", Usage: "Account email"},
			}, false, func(cmd *cli.Command, req *clientv1.RequestPasswordResetRequest) error {
				setStringFlag(cmd, "email", &req.Email)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.RequestPasswordResetRequest) (any, error) {
				return c.RequestPasswordReset(ctx, req)
			}),
			newSDKCommand("reset", "Complete a password reset", []cli.Flag{
				&cli.StringFlag{Name: "reset-token", Usage: "Reset token"},
				&cli.StringFlag{Name: "new-password", Usage: "New password"},
			}, false, func(cmd *cli.Command, req *clientv1.ResetPasswordRequest) error {
				setStringFlag(cmd, "reset-token", &req.ResetToken)
				setStringFlag(cmd, "new-password", &req.NewPassword)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ResetPasswordRequest) (any, error) {
				return c.ResetPassword(ctx, req)
			}),
			newSDKCommand("resend-verification", "Resend a verification email", []cli.Flag{
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
			}, true, func(cmd *cli.Command, req *clientv1.ResendVerificationEmailRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ResendVerificationEmailRequest) (any, error) {
				return c.ResendVerificationEmail(ctx, req)
			}),
			newSDKCommand("verify-email", "Verify an email address", []cli.Flag{
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
				&cli.StringFlag{Name: "verification-token", Usage: "Verification token"},
			}, false, func(cmd *cli.Command, req *clientv1.VerifyEmailRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				setStringFlag(cmd, "verification-token", &req.VerificationToken)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.VerifyEmailRequest) (any, error) {
				return c.VerifyEmail(ctx, req)
			}),
		},
	}
}

func newAuthMethodsCommand() *cli.Command {
	return &cli.Command{
		Name:  "methods",
		Usage: "Manage auth methods",
		Commands: []*cli.Command{
			newSDKCommand("list", "List auth methods", []cli.Flag{&cli.StringFlag{Name: "user-id", Usage: "User ID"}}, true, func(cmd *cli.Command, req *clientv1.ListAuthMethodsRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListAuthMethodsRequest) (any, error) {
				return c.ListAuthMethods(ctx, req)
			}),
			newSDKCommand("remove", "Remove an auth method", []cli.Flag{&cli.StringFlag{Name: "user-id", Usage: "User ID"}, &cli.StringFlag{Name: "method-type", Usage: "Auth method type"}}, true, func(cmd *cli.Command, req *clientv1.RemoveAuthMethodRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				if cmd.IsSet("method-type") {
					req.MethodType = parseAuthMethodType(cmd.String("method-type"))
				}
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.RemoveAuthMethodRequest) (any, error) {
				return c.RemoveAuthMethod(ctx, req)
			}),
			newSDKCommand("preferred", "Set preferred auth method", []cli.Flag{&cli.StringFlag{Name: "user-id", Usage: "User ID"}, &cli.StringFlag{Name: "method-type", Usage: "Auth method type"}}, true, func(cmd *cli.Command, req *clientv1.SetPreferredAuthMethodRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				if cmd.IsSet("method-type") {
					req.MethodType = parseAuthMethodType(cmd.String("method-type"))
				}
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.SetPreferredAuthMethodRequest) (any, error) {
				return c.SetPreferredAuthMethod(ctx, req)
			}),
		},
	}
}

func newAuthOAuthCommand() *cli.Command {
	return &cli.Command{
		Name:  "oauth",
		Usage: "OAuth provider helpers",
		Commands: []*cli.Command{
			newSDKCommand("initiate", "Initiate OAuth", []cli.Flag{
				&cli.StringFlag{Name: "provider", Usage: "Provider"},
				&cli.StringFlag{Name: "redirect-uri", Usage: "Redirect URI"},
				&cli.StringFlag{Name: "prompt", Usage: "OAuth prompt"},
			}, false, func(cmd *cli.Command, req *clientv1.InitiateOAuthRequest) error {
				setStringFlag(cmd, "provider", &req.Provider)
				setStringFlag(cmd, "redirect-uri", &req.RedirectURI)
				setStringFlag(cmd, "prompt", &req.Prompt)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.InitiateOAuthRequest) (any, error) {
				return c.InitiateOAuth(ctx, req)
			}),
			newSDKCommand("complete", "Complete OAuth", []cli.Flag{
				&cli.StringFlag{Name: "provider", Usage: "Provider"},
				&cli.StringFlag{Name: "code", Usage: "OAuth code"},
				&cli.StringFlag{Name: "state", Usage: "OAuth state"},
				&cli.StringFlag{Name: "redirect-uri", Usage: "Redirect URI"},
			}, false, func(cmd *cli.Command, req *clientv1.CompleteOAuthRequest) error {
				setStringFlag(cmd, "provider", &req.Provider)
				setStringFlag(cmd, "code", &req.Code)
				setStringFlag(cmd, "state", &req.State)
				setStringFlag(cmd, "redirect-uri", &req.RedirectURI)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.CompleteOAuthRequest) (any, error) {
				return c.CompleteOAuth(ctx, req)
			}),
			newSDKCommand("link", "Link an OAuth provider", []cli.Flag{
				&cli.StringFlag{Name: "provider", Usage: "Provider"},
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
				&cli.StringFlag{Name: "code", Usage: "OAuth code"},
				&cli.StringFlag{Name: "state", Usage: "OAuth state"},
				&cli.StringFlag{Name: "redirect-uri", Usage: "Redirect URI"},
			}, true, func(cmd *cli.Command, req *clientv1.LinkOAuthProviderRequest) error {
				setStringFlag(cmd, "provider", &req.Provider)
				setStringFlag(cmd, "user-id", &req.UserID)
				setStringFlag(cmd, "code", &req.Code)
				setStringFlag(cmd, "state", &req.State)
				setStringFlag(cmd, "redirect-uri", &req.RedirectURI)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.LinkOAuthProviderRequest) (any, error) {
				return c.LinkOAuthProvider(ctx, req)
			}),
			newSDKCommand("unlink", "Unlink an OAuth provider", []cli.Flag{
				&cli.StringFlag{Name: "provider", Usage: "Provider"},
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
			}, true, func(cmd *cli.Command, req *clientv1.UnlinkOAuthProviderRequest) error {
				setStringFlag(cmd, "provider", &req.Provider)
				setStringFlag(cmd, "user-id", &req.UserID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UnlinkOAuthProviderRequest) (any, error) {
				return c.UnlinkOAuthProvider(ctx, req)
			}),
		},
	}
}

func newAuthTokensCommand() *cli.Command {
	return &cli.Command{
		Name:  "tokens",
		Usage: "Token helpers",
		Commands: []*cli.Command{
			newSDKCommand("validate", "Validate an access token", []cli.Flag{
				&cli.StringFlag{Name: "access-token", Usage: "Access token"},
			}, false, func(cmd *cli.Command, req *clientv1.ValidateTokenRequest) error {
				setStringFlag(cmd, "access-token", &req.AccessToken)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ValidateTokenRequest) (any, error) {
				return c.ValidateToken(ctx, req)
			}),
		},
	}
}

func newAuthUsersCommand() *cli.Command {
	return &cli.Command{
		Name:  "users",
		Usage: "User account helpers",
		Commands: []*cli.Command{
			newSDKCommand("get", "Get a user", []cli.Flag{&cli.StringFlag{Name: "id", Usage: "User ID"}}, true, func(cmd *cli.Command, req *clientv1.GetUserRequest) error {
				setStringFlag(cmd, "id", &req.UserID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.GetUserRequest) (any, error) {
				return c.GetUser(ctx, req)
			}),
			newSDKCommand("update", "Update a user", []cli.Flag{
				&cli.StringFlag{Name: "id", Usage: "User ID"},
				&cli.StringFlag{Name: "email", Usage: "Email"},
				repeatedIDsFlag("role", "Role (repeatable or comma-separated)"),
			}, true, func(cmd *cli.Command, req *clientv1.UpdateUserRequest) error {
				setStringFlag(cmd, "id", &req.UserID)
				setStringPtrFlag(cmd, "email", &req.Email)
				if cmd.IsSet("role") {
					req.Roles = splitCSV(cmd.StringSlice("role"))
				}
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.UpdateUserRequest) (any, error) {
				return c.UpdateUser(ctx, req)
			}),
			newSDKCommand("delete", "Delete a user", []cli.Flag{&cli.StringFlag{Name: "id", Usage: "User ID"}}, true, func(cmd *cli.Command, req *clientv1.DeleteUserRequest) error {
				setStringFlag(cmd, "id", &req.UserID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.DeleteUserRequest) (any, error) {
				return c.DeleteUser(ctx, req)
			}),
		},
	}
}

func newAuthPasskeysCommand() *cli.Command {
	return &cli.Command{
		Name:  "passkeys",
		Usage: "Passkey helpers",
		Commands: []*cli.Command{
			newSDKCommand("initiate-registration", "Initiate passkey registration", []cli.Flag{
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
				&cli.StringFlag{Name: "display-name", Usage: "Display name"},
			}, true, func(cmd *cli.Command, req *clientv1.InitiatePasskeyRegistrationRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				setStringFlag(cmd, "display-name", &req.DisplayName)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.InitiatePasskeyRegistrationRequest) (any, error) {
				return c.InitiatePasskeyRegistration(ctx, req)
			}),
			newSDKCommand("complete-registration", "Complete passkey registration from raw WebAuthn JSON fields", nil, true, nil, func(ctx context.Context, c *clientv1.Client, req *clientv1.CompletePasskeyRegistrationRequest) (any, error) {
				return c.CompletePasskeyRegistration(ctx, req)
			}),
			newSDKCommand("initiate-authentication", "Initiate passkey authentication", []cli.Flag{&cli.StringFlag{Name: "email", Usage: "Account email"}}, false, func(cmd *cli.Command, req *clientv1.InitiatePasskeyAuthenticationRequest) error {
				setStringFlag(cmd, "email", &req.Email)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.InitiatePasskeyAuthenticationRequest) (any, error) {
				return c.InitiatePasskeyAuthentication(ctx, req)
			}),
			newSDKCommand("complete-authentication", "Complete passkey authentication from raw WebAuthn JSON fields", nil, false, nil, func(ctx context.Context, c *clientv1.Client, req *clientv1.CompletePasskeyAuthenticationRequest) (any, error) {
				return c.CompletePasskeyAuthentication(ctx, req)
			}),
			newSDKCommand("list", "List user passkeys", []cli.Flag{&cli.StringFlag{Name: "user-id", Usage: "User ID"}}, true, func(cmd *cli.Command, req *clientv1.ListUserPasskeysRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.ListUserPasskeysRequest) (any, error) {
				return c.ListUserPasskeys(ctx, req)
			}),
			newSDKCommand("delete", "Delete a passkey", []cli.Flag{
				&cli.StringFlag{Name: "user-id", Usage: "User ID"},
				&cli.StringFlag{Name: "credential-id", Usage: "Credential ID"},
			}, true, func(cmd *cli.Command, req *clientv1.DeletePasskeyRequest) error {
				setStringFlag(cmd, "user-id", &req.UserID)
				setStringFlag(cmd, "credential-id", &req.CredentialID)
				return nil
			}, func(ctx context.Context, c *clientv1.Client, req *clientv1.DeletePasskeyRequest) (any, error) {
				return c.DeletePasskey(ctx, req)
			}),
		},
	}
}

func parseAuthMethodType(v string) clientv1.AuthMethodType {
	switch v {
	case "password":
		return clientv1.AuthMethodTypePassword
	case "google":
		return clientv1.AuthMethodTypeOAuthGoogle
	case "discord":
		return clientv1.AuthMethodTypeOAuthDiscord
	case "apple":
		return clientv1.AuthMethodTypeOAuthApple
	case "passkey":
		return clientv1.AuthMethodTypePasskey
	default:
		return clientv1.AuthMethodType(v)
	}
}
